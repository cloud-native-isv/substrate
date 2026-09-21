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
	"testing"

	"github.com/agent-substrate/substrate/pkg/proto/ateapipb"
)

func assignTo(atespace, name string) *ateapipb.Assignment {
	return &ateapipb.Assignment{Actor: &ateapipb.ObjectRef{Atespace: atespace, Name: name}}
}

func workerWith(capacity int64, assignments ...*ateapipb.Assignment) *ateapipb.Worker {
	return &ateapipb.Worker{ActorCapacity: capacity, Assignments: assignments}
}

// F9 backward-compat invariant: an unset/zero/negative capacity means 1, so a
// worker hosts at most one actor exactly as upstream did.
func TestWorkerEffectiveCapacityDefaultsToOne(t *testing.T) {
	for _, c := range []int64{0, -1, -100} {
		if got := WorkerEffectiveCapacity(workerWith(c)); got != 1 {
			t.Errorf("WorkerEffectiveCapacity(cap=%d) = %d, want 1", c, got)
		}
	}
	if got := WorkerEffectiveCapacity(workerWith(5)); got != 5 {
		t.Errorf("WorkerEffectiveCapacity(cap=5) = %d, want 5", got)
	}
}

// A default-capacity worker is schedulable only while empty (== upstream's
// `GetAssignment() == nil` gate); a raised capacity admits N actors.
func TestWorkerHasCapacity(t *testing.T) {
	a := assignTo("team-a", "act-1")
	b := assignTo("team-a", "act-2")
	c := assignTo("team-a", "act-3")

	if !WorkerHasCapacity(workerWith(0)) {
		t.Error("empty default-capacity worker should have capacity")
	}
	if WorkerHasCapacity(workerWith(0, a)) {
		t.Error("default-capacity worker holding one actor must be full (upstream 1:1)")
	}
	if !WorkerHasCapacity(workerWith(3, a, b)) {
		t.Error("capacity-3 worker holding 2 actors should still have capacity")
	}
	if WorkerHasCapacity(workerWith(3, a, b, c)) {
		t.Error("capacity-3 worker holding 3 actors must be full")
	}
}

func TestFindAndHostsActor(t *testing.T) {
	w := workerWith(4, assignTo("team-a", "act-1"), assignTo("team-b", "act-2"))

	if got := FindWorkerAssignment(w, ActorRef{Atespace: "team-b", Name: "act-2"}); got == nil || got.GetActor().GetName() != "act-2" {
		t.Errorf("FindWorkerAssignment(team-b/act-2) = %v, want the act-2 assignment", got)
	}
	if !WorkerHostsActor(w, ActorRef{Atespace: "team-a", Name: "act-1"}) {
		t.Error("WorkerHostsActor(team-a/act-1) = false, want true")
	}
	// Same name, different atespace is a different actor — must not match.
	if WorkerHostsActor(w, ActorRef{Atespace: "team-c", Name: "act-1"}) {
		t.Error("WorkerHostsActor(team-c/act-1) = true, want false (atespace is part of identity)")
	}
	if FindWorkerAssignment(w, ActorRef{Atespace: "team-z", Name: "nope"}) != nil {
		t.Error("FindWorkerAssignment for an absent actor must be nil")
	}
}

// Upsert appends a new actor and idempotently replaces an existing one, never
// duplicating it and never disturbing siblings.
func TestUpsertWorkerAssignment(t *testing.T) {
	w := workerWith(4, assignTo("team-a", "act-1"))

	UpsertWorkerAssignment(w, assignTo("team-a", "act-2"))
	if len(w.GetAssignments()) != 2 {
		t.Fatalf("after appending act-2, len = %d, want 2", len(w.GetAssignments()))
	}

	// Re-upserting act-1 (e.g. a retried resume) replaces in place, not appends.
	UpsertWorkerAssignment(w, &ateapipb.Assignment{
		Actor:         &ateapipb.ObjectRef{Atespace: "team-a", Name: "act-1"},
		ActorTemplate: &ateapipb.KubeNamespacedObjectRef{Namespace: "ns", Name: "tmpl"},
	})
	if len(w.GetAssignments()) != 2 {
		t.Fatalf("re-upsert of act-1 duplicated it: len = %d, want 2", len(w.GetAssignments()))
	}
	got := FindWorkerAssignment(w, ActorRef{Atespace: "team-a", Name: "act-1"})
	if got.GetActorTemplate().GetName() != "tmpl" {
		t.Errorf("re-upsert did not replace the assignment: template = %v, want tmpl", got.GetActorTemplate())
	}
	if !WorkerHostsActor(w, ActorRef{Atespace: "team-a", Name: "act-2"}) {
		t.Error("upsert of act-1 disturbed sibling act-2")
	}
}

// The key F9 correctness point: removing one actor leaves its co-hosted siblings
// bound; removing the last yields nil (matching upstream `Assignment = nil`).
func TestRemoveWorkerAssignmentPreservesSiblings(t *testing.T) {
	w := workerWith(4, assignTo("team-a", "act-1"), assignTo("team-a", "act-2"), assignTo("team-b", "act-3"))

	RemoveWorkerAssignment(w, ActorRef{Atespace: "team-a", Name: "act-2"})
	if len(w.GetAssignments()) != 2 {
		t.Fatalf("after removing act-2, len = %d, want 2", len(w.GetAssignments()))
	}
	if WorkerHostsActor(w, ActorRef{Atespace: "team-a", Name: "act-2"}) {
		t.Error("act-2 still present after removal")
	}
	if !WorkerHostsActor(w, ActorRef{Atespace: "team-a", Name: "act-1"}) || !WorkerHostsActor(w, ActorRef{Atespace: "team-b", Name: "act-3"}) {
		t.Error("removing act-2 evicted a sibling (act-1 / act-3)")
	}

	// Removing a non-member is a no-op.
	RemoveWorkerAssignment(w, ActorRef{Atespace: "team-z", Name: "ghost"})
	if len(w.GetAssignments()) != 2 {
		t.Errorf("removing a non-member changed len to %d, want 2", len(w.GetAssignments()))
	}

	// Draining to empty yields nil, not an empty non-nil slice.
	RemoveWorkerAssignment(w, ActorRef{Atespace: "team-a", Name: "act-1"})
	RemoveWorkerAssignment(w, ActorRef{Atespace: "team-b", Name: "act-3"})
	if w.Assignments != nil {
		t.Errorf("after removing all actors, Assignments = %#v, want nil", w.Assignments)
	}
	if WorkerHasCapacity(workerWith(0)) == false {
		t.Error("sanity: empty default worker should have capacity")
	}
}
