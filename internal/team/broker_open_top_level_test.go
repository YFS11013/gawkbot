package team

import (
	"strings"
	"testing"
)

// Journeys 22, 25, 26 and 27 each split one human request across two top-level
// Issues: the second arrived seconds after the first and restated the same
// work, so the human was asked to approve half of it twice. Journey 26's
// duplicate declared no artifact and titled itself "两页变更说明已写入 wiki";
// Journey 27's pinned the path the first task's details already named.
func TestBotCannotSplitOneRequestAcrossTwoTopLevelTasks(t *testing.T) {
	cases := []struct {
		name   string
		open   teamTask
		create teamTask
	}{
		{
			name: "J26: restates the open task",
			open: teamTask{ID: "JOURN-3", Channel: "general", Owner: "cos", CreatedBy: "cos",
				Title:  "Journey 26：把这批改动写成两页（内部变更清单 + 对外说明）",
				status: "in_progress",
				Definition: &TaskDefinition{
					Goal:         "two pages",
					ArtifactPath: "team/reviews/j26-public.md",
				}},
			create: teamTask{ID: "JOURN-7", Channel: "general", Owner: "cos", CreatedBy: "cos",
				Title: "Journey 26：两页变更说明已写入 wiki"},
		},
		{
			name: "J27: pins a path the open task already names",
			open: teamTask{ID: "JOURN-2", Channel: "cos__human", Owner: "cos", CreatedBy: "cos",
				Title:   "Journey 27：两页变更清单（内部 + 对外）",
				Details: "Human asked for two review pages: internal changelog (team/reviews/j27-internal.md) and public note (team/reviews/j27-public.md).",
				status:  "in_progress"},
			create: teamTask{ID: "JOURN-7", Channel: "cos__human", Owner: "cos", CreatedBy: "cos",
				Title:      "Journey 27：对外说明页",
				Definition: &TaskDefinition{Goal: "public note", ArtifactPath: "team/reviews/j27-public.md"}},
		},
		{
			name: "same declared scope",
			open: teamTask{ID: "JOURN-3", Channel: "general", Owner: "cos", CreatedBy: "cos",
				Title:      "Renew the office lease",
				status:     "in_progress",
				Definition: &TaskDefinition{Goal: "lease", ArtifactPath: "team/accounts/lease.md"}},
			create: teamTask{ID: "JOB-4", Channel: "general", Owner: "cos", CreatedBy: "cos",
				Title:      "Lease renewal, draft two",
				Definition: &TaskDefinition{Goal: "draft", ArtifactPath: "team/accounts/lease.md"}},
		},
	}
	for _, tc := range cases {
		b := newTestBroker(t)
		b.mu.Lock()
		b.tasks = append(b.tasks, tc.open)
		err := b.applyOpenTopLevelExclusivityLocked(&tc.create, "cos", false)
		b.mu.Unlock()
		if err == nil {
			t.Fatalf("%s: a second top-level task for the same work must be refused", tc.name)
		}
		for _, want := range []string{tc.open.ID, "submit_for_review", "parent_issue_id=" + tc.open.ID} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("%s: the refusal must point at the open task and the two ways forward (%q missing): %v", tc.name, want, err)
			}
		}
	}
}

// The guard must not become a channel-wide lock: unrelated work stays legal, the
// opt-out stays, and humans re-split by design.
func TestUnrelatedTopLevelWorkStillCreates(t *testing.T) {
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{ID: "JOURN-3", Channel: "general", Owner: "cos", CreatedBy: "cos",
		Title:      "Journey 27：把这批改动写成两页",
		status:     "in_progress",
		Definition: &TaskDefinition{Goal: "two pages", ArtifactPath: "team/reviews/j27-public.md"}})
	other := teamTask{ID: "JOB-9", Channel: "general", Owner: "cos", CreatedBy: "cos",
		Title:      "Renew the office lease",
		Definition: &TaskDefinition{Goal: "lease", ArtifactPath: "team/accounts/lease.md"}}
	if err := b.applyOpenTopLevelExclusivityLocked(&other, "cos", false); err != nil {
		b.mu.Unlock()
		t.Fatalf("unrelated work must still be creatable: %v", err)
	}
	if err := b.applyOpenTopLevelExclusivityLocked(&other, "cos", true); err != nil {
		b.mu.Unlock()
		t.Fatalf("allow_duplicate must remain the opt-out: %v", err)
	}
	if err := b.applyOpenTopLevelExclusivityLocked(&other, "you", false); err != nil {
		b.mu.Unlock()
		t.Fatalf("humans re-split by design: %v", err)
	}
	b.mu.Unlock()
}
