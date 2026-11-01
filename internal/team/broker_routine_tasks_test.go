package team

// Tests for the routine-as-task fire path (broker_routine_tasks.go). An agent
// routine used to post a bare chat message; nothing carried its artifact and
// distillation never ran for a daily brief. These pin the replacement contract:
// one task per fire, no duplicate message, dedupe on a replayed slot, and a
// message fallback (recorded as a failed run) when task creation breaks.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func agentRoutineJob() schedulerJob {
	return schedulerJob{
		Slug:       "5",
		Kind:       "agent_routine",
		Label:      "每个交易日早上中文简报",
		TargetType: "agent",
		TargetID:   "forex-watch",
		Channel:    "forex-watch__human",
		NextRun:    "2026-11-02T08:30:00Z",
		Payload:    "每日外汇简报（全中文）。",
		Status:     "scheduled",
		Enabled:    true,
	}
}

func TestProcessBotJob_AgentRoutineCreatesTaskInsteadOfMessage(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	b.dueJobs = []schedulerJob{agentRoutineJob()}
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Date(2026, 11, 2, 8, 30, 0, 0, time.UTC))}

	sched.processOnce()

	if len(b.routineTasks) != 1 {
		t.Fatalf("expected exactly one routine task; got %+v", b.routineTasks)
	}
	req := b.routineTasks[0]
	if req.Action != "create" {
		t.Errorf("expected action=create; got %q", req.Action)
	}
	if req.Owner != "forex-watch" || req.Channel != "forex-watch__human" {
		t.Errorf("task must target the routine owner's channel; got owner=%q channel=%q", req.Owner, req.Channel)
	}
	if req.CreatedBy != routineTaskActor {
		t.Errorf("routine task must be created by an internal actor (skips the plan gate); got %q", req.CreatedBy)
	}
	if req.Title != "每个交易日早上中文简报" {
		t.Errorf("expected the routine label as title; got %q", req.Title)
	}
	if !strings.Contains(req.Details, "每日外汇简报") {
		t.Errorf("expected the routine payload as details; got %q", req.Details)
	}
	if req.SourceSignalID != routineTaskSignal("5", "2026-11-02T08:30:00Z") {
		t.Errorf("expected the due slot as dedupe signal; got %q", req.SourceSignalID)
	}
	// The task dispatch is the wake. A second message would run the owner twice,
	// the first time with no task attached.
	if len(b.automationPosts) != 0 {
		t.Errorf("agent routine must not also post a routine message; got %+v", b.automationPosts)
	}
	if last := b.jobStateUpdates[len(b.jobStateUpdates)-1]; last.status != "scheduled" {
		t.Errorf("expected the job re-armed as scheduled; got %+v", last)
	}
}

func TestProcessBotJob_AgentRoutineDedupesReplayedSlot(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	b.dueJobs = []schedulerJob{agentRoutineJob()}
	b.routineSignals = map[string]string{
		routineTaskSignal("5", "2026-11-02T08:30:00Z"): "ACMER-42",
	}
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Date(2026, 11, 2, 8, 30, 5, 0, time.UTC))}

	sched.processOnce()

	if len(b.routineTasks) != 0 {
		t.Fatalf("a replayed slot must not create a second task; got %+v", b.routineTasks)
	}
	if len(b.automationPosts) != 0 {
		t.Errorf("deduped fire must not post a message either; got %+v", b.automationPosts)
	}
}

func TestProcessBotJob_AgentRoutineFallsBackToMessageOnCreateFailure(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	b.dueJobs = []schedulerJob{agentRoutineJob()}
	b.routineCreatErr = errors.New("owner is not a registered bot")
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Date(2026, 11, 2, 8, 30, 0, 0, time.UTC))}

	sched.processOnce()

	if len(b.automationPosts) != 1 {
		t.Fatalf("expected the message fallback to fire once; got %+v", b.automationPosts)
	}
	if post := b.automationPosts[0]; post.channel != "forex-watch__human" || !strings.Contains(post.body, "每日外汇简报") {
		t.Errorf("fallback message must carry the routine body; got %+v", post)
	}
}

// A routine pointing at a room that no longer exists must fail loudly and NOT
// post: the fallback would land in that same missing room, wake the owner for
// nothing, and look like a delivered routine in the Runs tab.
func TestProcessBotJob_AgentRoutineSkipsFallbackWhenChannelIsGone(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	job := agentRoutineJob()
	job.Channel = "general"
	b.dueJobs = []schedulerJob{job}
	b.routineCreatErr = &TaskMutationError{Kind: TaskMutationNotFound, Message: "channel not found"}
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Date(2026, 11, 2, 8, 30, 0, 0, time.UTC))}

	sched.processOnce()

	if len(b.automationPosts) != 0 {
		t.Fatalf("a missing channel must not get a fallback message; got %+v", b.automationPosts)
	}
	if len(b.completedRuns) != 1 {
		t.Fatalf("expected the fire to record one run; got %+v", b.completedRuns)
	}
	run := b.completedRuns[0]
	if run.Status != "failed" {
		t.Errorf("a routine that could not be delivered must be a failed run; got %q", run.Status)
	}
	if !strings.Contains(run.Message, "channel not found") {
		t.Errorf("the run must carry the create error; got %q", run.Message)
	}
	if last := b.jobStateUpdates[len(b.jobStateUpdates)-1]; last.status == "disabled" {
		t.Errorf("a delivery failure must not silently disable the routine; got %+v", last)
	}
}

func TestProcessBotJob_NonAgentRoutineStillPostsMessage(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	job := agentRoutineJob()
	job.Kind = ""
	b.dueJobs = []schedulerJob{job}
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Date(2026, 11, 2, 8, 30, 0, 0, time.UTC))}

	sched.processOnce()

	if len(b.routineTasks) != 0 {
		t.Errorf("a non-agent job must keep the message path; got %+v", b.routineTasks)
	}
	if len(b.automationPosts) != 1 {
		t.Fatalf("expected one routine message; got %+v", b.automationPosts)
	}
}

// TestCreateRoutineTaskLandsDispatchableTask pins what the scheduler relies on
// when it hands a routine to the task engine: the created task is owned, sits
// in an executable lifecycle state (otherwise sendTaskUpdate drops the wake and
// the routine silently stops producing), and its signal makes a replayed slot
// dedupe.
func TestCreateRoutineTaskLandsDispatchableTask(t *testing.T) {
	b := newTestBroker(t)
	b.members = []officeMember{{Slug: "forex-watch", Name: "Forex Watch"}}
	b.channels = []teamChannel{{Slug: "forex-watch__human", Name: "Forex Watch DM", Members: []string{"human", "forex-watch"}}}

	signal := routineTaskSignal("5", "2026-11-02T08:30:00Z")
	resp, err := b.CreateRoutineTask(TaskPostRequest{
		Action:         "create",
		Channel:        "forex-watch__human",
		Title:          "每个交易日早上中文简报",
		Details:        "每日外汇简报（全中文）。",
		Owner:          "forex-watch",
		CreatedBy:      routineTaskActor,
		SourceSignalID: signal,
	})
	if err != nil {
		t.Fatalf("routine task create failed: %v", err)
	}
	task := resp.Task
	if task.ID == "" {
		t.Fatal("expected a task id")
	}
	if task.Owner != "forex-watch" {
		t.Errorf("expected owner forex-watch; got %q", task.Owner)
	}
	if !isExecutableTeamTaskStatus(task.LifecycleState) {
		t.Errorf("routine task must land in an executable lifecycle state, else the wake is dropped; got %q (status %q)", task.LifecycleState, task.Status())
	}
	if id, ok := b.RoutineTaskForSignal(signal); !ok || id != task.ID {
		t.Errorf("expected the signal to resolve to %q; got %q ok=%v", task.ID, id, ok)
	}
}

// TestRunTaskVerification_WikiArtifact pins the check that turns a routine
// completion into a verified outcome (the prerequisite for distillation).
func TestRunTaskVerification_WikiArtifact(t *testing.T) {
	wiki := t.TempDir()
	if err := os.MkdirAll(filepath.Join(wiki, "artifacts", "forex"), 0o755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(wiki, "artifacts", "forex", "2026-11-02.md")
	if err := os.WriteFile(good, []byte("# brief\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(wiki, "artifacts", "forex", "empty.md")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	v := TaskVerification{Kind: taskVerificationKindWikiArtifact, Spec: "artifact_path declared on complete", Required: true}

	cases := []struct {
		name     string
		artifact string
		wantPass bool
		wantSub  string
	}{
		{"declared deliverable exists", "artifacts/forex/2026-11-02.md", true, "wiki artifact"},
		{"no artifact declared", "", false, "no artifact_path"},
		{"artifact missing", "artifacts/forex/2026-11-03.md", false, "is not in the wiki"},
		{"artifact empty", "artifacts/forex/empty.md", false, "is empty"},
		{"path escapes the wiki", "../../etc/passwd", false, "escapes the wiki root"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runTaskVerificationIn(v, taskVerificationEnv{WikiRoot: wiki, Artifact: tc.artifact})
			if res.Pass != tc.wantPass {
				t.Fatalf("pass=%v want %v (detail %q)", res.Pass, tc.wantPass, res.Detail)
			}
			if !strings.Contains(res.Detail, tc.wantSub) {
				t.Errorf("detail %q does not mention %q", res.Detail, tc.wantSub)
			}
		})
	}
}

// TestRoutineTaskCarriesVerifiedArtifactGate proves the scheduler asks for the
// check, so a routine completion cannot land unverified and skip distillation.
func TestRoutineTaskCarriesVerifiedArtifactGate(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	b.dueJobs = []schedulerJob{agentRoutineJob()}
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Date(2026, 11, 2, 8, 30, 0, 0, time.UTC))}

	sched.processOnce()

	if len(b.routineTasks) != 1 {
		t.Fatalf("expected one routine task; got %+v", b.routineTasks)
	}
	req := b.routineTasks[0]
	if req.VerificationKind != taskVerificationKindWikiArtifact || !req.VerificationRequired {
		t.Fatalf("routine task must carry a required wiki_artifact check; got kind=%q required=%v", req.VerificationKind, req.VerificationRequired)
	}
}
