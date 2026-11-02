package team

// The stall guard exists because task-skill-2923 ran 42 times on identical
// input: every turn aborted on the same two missing fields, changed nothing,
// and recorded exactly that in its ledger -- which is capped at 20 entries and
// rotated, so the counter that mattered could never be read back.
//
// These tests pin the two halves that make it a guard rather than a retry cap:
// consecutive empty turns accumulate, and any turn that produces something
// resets the streak. A task doing real work is allowed to run indefinitely.

import (
	"testing"
	"time"
)

func TestEmptyTurnsAccumulateAndTripAfterThreshold(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Run skill: meeting-prep", "needs object_name", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i < maxStalledTurns; i++ {
		stalled, crossed := b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{
			Bot: "eng", Outcome: noDurableTraceOutcome,
			At: time.Now().UTC().Format(time.RFC3339),
		})
		if stalled != i {
			t.Fatalf("after %d empty turns: stalled=%d, want %d", i, stalled, i)
		}
		if crossed {
			t.Fatalf("turn %d must not cross the limit (limit is %d)", i, maxStalledTurns)
		}
	}

	stalled, crossed := b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{
		Bot: "eng", Outcome: noDurableTraceOutcome,
		At: time.Now().UTC().Format(time.RFC3339),
	})
	if !crossed {
		t.Fatalf("turn %d must cross the limit; stalled=%d", maxStalledTurns, stalled)
	}
}

func TestProductiveTurnResetsTheStreak(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Long but productive", "keeps landing changes", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < maxStalledTurns-1; i++ {
		b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{
			Bot: "eng", Outcome: noDurableTraceOutcome,
			At: time.Now().UTC().Format(time.RFC3339),
		})
	}
	// One turn that actually did something.
	b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{
		Bot: "eng", Outcome: "ok", Said: "fixed the fixture",
		At: time.Now().UTC().Format(time.RFC3339),
	})

	got := b.TaskByID(task.ID)
	if got.StalledTurns != 0 {
		t.Fatalf("a productive turn must reset the streak; got %d", got.StalledTurns)
	}

	// The next empty turn starts over at 1, not at the old count.
	stalled, crossed := b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{
		Bot: "eng", Outcome: noDurableTraceOutcome,
		At: time.Now().UTC().Format(time.RFC3339),
	})
	if stalled != 1 || crossed {
		t.Fatalf("after reset, first empty turn: stalled=%d crossed=%v, want 1/false", stalled, crossed)
	}
}

func TestStallCounterSurvivesLedgerRotation(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Rotated journal", "the counter must outlive the window", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}

	// Twice the ledger cap: the journal rotates, the counter must not.
	turns := taskLedgerMaxEntries * 2
	for i := 0; i < turns; i++ {
		b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{
			Bot: "eng", Outcome: noDurableTraceOutcome,
			At: time.Now().UTC().Format(time.RFC3339),
		})
	}

	got := b.TaskByID(task.ID)
	if len(got.Ledger) != taskLedgerMaxEntries {
		t.Fatalf("journal should rotate to %d entries; got %d", taskLedgerMaxEntries, len(got.Ledger))
	}
	if got.StalledTurns != turns {
		t.Fatalf("counter must not rotate with the journal: got %d, want %d", got.StalledTurns, turns)
	}
}

// The end-to-end direction: the launcher's own journal path must actually take
// the task out of rotation, not just increment a number.
func TestLauncherAutoCancelsAStalledTask(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Run skill: meeting-prep", "needs object_name", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}
	l := launcherForBrokerFixture(b)

	// Future timestamp: no message or action can fall inside the window, so
	// every turn assembles to "no durable trace" honestly rather than by
	// special-casing the ledger.
	startedAt := time.Now().UTC().Add(time.Hour)
	for i := 0; i < maxStalledTurns; i++ {
		l.recordTaskLedgerEntry("eng", headlessCodexTurn{TaskID: task.ID, Channel: task.Channel}, startedAt, nil)
	}

	got := b.TaskByID(task.ID)
	if got.status != "canceled" {
		t.Fatalf("a task that stalled %d times must be canceled; status=%q stalled=%d",
			maxStalledTurns, got.status, got.StalledTurns)
	}
	if !hasSchedulingCleared(*got) {
		t.Fatalf("auto-canceled task must not keep a wake schedule: follow_up=%q recheck=%q due=%q",
			got.FollowUpAt, got.RecheckAt, got.DueAt)
	}
	if got.Details == "" || !containsFold(got.Details, "no durable trace") {
		t.Fatalf("cancellation reason must be readable on the task itself; details=%q", got.Details)
	}
}

func hasSchedulingCleared(t teamTask) bool {
	return t.FollowUpAt == "" && t.RecheckAt == "" && t.ReminderAt == "" && t.DueAt == ""
}
