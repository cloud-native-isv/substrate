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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestDataPlaneRedirect(t *testing.T) {
	s := newTestServer(&fakeControl{})
	w := do(t, s, "POST", "/sandboxes/sbx-9/execute", `{"code":"1+1"}`, true)
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("code = %d, want 307", w.Code)
	}
	want := "https://sbx-9.tenant-a.actors.resources.test/sandboxes/sbx-9/execute"
	if got := w.Header().Get("Location"); got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

func TestDataPlane501WithoutDomain(t *testing.T) {
	s := New(Config{Control: &fakeControl{}, JWTSecret: testSecret, TemplateNamespace: "ate-wasm"})
	w := do(t, s, "POST", "/sandboxes/sbx-9/execute", "", true)
	if w.Code != http.StatusNotImplemented {
		t.Errorf("code = %d, want 501", w.Code)
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
