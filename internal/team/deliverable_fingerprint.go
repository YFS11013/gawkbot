package team

// Deliverable fingerprints: the send-back path needs to know whether the thing
// the human rejected is still the thing they are being asked to accept.
//
// Journey 18 (core-loop-runs/2026-11-11-journey18): the human sent the public
// release notes back with one specific defect. The office replied "已按反馈修改"
// and re-raised the same card; the file was byte-identical (724 B, same mtime)
// and the wiki had no commit for that path afterwards. A second send-back, this
// time quoting the byte count and the last commit, produced a reply that invented
// a new byte count ("475 bytes, 7 lines") for a file that never changed. The
// card was the only acceptance surface, and it could not tell "fixed" from
// "untouched" — so a human who reads the note approves what they just rejected.
//
// The fingerprint below is that missing comparison. It is content-addressed (not
// size/mtime), so a same-size rewrite still counts as a change, and it is taken
// over the deliverable SET (a lane's artifact, or every file an Issue's lanes
// landed) because the container card approves the set, not one file.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// deliverableRootLocked is the tree the set digest reads: the broker's own
// wiki when it has one, the process wiki root otherwise (a broker assembled
// without a worker still has to answer "did this change"). Caller holds b.mu.
func (b *Broker) deliverableRootLocked() string {
	if root := b.wikiRootLocked(); root != "" {
		return root
	}
	return WikiRootDir()
}

// deliverableFingerprint digests the current on-disk state of a deliverable set
// under the caller's wiki root. Paths are wiki-relative (the same form
// TaskPostRequest.ArtifactPath carries). A missing file hashes as "missing"
// rather than being dropped, so deleting a deliverable still registers as a
// change. Returns "" for an empty set, which every caller reads as "nothing to
// compare".
//
// The root is a parameter, not WikiRootDir(): the broker resolves its own wiki
// (tests build one in a temp dir, and the stamp and the comparison have to hash
// the same tree or "unchanged" becomes a statement about an unrelated file).
func deliverableFingerprint(root string, paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	sorted := make([]string, 0, len(paths))
	for _, p := range paths {
		if p = strings.TrimSpace(p); p != "" {
			sorted = append(sorted, p)
		}
	}
	if len(sorted) == 0 {
		return ""
	}
	sort.Strings(sorted)
	h := sha256.New()
	for _, p := range sorted {
		fmt.Fprintf(h, "%s\x00", p)
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			h.Write([]byte("missing\x00"))
			continue
		}
		sum := sha256.Sum256(data)
		h.Write(sum[:])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// deliverablePathsLocked lists the files a decision card is asking the human to
// accept: the task's own artifact plus every child lane's recorded artifact.
// "ships no file" declarations are not files and are skipped. Caller holds b.mu.
func (b *Broker) deliverablePathsLocked(task *teamTask) []string {
	if b == nil || task == nil {
		return nil
	}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		for _, seen := range out {
			if SameWikiPath(seen, p) {
				return
			}
		}
		out = append(out, p)
	}
	// Journey 23 (core-loop-runs/2026-11-13-journey23): the definition lists
	// every file the work has to produce, and office work routinely ships more
	// than the one path recorded as the task's artifact. JOURN-2 wrote both
	// pages, but the digest only read the internal one — so when the human sent
	// it back over the PUBLIC page, a real rewrite left the digest untouched,
	// the next card claimed "none of those files has changed", and the model
	// made the digest move by adding a trailing space to the internal page and
	// reverting it one commit later. A set that only sees one file teaches the
	// bot to nudge that file.
	if task.Definition != nil {
		for _, d := range task.Definition.Deliverables {
			add(wikiPathFromDeliverableName(d.Name))
		}
	}
	// What the task actually published beats what its definition hoped it
	// would: Journey 24's define carried no deliverables list, and the two
	// pages it wrote were only discoverable from the write ledger.
	for _, published := range task.Artifacts {
		add(published)
	}
	add(task.Artifact)
	for _, entry := range b.childDeliverablesLocked(task) {
		if strings.Contains(entry, " (ships no file)") {
			continue
		}
		add(entry)
	}
	return out
}

// wikiPathFromDeliverableName reads a deliverable's name as a wiki path when it
// is written like one ("team/reviews/acme.md"). Deliverables are also allowed
// to be plain descriptions ("renewal brief"), and those carry no path.
func wikiPathFromDeliverableName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, " ") || !strings.Contains(name, "/") {
		return ""
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown", ".txt":
		return name
	}
	return ""
}

// stampSentBackFingerprint records what the human rejected, at the moment they
// rejected it, so the next card for the same task can tell whether anything
// moved. Reads the deliverable list under the lock, does the file IO outside it,
// then re-finds the task by id (it may have been mutated while the digest ran).
func (b *Broker) stampSentBackFingerprint(taskID string, at string) {
	if b == nil {
		return
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	b.mu.Lock()
	task := b.taskByIDLocked(taskID)
	if task == nil {
		b.mu.Unlock()
		return
	}
	paths := b.deliverablePathsLocked(task)
	root := b.deliverableRootLocked()
	b.mu.Unlock()

	fingerprint := deliverableFingerprint(root, paths)

	b.mu.Lock()
	defer b.mu.Unlock()
	task = b.taskByIDLocked(taskID)
	if task == nil {
		return
	}
	task.SentBackFingerprint = fingerprint
	task.SentBackAt = strings.TrimSpace(at)
	if err := b.saveLocked(); err != nil {
		log.Printf("broker: persist send-back fingerprint for task %q: %v", taskID, err)
	}
}

// clearSentBackFingerprint drops the comparison point once the work is
// accepted: the task is closing, and a later reopen must not inherit a digest
// from an earlier round.
// the task is closing, and a later reopen must not inherit a stale digest.
func (b *Broker) clearSentBackFingerprint(taskID string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	task := b.taskByIDLocked(strings.TrimSpace(taskID))
	if task == nil || (task.SentBackFingerprint == "" && task.SentBackAt == "") {
		return
	}
	task.SentBackFingerprint = ""
	task.SentBackAt = ""
	if err := b.saveLocked(); err != nil {
		log.Printf("broker: clear send-back fingerprint for task %q: %v", taskID, err)
	}
}

func (b *Broker) clearSentBackFingerprintLocked(task *teamTask) {
	if task == nil {
		return
	}
	task.SentBackFingerprint = ""
	task.SentBackAt = ""
}

// sentBackDeliverablesUnchangedLocked reports whether the deliverable set is
// byte-for-byte what it was when the human last sent this task back. The digest
// reads a handful of small wiki files, which is why the card builder tolerates
// the IO under the lock: it only runs for a task carrying a send-back stamp,
// once per convergence sweep, and the files are the very set the human is
// looking at. Caller holds b.mu.
func (b *Broker) sentBackDeliverablesUnchangedLocked(task *teamTask) bool {
	if b == nil || task == nil {
		return false
	}
	stamp := strings.TrimSpace(task.SentBackFingerprint)
	if stamp == "" {
		return false
	}
	paths := b.deliverablePathsLocked(task)
	current := deliverableFingerprint(b.deliverableRootLocked(), paths)
	return current != "" && current == stamp
}

// sentBackDeliverablesUnchanged reports whether the task's deliverable set is
// still byte-identical to the version a send-back rejected, and when that
// send-back happened. Returns false for a task with no un-cleared stamp, and
// false while a human request is open for it: a lane waiting on the human is
// not dodging the rework. Reads the paths under the lock and hashes outside it.
func (b *Broker) sentBackDeliverablesUnchanged(taskID string) (bool, string) {
	if b == nil {
		return false, ""
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false, ""
	}
	b.mu.Lock()
	task := b.taskByIDLocked(taskID)
	if task == nil {
		b.mu.Unlock()
		return false, ""
	}
	stamp := strings.TrimSpace(task.SentBackFingerprint)
	if stamp == "" {
		b.mu.Unlock()
		return false, ""
	}
	sentBackAt := strings.TrimSpace(task.SentBackAt)
	// A lane waiting on the human is not dodging the rework.
	for _, req := range b.requests {
		if strings.TrimSpace(req.IssueID) == task.ID && requestIsActive(req) {
			b.mu.Unlock()
			return false, ""
		}
	}
	paths := b.deliverablePathsLocked(task)
	root := b.deliverableRootLocked()
	b.mu.Unlock()

	current := deliverableFingerprint(root, paths)
	if current == "" || current != stamp {
		return false, ""
	}
	return true, sentBackAt
}

// RecordReworkUnwritten leaves the audit row for a turn that reported a rework
// without writing one. The ledger entry is the steering signal (the next packet
// renders it); this row is what an operator reading the office log sees.
func (b *Broker) RecordReworkUnwritten(taskID, actor, sentBackAt string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	task := b.taskByIDLocked(strings.TrimSpace(taskID))
	if task == nil {
		return
	}
	summary := "no file changed since the send-back at " + strings.TrimSpace(sentBackAt)
	b.appendActionLocked("rework_unwritten", actor, normalizeChannelSlug(task.Channel), actor, truncateSummary(summary, 140), task.ID)
	if err := b.saveLocked(); err != nil {
		log.Printf("broker: persist rework_unwritten row for task %q: %v", task.ID, err)
	}
}
