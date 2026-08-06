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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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
	// (e.g. "actors.resources.example.com"). Empty disables redirects.
	ActorDomain string
}

type Server struct {
	cfg Config
	mux *http.ServeMux
}

func New(cfg Config) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /sandboxes", s.auth(s.createSandbox))
	s.mux.HandleFunc("GET /sandboxes", s.auth(s.listSandboxes))
	s.mux.HandleFunc("GET /sandboxes/{id}", s.auth(s.getSandbox))
	s.mux.HandleFunc("DELETE /sandboxes/{id}", s.auth(s.deleteSandbox))
	s.mux.HandleFunc("POST /sandboxes/{id}/pause", s.auth(s.pauseSandbox))
	s.mux.HandleFunc("POST /sandboxes/{id}/resume", s.auth(s.resumeSandbox))
	s.mux.HandleFunc("POST /sandboxes/{id}/timeout", s.auth(s.setTimeout))
	// Data plane: served by the actor itself via atenet (M4).
	s.mux.HandleFunc("/sandboxes/{id}/execute", s.auth(s.dataPlaneRedirect))
	s.mux.HandleFunc("/sandboxes/{id}/files", s.auth(s.dataPlaneRedirect))
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

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

func (s *Server) deleteSandbox(w http.ResponseWriter, r *http.Request) {
	if _, err := s.cfg.Control.DeleteActor(r.Context(), &ateapipb.DeleteActorRequest{Actor: s.ref(r)}); err != nil {
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

// dataPlaneRedirect points SDKs at the actor's atenet domain, where the
// in-actor HTTP surface (ateom-wasmd) serves /execute and /files (M4).
func (s *Server) dataPlaneRedirect(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ActorDomain == "" {
		writeErr(w, http.StatusNotImplemented, "data plane not wired: set --actor-domain")
		return
	}
	name := r.PathValue("id")
	atespace := atespaceFrom(r.Context())
	target := fmt.Sprintf("https://%s.%s.%s%s", name, atespace, s.cfg.ActorDomain, r.URL.Path)
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
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
	}
	if ts := actor.GetMetadata().GetCreateTime(); ts != nil {
		resp.StartedAt = ts.AsTime().UTC().Format(time.RFC3339)
	}
	return resp
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
