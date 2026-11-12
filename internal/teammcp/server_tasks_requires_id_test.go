package teammcp

import (
	"context"
	"strings"
	"testing"
)

// Journey 21 (core-loop-runs/2026-11-13-journey21): one turn called
// team_task action=define without `id` 65 times. Each call reached the broker,
// matched no task, and came back "404 task not found" — an error that names a
// missing task instead of the missing argument, so the model kept retrying the
// same shape. Non-create actions address an existing task by construction, and
// the broker never infers it from the channel, so the tool must say so before
// making the round trip.
func TestHandleTeamTaskNamesAMissingIDInsteadOf404(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WUPHF_AGENT_SLUG", "eng")
	ctx := context.Background()

	for _, action := range []string{"define", "complete", "submit_for_review", "comment", "request_changes", "claim"} {
		res, _, err := handleTeamTask(ctx, nil, TeamTaskArgs{Action: action})
		if err != nil {
			t.Fatalf("%s without id must be a tool error result, not a transport error: %v", action, err)
		}
		text := textFromResult(t, res)
		if !strings.Contains(text, "needs the task id") {
			t.Errorf("%s without id must name the missing argument, got %q", action, text)
		}
		if strings.Contains(text, "task not found") {
			t.Errorf("%s without id must not look like a missing task, got %q", action, text)
		}
	}
}
