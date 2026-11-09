package team

import (
	"errors"
	"strings"
	"testing"
)

// The task-scoped gate is the only thing standing between a bot and writing
// an arbitrary wiki path with no human in the loop, so pin every refusal.
func TestAuthorizeTaskArtifactWrite(t *testing.T) {
	b := newTestBroker(t)
	now := "2026-11-01T10:00:00Z"
	add := func(id, owner, status, declared string) {
		def := &TaskDefinition{Goal: "x", DefinedAt: now}
		if declared != "" {
			def.ArtifactPath = declared
		}
		b.mu.Lock()
		b.tasks = append(b.tasks, teamTask{ID: id, Owner: owner, status: status, Definition: def})
		b.mu.Unlock()
	}
	add("ACMER-900", "cos", "in_progress", "reviews/ACMER-899.md")
	add("ACMER-901", "cos", "done", "reviews/ACMER-898.md")
	add("ACMER-902", "cos", "in_progress", "")
	add("ACMER-903", "data-keeper", "in_progress", "team/notes/x.md")

	cases := []struct {
		name   string
		slug   string
		taskID string
		path   string
		want   string // "" = allowed
	}{
		{"declared path", "cos", "ACMER-900", "reviews/ACMER-899.md", ""},
		{"declared path, other separator style", "cos", "ACMER-900", `reviews\ACMER-899.md`, ""},
		{"declared path, leading ./", "cos", "ACMER-900", "./reviews/ACMER-899.md", ""},
		{"declared path, absolute", "cos", "ACMER-900", "/reviews/ACMER-899.md", ""},
		{"undeclared path", "cos", "ACMER-900", "team/people/nazz.md", "declares artifact_path"},
		{"not the owner", "data-keeper", "ACMER-900", "reviews/ACMER-899.md", "owned by cos"},
		{"unknown task", "cos", "ACMER-999", "reviews/ACMER-899.md", "not found"},
		{"task is closed", "cos", "ACMER-901", "reviews/ACMER-898.md", "already closed"},
		{"no declared path", "cos", "ACMER-902", "reviews/ACMER-898.md", "declares no artifact_path"},
		{"someone else's task", "cos", "ACMER-903", "team/notes/x.md", "owned by data-keeper"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := b.authorizeTaskArtifactWrite(tc.slug, tc.taskID, tc.path)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected the write to be allowed, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected a refusal mentioning %q, got none", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected refusal mentioning %q, got %q", tc.want, err.Error())
			}
		})
	}
}

// The review task must declare the path it is allowed to write; otherwise the
// reviewer falls back to needing a human go-ahead and the lane dead-ends.
func TestReviewTaskDeclaresItsArtifactPath(t *testing.T) {
	b := newTestBroker(t)
	b.members = []officeMember{{Slug: "cos", BuiltIn: true}, {Slug: "data-keeper"}}
	deliveredTaskForReview(t, b, "ACMER-904", "cos")
	b.reviewCompletedTask("ACMER-904")

	reviews := reviewsForTask(b, "ACMER-904")
	if len(reviews) != 1 {
		t.Fatalf("expected one review task, got %d", len(reviews))
	}
	declare := ""
	if reviews[0].Definition != nil {
		declare = reviews[0].Definition.ArtifactPath
	}
	if declare != "team/reviews/ACMER-904.md" {
		t.Fatalf("review must declare reviews/ACMER-904.md as its artifact path, got %q", declare)
	}
	if err := b.authorizeTaskArtifactWrite(reviews[0].Owner, reviews[0].ID, declare); err != nil {
		t.Fatalf("the reviewer must be able to write its own declared artifact: %v", err)
	}
}

// A refused write for a task with no pinned path must reach the human who can
// fix it, once. Journey 3/4: cos's planner created lanes without definitions,
// both owners were refused, both fell back to notebooks, and the human learned
// about the downgrade only at review.
func TestUndeclaredArtifactPathNudgesCreatorOnce(t *testing.T) {
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID: "JOURN-916", Channel: "analyst__cos", Title: "ICP hypothesis", Owner: "analyst",
		CreatedBy: "cos", status: "in_progress",
	}}
	b.mu.Unlock()

	for i := 0; i < 3; i++ {
		b.nudgeUndeclaredArtifactPath("JOURN-916", "analyst", "team/research/icp.md")
	}

	b.mu.Lock()
	var notices []channelMessage
	for _, m := range b.messages {
		if m.Kind == artifactPathMissingKind {
			notices = append(notices, m)
		}
	}
	actions := 0
	for _, a := range b.actions {
		if a.Kind == artifactPathMissingKind && a.RelatedID == "JOURN-916" {
			actions++
		}
	}
	b.mu.Unlock()

	if len(notices) != 1 || actions != 1 {
		t.Fatalf("three retries must leave one nudge, got %d messages / %d actions", len(notices), actions)
	}
	body := notices[0].Content
	for _, want := range []string{"@cos", "JOURN-916", "team/research/icp.md", "artifact_path", "@analyst"} {
		if !strings.Contains(body, want) {
			t.Errorf("nudge must contain %q, got %q", want, body)
		}
	}
	if notices[0].Channel != "analyst__cos" {
		t.Errorf("nudge must land on the task channel, got %q", notices[0].Channel)
	}
}

// The refusal itself must tell the calling bot what to do next and who owns
// the missing field — the old text only said "pass human_request instead" and
// never named the creator.
func TestUndeclaredArtifactPathRefusalNamesCreator(t *testing.T) {
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID: "JOURN-917", Channel: "team", Title: "email draft", Owner: "ae",
		CreatedBy: "cos", status: "in_progress",
	}}
	b.mu.Unlock()

	err := b.authorizeTaskArtifactWrite("ae", "JOURN-917", "team/emails/x.md")
	if err == nil {
		t.Fatal("expected refusal")
	}
	if !errors.Is(err, errTaskArtifactUndeclared) {
		t.Fatalf("refusal must be the undeclared sentinel, got %v", err)
	}
	for _, want := range []string{"@cos", "definition.artifact_path", "cannot succeed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal must contain %q, got %q", want, err.Error())
		}
	}
}
