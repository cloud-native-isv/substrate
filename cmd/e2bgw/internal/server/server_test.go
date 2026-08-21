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

package server

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agent-substrate/substrate/pkg/proto/ateapipb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testSecret = []byte("test-secret")

func signJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	signing := enc(map[string]string{"alg": "HS256", "typ": "JWT"}) + "." + enc(claims)
	mac := hmac.New(sha256.New, testSecret)
	mac.Write([]byte(signing))
	return signing + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// fakeControl records calls and plays back canned responses.
type fakeControl struct {
	created   []*ateapipb.CreateActorRequest
	resumed   []*ateapipb.ResumeActorRequest
	suspended []*ateapipb.SuspendActorRequest
	deleted   []*ateapipb.DeleteActorRequest

	createErr error
	resumeErr error
	getErr    error

	actor *ateapipb.Actor
}

func (f *fakeControl) actorOr(name, atespace string) *ateapipb.Actor {
	if f.actor != nil {
		return f.actor
	}
	return &ateapipb.Actor{
		Metadata:          &ateapipb.ResourceMetadata{Atespace: atespace, Name: name},
		ActorTemplateName: "code-interpreter",
		Status:            ateapipb.Actor_STATUS_RUNNING,
	}
}

func (f *fakeControl) CreateActor(_ context.Context, in *ateapipb.CreateActorRequest, _ ...grpc.CallOption) (*ateapipb.Actor, error) {
	f.created = append(f.created, in)
	if f.createErr != nil {
		return nil, f.createErr
	}
	return in.GetActor(), nil
}

func (f *fakeControl) GetActor(_ context.Context, in *ateapipb.GetActorRequest, _ ...grpc.CallOption) (*ateapipb.Actor, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.actorOr(in.GetActor().GetName(), in.GetActor().GetAtespace()), nil
}

func (f *fakeControl) DeleteActor(_ context.Context, in *ateapipb.DeleteActorRequest, _ ...grpc.CallOption) (*ateapipb.Actor, error) {
	f.deleted = append(f.deleted, in)
	return f.actorOr(in.GetActor().GetName(), in.GetActor().GetAtespace()), nil
}

func (f *fakeControl) SuspendActor(_ context.Context, in *ateapipb.SuspendActorRequest, _ ...grpc.CallOption) (*ateapipb.SuspendActorResponse, error) {
	f.suspended = append(f.suspended, in)
	return &ateapipb.SuspendActorResponse{Actor: f.actorOr(in.GetActor().GetName(), in.GetActor().GetAtespace())}, nil
}

func (f *fakeControl) ResumeActor(_ context.Context, in *ateapipb.ResumeActorRequest, _ ...grpc.CallOption) (*ateapipb.ResumeActorResponse, error) {
	f.resumed = append(f.resumed, in)
	if f.resumeErr != nil {
		return nil, f.resumeErr
	}
	return &ateapipb.ResumeActorResponse{
		Actor:   f.actorOr(in.GetActor().GetName(), in.GetActor().GetAtespace()),
		Resumed: true,
	}, nil
}

func (f *fakeControl) ListActors(_ context.Context, in *ateapipb.ListActorsRequest, _ ...grpc.CallOption) (*ateapipb.ListActorsResponse, error) {
	return &ateapipb.ListActorsResponse{Actors: []*ateapipb.Actor{f.actorOr("sbx-1", in.GetAtespace())}}, nil
}

func (f *fakeControl) ListActorSnapshots(_ context.Context, _ *ateapipb.ListActorSnapshotsRequest, _ ...grpc.CallOption) (*ateapipb.ListActorSnapshotsResponse, error) {
	return &ateapipb.ListActorSnapshotsResponse{}, nil
}

func newTestServer(f *fakeControl) *Server {
	return New(Config{
		Control:           f,
		JWTSecret:         testSecret,
		TemplateNamespace: "ate-wasm",
		ActorDomain:       "actors.resources.test",
	})
}

func do(t *testing.T, s *Server, method, path, body string, authed bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if authed {
		req.Header.Set("X-API-Key", signJWT(t, map[string]any{"atespace": "tenant-a"}))
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

// newProxyTestServer runs upstream as a TLS fake of the atenet edge and
// returns a gateway whose data-plane proxy is wired at it: the upstream
// certificate is trusted via Config.ActorCA (the --actor-ca path, injected
// as PEM like the flag would) and the transport dials the fake regardless
// of the actor authority in the URL.
//
// The httptest certificate is issued for example.com, not the actor authority,
// so the verification name is pinned through Config.ActorTLSServerName — the
// same --actor-tls-server-name path a real deployment uses when the atenet edge
// certificate carries no actor-domain SAN.
func newProxyTestServer(t *testing.T, upstream http.Handler) *Server {
	t.Helper()
	return newProxyTestServerTLSName(t, upstream, "example.com")
}

// newProxyTestServerTLSName is newProxyTestServer with an explicit TLS
// verification name; "" leaves verification against the actor authority.
func newProxyTestServerTLSName(t *testing.T, upstream http.Handler, tlsServerName string) *Server {
	t.Helper()
	fake := httptest.NewTLSServer(upstream)
	t.Cleanup(fake.Close)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: fake.Certificate().Raw})
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("appending upstream cert PEM")
	}
	s := New(Config{
		Control:            &fakeControl{},
		JWTSecret:          testSecret,
		TemplateNamespace:  "ate-wasm",
		ActorDomain:        "actors.resources.test",
		ActorCA:            pool,
		ActorTLSServerName: tlsServerName,
	})
	fakeAddr := strings.TrimPrefix(fake.URL, "https://")
	s.dataPlaneTransport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, fakeAddr)
	}
	return s
}

// TestDataPlaneProxyTLSServerName pins both halves of the
// --actor-tls-server-name contract: pinning the name lets the proxy verify an
// edge certificate that does not carry the actor domain, and leaving it empty
// keeps verifying the actor authority (so a mismatch is still refused rather
// than silently accepted).
func TestDataPlaneProxyTLSServerName(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("pinned name verifies against the edge cert", func(t *testing.T) {
		s := newProxyTestServerTLSName(t, ok, "example.com")
		w := do(t, s, "POST", "/sandboxes/sbx-1/execute", `{"code":"1"}`, true)
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, want 200; body %s", w.Code, w.Body)
		}
	})

	t.Run("without the override the actor authority is still verified", func(t *testing.T) {
		// The fake edge cert is for example.com; the proxy dials
		// sbx-1.tenant-a.actors.resources.test. Verification must fail, and the
		// ErrorHandler must turn that into a 502 rather than proceeding.
		s := newProxyTestServerTLSName(t, ok, "")
		w := do(t, s, "POST", "/sandboxes/sbx-1/execute", `{"code":"1"}`, true)
		if w.Code != http.StatusBadGateway {
			t.Fatalf("code = %d, want 502 (hostname verification must fail); body %s", w.Code, w.Body)
		}
	})
}

func TestAuthRejects(t *testing.T) {
	s := newTestServer(&fakeControl{})
	tests := []struct {
		name string
		key  string
		want int
	}{
		{"missing key", "", http.StatusUnauthorized},
		{"garbage key", "not-a-jwt", http.StatusUnauthorized},
		{"wrong secret", func() string {
			sig := base64.RawURLEncoding.EncodeToString([]byte("bad"))
			hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
			pl := base64.RawURLEncoding.EncodeToString([]byte(`{"atespace":"x"}`))
			return hdr + "." + pl + "." + sig
		}(), http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/sandboxes", nil)
			if tt.key != "" {
				req.Header.Set("X-API-Key", tt.key)
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("code = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestAuthAcceptsTenantClaim(t *testing.T) {
	f := &fakeControl{}
	s := newTestServer(f)
	req := httptest.NewRequest("GET", "/sandboxes", nil)
	req.Header.Set("Authorization", "Bearer "+signJWT(t, map[string]any{"tenant": "tenant-b"}))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body %s", w.Code, w.Body)
	}
}

func TestCreateSandbox(t *testing.T) {
	f := &fakeControl{}
	s := newTestServer(f)
	w := do(t, s, "POST", "/sandboxes", `{"templateID":"code-interpreter"}`, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("code = %d, body %s", w.Code, w.Body)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp["sandboxID"].(string), "sbx-") {
		t.Errorf("sandboxID = %v", resp["sandboxID"])
	}
	if resp["templateID"] != "code-interpreter" {
		t.Errorf("templateID = %v", resp["templateID"])
	}

	if len(f.created) != 1 || len(f.resumed) != 1 {
		t.Fatalf("created=%d resumed=%d, want 1/1", len(f.created), len(f.resumed))
	}
	actor := f.created[0].GetActor()
	if actor.GetMetadata().GetAtespace() != "tenant-a" {
		t.Errorf("atespace = %q, want tenant-a", actor.GetMetadata().GetAtespace())
	}
	if actor.GetActorTemplateNamespace() != "ate-wasm" || actor.GetActorTemplateName() != "code-interpreter" {
		t.Errorf("template ref = %s/%s", actor.GetActorTemplateNamespace(), actor.GetActorTemplateName())
	}
	if !f.resumed[0].GetBoot() {
		t.Error("initial resume must set boot=true")
	}
}

func TestCreateSandboxRollsBackOnResumeFailure(t *testing.T) {
	f := &fakeControl{resumeErr: status.Error(codes.ResourceExhausted, "no free workers")}
	s := newTestServer(f)
	w := do(t, s, "POST", "/sandboxes", `{"templateID":"code-interpreter"}`, true)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("code = %d, want 429; body %s", w.Code, w.Body)
	}
	if len(f.deleted) != 1 {
		t.Fatalf("deleted = %d, want rollback delete", len(f.deleted))
	}
}

func TestCreateSandboxValidation(t *testing.T) {
	s := newTestServer(&fakeControl{})
	if w := do(t, s, "POST", "/sandboxes", `{}`, true); w.Code != http.StatusBadRequest {
		t.Errorf("missing templateID: code = %d, want 400", w.Code)
	}
	if w := do(t, s, "POST", "/sandboxes", `{bad`, true); w.Code != http.StatusBadRequest {
		t.Errorf("bad json: code = %d, want 400", w.Code)
	}
}

// TestConnectSandbox pins the SDK 2.x Sandbox.connect() contract
// (POST /sandboxes/{id}/connect): a suspended sandbox is resumed, a running
// one is attached as-is (resume answers FailedPrecondition), and both paths
// return sandbox info carrying a fresh envdAccessToken.
func TestConnectSandbox(t *testing.T) {
	t.Run("resumes a suspended sandbox", func(t *testing.T) {
		f := &fakeControl{}
		s := newTestServer(f)
		w := do(t, s, "POST", "/sandboxes/sbx-1/connect", "", true)
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, body %s", w.Code, w.Body)
		}
		if len(f.resumed) != 1 {
			t.Fatalf("resumed = %d, want 1", len(f.resumed))
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if tok, _ := resp["envdAccessToken"].(string); tok == "" {
			t.Errorf("envdAccessToken missing from connect response: %s", w.Body)
		}
	})

	t.Run("attaches to a running sandbox as-is", func(t *testing.T) {
		f := &fakeControl{resumeErr: status.Error(codes.FailedPrecondition, "already running")}
		s := newTestServer(f)
		w := do(t, s, "POST", "/sandboxes/sbx-1/connect", `{"timeout": 60}`, true)
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, body %s", w.Code, w.Body)
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp["sandboxID"] != "sbx-1" {
			t.Errorf("sandboxID = %v", resp["sandboxID"])
		}
	})

	t.Run("unknown sandbox answers 404", func(t *testing.T) {
		f := &fakeControl{resumeErr: status.Error(codes.NotFound, "no such actor")}
		s := newTestServer(f)
		w := do(t, s, "POST", "/sandboxes/sbx-nope/connect", "", true)
		if w.Code != http.StatusNotFound {
			t.Fatalf("code = %d, want 404; body %s", w.Code, w.Body)
		}
	})
}

func TestLifecycleEndpoints(t *testing.T) {
	f := &fakeControl{}
	s := newTestServer(f)

	if w := do(t, s, "POST", "/sandboxes/sbx-1/pause", "", true); w.Code != http.StatusNoContent {
		t.Errorf("pause: code = %d", w.Code)
	}
	if got := f.suspended[0].GetActor(); got.GetAtespace() != "tenant-a" || got.GetName() != "sbx-1" {
		t.Errorf("suspend ref = %v", got)
	}

	if w := do(t, s, "POST", "/sandboxes/sbx-1/resume", "", true); w.Code != http.StatusOK {
		t.Errorf("resume: code = %d", w.Code)
	}
	if w := do(t, s, "DELETE", "/sandboxes/sbx-1", "", true); w.Code != http.StatusNoContent {
		t.Errorf("delete: code = %d", w.Code)
	}
	if w := do(t, s, "POST", "/sandboxes/sbx-1/timeout", `{"timeout":300}`, true); w.Code != http.StatusNoContent {
		t.Errorf("timeout: code = %d", w.Code)
	}
	if w := do(t, s, "GET", "/sandboxes/sbx-1", "", true); w.Code != http.StatusOK {
		t.Errorf("get: code = %d", w.Code)
	}
	if w := do(t, s, "GET", "/sandboxes", "", true); w.Code != http.StatusOK {
		t.Errorf("list: code = %d", w.Code)
	}
}

func TestGRPCErrorMapping(t *testing.T) {
	f := &fakeControl{getErr: status.Error(codes.NotFound, "no such actor")}
	s := newTestServer(f)
	if w := do(t, s, "GET", "/sandboxes/sbx-x", "", true); w.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", w.Code)
	}
}

func TestDataPlaneProxyRewritesPathAndHost(t *testing.T) {
	type seen struct {
		method, path, query, host, body, apiKey string
	}
	var got seen
	s := newProxyTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = seen{r.Method, r.URL.Path, r.URL.RawQuery, r.Host, string(body), r.Header.Get("X-API-Key")}
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"type":"end"}`)
	}))
	w := do(t, s, "POST", "/sandboxes/sbx-1/execute?foo=bar", `{"code":"1+1"}`, true)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body %s", w.Code, w.Body)
	}
	if got.method != "POST" || got.path != "/execute" {
		t.Errorf("upstream saw %s %s, want POST /execute", got.method, got.path)
	}
	if got.query != "foo=bar" {
		t.Errorf("query = %q, want foo=bar", got.query)
	}
	if want := "sbx-1.tenant-a.actors.resources.test"; got.host != want {
		t.Errorf("Host = %q, want %q", got.host, want)
	}
	if got.body != `{"code":"1+1"}` {
		t.Errorf("body = %q", got.body)
	}
	if got.apiKey != "" {
		t.Errorf("X-API-Key leaked to upstream: %q", got.apiKey)
	}
	if !strings.Contains(w.Body.String(), `"end"`) {
		t.Errorf("response body = %q", w.Body)
	}
}

func TestDataPlaneProxyRoutes(t *testing.T) {
	tests := []struct {
		name, method, path, body string
		wantPath, wantQuery      string
	}{
		{"files GET", "GET", "/sandboxes/sbx-2/files?path=%2Ftmp%2Fa.txt", "", "/files", "path=%2Ftmp%2Fa.txt"},
		{"files POST", "POST", "/sandboxes/sbx-2/files?path=%2Ftmp%2Fa.txt", "hello", "/files", "path=%2Ftmp%2Fa.txt"},
		{"filesystem Stat", "POST", "/sandboxes/sbx-2/filesystem.Filesystem/Stat", `{"path":"/tmp"}`, "/filesystem.Filesystem/Stat", ""},
		{"filesystem ListDir", "POST", "/sandboxes/sbx-2/filesystem.Filesystem/ListDir", `{"path":"/"}`, "/filesystem.Filesystem/ListDir", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath, gotQuery, gotBody string
			s := newProxyTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				gotMethod, gotPath, gotQuery, gotBody = r.Method, r.URL.Path, r.URL.RawQuery, string(body)
				w.WriteHeader(http.StatusOK)
			}))
			w := do(t, s, tt.method, tt.path, tt.body, true)
			if w.Code != http.StatusOK {
				t.Fatalf("code = %d, body %s", w.Code, w.Body)
			}
			if gotMethod != tt.method || gotPath != tt.wantPath || gotQuery != tt.wantQuery {
				t.Errorf("upstream saw %s %s?%s, want %s %s?%s",
					gotMethod, gotPath, gotQuery, tt.method, tt.wantPath, tt.wantQuery)
			}
			if gotBody != tt.body {
				t.Errorf("body = %q, want %q", gotBody, tt.body)
			}
		})
	}
}

func TestDataPlaneProxyStreamsNDJSON(t *testing.T) {
	firstLineRead := make(chan struct{})
	handlerDone := make(chan struct{})
	s := newProxyTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"type":"stdout","text":"hi"}`)
		w.(http.Flusher).Flush()
		// Hold the stream open until the client proves it received the
		// first line mid-stream (i.e. the proxy did not buffer).
		select {
		case <-firstLineRead:
		case <-time.After(5 * time.Second):
			t.Error("client never read the first line")
			return
		}
		fmt.Fprintln(w, `{"type":"end"}`)
	}))

	gw := httptest.NewServer(s)
	t.Cleanup(gw.Close)
	req, err := http.NewRequest("POST", gw.URL+"/sandboxes/sbx-3/execute", strings.NewReader(`{"code":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-API-Key", signJWT(t, map[string]any{"atespace": "tenant-a"}))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("code = %d", resp.StatusCode)
	}

	br := bufio.NewReader(resp.Body)
	line1, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("reading first frame: %v", err)
	}
	if !strings.Contains(line1, `"stdout"`) {
		t.Errorf("first frame = %q", line1)
	}
	// The upstream handler must still be alive: if the proxy had buffered
	// the response, the first line would only arrive after it returned.
	select {
	case <-handlerDone:
		t.Fatal("upstream handler already returned: stream was buffered, not flushed per line")
	default:
	}
	close(firstLineRead)

	line2, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("reading end frame: %v", err)
	}
	if !strings.Contains(line2, `"end"`) {
		t.Errorf("end frame = %q", line2)
	}
}

func TestDataPlane501WithoutDomain(t *testing.T) {
	s := New(Config{Control: &fakeControl{}, JWTSecret: testSecret, TemplateNamespace: "ate-wasm"})
	w := do(t, s, "POST", "/sandboxes/sbx-9/execute", "", true)
	if w.Code != http.StatusNotImplemented {
		t.Errorf("code = %d, want 501", w.Code)
	}
}

func TestDataPlane502WhenUpstreamUnreachable(t *testing.T) {
	s := newTestServer(&fakeControl{})
	// A freshly closed listener yields an address that refuses connections.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	deadAddr := lis.Addr().String()
	_ = lis.Close()
	s.dataPlaneTransport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, deadAddr)
	}
	w := do(t, s, "POST", "/sandboxes/sbx-9/execute", `{"code":"1"}`, true)
	if w.Code != http.StatusBadGateway {
		t.Errorf("code = %d, want 502; body %s", w.Code, w.Body)
	}
}

func TestStateMapping(t *testing.T) {
	for status, want := range map[ateapipb.Actor_Status]string{
		ateapipb.Actor_STATUS_RUNNING:   "running",
		ateapipb.Actor_STATUS_RESUMING:  "running",
		ateapipb.Actor_STATUS_SUSPENDED: "paused",
		ateapipb.Actor_STATUS_PAUSED:    "paused",
		ateapipb.Actor_STATUS_CRASHED:   "crashed",
	} {
		if got := e2bState(status); got != want {
			t.Errorf("e2bState(%v) = %q, want %q", status, got, want)
		}
	}
}

func TestSandboxIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := newSandboxID()
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
}
