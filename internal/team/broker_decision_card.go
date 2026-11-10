package team

// Decision-field cards: when a lane finishes review and lands in the
// `decision` lifecycle state, the ball is with the human — and until Journey 9
// nothing told them so.
//
// Journey 9's stop table (core-loop-runs/2026-11-09-journey9): JOURN-16 finished
// its email draft at 13:48:03, the reviewer convergence moved it to `decision`,
// and it sat there for the rest of the run. `pending_interview` was null and no
// request pointed at the task: the only way to learn that a decision was owed
// was cos's prose in the channel ("待你确认发送"), and the human's web inbox row
// was the only affordance. The office then read as silent for 26 minutes
// (F-H) because the one party who could move was never asked in the place they
// work — the exact failure `postRequestRaisedChatMessageLocked` exists to
// prevent for every other kind of ask ("a blocking interview sat buried in the
// Inbox for 44 minutes with no push").
//
// So: convergence raises ONE non-blocking approval card per lane, in the lane's
// channel, carrying the deliverable being approved. Answering it performs the
// task action the choice means — approve closes the lane, "send back" bounces it
// to its owner with the note. A card that only recorded an answer and left the
// task in `decision` would be worse than no card at all, which is why the answer
// mapping lives here and not in a separate follow-up.

import (
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	// decisionCardDedupePrefix marks a request as a decision-field card and
	// pins it to one lane. The prefix is also how the answer path recognises
	// which requests own a task mutation.
	decisionCardDedupePrefix = "decision:"

	decisionChoiceApprove = "approve_decision"
	decisionChoiceChanges = "request_changes_decision"
)

// requestIsDecisionCard reports whether a request was raised by reviewer
// convergence for a lane that now needs a human call.
func requestIsDecisionCard(req humanInterview) bool {
	return strings.HasPrefix(strings.TrimSpace(req.DedupeKey), decisionCardDedupePrefix)
}

// hasApprovableDeliverable reports whether a lane has anything for the human to
// accept: a recorded artifact, or an explicit not_required declaration. A lane
// that landed in decision with neither has no deliverable to approve, and a card
// asking to "accept" nothing would only train the human to ignore these cards.
func hasApprovableDeliverable(task *teamTask) bool {
	if task == nil {
		return false
	}
	return strings.TrimSpace(task.Artifact) != "" || strings.TrimSpace(task.ArtifactNotRequired) != ""
}

// raiseDecisionCardLocked raises the decision-field card for a lane that just
// landed in `decision`. Idempotent on the dedupe key, so repeated convergence
// sweeps (the 30-second ticker) never stack cards. Non-blocking by design: the
// lane is not parked on it, no bot's wake is suppressed, and `pendingInterview`
// stays with whichever request really is gating the office. Caller holds b.mu;
// persistence is the caller's responsibility.
func (b *Broker) raiseDecisionCardLocked(task *teamTask) string {
	if b == nil || task == nil || !hasApprovableDeliverable(task) {
		return ""
	}
	dedupeKey := decisionCardDedupePrefix + strings.TrimSpace(task.ID)
	for i := range b.requests {
		if !requestIsActive(b.requests[i]) {
			continue
		}
		if strings.TrimSpace(b.requests[i].DedupeKey) == dedupeKey {
			// Already asked. "" (not the existing id) so the convergence hook
			// does not re-persist the whole broker state on every 30-second
			// sweep for a card that has been sitting there all along.
			return ""
		}
	}

	from := strings.TrimSpace(task.Owner)
	if from == "" {
		from = "office"
	}
	channel := normalizeChannelSlug(task.Channel)
	if strings.TrimSpace(task.Channel) == "" {
		if home, err := b.homeChannelForWriterLocked(from, task.Owner, from); err == nil {
			channel = home
		}
	}
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = task.ID
	}

	// Lead with what the human is being asked to accept and what happens either
	// way — the lesson from the plan-approval copy rewrite, where the bot's
	// internal framing ("Plan ready for X. Review it and approve…") left the
	// human unable to tell what it was for.
	var qb strings.Builder
	fmt.Fprintf(&qb, "%s finished %s (%s) and the review is in. Nothing else moves until you call it.\n\n", ownerLabelForPlan(from), title, task.ID)
	if artifact := strings.TrimSpace(task.Artifact); artifact != "" {
		fmt.Fprintf(&qb, "The deliverable is %s.\n", artifact)
	} else {
		fmt.Fprintf(&qb, "The owner declared this lane ships no file (%s).\n", strings.TrimSpace(task.ArtifactNotRequired))
	}
	qb.WriteString("Approve and the lane closes. Send it back and the owner picks up your note as the rework.")

	options := []interviewOption{
		{ID: decisionChoiceApprove, Label: "Approve", Description: "Accept the deliverable and close this lane."},
		{ID: decisionChoiceChanges, Label: "Send back", Description: "Hand it back to the owner with what must change.", RequiresText: true, TextHint: "Type what must change before this can be accepted."},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	b.counter++
	req := humanInterview{
		ID:            fmt.Sprintf("request-%d", b.counter),
		Kind:          "approval",
		Status:        "pending",
		From:          from,
		Channel:       b.requestChannelForLocked(from, channel),
		Title:         "Approve " + title + "?",
		Question:      qb.String(),
		Options:       options,
		RecommendedID: decisionChoiceApprove,
		DedupeKey:     dedupeKey,
		IssueID:       strings.TrimSpace(task.ID),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	b.scheduleRequestLifecycleLocked(&req)
	b.postRequestRaisedChatMessageLocked(&req)
	b.requests = append(b.requests, req)
	if ch := strings.TrimSpace(req.Channel); ch != "" {
		b.appendActionLocked("request_created", "office", ch, from,
			truncateSummary(req.Title+" "+req.Question, 140), req.ID)
	}
	return req.ID
}

// raiseDecisionCardAfterConvergenceLocked is the convergence hook: raise the
// card for a lane that just entered decision, and persist. Caller holds b.mu.
func (b *Broker) raiseDecisionCardAfterConvergenceLocked(taskID string) {
	if b == nil {
		return
	}
	task := b.taskByIDLocked(strings.TrimSpace(taskID))
	if task == nil || task.LifecycleState != LifecycleStateDecision {
		return
	}
	if b.raiseDecisionCardLocked(task) == "" {
		return
	}
	if err := b.saveLocked(); err != nil {
		log.Printf("broker: persist decision card for task %q: %v", taskID, err)
	}
}

// applyDecisionCardAnswer performs the decision a card's answer means. MUST be
// called with b.mu RELEASED: it goes through RecordTaskDecisionWithComment, the
// same chokepoint the web Inbox's Approve / Request-changes buttons use, which
// takes the lock itself and owns every consequence — verification + artifact
// gates, the Review/Decision → Approved transition, the wiki Decision article,
// the channel broadcast, distillation, and the retirement of the requests
// anchored to the lane (which is what makes this card one-shot).
//
// Approving through MutateTask's "approve" instead would have set status=done
// while leaving the task in the decision lifecycle bucket — a split state the
// Inbox would keep listing as "decision required".
//
// Choices map as:
//   - approve_decision         → RecordDecisionApprove (the lane closes)
//   - request_changes_decision → RecordDecisionRequestChanges with the note
//   - a free-text thread reply on the card → request_changes with that text
//     (the human answered in words, which is a rework instruction, not consent)
func (b *Broker) applyDecisionCardAnswer(requestID, actor string) error {
	if b == nil {
		return nil
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil
	}
	var (
		taskID, choice, note string
		state                LifecycleState
		found                bool
	)
	b.mu.Lock()
	for i := range b.requests {
		if b.requests[i].ID != requestID {
			continue
		}
		req := b.requests[i]
		if !requestIsDecisionCard(req) || req.Answered == nil {
			break
		}
		choice = strings.ToLower(strings.TrimSpace(req.Answered.ChoiceID))
		note = strings.TrimSpace(req.Answered.CustomText)
		taskID = strings.TrimSpace(req.IssueID)
		if task := b.taskByIDLocked(taskID); task != nil {
			state = task.LifecycleState
		}
		found = true
		break
	}
	b.mu.Unlock()
	if !found || taskID == "" {
		return nil
	}

	var action recordDecisionAction
	switch choice {
	case decisionChoiceApprove:
		action = RecordDecisionApprove
	case decisionChoiceChanges:
		action = RecordDecisionRequestChanges
	default:
		if choice != "" && choice != "answer_directly" {
			return nil
		}
		action = RecordDecisionRequestChanges
	}
	// This executor is reachable from the HTTP answer endpoint, the Slack gate
	// and both message handlers, so a second click on the same card re-enters it
	// with the answer already recorded. Walking the decision twice would
	// re-broadcast it and re-queue distillation for a task that already shipped;
	// a lane already sitting in the state this action produces means the work is
	// done — report success, touch nothing.
	switch action {
	case RecordDecisionApprove:
		if state == LifecycleStateApproved {
			return nil
		}
	case RecordDecisionRequestChanges:
		if state == LifecycleStateChangesRequested {
			return nil
		}
	}
	// A refusal has nobody else to tell: the answer is already recorded, so hand
	// the reason back to whoever answered. A click that reports success and
	// changes nothing is the exact failure this card exists to end.
	if err := b.RecordTaskDecisionWithComment(taskID, string(action), note, actor); err != nil {
		log.Printf("broker: decision card %s answered %q but the %s decision on task %s was refused: %v", requestID, choice, action, taskID, err)
		return err
	}
	return nil
}

// reportDecisionAnswerFailure tells the channel that a human's reply on a
// decision card answered the card but the decision itself was refused. The
// click path returns the refusal to the caller (409); a thread reply has no
// response body to carry it, and a human who wrote "…send it back" deserves to
// know the lane did not move.
func (b *Broker) reportDecisionAnswerFailure(channel string, err error) {
	if b == nil || err == nil {
		return
	}
	b.PostSystemMessage(channel, "That reply answered the decision card, but the decision could not be applied: "+err.Error(), "decision_card")
}
