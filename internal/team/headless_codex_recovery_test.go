package team

// headless_codex_recovery_test.go — regression coverage for the transient
// provider-stream retry path. Live incident (2026-07-04, OFFICE-857): a
// provider connection drop mid-build killed the app-builder's headless
// claude session; the office-mode turn was never retried
// (shouldRetryHeadlessTurn returns false for non-worktree, non-external
// tasks), so the turn fell through to recoverFailedHeadlessTurn/BlockTask
// and a 60s build stalled for minutes with zero UI feedback. The queue must
// re-run the SAME turn immediately on a transient failure, emit a
// "reconnecting" HeadlessEvent (wire contract with the web build-activity
// feed), and never block the task — while NON-transient failures keep the
// old recovery path.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitForTurnError(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a headless runner turn")
		return nil
	}
}

// newOfficeModeTaskForTest creates a plain office-execution task owned by the
// app-builder (the live persona the incident hit). The owner is deliberately
// NOT a codingBotSlugs member and the title carries no external-integration
// keywords, so neither the durability guard nor the external-action rules
// interfere with the queue behavior under test.
func newOfficeModeTaskForTest(t *testing.T, b *Broker) teamTask {
	return newOfficeModeTaskForTestWithOwner(t, b, "app-builder")
}

// newOfficeModeTaskForTestWithOwner lets GENERIC office-contract tests pick a
// non-builder owner: the App Builder now has its own retry carve-out
// (resume-requeue on failure/timeout), so tests pinning the generic
// block-on-failure contract must not run under its slug.
func newOfficeModeTaskForTestWithOwner(t *testing.T, b *Broker, owner string) teamTask {
	t.Helper()
	task, reused, err := b.EnsurePlannedTask(plannedTaskInput{
		Channel:       "team",
		Title:         "Refresh the weekly metrics summary layout",
		Owner:         owner,
		CreatedBy:     "cos",
		TaskType:      "feature",
		ExecutionMode: "office",
	})
	if err != nil || reused {
		t.Fatalf("ensure planned task: %v reused=%v", err, reused)
	}
	return task
}

func taskByIDForTest(t *testing.T, b *Broker, id string) teamTask {
	t.Helper()
	for _, candidate := range b.AllTasks() {
		if candidate.ID == id {
			return candidate
		}
	}
	t.Fatalf("expected to find task %s", id)
	return teamTask{}
}

func TestHeadlessQueueRetriesOfficeTurnAfterTransientProviderError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	b := newTestBroker(t)
	task := newOfficeModeTaskForTest(t, b)

	var mu sync.Mutex
	var notifications []string
	turnDone := make(chan error, 4)
	setHeadlessCodexRunTurnForTest(t, func(_ *Launcher, _ context.Context, slug, notification string, _ ...string) error {
		if slug != "app-builder" {
			return nil
		}
		mu.Lock()
		notifications = append(notifications, notification)
		call := len(notifications)
		mu.Unlock()
		var turnErr error
		if call == 1 {
			// The live failure shape: the claude CLI dies on a dropped
			// provider stream and runHeadlessClaudeTurn returns the wrapped
			// cmd.Wait error carrying the provider's connection detail.
			turnErr = errors.New("exit status 1: Connection error.")
		}
		turnDone <- turnErr
		return turnErr
	})

	l := newHeadlessLauncherForTest(t)
	l.broker = b

	l.enqueueHeadlessCodexTurnRecord("app-builder", headlessCodexTurn{
		Prompt:  "Work the team build for #" + task.ID,
		Channel: task.Channel,
		TaskID:  task.ID,
	})

	if err := waitForTurnError(t, turnDone); err == nil {
		t.Fatal("expected the first scripted turn to fail with a transient error")
	}
	// Pre-fix this is where the test fails: the office-mode turn is never
	// retried (recoverFailedHeadlessTurn blocks the task instead), so a
	// second runner call never arrives.
	if err := waitForTurnError(t, turnDone); err != nil {
		t.Fatalf("expected the transient retry to succeed, got %v", err)
	}
	l.waitForHeadlessIdle(t)

	mu.Lock()
	got := append([]string(nil), notifications...)
	mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("expected exactly 2 runner calls (original + transient retry), got %d: %q", len(got), got)
	}
	if got[0] != got[1] {
		t.Fatalf("expected the transient retry to re-run the SAME turn, got %q then %q", got[0], got[1])
	}

	updated := taskByIDForTest(t, b, task.ID)
	if updated.Blocked() || updated.Status() == "blocked" {
		t.Fatalf("expected transient retry to keep the task unblocked, got status=%s blocked=%v", updated.Status(), updated.Blocked())
	}

	// The retry must be visible on the run's event stream (the same
	// task-scoped channel /apps/{id}/activity serves): one HeadlessEvent of
	// type "reconnecting" — an exact wire contract with the web feed — with
	// a turn_id and a short human note.
	lines := b.BotStream("app-builder").recentTask(task.ID)
	sawReconnecting := false
	for _, line := range lines {
		if !strings.Contains(line, `"type":"reconnecting"`) {
			continue
		}
		sawReconnecting = true
		if !strings.Contains(line, "Connection dropped") {
			t.Fatalf("expected reconnecting event to carry a human note, got %q", line)
		}
		if !strings.Contains(line, `"turn_id"`) {
			t.Fatalf("expected reconnecting event to carry a turn_id, got %q", line)
		}
	}
	if !sawReconnecting {
		t.Fatalf("expected a reconnecting HeadlessEvent on the task stream, got %q", lines)
	}
}

// TestHeadlessQueueFallsBackThroughTheWholeRing drives a turn whose provider
// keeps failing transiently: same-model reconnects first, then each model of
// the ring in order, once each. A free-model office lives on this — quotas
// throttle per model, so the second throttled model must hand over to the
// third instead of ending the turn.
func TestHeadlessQueueFallsBackThroughTheWholeRing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(headlessModelFallbackEnv, "opencode-go/iceman-free,opencode-go/beam-free")
	t.Setenv(headlessSilentTurnBudgetEnv, "0")

	b := newTestBroker(t)
	task := newOfficeModeTaskForTestWithOwner(t, b, "cmo")

	var mu sync.Mutex
	var models []string
	done := make(chan struct{}, 8)
	setHeadlessCodexRunTurnForTest(t, func(_ *Launcher, ctx context.Context, slug, _ string, _ ...string) error {
		if slug != "cmo" {
			return nil
		}
		mu.Lock()
		models = append(models, headlessModelOverride(ctx))
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
		return errors.New("stream error: 429 too many requests")
	})

	l := newHeadlessLauncherForTest(t)
	l.broker = b

	l.enqueueHeadlessCodexTurnRecord("cmo", headlessCodexTurn{
		Prompt:  "Work the team build for #" + task.ID,
		Channel: task.Channel,
		TaskID:  task.ID,
	})

	// One primary attempt, two same-model reconnects, one attempt per ring
	// model, then the failure path.
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			mu.Lock()
			got := append([]string(nil), models...)
			mu.Unlock()
			logBytes, _ := os.ReadFile(filepath.Join(wuphfLogDir(), "headless-codex-cmo.log"))
			t.Fatalf("expected attempt %d to run, saw models %v; log:\n%s", i+1, got, logBytes)
		}
	}
	l.waitForHeadlessIdle(t)

	mu.Lock()
	got := append([]string(nil), models...)
	mu.Unlock()
	want := []string{"", "opencode-go/iceman-free", "opencode-go/beam-free"}
	var seen []string
	for _, model := range got {
		if len(seen) == 0 || seen[len(seen)-1] != model {
			seen = append(seen, model)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("model order = %v (attempts %v), want %v", seen, got, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("model order = %v, want %v", seen, want)
		}
	}

	logBytes, err := os.ReadFile(filepath.Join(wuphfLogDir(), "headless-codex-cmo.log"))
	if err != nil {
		t.Fatalf("read headless log: %v", err)
	}
	logText := string(logBytes)
	for _, model := range []string{"opencode-go/iceman-free", "opencode-go/beam-free"} {
		if !strings.Contains(logText, "switching this turn to "+model) {
			t.Fatalf("expected the ring to reach %s, log:\n%s", model, logText)
		}
	}
}

// TestHeadlessQueueSilentStallCancelsEarlyAndRetries proves the zero-output
// budget: an attempt that never emits anything is cancelled well before the
// turn's own timeout, and the work moves to the fallback model. This is the
// only lever that shortens the Journey 17 stall (per-provider timeouts in
// opencode's config do not apply to its builtin providers).
func TestHeadlessQueueSilentStallCancelsEarlyAndRetries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WUPHF_HEADLESS_MODEL_FALLBACK", "opencode-go/iceman-contributor-free")
	t.Setenv(headlessSilentTurnBudgetEnv, "600ms")

	b := newTestBroker(t)
	task := newOfficeModeTaskForTestWithOwner(t, b, "cmo")

	var mu sync.Mutex
	calls := 0
	done := make(chan struct{}, 4)
	setHeadlessCodexRunTurnForTest(t, func(_ *Launcher, ctx context.Context, slug, _ string, _ ...string) error {
		if slug != "cmo" {
			return nil
		}
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
		if n > 1 {
			return nil // the fallback turn answers
		}
		<-ctx.Done() // never emits a single event: exactly the stall
		return ctx.Err()
	})

	l := newHeadlessLauncherForTest(t)
	l.broker = b

	l.enqueueHeadlessCodexTurnRecord("cmo", headlessCodexTurn{
		Prompt:  "Work the team build for #" + task.ID,
		Channel: task.Channel,
		TaskID:  task.ID,
	})

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("expected attempt %d to run", i+1)
		}
	}
	l.waitForHeadlessIdle(t)

	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls != 2 {
		t.Fatalf("expected the silent attempt plus one fallback attempt, got %d calls", gotCalls)
	}

	logBytes, err := os.ReadFile(filepath.Join(wuphfLogDir(), "headless-codex-cmo.log"))
	if err != nil {
		t.Fatalf("read headless log: %v", err)
	}
	logText := string(logBytes)
	if !strings.Contains(logText, "silent-stall: no model output in") {
		t.Fatalf("expected a silent-stall log line, got:\n%s", logText)
	}
	if !strings.Contains(logText, "retrying on opencode-go/iceman-contributor-free") {
		t.Fatalf("expected the retry to name the fallback model, got:\n%s", logText)
	}
}

// TestHeadlessQueueSilentTimeoutSwitchesToFallbackModel drives the real worker
// through the Journey 17 failure: the turn times out without producing a byte
// of model output. That turn must get one attempt on the configured fallback
// model instead of going straight to BlockTask.
func TestHeadlessQueueSilentTimeoutSwitchesToFallbackModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WUPHF_HEADLESS_MODEL_FALLBACK", "opencode-go/iceman-contributor-free")

	b := newTestBroker(t)
	task := newOfficeModeTaskForTestWithOwner(t, b, "cmo")

	var mu sync.Mutex
	calls := 0
	done := make(chan struct{}, 4)
	setHeadlessCodexRunTurnForTest(t, func(_ *Launcher, _ context.Context, slug, _ string, _ ...string) error {
		if slug != "cmo" {
			return nil
		}
		mu.Lock()
		calls++
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
		return context.DeadlineExceeded
	})

	l := newHeadlessLauncherForTest(t)
	l.broker = b

	l.enqueueHeadlessCodexTurnRecord("cmo", headlessCodexTurn{
		Prompt:  "Work the team build for #" + task.ID,
		Channel: task.Channel,
		TaskID:  task.ID,
	})

	// First attempt times out, then the fallback attempt starts.
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("expected attempt %d to run", i+1)
		}
	}
	l.waitForHeadlessIdle(t)

	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls != 2 {
		t.Fatalf("expected exactly one fallback attempt after the silent timeout, got %d calls", gotCalls)
	}

	logBytes, err := os.ReadFile(filepath.Join(wuphfLogDir(), "headless-codex-cmo.log"))
	if err != nil {
		t.Fatalf("read headless log: %v", err)
	}
	if !strings.Contains(string(logBytes), "retrying on opencode-go/iceman-contributor-free") {
		t.Fatalf("expected a model-fallback log line naming the fallback, got:\n%s", logBytes)
	}
}

func TestHeadlessQueueDoesNotRetryOfficeTurnAfterNonTransientError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	b := newTestBroker(t)
	task := newOfficeModeTaskForTestWithOwner(t, b, "cmo")

	var mu sync.Mutex
	calls := 0
	turnDone := make(chan error, 4)
	setHeadlessCodexRunTurnForTest(t, func(_ *Launcher, _ context.Context, slug, _ string, _ ...string) error {
		if slug != "cmo" {
			return nil
		}
		mu.Lock()
		calls++
		mu.Unlock()
		err := errors.New("app verify gate failed: missing export")
		turnDone <- err
		return err
	})

	l := newHeadlessLauncherForTest(t)
	l.broker = b

	l.enqueueHeadlessCodexTurnRecord("cmo", headlessCodexTurn{
		Prompt:  "Work the team build for #" + task.ID,
		Channel: task.Channel,
		TaskID:  task.ID,
	})

	if err := waitForTurnError(t, turnDone); err == nil {
		t.Fatal("expected the scripted turn to fail")
	}
	l.waitForHeadlessIdle(t)

	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls != 1 {
		t.Fatalf("expected a non-transient failure to run exactly once, got %d calls", gotCalls)
	}

	updated := taskByIDForTest(t, b, task.ID)
	if !updated.Blocked() && updated.Status() != "blocked" {
		t.Fatalf("expected non-transient failure to keep the BlockTask recovery path, got status=%s blocked=%v", updated.Status(), updated.Blocked())
	}

	for _, line := range b.BotStream("cmo").recentTask(task.ID) {
		if strings.Contains(line, `"type":"reconnecting"`) {
			t.Fatalf("expected no reconnecting event for a non-transient failure, got %q", line)
		}
	}
}

// TestHeadlessTurnEmittedNothing pins the silence check that routes a
// zero-event timeout to the fallback model. Journey 17: a rate-limited primary
// kept the runner alive for the whole turn budget without emitting anything,
// so the broker only saw a timeout and blocked the task.
func TestHeadlessTurnProducedNoOutput(t *testing.T) {
	start := time.Now().Add(-time.Minute)
	start = start.Truncate(time.Second)

	fresh := &botStreamBuffer{subs: make(map[int]botStreamSubscriber)}
	if !headlessTurnProducedNoOutput(fresh, "task-1", start) {
		t.Fatalf("empty stream should read as silent")
	}

	spoke := &botStreamBuffer{subs: make(map[int]botStreamSubscriber)}
	pushHeadlessEvent(spoke, HeadlessEvent{Type: HeadlessEventTypeText, Bot: "cos", TaskID: "task-1", Text: "hi"})
	if headlessTurnProducedNoOutput(spoke, "task-1", start) {
		t.Fatalf("a turn that emitted model output is not silent")
	}

	tooled := &botStreamBuffer{subs: make(map[int]botStreamSubscriber)}
	pushHeadlessEvent(tooled, HeadlessEvent{Type: HeadlessEventTypeToolUse, Bot: "cos", TaskID: "task-1", ToolName: "bash"})
	if headlessTurnProducedNoOutput(tooled, "task-1", start) {
		t.Fatalf("a tool call counts as model output")
	}

	// A dead child still emits an error + manifest pair; that bookkeeping must
	// not be mistaken for the model having spoken (this is what made the live
	// Journey 17 timeout look productive).
	bookkeeping := &botStreamBuffer{subs: make(map[int]botStreamSubscriber)}
	pushHeadlessEvent(bookkeeping, HeadlessEvent{Type: HeadlessEventTypeError, Bot: "cos", TaskID: "task-1", Text: "exit status 1"})
	pushHeadlessEvent(bookkeeping, HeadlessEvent{Type: HeadlessEventTypeManifest, Bot: "cos", TaskID: "task-1", Text: "exit status 1"})
	if !headlessTurnProducedNoOutput(bookkeeping, "task-1", start) {
		t.Fatalf("terminal bookkeeping events must not count as model output")
	}

	// Events from an EARLIER attempt must not make this attempt look alive.
	stale := &botStreamBuffer{subs: make(map[int]botStreamSubscriber)}
	pushHeadlessEvent(stale, HeadlessEvent{
		Type: HeadlessEventTypeText, Bot: "cos", TaskID: "task-1", Text: "old",
		StartedAt: start.Add(-5 * time.Minute).UTC().Format(time.RFC3339),
	})
	if !headlessTurnProducedNoOutput(stale, "task-1", start) {
		t.Fatalf("an event stamped before the attempt started must not count")
	}

	// No task id means nothing to judge; stay conservative (not silent).
	if headlessTurnProducedNoOutput(spoke, " ", start) {
		t.Fatalf("blank task id should not be classified as silent")
	}
}

func TestIsTransientProviderError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		// Live 2026-07-04 shapes from runHeadlessClaudeTurn's cmd.Wait wrap.
		{"claude connection error detail", errors.New("exit status 1: Connection error."), true},
		{"bare exit status wrap", errors.New("exit status 1: exit status 1"), true},
		{"bare exit status", errors.New("exit status 1"), true},
		{"connection reset by peer", errors.New("exit status 1: read tcp 10.0.0.5:52344->104.18.2.1:443: read: connection reset by peer"), true},
		{"node econnreset", errors.New("exit status 1: Error: read ECONNRESET"), true},
		{"socket hang up", errors.New("exit status 1: FetchError: socket hang up"), true},
		{"provider overloaded", errors.New(`exit status 1: API Error: 529 {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`), true},
		{"provider 5xx", errors.New("exit status 1: API Error: 500 internal server error"), true},
		// Chain-typed stream failures survive the parse-path wrap.
		{"wrapped unexpected EOF", fmt.Errorf("read claude json stream: %w", io.ErrUnexpectedEOF), true},
		{"wrapped closed pipe", fmt.Errorf("read claude json stream: %w", io.ErrClosedPipe), true},
		// Quota exhaustion (Journey 17): the real body from opencode-go.
		{"provider rate limit", errors.New(`exit status 1: {"error":{"type":"rate_limit_error","message":"Rate limit exceeded. Please try again later."}}`), true},
		{"rate limit underscore type", errors.New("stream error: rate_limit_error"), true},
		{"too many requests", errors.New("http 429 Too Many Requests"), true},
		// Non-transient failures keep the old recovery path.
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"max turns", errors.New("Reached maximum number of turns (15)"), false},
		{"killed", errors.New("signal: killed: signal: killed"), false},
		{"cli missing", errors.New(`claude not found: exec: "claude": executable file not found in $PATH`), false},
		{"detailed exit failure", errors.New("exit status 1: app verify gate failed: missing export"), false},
		{"durability reason", errors.New("coding turn for #task-1 completed without durable task state or completion evidence"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTransientProviderError(tc.err); got != tc.want {
				t.Fatalf("isTransientProviderError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestShouldRetryHeadlessTurnAllowsOneTransientOfficeRetry(t *testing.T) {
	office := &teamTask{ID: "task-1", Title: "Refresh the weekly metrics summary layout", ExecutionMode: "office"}

	if !shouldRetryHeadlessTurn(office, headlessCodexTurn{Attempts: 0}, true) {
		t.Fatal("expected a transient office failure to earn one recovery retry")
	}
	if shouldRetryHeadlessTurn(office, headlessCodexTurn{Attempts: 1}, true) {
		t.Fatal("expected the transient office retry to be capped at one")
	}
	if shouldRetryHeadlessTurn(office, headlessCodexTurn{Attempts: 0}, false) {
		t.Fatal("expected a non-transient office failure to keep the no-retry path")
	}

	worktree := &teamTask{ID: "task-2", Title: "Implement queue mode", ExecutionMode: "local_worktree"}
	if !shouldRetryHeadlessTurn(worktree, headlessCodexTurn{Attempts: 0}, false) {
		t.Fatal("expected worktree retry behavior to be unchanged")
	}
}

// 2026-08-16 fresh-workspace QA regression: a timed-out App Builder BUILD
// turn must requeue promptly with a resume prompt — the old path blocked the
// task into the slow self-heal lane, and the operator watched "Building" for
// 41 silent minutes before a recovery turn restarted the build from scratch.
func TestRecoverTimedOutHeadlessTurnRequeuesAppBuilderBuild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	b := newTestBroker(t)
	task := newOfficeModeTaskForTest(t, b)

	l := newHeadlessLauncherForTest(t)
	l.broker = b

	turn := headlessCodexTurn{
		Prompt:  "Build the Chase Bot app for #" + task.ID,
		Channel: task.Channel,
		TaskID:  task.ID,
	}
	lane := l.laneForTurn("app-builder", turn)
	l.headless.mu.Lock()
	l.headless.workers[lane] = true // keep the queue inspectable: no worker spawns
	l.headless.mu.Unlock()

	l.recoverTimedOutHeadlessTurn("app-builder", turn, time.Now().UTC().Add(-2*time.Second), 10*time.Minute)

	l.headless.mu.Lock()
	queued := append([]headlessCodexTurn(nil), l.headless.queues[lane]...)
	l.headless.mu.Unlock()
	if len(queued) != 1 {
		t.Fatalf("expected one queued timeout-recovery retry for the build, got %+v", queued)
	}
	retry := queued[0]
	if retry.Attempts != 1 {
		t.Fatalf("expected retry attempt 1, got %+v", retry)
	}
	if !strings.Contains(retry.Prompt, "RESUME from it") {
		t.Fatalf("expected the resume-not-restart note in the retry prompt, got %q", retry.Prompt)
	}

	updated := taskByIDForTest(t, b, task.ID)
	if updated.Blocked() || updated.Status() == "blocked" {
		t.Fatalf("expected the build task to stay active during the prompt retry, got status=%s blocked=%v", updated.Status(), updated.Blocked())
	}

	// Simulate the retry running and timing out again: drain the queue (a
	// pending queued turn reads as recovery-in-progress, correctly masking a
	// duplicate requeue) and recover with the retried turn. Budget spent
	// (attempts >= 2): the old BlockTask fallback takes over.
	l.headless.mu.Lock()
	l.headless.queues[lane] = nil
	l.headless.mu.Unlock()
	second := retry
	second.Attempts = 2
	l.recoverTimedOutHeadlessTurn("app-builder", second, time.Now().UTC().Add(time.Second), 10*time.Minute)
	updated = taskByIDForTest(t, b, task.ID)
	if !updated.Blocked() && updated.Status() != "blocked" {
		t.Fatalf("expected the budget-spent timeout to fall back to BlockTask, got status=%s blocked=%v", updated.Status(), updated.Blocked())
	}
}

func TestRecoverFailedHeadlessTurnRetriesOfficeTaskOnceOnTransientFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	b := newTestBroker(t)
	task := newOfficeModeTaskForTestWithOwner(t, b, "cmo")

	l := newHeadlessLauncherForTest(t)
	l.broker = b

	turn := headlessCodexTurn{
		Prompt:  "Work the team build for #" + task.ID,
		Channel: task.Channel,
		TaskID:  task.ID,
	}
	lane := l.laneForTurn("cmo", turn)
	l.headless.mu.Lock()
	l.headless.workers[lane] = true // keep the queue inspectable: no worker spawns
	l.headless.mu.Unlock()

	detail := "exit status 1: read tcp 10.0.0.5:52344->104.18.2.1:443: read: connection reset by peer"
	l.recoverFailedHeadlessTurn("cmo", turn, time.Now().UTC().Add(-2*time.Second), detail)

	l.headless.mu.Lock()
	queued := append([]headlessCodexTurn(nil), l.headless.queues[lane]...)
	l.headless.mu.Unlock()
	if len(queued) != 1 {
		t.Fatalf("expected one queued transient recovery retry, got %+v", queued)
	}
	retry := queued[0]
	if retry.Attempts != 1 {
		t.Fatalf("expected recovery retry attempt 1, got %+v", retry)
	}
	if !strings.Contains(retry.Prompt, "Previous attempt by @cmo failed") {
		t.Fatalf("expected retry prompt note, got %q", retry.Prompt)
	}

	updated := taskByIDForTest(t, b, task.ID)
	if updated.Blocked() || updated.Status() == "blocked" {
		t.Fatalf("expected task to remain active during transient recovery retry, got status=%s blocked=%v", updated.Status(), updated.Blocked())
	}

	// Second transient failure of the recovery retry: the one-retry budget is
	// spent, so the old BlockTask path takes over.
	l.recoverFailedHeadlessTurn("cmo", retry, time.Now().UTC().Add(-1*time.Second), detail)
	updated = taskByIDForTest(t, b, task.ID)
	if !updated.Blocked() && updated.Status() != "blocked" {
		t.Fatalf("expected the second transient failure to block the task, got status=%s blocked=%v", updated.Status(), updated.Blocked())
	}
}

func TestHeadlessReplyToIDUsesInstructionLine(t *testing.T) {
	notification := "--- packet history ---\n- @cos (channel: \"cos__human\", reply_to_id: \"msg-old\") history text\n---\n[New from @human]: question\nReply using team_broadcast with my_slug \"cos\" and channel \"cos__human\" reply_to_id \"msg-new\". Once you have posted."
	if got := headlessReplyToID(notification); got != "msg-new" {
		t.Fatalf("expected trigger reply id msg-new, got %q", got)
	}
}

func TestHeadlessReplyToIDEmptyWithoutInstruction(t *testing.T) {
	notification := "--- packet history ---\n- @cos (channel: \"cos__human\", reply_to_id: \"msg-old\") history text\n"
	if got := headlessReplyToID(notification); got != "" {
		t.Fatalf("expected empty without instruction line, got %q", got)
	}
}
