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
	"log/slog"
	"sync"
	"time"

	"github.com/agent-substrate/substrate/pkg/proto/ateapipb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// defaultSandboxTTL matches the E2B default sandbox timeout (5 minutes).
const defaultSandboxTTL = 300 * time.Second

// suspendCallTimeout bounds the control-plane call made on TTL expiry.
const suspendCallTimeout = 60 * time.Second

// ttlTable schedules gateway-side sandbox timeouts. E2B semantics: a sandbox
// lives `timeout` seconds unless extended (create/resume/timeout calls all
// reset the clock); on expiry the actor is suspended — with substrate's
// request parking + auto-resume that is strictly gentler than E2B's
// kill-on-timeout, since a late request transparently wakes the actor
// instead of failing.
//
// State is in-memory by design (e2bgw runs single-replica): a gateway
// restart drops pending timers, which only means an idle sandbox keeps
// running until paused or killed explicitly — never data loss.
type ttlTable struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
}

func (t *ttlTable) arm(key string, d time.Duration, expire func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timers == nil {
		t.timers = map[string]*time.Timer{}
	}
	if old, ok := t.timers[key]; ok {
		old.Stop()
	}
	var timer *time.Timer
	timer = time.AfterFunc(d, func() {
		t.mu.Lock()
		// Fire only while still the registered timer: a concurrent re-arm
		// or cancel invalidates this schedule (Stop cannot win a race
		// against an already-started AfterFunc, so re-check under the lock).
		if t.timers[key] != timer {
			t.mu.Unlock()
			return
		}
		delete(t.timers, key)
		t.mu.Unlock()
		expire()
	})
	t.timers[key] = timer
}

func (t *ttlTable) cancel(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timer, ok := t.timers[key]; ok {
		timer.Stop()
		delete(t.timers, key)
	}
}

// armTTL (re)schedules the sandbox's timeout; seconds <= 0 uses the E2B
// default.
func (s *Server) armTTL(atespace, name string, seconds int64) {
	d := defaultSandboxTTL
	if seconds > 0 {
		d = time.Duration(seconds) * time.Second
	}
	key := atespace + "/" + name
	s.ttl.arm(key, d, func() {
		ctx, cancel := context.WithTimeout(context.Background(), suspendCallTimeout)
		defer cancel()
		_, err := s.cfg.Control.SuspendActor(ctx, &ateapipb.SuspendActorRequest{
			Actor: &ateapipb.ObjectRef{Atespace: atespace, Name: name},
		})
		switch status.Code(err) {
		case codes.OK:
			slog.Info("sandbox TTL expired; suspended", slog.String("actor", key))
		case codes.NotFound, codes.FailedPrecondition:
			// Deleted or already suspended/suspending — nothing to do.
		default:
			slog.Warn("sandbox TTL expiry suspend failed",
				slog.String("actor", key), slog.Any("err", err))
		}
	})
}

func (s *Server) cancelTTL(atespace, name string) {
	s.ttl.cancel(atespace + "/" + name)
}
