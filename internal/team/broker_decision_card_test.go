package team

// broker_decision_card_test.go covers the Journey 9 D2 gap: a lane that lands
// in `decision` owes the human a call, and until now the only surface that said
// so was a web inbox row. JOURN-16 sat in that state for the rest of the run
// with pending_interview=null, which is what the run's 26-minute "silence"
// (F-H) actually was.

import (
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
