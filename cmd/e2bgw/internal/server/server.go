// Copyright 2026 The xuanji authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package server implements the E2B REST → ateapi Control translation.
package server

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/agent-substrate/substrate/pkg/proto/ateapipb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ControlAPI is the subset of ateapipb.ControlClient the gateway consumes;
// narrowed for testability (satisfied by the generated client).
type ControlAPI interface {
	CreateActor(ctx context.Context, in *ateapipb.CreateActorRequest, opts ...grpc.CallOption) (*ateapipb.Actor, error)
	GetActor(ctx context.Context, in *ateapipb.GetActorRequest, opts ...grpc.CallOption) (*ateapipb.Actor, error)
	DeleteActor(ctx context.Context, in *ateapipb.DeleteActorRequest, opts ...grpc.CallOption) (*ateapipb.Actor, error)
	SuspendActor(ctx context.Context, in *ateapipb.SuspendActorRequest, opts ...grpc.CallOption) (*ateapipb.SuspendActorResponse, error)
	ResumeActor(ctx context.Context, in *ateapipb.ResumeActorRequest, opts ...grpc.CallOption) (*ateapipb.ResumeActorResponse, error)
	ListActors(ctx context.Context, in *ateapipb.ListActorsRequest, opts ...grpc.CallOption) (*ateapipb.ListActorsResponse, error)
}

type Config struct {
	Control           ControlAPI
	JWTSecret         []byte
	TemplateNamespace string
	// ActorDomain is the atenet suffix for per-actor data-plane URLs
	// (e.g. "actors.resources.example.com"). Empty disables the data-plane
	// proxy (those routes answer 501).
	ActorDomain string
	// ActorCA optionally overrides the root CAs used to verify the atenet
	// edge certificate fronting actor data planes (self-signed test
	// environments). Nil uses the system roots.
	ActorCA *x509.CertPool
	// ActorTLSServerName overrides the hostname verified against the atenet
	// edge certificate. Empty verifies against the actor authority in the URL,
	// which is the right default but requires that certificate to carry the
	// actor domain among its SANs.
	//
	// It usually does not: upstream's servicedns signer derives SANs purely
	// from the Services covering the pod (`<svc>.<ns>.svc`; see
	// cmd/podcertcontroller/internal/servicednssigner), so the edge cert has no
	// actor-domain SAN and every data-plane request would fail hostname
	// verification. Fix it either by adding the actor domain to the certificate
	// (preferred — see manifests/ate-install/cert-manager-pki/certificates.yaml)
	// or by pinning the name here. Both keep chain verification intact, unlike
	// disabling verification.
	ActorTLSServerName string
}

type Server struct {
	cfg Config
	mux *http.ServeMux
	// dataPlaneTransport dials the atenet edge for proxied data-plane
	// requests; tests rewire its dialer at a fake upstream.
	dataPlaneTransport *http.Transport
}

func New(cfg Config) *Server {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.ActorCA != nil || cfg.ActorTLSServerName != "" {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		if cfg.ActorCA != nil {
			tlsCfg.RootCAs = cfg.ActorCA
		}
		tlsCfg.ServerName = cfg.ActorTLSServerName
		transport.TLSClientConfig = tlsCfg
	}
	s := &Server{cfg: cfg, mux: http.NewServeMux(), dataPlaneTransport: transport}
	s.mux.HandleFunc("POST /sandboxes", s.auth(s.createSandbox))
	s.mux.HandleFunc("GET /sandboxes", s.auth(s.listSandboxes))
	s.mux.HandleFunc("GET /sandboxes/{id}", s.auth(s.getSandbox))
	s.mux.HandleFunc("DELETE /sandboxes/{id}", s.auth(s.deleteSandbox))
	s.mux.HandleFunc("POST /sandboxes/{id}/pause", s.auth(s.pauseSandbox))
	s.mux.HandleFunc("POST /sandboxes/{id}/resume", s.auth(s.resumeSandbox))
	s.mux.HandleFunc("POST /sandboxes/{id}/timeout", s.auth(s.setTimeout))
	// Data plane, path-based: reverse-proxied to the actor's own HTTP surface
	// (ateom-wasmd) through the atenet edge (M4). Watch-family filesystem
	// RPCs are not registered: the actor answers them unimplemented.
	s.mux.HandleFunc("/sandboxes/{id}/execute", s.auth(s.dataPlaneProxy))
	s.mux.HandleFunc("/sandboxes/{id}/files", s.auth(s.dataPlaneProxy))
	for _, rpc := range []string{"Stat", "ListDir", "MakeDir", "Move", "Remove"} {
		s.mux.HandleFunc("POST /sandboxes/{id}/filesystem.Filesystem/"+rpc, s.auth(s.dataPlaneProxy))
	}
	// Data plane, host-based ("{port}-{sandboxID}.{domain}") is intercepted
	// in ServeHTTP before the mux: unmodified E2B SDKs address envd that way
	// and authenticate with X-Access-Token, not the API key.
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The E2B SDK reaches the data plane as https://{port}-{sandboxID}.{domain}
	// (host carries the routing, path is root-relative: /execute, /files, ...).
	// Recognize that host shape before mux dispatch; every other host (api.*,
	// Service names, pod IPs) falls through to the REST mux.
	if id, ok := sandboxIDFromHost(r.Host); ok {
		s.hostDataPlaneProxy(w, r, id)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// sandboxIDFromHost matches the E2B data-plane host shape
// "{port}-{sandboxID}.{domain}" and extracts the sandbox ID. The port label
// is advisory (the actor serves one HTTP surface behind atunnel :443).
func sandboxIDFromHost(host string) (string, bool) {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	label, _, _ := strings.Cut(host, ".")
	port, id, ok := strings.Cut(label, "-")
	if !ok || port == "" || id == "" {
		return "", false
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return "", false
		}
	}
	return id, true
}

// --- auth ---

type atespaceKey struct{}

// auth verifies the E2B API key (an HS256 JWT carried in X-API-Key or
// Authorization: Bearer) and injects the caller's atespace into the context.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Key")
		if token == "" {
			token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if token == "" {
			writeErr(w, http.StatusUnauthorized, "missing API key")
			return
		}
		claims, err := verifyHS256(token, s.cfg.JWTSecret)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid API key")
			return
		}
		atespace, _ := claims["atespace"].(string)
		if atespace == "" {
			// Compatibility with the previous gateway's tokens.
			atespace, _ = claims["tenant"].(string)
		}
		if atespace == "" {
			writeErr(w, http.StatusUnauthorized, "API key carries no atespace/tenant claim")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), atespaceKey{}, atespace)))
	}
}

func atespaceFrom(ctx context.Context) string {
	v, _ := ctx.Value(atespaceKey{}).(string)
	return v
}

// --- handlers ---

type createSandboxRequest struct {
	TemplateID string            `json:"templateID"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timeout    int64             `json:"timeout,omitempty"`
}

type sandboxResponse struct {
	SandboxID   string `json:"sandboxID"`
	TemplateID  string `json:"templateID"`
	ClientID    string `json:"clientID"`
	State       string `json:"state,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	EnvdVersion string `json:"envdVersion,omitempty"`
	// EnvdAccessToken authenticates host-based data-plane requests: SDKs echo
	// it as X-Access-Token (see hostDataPlaneProxy).
	EnvdAccessToken string `json:"envdAccessToken,omitempty"`
}

func (s *Server) createSandbox(w http.ResponseWriter, r *http.Request) {
	var req createSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.TemplateID == "" {
		writeErr(w, http.StatusBadRequest, "templateID is required")
		return
	}
	atespace := atespaceFrom(r.Context())
	name := newSandboxID()

	actor := &ateapipb.Actor{
		Metadata:               &ateapipb.ResourceMetadata{Atespace: atespace, Name: name},
		ActorTemplateNamespace: s.cfg.TemplateNamespace,
		ActorTemplateName:      req.TemplateID,
	}
	if _, err := s.cfg.Control.CreateActor(r.Context(), &ateapipb.CreateActorRequest{Actor: actor}); err != nil {
		writeGRPCErr(w, "CreateActor", err)
		return
	}
	resumed, err := s.cfg.Control.ResumeActor(r.Context(), &ateapipb.ResumeActorRequest{
		Actor: &ateapipb.ObjectRef{Atespace: atespace, Name: name},
		Boot:  true,
	})
	if err != nil {
		// Best-effort rollback so a failed boot does not leak an actor.
		if _, derr := s.cfg.Control.DeleteActor(r.Context(), &ateapipb.DeleteActorRequest{
			Actor: &ateapipb.ObjectRef{Atespace: atespace, Name: name},
		}); derr != nil {
			slog.Warn("rollback DeleteActor failed", slog.String("actor", name), slog.Any("err", derr))
		}
		writeGRPCErr(w, "ResumeActor", err)
		return
	}
	writeJSON(w, http.StatusCreated, s.sandboxResponse(resumed.GetActor(), req.TemplateID))
}

func (s *Server) listSandboxes(w http.ResponseWriter, r *http.Request) {
	atespace := atespaceFrom(r.Context())
	var out []sandboxResponse
	pageToken := ""
	for {
		resp, err := s.cfg.Control.ListActors(r.Context(), &ateapipb.ListActorsRequest{
			Atespace:  atespace,
			PageToken: pageToken,
		})
		if err != nil {
			writeGRPCErr(w, "ListActors", err)
			return
		}
		for _, a := range resp.GetActors() {
			out = append(out, s.sandboxResponse(a, a.GetActorTemplateName()))
		}
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getSandbox(w http.ResponseWriter, r *http.Request) {
	actor, err := s.cfg.Control.GetActor(r.Context(), &ateapipb.GetActorRequest{
		Actor: s.ref(r),
	})
	if err != nil {
		writeGRPCErr(w, "GetActor", err)
		return
	}
	writeJSON(w, http.StatusOK, s.sandboxResponse(actor, actor.GetActorTemplateName()))
}

// deleteSandbox implements E2B kill(): it destroys the sandbox regardless of
// state. ateapi only deletes SUSPENDED/CRASHED actors (MarkDeletingStep
// prerequisite), so a RUNNING actor is suspended first and the delete
// retried — mirroring how the E2B API tears down live sandboxes.
func (s *Server) deleteSandbox(w http.ResponseWriter, r *http.Request) {
	ref := s.ref(r)
	_, err := s.cfg.Control.DeleteActor(r.Context(), &ateapipb.DeleteActorRequest{Actor: ref})
	if status.Code(err) == codes.FailedPrecondition {
		if _, serr := s.cfg.Control.SuspendActor(r.Context(), &ateapipb.SuspendActorRequest{Actor: ref}); serr != nil {
			// A concurrent suspend may have won the race; the retry below
			// settles it either way.
			slog.Warn("kill: suspend before delete failed",
				slog.String("actor", ref.GetName()), slog.Any("err", serr))
		}
		_, err = s.cfg.Control.DeleteActor(r.Context(), &ateapipb.DeleteActorRequest{Actor: ref})
	}
	if err != nil {
		writeGRPCErr(w, "DeleteActor", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pauseSandbox(w http.ResponseWriter, r *http.Request) {
	if _, err := s.cfg.Control.SuspendActor(r.Context(), &ateapipb.SuspendActorRequest{Actor: s.ref(r)}); err != nil {
		writeGRPCErr(w, "SuspendActor", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resumeSandbox(w http.ResponseWriter, r *http.Request) {
	resp, err := s.cfg.Control.ResumeActor(r.Context(), &ateapipb.ResumeActorRequest{Actor: s.ref(r)})
	if err != nil {
		writeGRPCErr(w, "ResumeActor", err)
		return
	}
	writeJSON(w, http.StatusOK, s.sandboxResponse(resp.GetActor(), resp.GetActor().GetActorTemplateName()))
}

// setTimeout accepts the E2B timeout call. Gateway-side TTL→Suspend scheduling
// is a follow-up; acknowledging keeps SDK flows moving.
func (s *Server) setTimeout(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// dataPlaneProxy reverse-proxies path-based data-plane calls
// (/sandboxes/{id}/execute, .../files, and the filesystem.Filesystem RPCs) to
// the actor's atenet domain, where the in-actor HTTP surface (ateom-wasmd)
// serves them at the root path (M4). Authenticated by the API key (atespace
// from the request context).
func (s *Server) dataPlaneProxy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("id")
	// Strip the gateway prefix: /sandboxes/{id}/execute → /execute.
	upstreamPath := strings.TrimPrefix(r.URL.Path, "/sandboxes/"+name)
	s.proxyToActor(w, r, atespaceFrom(r.Context()), name, upstreamPath)
}

// hostDataPlaneProxy serves host-based data-plane requests
// ({port}-{sandboxID}.{domain}). The caller is authenticated by the envd
// access token (X-Access-Token) minted at create/resume time: its claims pin
// both the atespace (needed to build the actor authority — the host name
// alone does not carry it) and the sandbox ID (so a token for one sandbox
// cannot reach another).
func (s *Server) hostDataPlaneProxy(w http.ResponseWriter, r *http.Request, sandboxID string) {
	token := r.Header.Get("X-Access-Token")
	if token == "" {
		writeErr(w, http.StatusUnauthorized, "missing X-Access-Token")
		return
	}
	claims, err := verifyHS256(token, s.cfg.JWTSecret)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid access token")
		return
	}
	atespace, _ := claims["atespace"].(string)
	sandbox, _ := claims["sandbox"].(string)
	if atespace == "" || sandbox != sandboxID {
		writeErr(w, http.StatusUnauthorized, "access token does not match sandbox")
		return
	}
	s.proxyToActor(w, r, atespace, sandboxID, r.URL.Path)
}

// proxyToActor reverse-proxies the request to the actor's atenet domain.
// The outgoing Host must be the actor authority: the atenet router selects
// the worker by :authority and atunnel authorizes the actor DNS name.
func (s *Server) proxyToActor(w http.ResponseWriter, r *http.Request, atespace, name, upstreamPath string) {
	if s.cfg.ActorDomain == "" {
		writeErr(w, http.StatusNotImplemented, "data plane not wired: set --actor-domain")
		return
	}
	authority := fmt.Sprintf("%s.%s.%s", name, atespace, s.cfg.ActorDomain)
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "https"
			pr.Out.URL.Host = authority
			pr.Out.URL.Path = upstreamPath
			pr.Out.URL.RawPath = ""
			pr.Out.Host = authority
			// Gateway credentials stay off the actor.
			pr.Out.Header.Del("X-API-Key")
			pr.Out.Header.Del("Authorization")
			pr.Out.Header.Del("X-Access-Token")
		},
		Transport: s.dataPlaneTransport,
		// /execute is a long-lived NDJSON stream: flush every write instead of
		// buffering. Note httputil already forces this for streaming
		// responses — its flushInterval() returns -1 whenever
		// res.ContentLength == -1 — so setting it here only changes behaviour
		// for responses that *do* carry a Content-Length. Kept explicit so the
		// intent survives; what would actually break streaming is buffering the
		// body (e.g. a ModifyResponse that reads it whole), which
		// TestDataPlaneProxyStreamsNDJSON does catch.
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("data-plane proxy failed",
				slog.String("authority", authority), slog.Any("err", err))
			// Plain 502, not an E2B error body: this is a gateway-level
			// transport failure, not an API-mapped error.
			http.Error(w, "upstream actor unreachable", http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(w, r)
}

// --- helpers ---

func (s *Server) ref(r *http.Request) *ateapipb.ObjectRef {
	return &ateapipb.ObjectRef{Atespace: atespaceFrom(r.Context()), Name: r.PathValue("id")}
}

func (s *Server) sandboxResponse(actor *ateapipb.Actor, templateID string) sandboxResponse {
	resp := sandboxResponse{
		SandboxID:   actor.GetMetadata().GetName(),
		TemplateID:  templateID,
		ClientID:    "e2bgw",
		State:       e2bState(actor.GetStatus()),
		EnvdVersion: "0.2.0",
		EnvdAccessToken: s.mintAccessToken(
			actor.GetMetadata().GetAtespace(), actor.GetMetadata().GetName()),
	}
	if ts := actor.GetMetadata().GetCreateTime(); ts != nil {
		resp.StartedAt = ts.AsTime().UTC().Format(time.RFC3339)
	}
	return resp
}

// mintAccessToken issues the envd access token for one sandbox. Stateless by
// design: any gateway replica can verify it with the shared JWT secret, and
// a fresh one is handed out on every create/get/resume response (so
// reconnecting SDKs never hold an expired token for a live sandbox).
func (s *Server) mintAccessToken(atespace, name string) string {
	return signHS256(map[string]any{
		"atespace": atespace,
		"sandbox":  name,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}, s.cfg.JWTSecret)
}

func e2bState(s ateapipb.Actor_Status) string {
	switch s {
	case ateapipb.Actor_STATUS_RUNNING, ateapipb.Actor_STATUS_RESUMING:
		return "running"
	case ateapipb.Actor_STATUS_SUSPENDED, ateapipb.Actor_STATUS_SUSPENDING,
		ateapipb.Actor_STATUS_PAUSED, ateapipb.Actor_STATUS_PAUSING:
		return "paused"
	default:
		return strings.ToLower(strings.TrimPrefix(s.String(), "STATUS_"))
	}
}

func newSandboxID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // crypto/rand failure is not recoverable
	}
	return "sbx-" + hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"code": code, "message": msg})
}

// writeGRPCErr maps Control API errors onto E2B HTTP statuses.
func writeGRPCErr(w http.ResponseWriter, op string, err error) {
	st, ok := status.FromError(err)
	if !ok {
		var ue *json.UnsupportedValueError
		if errors.As(err, &ue) {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("%s: %v", op, err))
		return
	}
	code := http.StatusBadGateway
	switch st.Code() {
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.AlreadyExists:
		code = http.StatusConflict
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.PermissionDenied:
		code = http.StatusForbidden
	case codes.Unauthenticated:
		code = http.StatusUnauthorized
	case codes.FailedPrecondition:
		code = http.StatusConflict
	case codes.ResourceExhausted:
		code = http.StatusTooManyRequests
	case codes.Unavailable:
		code = http.StatusServiceUnavailable
	}
	writeErr(w, code, fmt.Sprintf("%s: %s", op, st.Message()))
}
