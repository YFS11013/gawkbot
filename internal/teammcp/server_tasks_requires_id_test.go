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

// Journey 23 (core-loop-runs/2026-11-13-journey23): the model called define
// without id 54 times in one turn. The message told it what to pass ("the id
// create returned") but never which value, so every retry was a fresh guess.
// When the agent owns exactly one live task, the error names it.
func TestMissingTaskIDErrorNamesTheAgentsOwnTask(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WUPHF_AGENT_SLUG", "cos")
	srv, _ := stubBroker(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"JOURN-2","owner":"cos","status":"planning","channel":"general"},
			{"id":"OLD-1","owner":"cos","status":"done","channel":"general"},
			{"id":"OTHER-9","owner":"nazz","status":"running","channel":"general"}]}`))
	})
	defer srv.Close()
	withBrokerURL(t, srv.URL)

	res, _, err := handleTeamTask(context.Background(), nil, TeamTaskArgs{Action: "define", Channel: "general"})
	if err != nil {
		t.Fatalf("must be a tool error result: %v", err)
	}
	text := textFromResult(t, res)
	if !strings.Contains(text, "JOURN-2") {
		t.Fatalf("the error must name the agent's one live task, got %q", text)
	}
	if strings.Contains(text, "OLD-1") || strings.Contains(text, "OTHER-9") {
		t.Fatalf("closed tasks and other agents' tasks are not candidates, got %q", text)
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
