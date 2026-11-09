package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A write to the wrong path is refused with both paths named, on the call, and
// nowhere else. Journey 8: four lanes hit this and every one corrected inside
// five seconds — off the 403 body, not a channel broadcast. A chat notice would
// repeat the same two paths to an audience that cannot act on them, while the
// "nobody declared a path" case needs the creator and does get one.
func TestWriteTaskMismatchStaysOnTheCall(t *testing.T) {
	b, repo, stop := newKnowledgeBroker(t)
	defer stop()
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: "JOURN-918", Channel: "team", Title: "QA pass", Owner: "dwight",
		CreatedBy: "cos", status: "in_progress",
		Definition: &TaskDefinition{
			Goal: "QA the quote", DefinedAt: "2026-11-09T09:30:00Z",
			ArtifactPath: "team/reviews/journ-918.md",
		},
	})
	beforeMessages, beforeActions := len(b.messages), len(b.actions)
	b.mu.Unlock()

	post := func(writePath string) *httptest.ResponseRecorder {
		body := strings.NewReader(`{"slug":"dwight","task_id":"JOURN-918","path":"` + writePath + `","content":"# QA\n","mode":"create"}`)
		rec := httptest.NewRecorder()
		b.handleWikiWriteTask(rec, httptest.NewRequest(http.MethodPost, "/wiki/write-task", body))
		return rec
	}

	rec := post("team/reports/journ-918.md")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a write outside the declared scope must be 403, got %d: %s", rec.Code, rec.Body.String())
	}
	var refusal struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("refusal must be JSON: %v (%s)", err, rec.Body.String())
	}
	for _, want := range []string{"team/reviews/journ-918.md", "team/reports/journ-918.md", "declares artifact_path"} {
		if !strings.Contains(refusal.Error, want) {
			t.Errorf("the refusal must name %q so the bot can retry without asking: got %q", want, refusal.Error)
		}
	}

	b.mu.Lock()
	extraMessages, extraActions := len(b.messages)-beforeMessages, len(b.actions)-beforeActions
	b.mu.Unlock()
	if extraMessages != 0 || extraActions != 0 {
		t.Fatalf("a path mismatch must not post to the channel: %d message(s), %d action(s)", extraMessages, extraActions)
	}

	rec = post("team/reviews/journ-918.md")
	if rec.Code != http.StatusOK {
		t.Fatalf("the declared path must be accepted on the retry, got %d: %s", rec.Code, rec.Body.String())
	}
	// The worker commits off the request goroutine; poll rather than sleep a
	// fixed slice that a slow CI box would outrun.
	want := filepath.Join(repo.Root(), "team/reviews/journ-918.md")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(want); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the accepted write never reached %s", want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
