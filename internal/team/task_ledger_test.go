package team

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendTaskLedgerEntryCapsAndPersists(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Long-running build", "many turns", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < taskLedgerMaxEntries+5; i++ {
		b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{Bot: "eng", Outcome: "ok"})
	}
	got := b.TaskByID(task.ID)
	if len(got.Ledger) != taskLedgerMaxEntries {
		t.Fatalf("want cap %d; got %d", taskLedgerMaxEntries, len(got.Ledger))
	}
}

func TestRecordTaskLedgerEntryAssemblesFromBrokerFacts(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Instrumented work", "watch the journal", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}
	l := launcherForBrokerFixture(b)
	startedAt := time.Now().UTC().Add(-1 * time.Second)

	if _, err := b.PostMessage("eng", task.Channel, "Tried the flag approach; tests still red on auth_test.go", nil, ""); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	b.appendActionLocked("task_updated", "office", task.Channel, "eng", "marked blocked on flaky fixture", task.ID)
	b.mu.Unlock()

	l.recordTaskLedgerEntry("eng", headlessCodexTurn{TaskID: task.ID, Channel: task.Channel}, startedAt, errors.New("turn timed out after 20m"))

	got := b.TaskByID(task.ID)
	if len(got.Ledger) != 1 {
		t.Fatalf("want 1 entry; got %d", len(got.Ledger))
	}
	e := got.Ledger[0]
	if e.Bot != "eng" || !strings.Contains(e.Outcome, "timed out") {
		t.Fatalf("entry bot/outcome wrong: %+v", e)
	}
	if !strings.Contains(e.Said, "flag approach") {
		t.Fatalf("entry must carry the bot's last message; got %q", e.Said)
	}
	if len(e.Actions) == 0 || !strings.Contains(e.Actions[0], "flaky fixture") {
		t.Fatalf("entry must carry the task mutations; got %v", e.Actions)
	}

	packet := l.notifyCtx().BuildTaskExecutionPacket("eng", officeActionLog{Actor: "cos"}, *got, "Next attempt.")
	if !strings.Contains(packet, "TASK JOURNAL") || !strings.Contains(packet, "flag approach") {
		t.Fatalf("packet must carry the journal; got:\n%s", packet)
	}
}

func TestRecordTaskLedgerEntrySkipsTasklessTurns(t *testing.T) {
	b := newVerificationTestBroker(t)
	l := launcherForBrokerFixture(b)
	l.recordTaskLedgerEntry("eng", headlessCodexTurn{Channel: "team"}, time.Now(), nil)
	// Nothing to assert on a specific task — the contract is simply that a
	// task-less turn must not panic or write anywhere.
}

// TestRecordTaskLedgerEntryCarriesContextUsed pins the B4 transparency
// contract: the packet-build manifest stamped on the turn reaches the
// ledger entry verbatim and survives the JSON wire (additive field).
func TestRecordTaskLedgerEntryCarriesContextUsed(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Context audit work", "track what was injected", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}
	l := launcherForBrokerFixture(b)
	manifest := []string{"learning:l-1", "wiki:companies/acme", "upstream:OFFICE-2"}
	l.recordTaskLedgerEntry("eng", headlessCodexTurn{TaskID: task.ID, Channel: task.Channel, ContextUsed: manifest}, time.Now().UTC(), nil)

	got := b.TaskByID(task.ID)
	if len(got.Ledger) != 1 {
		t.Fatalf("want 1 entry; got %d", len(got.Ledger))
	}
	if strings.Join(got.Ledger[0].ContextUsed, ",") != strings.Join(manifest, ",") {
		t.Fatalf("context manifest lost: %v", got.Ledger[0].ContextUsed)
	}

	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blob), `"context_used"`) {
		t.Fatalf("ledger entry wire must carry context_used: %s", blob)
	}
	var roundTripped teamTask
	if err := json.Unmarshal(blob, &roundTripped); err != nil {
		t.Fatal(err)
	}
	if strings.Join(roundTripped.Ledger[0].ContextUsed, ",") != strings.Join(manifest, ",") {
		t.Fatalf("context manifest lost on round-trip: %v", roundTripped.Ledger[0].ContextUsed)
	}
}

// TestIssueActivityCarriesTurnEvents pins the Activity-rail surface: every
// settled turn appears as a kind="turn" event carrying its context manifest.
func TestIssueActivityCarriesTurnEvents(t *testing.T) {
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Activity rail work", "surface the turns", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}
	b.AppendTaskLedgerEntry(task.ID, TaskLedgerEntry{
		Bot: "eng", Outcome: "ok", Said: "shipped the slice",
		ContextUsed: []string{"learning:l-9", "wiki:people/sam"},
	})
	b.mu.Lock()
	events := b.collectIssueActivityLocked(task.ID)
	b.mu.Unlock()
	var turn *IssueActivityEvent
	for i := range events {
		if events[i].Kind == IssueActivityKindTurn {
			turn = &events[i]
		}
	}
	if turn == nil {
		t.Fatalf("no turn event in activity feed: %+v", events)
	}
	if turn.Actor != "eng" || turn.Detail != "shipped the slice" {
		t.Fatalf("turn event shape wrong: %+v", turn)
	}
	if strings.Join(turn.ContextUsed, ",") != "learning:l-9,wiki:people/sam" {
		t.Fatalf("turn event lost the context manifest: %v", turn.ContextUsed)
	}
}

// Journey 18: the office replied "已按反馈修改" twice while the file never moved,
// and because it posted a message each time the ledger counted both turns as
// progress. A claim is not a trace — the broker compares the deliverable set
// against what the send-back rejected, so the journal says what happened.
func TestTurnThatClaimsAReworkWithoutWritingOneIsNotATrace(t *testing.T) {
	t.Setenv("WUPHF_RUNTIME_HOME", t.TempDir())
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Journey 18 release notes", "write the public page", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}
	const artifact = "team/reviews/rework-probe.md"
	full := filepath.Join(WikiRootDir(), filepath.FromSlash(artifact))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("- five bullets\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	b.mu.Lock()
	b.taskByIDLocked(task.ID).Artifact = artifact
	b.mu.Unlock()
	b.stampSentBackFingerprint(task.ID, "2026-11-12T00:14:59Z")

	l := launcherForBrokerFixture(b)
	startedAt := time.Now().UTC().Add(-1 * time.Second)
	if _, err := b.PostMessage("eng", task.Channel, "已按反馈修改，第 5 条已删除（新字节数 475）", nil, ""); err != nil {
		t.Fatalf("post: %v", err)
	}
	l.recordTaskLedgerEntry("eng", headlessCodexTurn{TaskID: task.ID, Channel: task.Channel}, startedAt, nil)

	got := b.TaskByID(task.ID)
	if len(got.Ledger) != 1 {
		t.Fatalf("want 1 ledger entry, got %d", len(got.Ledger))
	}
	if got.Ledger[0].Outcome != reworkUnwrittenOutcome {
		t.Fatalf("a message claiming a rework with the same bytes is not progress; outcome=%q", got.Ledger[0].Outcome)
	}
	if got.StalledTurns != 1 {
		t.Fatalf("a claimed-but-unwritten rework must count against the stall streak, got %d", got.StalledTurns)
	}
	rowFound := false
	for _, a := range b.ActionsSince(startedAt.UTC().Add(-time.Minute).Format(time.RFC3339)) {
		if a.Kind == "rework_unwritten" && a.RelatedID == task.ID {
			rowFound = true
		}
	}
	if !rowFound {
		t.Fatal("the office log must carry a rework_unwritten row")
	}
	packet := l.notifyCtx().BuildTaskExecutionPacket("eng", officeActionLog{Actor: "cos"}, *got, "Next attempt.")
	if !strings.Contains(packet, "rework-unwritten") {
		t.Fatalf("the next turn must be told the rework is not on disk; packet:\n%s", packet)
	}

	// A real write clears it: the same turn shape becomes ordinary progress.
	if err := os.WriteFile(full, []byte("- four bullets\n"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if _, err := b.PostMessage("eng", task.Channel, "改完了", nil, ""); err != nil {
		t.Fatalf("post: %v", err)
	}
	l.recordTaskLedgerEntry("eng", headlessCodexTurn{TaskID: task.ID, Channel: task.Channel}, time.Now().UTC().Add(-1*time.Second), nil)
	got = b.TaskByID(task.ID)
	if last := got.Ledger[len(got.Ledger)-1]; last.Outcome != "ok" {
		t.Fatalf("after a real write the outcome must be ok, got %q", last.Outcome)
	}
	if got.StalledTurns != 0 {
		t.Fatalf("a real write must reset the streak, got %d", got.StalledTurns)
	}
}

// Waiting on the human is not dodging the rework: while a request anchored to
// the task is open, the turn is not labelled.
func TestClaimedReworkIsNotFlaggedWhileWaitingOnTheHuman(t *testing.T) {
	t.Setenv("WUPHF_RUNTIME_HOME", t.TempDir())
	b := newVerificationTestBroker(t)
	task, _, err := b.EnsureTask("team", "Bounced page", "rework it", "eng", "cos", "")
	if err != nil {
		t.Fatal(err)
	}
	const artifact = "team/reviews/waiting-probe.md"
	full := filepath.Join(WikiRootDir(), filepath.FromSlash(artifact))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("v1"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	b.mu.Lock()
	b.taskByIDLocked(task.ID).Artifact = artifact
	b.mu.Unlock()
	b.stampSentBackFingerprint(task.ID, "2026-11-12T00:14:59Z")

	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.requests = append(b.requests, humanInterview{
		ID: "request-waiting-1", Kind: "interview", Status: "pending", From: "eng",
		Channel: task.Channel, Title: "Which tone?", Question: "Which tone?",
		IssueID: task.ID, CreatedAt: now, UpdatedAt: now,
	})
	b.mu.Unlock()
	l := launcherForBrokerFixture(b)
	startedAt := time.Now().UTC().Add(-1 * time.Second)
	if _, err := b.PostMessage("eng", task.Channel, "我这边等你的答复再改", nil, ""); err != nil {
		t.Fatalf("post: %v", err)
	}
	l.recordTaskLedgerEntry("eng", headlessCodexTurn{TaskID: task.ID, Channel: task.Channel}, startedAt, nil)
	got := b.TaskByID(task.ID)
	if last := got.Ledger[len(got.Ledger)-1]; last.Outcome == reworkUnwrittenOutcome {
		t.Fatal("a lane waiting on an open human request must not be labelled a fake rework")
	}
}
