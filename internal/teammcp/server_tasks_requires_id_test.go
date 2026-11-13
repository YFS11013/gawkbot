package teammcp

import (
	"context"
	"net/http"
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

// Journey 23 named the agent's task in the error; Journey 24 still called
// define without id 45 times against that message. The tool now resolves the
// sole live task itself and says which one it used.
func TestMissingTaskIDResolvesTheAgentsOwnTask(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WUPHF_AGENT_SLUG", "cos")
	srv, auth := stubBroker(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/tasks" {
			_, _ = w.Write([]byte(`{"tasks":[
				{"id":"JOURN-2","owner":"cos","status":"in_progress","channel":"general"},
				{"id":"OLD-1","owner":"cos","status":"done","channel":"general"},
				{"id":"OTHER-9","owner":"nazz","status":"running","channel":"general"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"task":{"id":"JOURN-2","owner":"cos","status":"in_progress","title":"two pages"}}`))
	})
	defer srv.Close()
	withBrokerURL(t, srv.URL)

	res, _, err := handleTeamTask(context.Background(), nil, TeamTaskArgs{Action: "define", Channel: "general", Goal: "two pages"})
	if err != nil {
		t.Fatalf("must be a tool result: %v", err)
	}
	text := textFromResult(t, res)
	if !strings.Contains(text, "JOURN-2") {
		t.Fatalf("the result must name the task it acted on, got %q", text)
	}
	if !strings.Contains(text, "only open task") {
		t.Fatalf("the result must say the id was inferred, got %q", text)
	}
	if !strings.Contains(auth.lastBody, `"id":"JOURN-2"`) {
		t.Fatalf("the broker call must carry the resolved id, got %q", auth.lastBody)
	}
}

// Two live tasks means no single right answer, so the message stays generic.
func TestMissingTaskIDErrorStaysGenericWithTwoLiveTasks(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WUPHF_AGENT_SLUG", "cos")
	srv, _ := stubBroker(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"JOURN-2","owner":"cos","status":"planning"},
			{"id":"JOURN-3","owner":"cos","status":"running"}]}`))
	})
	defer srv.Close()
	withBrokerURL(t, srv.URL)

	res, _, err := handleTeamTask(context.Background(), nil, TeamTaskArgs{Action: "complete", Channel: "general"})
	if err != nil {
		t.Fatalf("must be a tool error result: %v", err)
	}
	text := textFromResult(t, res)
	if !strings.Contains(text, "needs the task id") {
		t.Fatalf("must still name the missing argument, got %q", text)
	}
	if strings.Contains(text, "JOURN-2") || strings.Contains(text, "JOURN-3") {
		t.Fatalf("with two candidates the error must not pick one, got %q", text)
	}
}
