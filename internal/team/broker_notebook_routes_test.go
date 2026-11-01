package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// notebookTestServe wires registerNotebookRoutes behind a mux with a valid
// bearer token, so the tests exercise the same auth + path registration the
// broker serves.
func notebookTestServe(t *testing.T, b *Broker, method, target string, body string) *httptest.ResponseRecorder {
	t.Helper()
	b.token = "notebook-test-token"
	mux := http.NewServeMux()
	b.registerNotebookRoutes(mux)
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+b.Token())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestNotebookWriteCommitsToCallersOwnShelf(t *testing.T) {
	b, repo, teardown := newKnowledgeBroker(t)
	defer teardown()

	rec := notebookTestServe(t, b, http.MethodPost, "/notebook/write", `{
		"slug":"dwight",
		"path":"agents/dwight/notebook/beet-farm-2026-11-01.md",
		"mode":"create",
		"content":"# Beet farm\n\nQuota notes.\n",
		"commit_message":"notebook: beet farm notes"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Path         string `json:"path"`
		CommitSHA    string `json:"commit_sha"`
		BytesWritten int    `json:"bytes_written"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.CommitSHA == "" || got.BytesWritten == 0 {
		t.Fatalf("want a commit sha and bytes written, got %+v", got)
	}

	abs := filepath.Join(repo.Root(), "agents", "dwight", "notebook", "beet-farm-2026-11-01.md")
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("note not on disk: %v", err)
	}
	if !strings.Contains(string(data), "Quota notes.") {
		t.Fatalf("note body not written: %q", string(data))
	}
	// The commit must be in git, attributed to the author — a notebook note
	// that only lands on disk is lost on the next checkout. Repo.Log restricts
	// itself to team/ articles, so read the history directly.
	out, err := exec.Command("git", "-C", repo.Root(), "log", "-1", "--pretty=%s", "--",
		"agents/dwight/notebook/beet-farm-2026-11-01.md").CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "beet farm notes") {
		t.Fatalf("want the commit in history, got %q", string(out))
	}
	// And it must NOT leak into the shared catalog.
	index, _ := os.ReadFile(filepath.Join(repo.Root(), "index", "all.md"))
	if strings.Contains(string(index), "beet-farm") {
		t.Fatalf("private note leaked into index/all.md")
	}
}

func TestNotebookWriteRejectsAnotherBotsShelf(t *testing.T) {
	b, repo, teardown := newKnowledgeBroker(t)
	defer teardown()

	rec := notebookTestServe(t, b, http.MethodPost, "/notebook/write", `{
		"slug":"dwight",
		"path":"agents/kevin/notebook/chili-recipe.md",
		"mode":"create",
		"content":"# Chili\n"
	}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("cross-shelf write: want 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "notebook_path_not_author_owned") {
		t.Fatalf("want the ownership error named, got %s", rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), "agents", "kevin", "notebook", "chili-recipe.md")); err == nil {
		t.Fatalf("cross-shelf note was created")
	}
}

func TestNotebookWriteCreateRefusesExistingNote(t *testing.T) {
	b, repo, teardown := newKnowledgeBroker(t)
	defer teardown()
	writeNote(t, repo, "dwight", "existing.md", "# Existing\n")

	rec := notebookTestServe(t, b, http.MethodPost, "/notebook/write", `{
		"slug":"dwight",
		"path":"agents/dwight/notebook/existing.md",
		"mode":"create",
		"content":"# Replaced\n"
	}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for create-over-existing, got %d: %s", rec.Code, rec.Body.String())
	}
	data, _ := os.ReadFile(filepath.Join(repo.Root(), "agents", "dwight", "notebook", "existing.md"))
	if !strings.Contains(string(data), "Existing") {
		t.Fatalf("existing note was clobbered: %q", string(data))
	}

	// append_section is the supported way to add to it.
	rec = notebookTestServe(t, b, http.MethodPost, "/notebook/write", `{
		"slug":"dwight",
		"path":"agents/dwight/notebook/existing.md",
		"mode":"append_section",
		"content":"## Later\n\nMore.\n"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("append: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data, _ = os.ReadFile(filepath.Join(repo.Root(), "agents", "dwight", "notebook", "existing.md"))
	if !strings.Contains(string(data), "Existing") || !strings.Contains(string(data), "More.") {
		t.Fatalf("append lost content: %q", string(data))
	}
}

func TestNotebookCatalogListsShelvesWithCounts(t *testing.T) {
	b, repo, teardown := newKnowledgeBroker(t)
	defer teardown()
	writeNote(t, repo, "dwight", "a.md", "# A\n")
	writeNote(t, repo, "dwight", "b.md", "# B\n")
	writeNote(t, repo, "kevin", "c.md", "# C\n")

	rec := notebookTestServe(t, b, http.MethodGet, "/notebook/catalog", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Agents []notebookCatalogEntry `json:"agents"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	counts := map[string]int{}
	for _, e := range got.Agents {
		counts[e.AgentSlug] = e.FileCount
	}
	if counts["dwight"] != 2 || counts["kevin"] != 1 {
		t.Fatalf("want dwight=2 kevin=1, got %+v", got.Agents)
	}
}

func TestNotebookSearchStaysInsideOneShelf(t *testing.T) {
	b, repo, teardown := newKnowledgeBroker(t)
	defer teardown()
	writeNote(t, repo, "dwight", "own.md", "# Own\n\nthe beet quota is 40\n")
	writeNote(t, repo, "kevin", "other.md", "# Other\n\nthe beet quota is a secret\n")

	rec := notebookTestServe(t, b, http.MethodGet, "/notebook/search?slug=dwight&q=beet", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Hits []WikiSearchHit `json:"hits"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Hits) != 1 {
		t.Fatalf("want exactly the caller's own hit, got %+v", got.Hits)
	}
	if !strings.HasPrefix(got.Hits[0].Path, "agents/dwight/notebook/") {
		t.Fatalf("hit escaped the caller's shelf: %+v", got.Hits[0])
	}

	rec = notebookTestServe(t, b, http.MethodGet, "/notebook/search?q=beet", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing slug must be 400, got %d", rec.Code)
	}
}

func TestNotebookPromoteRaisesPendingHumanCard(t *testing.T) {
	b, repo, teardown := newKnowledgeBroker(t)
	defer teardown()
	writeNote(t, repo, "dwight", "shareable.md", "# Beet know-how\n\nHow to count beets.\n")

	body := `{
		"my_slug":"dwight",
		"source_path":"agents/dwight/notebook/shareable.md",
		"target_wiki_path":"team/notes/beet-counting.md",
		"rationale":"the team recounts beets every quarter"
	}`
	rec := notebookTestServe(t, b, http.MethodPost, "/notebook/promote", body)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		PromotionID string `json:"promotion_id"`
		State       string `json:"state"`
		HumanOnly   bool   `json:"human_only"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.PromotionID == "" || got.State != knowledgePromotionStatePending || !got.HumanOnly {
		t.Fatalf("want a pending human-only card, got %+v", got)
	}
	// The bot's ask must not have created the team article.
	if _, err := os.Stat(filepath.Join(repo.Root(), "team", "notes", "beet-counting.md")); err == nil {
		t.Fatalf("promotion wrote the team wiki without approval")
	}

	// Same content twice = one card, not two.
	rec = notebookTestServe(t, b, http.MethodPost, "/notebook/promote", body)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("repeat: want 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var again struct {
		PromotionID string `json:"promotion_id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&again)
	if again.PromotionID != got.PromotionID {
		t.Fatalf("duplicate card: %q != %q", again.PromotionID, got.PromotionID)
	}
}

func TestNotebookPromoteRequiresRationaleAndTeamTarget(t *testing.T) {
	b, repo, teardown := newKnowledgeBroker(t)
	defer teardown()
	writeNote(t, repo, "dwight", "shareable.md", "# Beet know-how\n")

	cases := []struct{ name, body, want string }{
		{
			name: "no rationale",
			body: `{"my_slug":"dwight","source_path":"agents/dwight/notebook/shareable.md","target_wiki_path":"team/x.md"}`,
			want: "rationale is required",
		},
		{
			name: "target outside team/",
			body: `{"my_slug":"dwight","source_path":"agents/dwight/notebook/shareable.md","target_wiki_path":"agents/dwight/notebook/x.md","rationale":"r"}`,
			want: "must start with team/",
		},
		{
			name: "source outside own shelf",
			body: `{"my_slug":"dwight","source_path":"agents/kevin/notebook/x.md","target_wiki_path":"team/x.md","rationale":"r"}`,
			want: "notebook_path_not_author_owned",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := notebookTestServe(t, b, http.MethodPost, "/notebook/promote", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("want 400, got %d: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("want %q in body, got %s", tc.want, rec.Body.String())
			}
		})
	}
}

func TestNotebookRoutesRequireBearer(t *testing.T) {
	b, _, teardown := newKnowledgeBroker(t)
	defer teardown()
	b.token = "notebook-test-token"
	mux := http.NewServeMux()
	b.registerNotebookRoutes(mux)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/notebook/catalog"},
		{http.MethodGet, "/notebook/search?slug=dwight&q=x"},
		{http.MethodPost, "/notebook/write"},
		{http.MethodPost, "/notebook/promote"},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without bearer: want 401, got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestNotebookRoutesRejectWrongMethodWithAllow(t *testing.T) {
	b, _, teardown := newKnowledgeBroker(t)
	defer teardown()

	rec := notebookTestServe(t, b, http.MethodGet, "/notebook/write", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow: want POST, got %q", allow)
	}
}
