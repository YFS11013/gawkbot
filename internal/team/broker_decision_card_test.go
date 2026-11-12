package team

// broker_decision_card_test.go covers the Journey 9 D2 gap: a lane that lands
// in `decision` owes the human a call, and until now the only surface that said
// so was a web inbox row. JOURN-16 sat in that state for the rest of the run
// with pending_interview=null, which is what the run's 26-minute "silence"
// (F-H) actually was.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// decisionCardFixture seeds one lane in review with a deliverable and drives
// convergence, which is the path every real lane takes into `decision`.
func decisionCardFixture(t *testing.T, artifact string) (*Broker, string) {
	t.Helper()
	clk := newRoutingFakeClockAt(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	b := newTestBroker(t)
	b.nowFn = clk.Now
	seedTaskInReview(t, b, "task-1", nil, 600)
	b.mu.Lock()
	task := b.taskByIDLocked("task-1")
	task.Owner = "eng"
	task.Artifact = artifact
	b.mu.Unlock()
	if err := b.EvaluateConvergence("task-1"); err != nil {
		t.Fatalf("evaluate convergence: %v", err)
	}
	return b, "task-1"
}

func decisionCards(b *Broker) []humanInterview {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []humanInterview
	for _, req := range b.requests {
		if requestIsDecisionCard(req) {
			out = append(out, cloneHumanInterview(req))
		}
	}
	return out
}

func TestConvergenceRaisesOneNonBlockingDecisionCard(t *testing.T) {
	b, taskID := decisionCardFixture(t, "team/reviews/acme-renewal-check.md")

	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("convergence must raise exactly one decision card, got %d", len(cards))
	}
	card := cards[0]
	if card.IssueID != taskID {
		t.Fatalf("the card must be anchored to the lane, got issue_id=%q", card.IssueID)
	}
	if card.Blocking || card.Required {
		t.Fatalf("the card must not gate the office: blocking=%v required=%v", card.Blocking, card.Required)
	}
	if card.DedupeKey != decisionCardDedupePrefix+taskID {
		t.Fatalf("dedupe key: want %q, got %q", decisionCardDedupePrefix+taskID, card.DedupeKey)
	}
	if !strings.Contains(card.Question, "team/reviews/acme-renewal-check.md") {
		t.Fatalf("the ask must name what is being approved, got %q", card.Question)
	}
	if len(card.Options) != 2 {
		t.Fatalf("want approve + send back, got %+v", card.Options)
	}
	// Non-blocking means no bot is parked and pendingInterview stays clear.
	b.mu.Lock()
	pending := b.pendingInterview
	b.mu.Unlock()
	if pending != nil {
		t.Fatalf("a non-blocking decision card must not become pendingInterview, got %q", pending.ID)
	}
	// The ask must be visible where the human works, not only in the inbox.
	b.mu.Lock()
	announced := false
	for _, msg := range b.messages {
		if msg.Channel == normalizeChannelSlug(card.Channel) && strings.Contains(msg.Content, card.ID) {
			announced = true
			break
		}
	}
	b.mu.Unlock()
	if !announced {
		t.Fatalf("the card must announce itself in %q so the human can answer without hunting", card.Channel)
	}

	// The 30-second sweeper re-evaluates every review-state lane; a repeated
	// convergence pass must not stack a second card.
	b.SweepReviewConvergence()
	b.SweepReviewConvergence()
	// A second evaluation for the same lane (a race between the sweeper and a
	// grade append, or a restart replaying convergence) must land on the live
	// card instead of asking the same question twice.
	b.mu.Lock()
	b.raiseDecisionCardAfterConvergenceLocked(taskID)
	b.mu.Unlock()
	if got := len(decisionCards(b)); got != 1 {
		t.Fatalf("repeated convergence must not stack cards, got %d", got)
	}
}

func TestDecisionCardSkippedWithoutADeliverable(t *testing.T) {
	b, _ := decisionCardFixture(t, "")
	if cards := decisionCards(b); len(cards) != 0 {
		t.Fatalf("a lane with nothing to accept must not ask for acceptance, got %+v", cards)
	}
}

func TestDecisionCardApproveClosesTheLane(t *testing.T) {
	b, taskID := decisionCardFixture(t, "team/reviews/acme-renewal-check.md")
	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("want one card, got %d", len(cards))
	}

	if status, msg := b.answerRequestFromActor("you", cards[0].ID, decisionChoiceApprove, "", ""); status != 200 {
		t.Fatalf("answer: %d %s", status, msg)
	}
	task := b.TaskByID(taskID)
	if task == nil {
		t.Fatal("task vanished")
	}
	if !strings.EqualFold(strings.TrimSpace(task.status), "done") {
		t.Fatalf("approving the card must close the lane, got status=%q review=%q", task.status, task.reviewState)
	}
	if task.LifecycleState != LifecycleStateApproved {
		t.Fatalf("approving the card must land the decision state, got %q — a done status on a lane still in the decision bucket is the split the Inbox keeps listing", task.LifecycleState)
	}
	// One-shot: the answer retires the card, so a second click cannot exist.
	b.mu.Lock()
	active := requestIsActive(findRequestLocked(b.requests, cards[0].ID))
	b.mu.Unlock()
	if active {
		t.Fatal("answering the card must retire it")
	}
}

// findRequestLocked returns the request with this id, or a zero-value request
// when absent (requestIsActive reports false for it).
func findRequestLocked(reqs []humanInterview, id string) humanInterview {
	for _, req := range reqs {
		if req.ID == id {
			return req
		}
	}
	return humanInterview{}
}

func TestDecisionCardSendBackBouncesToTheOwner(t *testing.T) {
	b, taskID := decisionCardFixture(t, "team/reviews/acme-renewal-check.md")
	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("want one card, got %d", len(cards))
	}
	const note = "cut the pricing table down to the two options we sell"

	if status, msg := b.answerRequestFromActor("you", cards[0].ID, decisionChoiceChanges, "", note); status != 200 {
		t.Fatalf("answer: %d %s", status, msg)
	}
	task := b.TaskByID(taskID)
	if task == nil {
		t.Fatal("task vanished")
	}
	if strings.EqualFold(strings.TrimSpace(task.status), "done") {
		t.Fatalf("sending it back must not close the lane")
	}
	if task.LifecycleState != LifecycleStateChangesRequested {
		t.Fatalf("want lifecycle_state=changes_requested, got %q", task.LifecycleState)
	}
	if task.ChangesRequested == nil || !strings.Contains(task.ChangesRequested.Body, "pricing table") {
		t.Fatalf("the human's note must reach the owner as the rework, got %+v", task.ChangesRequested)
	}
	if task.Artifact != "team/reviews/acme-renewal-check.md" {
		t.Fatalf("sending back must leave the deliverable in place, got %q", task.Artifact)
	}
}

func TestDecisionCardIsCancelledWhenTheLaneCloses(t *testing.T) {
	b, taskID := decisionCardFixture(t, "team/reviews/acme-renewal-check.md")
	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("want one card, got %d", len(cards))
	}
	// The lane closes some other way — the Inbox's own Approve button (the
	// canonical close: RecordTaskDecision, not a /tasks approve). A card that
	// outlives its lane is a click that can no longer do anything.
	if err := b.RecordTaskDecisionWithComment(taskID, string(RecordDecisionApprove), "", "cos"); err != nil {
		t.Fatalf("close lane: %v", err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, req := range b.requests {
		if req.ID != cards[0].ID {
			continue
		}
		if requestIsActive(req) {
			t.Fatalf("a closed lane must retire its decision card, still %q", req.Status)
		}
		return
	}
	t.Fatal("card vanished")
}

// A stale card must report its refusal, not a cheerful 200. The lane can be
// closed from the Inbox while the card is still open (another human, or the
// owner resolving it), and clicking the stale card then asks the decision
// chokepoint to transition an already-approved task.
func TestStaleDecisionCardReportsItsRefusal(t *testing.T) {
	b, taskID := decisionCardFixture(t, "team/reviews/acme-renewal-check.md")
	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("want one card, got %d", len(cards))
	}
	if err := b.RecordTaskDecisionWithComment(taskID, string(RecordDecisionApprove), "", "cos"); err != nil {
		t.Fatalf("close lane first: %v", err)
	}
	status, msg := b.answerRequestFromActor("you", cards[0].ID, decisionChoiceApprove, "", "")
	if status == 200 {
		t.Fatalf("a stale card must not report success, got 200")
	}
	if strings.TrimSpace(msg) == "" {
		t.Fatal("the refusal must carry a reason the human can read")
	}
}

// The executor is called from three places (the HTTP answer endpoint, the Slack
// gate, and both message handlers). Re-applying the same answer must not walk
// the decision a second time — an approve that lands twice would re-broadcast a
// decision and re-queue distillation for a task that already shipped.
func TestDecisionCardAnswerIsAppliedOnce(t *testing.T) {
	b, taskID := decisionCardFixture(t, "team/reviews/acme-renewal-check.md")
	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("want one card, got %d", len(cards))
	}
	if status, msg := b.answerRequestFromActor("you", cards[0].ID, decisionChoiceApprove, "", ""); status != 200 {
		t.Fatalf("first answer: %d %s", status, msg)
	}
	before := b.TaskByID(taskID)
	if before == nil || before.LifecycleState != LifecycleStateApproved {
		t.Fatalf("lane should be approved, got %+v", before)
	}
	if err := b.applyDecisionCardAnswer(cards[0].ID, "you"); err != nil {
		t.Fatalf("a repeat application must be a no-op, not an error: %v", err)
	}
	after := b.TaskByID(taskID)
	if after == nil || after.LifecycleState != LifecycleStateApproved {
		t.Fatalf("the repeat application must leave the lane alone, got %+v", after)
	}
	if after.UpdatedAt != before.UpdatedAt {
		t.Fatalf("the repeat application rewrote the task: %q -> %q", before.UpdatedAt, after.UpdatedAt)
	}
}

// Journey 10 (core-loop-runs/2026-11-10-journey10) F-I: the run's parent Issue
// JOURN-14 reached `decision` at 02:54:20Z with both lanes already shipped
// (dce4efc team/accounts/acme.md, 5c3c6b3 team/emails/acme-renewal-followup.md)
// but with its own artifact fields null. The card predicate read only those two
// fields, so no card was raised and the human had nothing to click for ten
// minutes; approving by hand with no artifact_path was refused 409 with advice
// aimed at the owner. An Issue is a container: what the human accepts is what
// its lanes landed.
func seedDecisionCardParent(t *testing.T) (*Broker, string) {
	t.Helper()
	b := newTestBroker(t)
	now := time.Date(2026, 11, 10, 2, 54, 20, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: "JOURN-14", Title: "Journey 10: Acme renewal", Channel: "cos__human", Owner: "cos",
			TaskType: "issue", status: "review", CreatedBy: "human", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-21", Title: "Acme quote one-pager", Channel: "task-journ-21", Owner: "revops",
			ParentIssueID: "JOURN-14", status: "done", Artifact: "team/accounts/acme.md",
			CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-23", Title: "Acme follow-up email", Channel: "task-journ-23", Owner: "ae",
			ParentIssueID: "JOURN-14", status: "done", Artifact: "team/emails/acme-renewal-followup.md",
			CreatedAt: now, UpdatedAt: now},
	)
	parent := b.taskByIDLocked("JOURN-14")
	parent.LifecycleState = LifecycleStateReview
	parent.pipelineStage = "review"
	parent.reviewState = "ready_for_review"
	b.mu.Unlock()
	return b, "JOURN-14"
}

func TestDecisionCardRaisedForParentIssueWithDeliveredLanes(t *testing.T) {
	b, taskID := seedDecisionCardParent(t)
	// The live path: cos submits the Issue for review, no reviewer is assigned,
	// the 30-second convergence sweep parks it on the human and must ask.
	if err := b.EvaluateConvergence(taskID); err != nil {
		t.Fatalf("convergence: %v", err)
	}
	if got := b.TaskByID(taskID).LifecycleState; got != LifecycleStateDecision {
		t.Fatalf("an Issue with no reviewers must land in decision, got %q", got)
	}

	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("a parent Issue whose lanes delivered owes the human a card, got %d", len(cards))
	}
	card := cards[0]
	if card.IssueID != taskID {
		t.Fatalf("the card must be anchored to the Issue, got issue_id=%q", card.IssueID)
	}
	if card.Blocking || card.Required {
		t.Fatalf("the parent card must not gate the office: blocking=%v required=%v", card.Blocking, card.Required)
	}
	// The ask must name every file, not one arbitrary lane's path: the human is
	// signing off the Issue, and both files are what they accept.
	for _, want := range []string{"team/accounts/acme.md", "team/emails/acme-renewal-followup.md"} {
		if !strings.Contains(card.Question, want) {
			t.Fatalf("the card must name %q, got %q", want, card.Question)
		}
	}

	// Clicking Approve must land, not 409: the artifact gate blocks vapor, and
	// an Issue whose lanes each shipped a file is not vapor.
	if status, msg := b.answerRequestFromActor("you", card.ID, decisionChoiceApprove, "", ""); status != 200 {
		t.Fatalf("approving a delivered Issue must not be refused: %d %s", status, msg)
	}
	task := b.TaskByID(taskID)
	if task == nil || task.LifecycleState != LifecycleStateApproved {
		t.Fatalf("approving the card must approve the Issue, got %+v", task)
	}
	if task.HollowCompletion {
		t.Fatal("an Issue whose lanes shipped files must not be recorded as a hollow completion")
	}
}

// The aggregate path must not swallow the vapor case: a parent Issue whose
// lanes finished without artifacts (and without declaring they ship none) has
// still delivered nothing, so no card and no approve.
func TestDecisionCardSkippedForParentIssueWithHollowLanes(t *testing.T) {
	b, taskID := seedDecisionCardParent(t)
	b.mu.Lock()
	for i := range b.tasks {
		if b.tasks[i].ParentIssueID == taskID {
			b.tasks[i].Artifact = ""
		}
	}
	b.mu.Unlock()
	if err := b.EvaluateConvergence(taskID); err != nil {
		t.Fatalf("convergence: %v", err)
	}

	if cards := decisionCards(b); len(cards) != 0 {
		t.Fatalf("lanes that shipped nothing must not ask for acceptance, got %d", len(cards))
	}
	if err := b.RecordTaskDecisionWithComment(taskID, string(RecordDecisionApprove), "", "you"); err == nil {
		t.Fatal("approving an Issue whose lanes shipped nothing must still be refused")
	}
}

// Journey 11 (core-loop-runs/2026-11-10-journey11): the office itself stamps a
// single lane's path onto the Issue when cos submits it for review. The card
// then read that field first and told the human "The deliverable is
// team/accounts/acme-renewal-quote.md" while the follow-up email sat in the same
// Issue, unreported. An Issue is a container: when its lanes delivered, the card
// names every one of them, and the Issue's own field is only mentioned if it
// points somewhere the lanes did not.
func TestDecisionCardNamesEveryLaneEvenWhenIssueCarriesOneArtifact(t *testing.T) {
	b, taskID := seedDecisionCardParent(t)
	b.mu.Lock()
	parent := b.taskByIDLocked(taskID)
	parent.Artifact = "team/accounts/acme.md"
	b.mu.Unlock()

	if err := b.EvaluateConvergence(taskID); err != nil {
		t.Fatalf("convergence: %v", err)
	}
	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("a parent Issue whose lanes delivered owes the human a card, got %d", len(cards))
	}
	question := cards[0].Question
	for _, want := range []string{"team/accounts/acme.md", "team/emails/acme-renewal-followup.md"} {
		if !strings.Contains(question, want) {
			t.Fatalf("the card must name every delivered lane file, missing %q in %q", want, question)
		}
	}
	if strings.Contains(question, "The Issue itself also records") {
		t.Fatalf("the Issue artifact is one of the lane files, so it must not be restated: %q", question)
	}
	if strings.Count(question, "team/accounts/acme.md") != 1 {
		t.Fatalf("the card must name each file once, got %q", question)
	}

	// A gutted Issue whose own artifact is the only thing left still has to
	// report that file, and must say it is the Issue's, not a lane's.
	b2, taskID2 := seedDecisionCardParent(t)
	b2.mu.Lock()
	parent = b2.taskByIDLocked(taskID2)
	parent.Artifact = "team/accounts/issue-level.md"
	for i := range b2.tasks {
		if b2.tasks[i].ParentIssueID == taskID2 {
			b2.tasks[i].Artifact = ""
			b2.tasks[i].ArtifactNotRequired = "review notes only, no file"
		}
	}
	b2.mu.Unlock()
	if err := b2.EvaluateConvergence(taskID2); err != nil {
		t.Fatalf("convergence: %v", err)
	}
	cards = decisionCards(b2)
	if len(cards) != 1 {
		t.Fatalf("lanes that shipped nothing leave the Issue's own artifact as the only deliverable, got %d cards", len(cards))
	}
	if q := cards[0].Question; !strings.Contains(q, "team/accounts/issue-level.md") || !strings.Contains(q, "The Issue itself also records") {
		t.Fatalf("an Issue artifact no lane claimed must still be named, got %q", q)
	}
}

// Journey 15 (core-loop-runs/2026-11-10-journey15): both lanes raised their own
// decision cards and then approved themselves 38 seconds later (no reviewer was
// configured), so the operator's two clicks landed on closed lanes. The Issue's
// card is the surface that can still move: it lists every lane file and its
// approve is what closes the container. A lane therefore defers to it.
//
// The exception matters as much: a lane whose Issue is already closed has no
// container card left, and its own card is the last place the human can see the
// work.
func TestLaneDecisionCardDefersToItsLiveIssue(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{
		{ID: "JOURN-1", Title: "Issue", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", LifecycleState: LifecycleStateRunning, CreatedAt: now, UpdatedAt: now},
		{ID: "JOURN-15", Title: "Lane", Channel: testTeamRoom, Owner: "cos",
			ParentIssueID: "JOURN-1", status: "review", LifecycleState: LifecycleStateDecision,
			Artifact: "team/accounts/acme-renewal-quote.md", CreatedAt: now, UpdatedAt: now},
		{ID: "SOLO-1", Title: "Standalone", Channel: testTeamRoom, Owner: "cos",
			status: "review", LifecycleState: LifecycleStateDecision,
			Artifact: "team/reports/solo.md", CreatedAt: now, UpdatedAt: now},
		{ID: "JOURN-9", Title: "Closed issue", Channel: testTeamRoom, Owner: "cos",
			status: "done", LifecycleState: LifecycleStateApproved, CreatedAt: now, UpdatedAt: now},
		{ID: "JOURN-90", Title: "Orphaned lane", Channel: testTeamRoom, Owner: "cos",
			ParentIssueID: "JOURN-9", status: "review", LifecycleState: LifecycleStateDecision,
			Artifact: "team/reports/orphan.md", CreatedAt: now, UpdatedAt: now},
	}
	b.mu.Unlock()

	for _, id := range []string{"JOURN-15", "SOLO-1", "JOURN-90"} {
		b.raiseDecisionCardAfterConvergenceLocked(id)
	}

	cards := map[string]string{}
	for _, card := range decisionCards(b) {
		cards[card.IssueID] = card.ID
	}
	if id, ok := cards["JOURN-15"]; ok {
		t.Fatalf("a lane under a live Issue must not raise its own card, got %s", id)
	}
	if _, ok := cards["SOLO-1"]; !ok {
		t.Fatal("a task with no Issue still needs its own card — it is the only surface")
	}
	if _, ok := cards["JOURN-90"]; !ok {
		t.Fatal("a lane whose Issue is closed keeps its own card — nothing else can accept it")
	}
}

// Journey 18: the office twice reported a rework it never wrote — the file was
// byte-identical (724 B) after the first send-back, and the second reply
// invented a byte count ("475 bytes") for content that never existed. The card
// was the only acceptance surface, so it has to say when the thing being
// offered is the thing that was just rejected.
func TestDecisionCardSaysWhenTheRejectedWorkWasNeverRewritten(t *testing.T) {
	t.Setenv("WUPHF_RUNTIME_HOME", t.TempDir())
	const artifact = "team/reviews/j18-public.md"
	full := filepath.Join(WikiRootDir(), filepath.FromSlash(artifact))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("AAAAA"), 0o644); err != nil {
		t.Fatalf("seed deliverable: %v", err)
	}

	b, taskID := decisionCardFixture(t, artifact)
	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("want one card, got %d", len(cards))
	}
	if strings.Contains(cards[0].Question, "has changed since you sent this back") {
		t.Fatal("the first card has nothing to compare against and must not warn")
	}

	if status, msg := b.answerRequestFromActor("you", cards[0].ID, decisionChoiceChanges, "", "drop the fifth bullet"); status != 200 {
		t.Fatalf("answer: %d %s", status, msg)
	}
	if task := b.TaskByID(taskID); task == nil || task.SentBackFingerprint == "" {
		t.Fatalf("a send-back must record what was rejected, got %+v", task)
	}

	// The owner reports "done" and the lane converges again. Nothing changed.
	reopenInDecision(t, b, taskID)
	cards = decisionCards(b)
	if len(cards) != 2 {
		t.Fatalf("want a second card, got %d", len(cards))
	}
	second := cards[1]
	if !strings.Contains(second.Question, "has changed since you sent this back") {
		t.Fatalf("an unchanged deliverable must be flagged on the card, got %q", second.Question)
	}

	// A same-size rewrite is still a change: the digest is content-addressed.
	if status, msg := b.answerRequestFromActor("you", second.ID, decisionChoiceChanges, "", "still not right"); status != 200 {
		t.Fatalf("answer: %d %s", status, msg)
	}
	if err := os.WriteFile(full, []byte("BBBBB"), 0o644); err != nil {
		t.Fatalf("rewrite deliverable: %v", err)
	}
	reopenInDecision(t, b, taskID)
	cards = decisionCards(b)
	if len(cards) != 3 {
		t.Fatalf("want a third card, got %d", len(cards))
	}
	if strings.Contains(cards[2].Question, "has changed since you sent this back") {
		t.Fatalf("a rewritten deliverable must clear the warning, got %q", cards[2].Question)
	}

	// Approval closes the task and drops the comparison point.
	if status, msg := b.answerRequestFromActor("you", cards[2].ID, decisionChoiceApprove, "", ""); status != 200 {
		t.Fatalf("answer: %d %s", status, msg)
	}
	if task := b.TaskByID(taskID); task != nil && (task.SentBackFingerprint != "" || task.SentBackAt != "") {
		t.Fatalf("approve must clear the send-back digest, got %+v", task)
	}
}

// reopenInDecision puts a task back where convergence can raise its card, which
// is what an owner's re-submission produces in production.
func reopenInDecision(t *testing.T, b *Broker, taskID string) {
	t.Helper()
	b.mu.Lock()
	task := b.taskByIDLocked(taskID)
	if task == nil {
		b.mu.Unlock()
		t.Fatalf("task %s vanished", taskID)
	}
	task.status = "review"
	task.LifecycleState = LifecycleStateDecision
	b.raiseDecisionCardAfterConvergenceLocked(taskID)
	b.mu.Unlock()
}

// The Journey 18 failure was a CONTAINER card: the Issue accepted two lane
// files, the human sent it back over one of them, and the re-raised card was
// identical. The comparison therefore has to span the lane set, not the Issue's
// own artifact (an Issue usually has none).
func TestContainerCardFlagsAnUnchangedLaneSet(t *testing.T) {
	t.Setenv("WUPHF_RUNTIME_HOME", t.TempDir())
	internalPath := filepath.Join(WikiRootDir(), "team", "reviews", "j18-internal.md")
	publicPath := filepath.Join(WikiRootDir(), "team", "reviews", "j18-public.md")
	if err := os.MkdirAll(filepath.Dir(internalPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(internalPath, []byte("23 rows"), 0o644); err != nil {
		t.Fatalf("seed internal: %v", err)
	}
	if err := os.WriteFile(publicPath, []byte("5 bullets"), 0o644); err != nil {
		t.Fatalf("seed public: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: "ISSUE-1", Title: "release notes", Channel: "team", Owner: "cos",
			status: "in_progress", LifecycleState: LifecycleStateDecision, CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "LANE-1", ParentIssueID: "ISSUE-1", Channel: "team", Owner: "cos",
			status: "done", LifecycleState: LifecycleStateApproved, CreatedAt: now, UpdatedAt: now,
			Artifact: "team/reviews/j18-internal.md"},
		teamTask{ID: "LANE-2", ParentIssueID: "ISSUE-1", Channel: "team", Owner: "cos",
			status: "done", LifecycleState: LifecycleStateApproved, CreatedAt: now, UpdatedAt: now,
			Artifact: "team/reviews/j18-public.md"},
	)
	b.raiseDecisionCardAfterConvergenceLocked("ISSUE-1")
	b.mu.Unlock()

	cards := decisionCards(b)
	if len(cards) != 1 {
		t.Fatalf("want one container card, got %d", len(cards))
	}
	if !strings.Contains(cards[0].Question, "j18-internal.md") || !strings.Contains(cards[0].Question, "j18-public.md") {
		t.Fatalf("the container card must name every lane file, got %q", cards[0].Question)
	}
	if strings.Contains(cards[0].Question, "has changed since you sent this back") {
		t.Fatal("no send-back has happened yet, so there is nothing to warn about")
	}

	if status, msg := b.answerRequestFromActor("you", cards[0].ID, decisionChoiceChanges, "", "drop the fifth bullet"); status != 200 {
		t.Fatalf("answer: %d %s", status, msg)
	}
	reopenInDecision(t, b, "ISSUE-1")
	cards = decisionCards(b)
	if len(cards) != 2 {
		t.Fatalf("want a second card, got %d", len(cards))
	}
	if !strings.Contains(cards[1].Question, "has changed since you sent this back") {
		t.Fatalf("an untouched lane set must be flagged, got %q", cards[1].Question)
	}

	// Rewriting the rejected file clears it; the untouched sibling is not the
	// comparison — the set is.
	if err := os.WriteFile(publicPath, []byte("4 bullets"), 0o644); err != nil {
		t.Fatalf("rewrite public: %v", err)
	}
	if status, msg := b.answerRequestFromActor("you", cards[1].ID, decisionChoiceChanges, "", "still not right"); status != 200 {
		t.Fatalf("answer: %d %s", status, msg)
	}
	if err := os.WriteFile(publicPath, []byte("4 bullets, fixed"), 0o644); err != nil {
		t.Fatalf("rewrite again: %v", err)
	}
	reopenInDecision(t, b, "ISSUE-1")
	cards = decisionCards(b)
	if len(cards) != 3 {
		t.Fatalf("want a third card, got %d", len(cards))
	}
	if strings.Contains(cards[2].Question, "has changed since you sent this back") {
		t.Fatalf("a reworked lane set must clear the warning, got %q", cards[2].Question)
	}
}
