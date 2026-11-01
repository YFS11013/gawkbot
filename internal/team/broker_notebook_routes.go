package team

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// broker_notebook_routes.go serves /notebook/*, the surface the MCP context_*
// tools call (internal/teammcp/context_tools.go). Until these existed the
// broker registered no /notebook/ handler at all, so context_capture,
// context_lookup's notebook corpus, and context_promote all reached a 404 —
// while the client reported it as a RETRYABLE backend failure. Bots wrote
// "remember this" and the note was gone; one of them stopped trying and closed
// a task with the artifact marked "pending on the storage side".
//
// Boundary: a bot may write only under agents/{its own slug}/notebook/. Reads
// (catalog, search) span all bots' shelves because context_lookup needs to
// know who has notes at all; the search itself is scoped per slug so one bot's
// private note never leaks into another's result set. Promotion never writes
// the team wiki — it raises a human approval card, exactly like
// /agent-knowledge/promotion-request.
func (b *Broker) registerNotebookRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/notebook/catalog", b.requireAuth(b.handleNotebookCatalog))
	mux.HandleFunc("/notebook/search", b.requireAuth(b.handleNotebookSearch))
	mux.HandleFunc("/notebook/write", b.requireAuth(b.handleNotebookWrite))
	mux.HandleFunc("/notebook/promote", b.requireAuth(b.handleNotebookPromote))
}

func notebookMethodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

// handleNotebookCatalog lists every bot that has a notebook shelf.
//
//	GET /notebook/catalog -> {"agents":[{"agent_slug","path","file_count"}]}
func (b *Broker) handleNotebookCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		notebookMethodNotAllowed(w, http.MethodGet)
		return
	}
	worker := b.requireWikiWorker(w, "notebook")
	if worker == nil {
		return
	}
	entries, err := catalogBotNotebooks(worker.Repo().Root())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": entries})
}

// handleNotebookSearch does a literal-substring search inside ONE bot's shelf.
//
//	GET /notebook/search?slug=cos&q=launch -> {"hits":[{"path","line","snippet"}]}
func (b *Broker) handleNotebookSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		notebookMethodNotAllowed(w, http.MethodGet)
		return
	}
	worker := b.requireWikiWorker(w, "notebook")
	if worker == nil {
		return
	}
	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		slug = strings.TrimSpace(r.URL.Query().Get("agent_slug"))
	}
	if err := validateNotebookSlug(slug); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug is required: " + err.Error()})
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("pattern"))
	}
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "q is required"})
		return
	}
	hits, err := searchBotNotebook(worker.Repo().Root(), slug, query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

// handleNotebookWrite commits one note to the caller's own shelf.
//
//	POST /notebook/write {slug, path, mode, content, commit_message}
//	  -> 200 {path, commit_sha, bytes_written}
func (b *Broker) handleNotebookWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		notebookMethodNotAllowed(w, http.MethodPost)
		return
	}
	worker := b.requireWikiWorker(w, "notebook")
	if worker == nil {
		return
	}
	var body struct {
		Slug          string `json:"slug"`
		Path          string `json:"path"`
		Mode          string `json:"mode"`
		Content       string `json:"content"`
		CommitMessage string `json:"commit_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	mode := strings.TrimSpace(body.Mode)
	if mode == "" {
		mode = "create"
	}
	switch mode {
	case "create", "replace", "append_section":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("mode must be one of create | replace | append_section; got %q", mode)})
		return
	}
	if err := validateNotebookWritePath(body.Slug, body.Path); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	sha, n, err := worker.EnqueueNotebookNote(r.Context(), strings.TrimSpace(body.Slug), strings.TrimSpace(body.Path), body.Content, mode, body.CommitMessage)
	if err != nil {
		writeNotebookWriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":          strings.TrimSpace(body.Path),
		"commit_sha":    sha,
		"bytes_written": n,
	})
}

// handleNotebookPromote asks a human to promote one note into the team wiki.
// It never writes team/: the approval card carries a snapshot of the note, and
// an edit after the card is raised cannot change what lands.
//
//	POST /notebook/promote {my_slug, source_path, target_wiki_path, rationale}
//	  -> 202 {promotion_id, reviewer_slug, state, human_only}
func (b *Broker) handleNotebookPromote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		notebookMethodNotAllowed(w, http.MethodPost)
		return
	}
	var body struct {
		MySlug     string `json:"my_slug"`
		SourcePath string `json:"source_path"`
		TargetPath string `json:"target_wiki_path"`
		Rationale  string `json:"rationale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	slug := strings.TrimSpace(body.MySlug)
	source := strings.TrimSpace(body.SourcePath)
	target := strings.TrimSpace(body.TargetPath)
	rationale := strings.TrimSpace(body.Rationale)
	if err := validateNotebookWritePath(slug, source); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	switch {
	case rationale == "":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rationale is required: say why this note belongs in the team wiki"})
		return
	case !strings.HasPrefix(target, "team/"):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("target_wiki_path %q must start with team/", target)})
		return
	}
	if err := validateArticlePath(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	id, err := b.requestKnowledgePromotion(slug, source, target, rationale)
	if err != nil {
		writeKnowledgePromotionError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"promotion_id":  id,
		"reviewer_slug": "",
		"state":         knowledgePromotionStatePending,
		"human_only":    true,
	})
}

// writeNotebookWriteError keeps the honest status codes: a saturated queue is a
// 429 the caller may retry, an inactive backend is a 503, and a rejected path
// or an existing file is a 400 the caller must fix. The previous surface sent
// everything as "retryable backend_error", including the 404 that never healed.
func writeNotebookWriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrQueueSaturated):
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrWorkerStopped), strings.Contains(err.Error(), "backend is not active"):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
}

// notebookCatalogEntry is one row of GET /notebook/catalog.
type notebookCatalogEntry struct {
	AgentSlug string `json:"agent_slug"`
	Path      string `json:"path"`
	FileCount int    `json:"file_count"`
}

// catalogBotNotebooks lists agents/{slug}/notebook/ directories that exist on
// disk. Shelf order is stable (slug asc) so a caller diffing two catalogs sees
// only real additions.
func catalogBotNotebooks(root string) ([]notebookCatalogEntry, error) {
	agentsDir := filepath.Join(root, "agents")
	dirs, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []notebookCatalogEntry{}, nil
		}
		return nil, fmt.Errorf("notebook: read agents dir: %w", err)
	}
	entries := make([]notebookCatalogEntry, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() || validateNotebookSlug(dir.Name()) != nil {
			continue
		}
		shelf := filepath.Join(agentsDir, dir.Name(), "notebook")
		info, statErr := os.Stat(shelf)
		if statErr != nil || !info.IsDir() {
			continue
		}
		files, readErr := os.ReadDir(shelf)
		if readErr != nil {
			continue
		}
		count := 0
		for _, f := range files {
			if !f.IsDir() && strings.EqualFold(filepath.Ext(f.Name()), ".md") {
				count++
			}
		}
		entries = append(entries, notebookCatalogEntry{
			AgentSlug: dir.Name(),
			Path:      "agents/" + dir.Name() + "/notebook",
			FileCount: count,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].AgentSlug < entries[j].AgentSlug })
	return entries, nil
}

// searchBotNotebook walks ONE shelf and returns literal-substring hits. Same
// shape and 100-hit ceiling as the team wiki search, and deliberately literal:
// bots never get to inject a regex.
func searchBotNotebook(root, slug, pattern string) ([]WikiSearchHit, error) {
	shelf := filepath.Join(root, "agents", slug, "notebook")
	const maxHits = 100
	hits := make([]WikiSearchHit, 0, 16)
	if _, err := os.Stat(shelf); err != nil {
		if os.IsNotExist(err) {
			return hits, nil
		}
		return nil, fmt.Errorf("notebook: stat %s: %w", slug, err)
	}
	err := filepath.Walk(shelf, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		if len(hits) >= maxHits {
			return filepath.SkipDir
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil //nolint:nilerr // non-fatal: skip unreadable file (race with delete)
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil //nolint:nilerr // non-fatal: path outside root is not a shelf entry
		}
		rel = filepath.ToSlash(rel)
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if !strings.Contains(line, pattern) {
				continue
			}
			hits = append(hits, WikiSearchHit{Path: rel, Line: lineNo, Snippet: strings.TrimSpace(line)})
			if len(hits) >= maxHits {
				return filepath.SkipDir
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("notebook: scan %s: %w", rel, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("notebook: search walk: %w", err)
	}
	return hits, nil
}
