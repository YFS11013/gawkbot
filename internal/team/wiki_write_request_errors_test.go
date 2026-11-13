package team

// Journey 27's bot wrote its page twice: the first call used mode=create on a
// path that already existed and got "broker POST /wiki/write-task failed: 500
// Internal Server Error {article already exists … use replace or
// append_section}". The text told the bot exactly what to do; the code told it
// the broker was broken. A caller-shaped failure is a 4xx.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWikiWriteRequestErrorsAreClientErrors(t *testing.T) {
	b, _, stop := newKnowledgeBroker(t)
	defer stop()
	const path = "team/reviews/j27-public.md"
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{ID: "JOURN-7", Channel: "team", Title: "Public note",
		Owner: "cos", CreatedBy: "cos", status: "in_progress", LifecycleState: LifecycleStateRunning,
		Definition: &TaskDefinition{Goal: "public note", ArtifactPath: path, DefinedAt: now},
		CreatedAt:  now, UpdatedAt: now})
	b.mu.Unlock()
	if _, _, err := b.WikiWorker().EnqueueTaskArtifact(t.Context(), "cos", path, "first\n", "create", ""); err != nil {
		t.Fatalf("seed the page: %v", err)
	}

	cases := []struct {
		name string
		body string
		want int
	}{
		{"create on an existing page", `{"slug":"cos","task_id":"JOURN-7","path":"` + path + `","content":"second\n","mode":"create"}`, http.StatusConflict},
		{"unknown mode", `{"slug":"cos","task_id":"JOURN-7","path":"` + path + `","content":"x\n","mode":"upsert"}`, http.StatusBadRequest},
		{"empty content", `{"slug":"cos","task_id":"JOURN-7","path":"` + path + `","content":"   ","mode":"replace"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		b.handleWikiWriteTask(rec, httptest.NewRequest(http.MethodPost, "/wiki/write-task", strings.NewReader(tc.body)))
		if rec.Code != tc.want {
			t.Fatalf("%s: want %d, got %d: %s", tc.name, tc.want, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "Internal Server Error") {
			t.Fatalf("%s: a request-shaped failure must not read as a server fault: %s", tc.name, rec.Body.String())
		}
	}
}
