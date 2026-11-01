package team

import (
	"strings"
	"testing"
	"time"
)

// deliveredTaskForReview is the smallest state the review lane acts on: a task
// that reached done with a Definition, an artifact, and a passing check.
func deliveredTaskForReview(t *testing.T, b *Broker, id, owner string) teamTask {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	task := teamTask{
		ID:        id,
		Channel:   testTeamRoom,
		Title:     "外汇每日简报",
		Details:   "给今天的外汇做一份简报",
		Owner:     owner,
		status:    "done",
		CreatedBy: "human",
		TaskType:  "issue",
		CreatedAt: now,
		UpdatedAt: now,
		Artifact:  "artifacts/forex/2026-11-01.md",
		Definition: &TaskDefinition{
			Goal:            "产出一份外汇简报",
			SuccessCriteria: []string{"给出三个主要货币对的方向", "写明数据来源"},
		},
		Verification: &TaskVerification{
			Kind:     taskVerificationKindWikiArtifact,
			Spec:     "artifact_path declared on complete",
			Required: true,
		},
		VerificationResult: &TaskVerificationResult{Pass: true, Kind: taskVerificationKindWikiArtifact, CheckedAt: now},
	}
	b.mu.Lock()
	b.tasks = append(b.tasks, task)
	b.mu.Unlock()
	return task
}

func reviewsForTask(b *Broker, taskID string) []teamTask {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []teamTask
	for _, t := range b.tasks {
		if strings.TrimSpace(t.SourceSignalID) == taskReviewSignalPrefix+taskID {
			out = append(out, t)
		}
	}
	return out
}

// A delivered task gets exactly one review, owned by a bot that is not the
// author, carrying its own artifact gate — and a second call does not fan out
// a duplicate (approve-after-complete replays the done branch).
func TestReviewCompletedTaskOpensCleanContextReview(t *testing.T) {
	b := newTestBroker(t)
	b.members = []officeMember{{Slug: "cos", BuiltIn: true}, {Slug: "data-keeper"}}

	deliveredTaskForReview(t, b, "ACMER-900", "cos")
	b.reviewCompletedTask("ACMER-900")
	b.reviewCompletedTask("ACMER-900")

	reviews := reviewsForTask(b, "ACMER-900")
	if len(reviews) != 1 {
		t.Fatalf("expected exactly one review task, got %d", len(reviews))
	}
	review := reviews[0]
	if review.Owner == "cos" {
		t.Fatalf("the review must be read by someone other than the author, got %q", review.Owner)
	}
	if review.Owner != "data-keeper" {
		t.Errorf("expected the first non-author member (data-keeper), got %q", review.Owner)
	}
	if review.Definition == nil || len(review.Definition.SuccessCriteria) == 0 {
		t.Fatalf("a review without a contract cannot be graded; got %+v", review.Definition)
	}
	if review.Verification == nil || review.Verification.Kind != taskVerificationKindWikiArtifact || !review.Verification.Required {
		t.Fatalf("the review must carry the wiki_artifact gate, got %+v", review.Verification)
	}
	if want := "team/reviews/ACMER-900.md"; !strings.Contains(review.Details, want) {
		t.Errorf("the reviewer must be told where to write %q; details %q", want, review.Details)
	}
	// Clean context: the author's plan/reasoning is not in the brief, only the
	// contract and the deliverable path.
	if strings.Contains(review.Details, "Wed") || strings.Contains(review.Details, "plan") {
		t.Errorf("review brief must not carry the author's process; got %q", review.Details)
	}
	if !strings.Contains(review.Details, "给出三个主要货币对的方向") {
		t.Errorf("review brief must carry the original success criteria; got %q", review.Details)
	}
}

// Nothing to review means nothing is created: no Definition, no artifact, a
// failed check, or a review-of-a-review all short-circuit.
func TestReviewCompletedTaskSkipsWhenNothingToReview(t *testing.T) {
	cases := map[string]func(task *teamTask){
		"no definition": func(task *teamTask) { task.Definition = nil },
		"no artifact":   func(task *teamTask) { task.Artifact = "" },
		"check failed": func(task *teamTask) {
			task.VerificationResult = &TaskVerificationResult{Pass: false, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
		},
		"already a review": func(task *teamTask) { task.SourceSignalID = taskReviewSignalPrefix + "ACMER-1" },
		"not done":         func(task *teamTask) { task.status = "review" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b := newTestBroker(t)
			b.members = []officeMember{{Slug: "cos", BuiltIn: true}, {Slug: "data-keeper"}}
			deliveredTaskForReview(t, b, "ACMER-901", "cos")
			b.mu.Lock()
			mutate(b.taskByIDLocked("ACMER-901"))
			b.mu.Unlock()

			b.reviewCompletedTask("ACMER-901")

			if got := reviewsForTask(b, "ACMER-901"); len(got) != 0 {
				t.Fatalf("expected no review task, got %+v", got)
			}
		})
	}
}

// A one-bot office cannot review itself: the lane stays silent rather than
// asking the author to grade their own work.
func TestReviewCompletedTaskNeedsAnIndependentReviewer(t *testing.T) {
	b := newTestBroker(t)
	b.members = []officeMember{{Slug: "cos", BuiltIn: true}}

	deliveredTaskForReview(t, b, "ACMER-902", "cos")
	b.reviewCompletedTask("ACMER-902")

	if got := reviewsForTask(b, "ACMER-902"); len(got) != 0 {
		t.Fatalf("a solo office must not self-review; got %+v", got)
	}
}
