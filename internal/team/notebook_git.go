package team

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// notebook_git.go commits one bot-authored notebook note under
// agents/{slug}/notebook/{file}.md.
//
// Repo.Commit cannot serve this: it requires a team/ article path and
// regenerates index/all.md on every write, so routing a private note through it
// would either reject the path or leak the note into the shared catalog. The
// commit recipe here mirrors Repo.CommitRichArtifact — write, stage the one
// path, skip an empty diff, commit under the author's slug — minus the catalog.
//
// Callers must validate before reaching git (see validateNotebookWritePath);
// this repeats the check because the method is exported to the worker and must
// not depend on the transport having done it.
func (r *Repo) CommitNotebookNote(ctx context.Context, slug, relPath, content, mode, message string) (string, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := validateNotebookWritePath(slug, relPath); err != nil {
		return "", 0, err
	}
	switch mode {
	case "create", "replace", "append_section":
	default:
		return "", 0, fmt.Errorf("notebook: unknown write mode %q; expected create|replace|append_section", mode)
	}
	if strings.TrimSpace(content) == "" {
		return "", 0, fmt.Errorf("notebook: content is required")
	}

	rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(relPath)))
	full := filepath.Join(r.root, filepath.FromSlash(rel))
	if !isPathWithin(r.root, full) {
		return "", 0, fmt.Errorf("notebook: %q resolves outside the wiki root", relPath)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return "", 0, fmt.Errorf("notebook: mkdir %s: %w", filepath.Dir(full), err)
	}

	existing, readErr := os.ReadFile(full)
	exists := readErr == nil
	if readErr != nil && !os.IsNotExist(readErr) {
		return "", 0, fmt.Errorf("notebook: read %s: %w", rel, readErr)
	}
	if mode == "create" && exists {
		return "", 0, fmt.Errorf("notebook: %q already exists; use replace or append_section", rel)
	}

	var bytesWritten int
	switch {
	case strings.EqualFold(mode, "append_section"):
		var buf []byte
		if len(existing) > 0 {
			buf = append(buf, existing...)
			if !strings.HasSuffix(string(existing), "\n") {
				buf = append(buf, '\n')
			}
			buf = append(buf, '\n')
		}
		buf = append(buf, []byte(content)...)
		if err := os.WriteFile(full, buf, 0o600); err != nil {
			return "", 0, fmt.Errorf("notebook: write %s: %w", rel, err)
		}
		bytesWritten = len(content)
	case exists && string(existing) == content:
		// Byte-identical rewrite: leave the file (and its mtime) alone and let
		// the empty-diff check below return HEAD.
		bytesWritten = len(content)
	default:
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			return "", 0, fmt.Errorf("notebook: write %s: %w", rel, err)
		}
		bytesWritten = len(content)
	}

	if out, err := r.runGitLocked(ctx, slug, "add", "--", rel); err != nil {
		return "", 0, fmt.Errorf("notebook: git add %s: %w: %s", rel, err, out)
	}
	empty, err := r.cachedDiffEmptyLocked(ctx, slug)
	if err != nil {
		return "", 0, err
	}
	if empty {
		head, err := r.currentHeadLocked(ctx)
		return head, bytesWritten, err
	}

	commitMsg := strings.TrimSpace(message)
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("notebook: update %s", rel)
	}
	if out, err := r.runGitLocked(ctx, slug, "commit", "-q", "-m", commitMsg); err != nil {
		return "", 0, fmt.Errorf("notebook: git commit: %w: %s", err, out)
	}
	head, err := r.currentHeadLocked(ctx)
	return head, bytesWritten, err
}
