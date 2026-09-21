// Copyright 2026 Google LLC
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

package resources

import (
	"github.com/agent-substrate/substrate/pkg/proto/ateapipb"
)

// WorkerActorCapacityAnnotation is the worker-pod annotation carrying the number
// of concurrent actors the pod's ateom runtime hosts (F9). It is the contract
// between three parties, all of which reference this single constant to prevent
// drift: the atecontroller projects WorkerPool.Spec.ActorCapacity onto the worker
// pod as this annotation (and as the WASM_MAX_ACTORS env); the ateapi syncer
// reads the annotation into Worker.actor_capacity; and the in-pod ateom-wasmd
// runtime reads WASM_MAX_ACTORS to enforce the same N (sandbox S12). Absent or
// invalid means 0 → effective capacity 1 (upstream single-actor behavior).
const WorkerActorCapacityAnnotation = "ate.dev/actor-capacity"

// Worker assignment helpers (xuanji F9 — multi active actor per worker pod).
//
// Upstream modeled a worker as hosting at most one active actor (a singular
// Worker.assignment). F9 generalizes this to Worker.assignments (repeated) so a
// worker pod — one digital employee, one S trust domain — may host N concurrent
// actors (spatial multiplexing, docs/concepts/trust-domain-panorama.md §2.1; the
// in-pod side is sandbox S12). These helpers centralize the two semantics that
// are easy to get wrong when going from 1:1 to N:1:
//
//   - capacity: a worker is schedulable while len(assignments) < effective
//     capacity, where effective capacity defaults to 1 (byte-identical to the
//     upstream "free worker" check) unless explicitly raised; and
//   - per-actor removal: releasing one actor (suspend/pause/crash) removes only
//     ITS assignment, never the siblings sharing the worker. Upstream's
//     `worker.Assignment = nil` would wrongly evict every co-hosted actor.

// WorkerEffectiveCapacity returns how many actors w may host concurrently. An
// unset or non-positive ActorCapacity means 1, reproducing the upstream
// "1 worker pod = 1 active actor" behavior exactly; F9's N:1 multiplexing is
// opt-in via a positive capacity.
func WorkerEffectiveCapacity(w *ateapipb.Worker) int64 {
	if c := w.GetActorCapacity(); c > 0 {
		return c
	}
	return 1
}

// WorkerHasCapacity reports whether w can host one more actor. With the default
// capacity of 1 this is true only when w has no assignments — identical to the
// upstream `worker.GetAssignment() == nil` scheduling gate.
func WorkerHasCapacity(w *ateapipb.Worker) bool {
	return int64(len(w.GetAssignments())) < WorkerEffectiveCapacity(w)
}

// FindWorkerAssignment returns w's assignment for ref, or nil if w does not host
// that actor.
func FindWorkerAssignment(w *ateapipb.Worker, ref ActorRef) *ateapipb.Assignment {
	for _, a := range w.GetAssignments() {
		if ActorRefFromObjectRef(a.GetActor()) == ref {
			return a
		}
	}
	return nil
}

// WorkerHostsActor reports whether ref is currently assigned to w.
func WorkerHostsActor(w *ateapipb.Worker, ref ActorRef) bool {
	return FindWorkerAssignment(w, ref) != nil
}

// UpsertWorkerAssignment records assignment a on w, replacing any existing
// assignment for the same actor (idempotent — re-resuming an actor already bound
// to w does not duplicate it) and leaving sibling actors intact. Callers gate on
// WorkerHasCapacity before placing a NEW actor; this does not itself enforce
// capacity.
func UpsertWorkerAssignment(w *ateapipb.Worker, a *ateapipb.Assignment) {
	ref := ActorRefFromObjectRef(a.GetActor())
	for i, ex := range w.GetAssignments() {
		if ActorRefFromObjectRef(ex.GetActor()) == ref {
			w.Assignments[i] = a
			return
		}
	}
	w.Assignments = append(w.Assignments, a)
}

// RemoveWorkerAssignment removes ref's assignment from w, leaving every other
// actor's assignment intact. Removing the last assignment yields a nil slice,
// matching upstream's `worker.Assignment = nil`. This is the N:1 generalization
// of that clearing operation and the key correctness point of F9: releasing one
// actor must not evict its co-hosted siblings.
func RemoveWorkerAssignment(w *ateapipb.Worker, ref ActorRef) {
	var out []*ateapipb.Assignment
	for _, a := range w.GetAssignments() {
		if ActorRefFromObjectRef(a.GetActor()) != ref {
			out = append(out, a)
		}
	}
	w.Assignments = out
}
