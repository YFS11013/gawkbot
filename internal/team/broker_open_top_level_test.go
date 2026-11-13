package team

import (
	"strings"
	"testing"
)

// Journeys 22, 25 and 26 each split one human request across two top-level
// Issues: the second arrived seconds after the first and restated the same
// work, so the human was asked to approve half of it twice. Journey 26's
// duplicate declared no artifact and titled itself "the two pages are already
// in the wiki" — it was created to close work out, and spent the turn on
// "still in planning" refusals instead.
func TestBotCannotSplitOneRequestAcrossTwoTopLevelTasks(t *testing.T) {
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: "JOURN-3", Channel: "general", Owner: "cos", CreatedBy: "cos",
		Title:  "Journey 26：把这批改动写成两页（内部变更清单 + 对外说明）",
		status: "in_progress",
		Definition: &TaskDefinition{
			Goal:         "two pages",
			ArtifactPath: "team/reviews/j26-public.md",
		},
	})
	b.mu.Unlock()

	cases := []struct {
		name  string
		task  teamTask
		actor string
	}{
		{
			name: "same declared scope",
			task: teamTask{ID: "JOURN-7", Channel: "general", Owner: "cos", CreatedBy: "cos",
				Title: "对外说明 j26-public.md", ParentIssueID: "",
				Definition: &TaskDefinition{Goal: "public note", ArtifactPath: "team/reviews/j26-public.md"}},
			actor: "cos",
		},
		{
			name: "restates the open task",
			task: teamTask{ID: "JOURN-7", Channel: "general", Owner: "cos", CreatedBy: "cos",
				Title: "Journey 26：两页变更说明已写入 wiki", ParentIssueID: ""},
			actor: "cos",
		},
	}
	for _, tc := range cases {
		b.mu.Lock()
		err := b.applyOpenTopLevelExclusivityLocked(&tc.task, tc.actor, false)
		b.mu.Unlock()
		if err == nil {
			t.Fatalf("%s: a second top-level task for the same work must be refused", tc.name)
		}
		if !strings.Contains(err.Error(), "JOURN-3") || !strings.Contains(err.Error(), "submit_for_review") {
			t.Fatalf("%s: the refusal must name the open task and the closing verbs, got %v", tc.name, err)
		}
	}

	// Separate work stays legal, and so does the explicit opt-out.
	other := teamTask{ID: "JOB-9", Channel: "general", Owner: "cos", CreatedBy: "cos",
		Title: "Renew the office lease", ParentIssueID: "",
		Definition: &TaskDefinition{Goal: "lease", ArtifactPath: "team/accounts/lease.md"}}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.applyOpenTopLevelExclusivityLocked(&other, "cos", false); err != nil {
		t.Fatalf("unrelated work must still be creatable: %v", err)
	}
	if err := b.applyOpenTopLevelExclusivityLocked(&other, "cos", true); err != nil {
		t.Fatalf("allow_duplicate must remain the opt-out: %v", err)
	}
	if err := b.applyOpenTopLevelExclusivityLocked(&other, "you", false); err != nil {
		t.Fatalf("humans re-split by design: %v", err)
	}
}
