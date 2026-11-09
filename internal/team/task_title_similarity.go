package team

import (
	"strings"
	"unicode"
)

// task_title_similarity.go provides normalized, fuzzy title matching used to
// catch duplicate and shallow-restating tasks that exact-title dedup misses
// ("Ship the MVP" vs "Build the first MVP slice" vs "ship mvp!"). The exact
// EqualFold check the reuse path used only collapses byte-identical titles, so
// the model spawned near-duplicates freely; this widens the net deterministically.

// titleStopwords are low-signal tokens dropped before comparing titles, so that
// filler and politeness words do not make two restatements of the same work
// look different. Kept deliberately small — removing real nouns/verbs would
// over-merge distinct work.
var titleStopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "to": {}, "for": {}, "of": {}, "and": {},
	"or": {}, "in": {}, "on": {}, "with": {}, "our": {}, "my": {}, "your": {},
	"this": {}, "that": {}, "please": {}, "can": {}, "you": {}, "we": {},
	"i": {}, "us": {}, "it": {}, "is": {}, "be": {}, "into": {}, "up": {},
	"first": {}, "new": {}, "task": {}, "issue": {},
}

// normalizeTitleTokens lowercases the title, strips punctuation, splits on
// whitespace, and drops stopwords. Returns the surviving content tokens in
// order. Empty input (or input that is all stopwords/punctuation) yields nil.
func normalizeTitleTokens(title string) []string {
	lowered := strings.ToLower(strings.TrimSpace(title))
	if lowered == "" {
		return nil
	}
	fields := strings.FieldsFunc(lowered, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		// Stem BEFORE the stopword check so plural stopwords drop too: "issues"
		// stems to "issue" (a stopword) and is removed, keeping "Create issue"
		// and "Create issues" identical instead of diverging.
		stemmed := stemTitleToken(f)
		if _, stop := titleStopwords[stemmed]; stop {
			continue
		}
		out = append(out, stemmed)
	}
	return out
}

// stemTitleToken applies a deliberately tiny stem so trivial singular/plural
// variants collapse ("webhooks" == "webhook", "emails" == "email"). It strips a
// trailing "s" from tokens longer than three letters; anything fancier risks
// over-merging distinct words, and this is only a dedup heuristic.
func stemTitleToken(tok string) string {
	// Skip "ss" (address) and "us" (focus, status, nexus) endings — stripping
	// their trailing "s" produces non-words and never reflects a real plural.
	if len(tok) > 3 && strings.HasSuffix(tok, "s") &&
		!strings.HasSuffix(tok, "ss") && !strings.HasSuffix(tok, "us") {
		return tok[:len(tok)-1]
	}
	return tok
}

// normalizeTitleKey joins the normalized content tokens into a stable, sorted-
// free key. Two titles with the same key (after stopword/punctuation removal,
// same token order) are treated as exact restatements.
func normalizeTitleKey(title string) string {
	return strings.Join(normalizeTitleTokens(title), " ")
}

// titleTokenSimilarity returns the Jaccard similarity (0..1) of the two titles'
// normalized content-token SETS. Order-insensitive so "MVP onboarding flow" and
// "onboarding flow for the MVP" score high. Returns 0 when either side has no
// content tokens.
func titleTokenSimilarity(a, b string) float64 {
	ta := normalizeTitleTokens(a)
	tb := normalizeTitleTokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	setA := make(map[string]struct{}, len(ta))
	for _, t := range ta {
		setA[t] = struct{}{}
	}
	setB := make(map[string]struct{}, len(tb))
	for _, t := range tb {
		setB[t] = struct{}{}
	}
	inter := 0
	for t := range setA {
		if _, ok := setB[t]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// titleSimilarityThreshold is the Jaccard cutoff at/above which two titles are
// treated as the same intent. Set conservatively high: dedup with title+owner
// only already risks over-merging (see findReusableTaskLocked), so the fuzzy
// widening must stay tight enough that genuinely-distinct work is not collapsed.
// 0.85 specifically clears the subset trap — a 4-token title that is a strict
// subset of a 5-token title scores 4/5=0.8, which must NOT merge (those are a
// scope refinement, not a duplicate). Exact restatements still match via the
// normalized-key equality check, which is independent of this threshold.
const titleSimilarityThreshold = 0.85

// titlesAreSimilar reports whether two titles name the same work for dedup /
// shallow-subtask purposes. True when their normalized keys match exactly OR
// their token-set Jaccard similarity meets titleSimilarityThreshold.
func titlesAreSimilar(a, b string) bool {
	keyA := normalizeTitleKey(a)
	keyB := normalizeTitleKey(b)
	if keyA == "" || keyB == "" {
		return false
	}
	if keyA == keyB {
		return true
	}
	return titleTokenSimilarity(a, b) >= titleSimilarityThreshold
}

// cjkBigrams returns the contiguous two-rune shingles of every CJK/Han run in
// s. Whitespace, digits, and Latin letters separate runs and are not shingled.
//
// Why this exists: CJK text has no spaces, so normalizeTitleTokens collapses a
// whole clause into ONE token and the similarity above degenerates into
// exact-phrase matching. Journey 8 created two lanes for the same slice one
// second apart — "JOURN-6-A: Acme 一页报价说明初稿" and "子任务 A — 一页报价说明初稿"
// — and token Jaccard scored them 0.20 (0.43 for the 48h-follow-up pair), far
// under the 0.85 cutoff, so the sibling guard let both through. Shingles give
// the overlap something to count.
func cjkBigrams(s string) []string {
	var out []string
	run := make([]rune, 0, 16)
	flush := func() {
		if len(run) == 1 {
			out = append(out, string(run[0]))
		}
		for i := 0; i+1 < len(run); i++ {
			out = append(out, string(run[i:i+2]))
		}
		run = run[:0]
	}
	for _, r := range strings.ToLower(s) {
		if unicode.Is(unicode.Han, r) {
			run = append(run, r)
			continue
		}
		flush()
	}
	flush()
	return out
}

// CJK phrase matching thresholds. Measured against the Journey 8 sibling pairs
// and the cases the guard must NOT merge:
//
//	                                      sharing |A∩B|/min(|A|,|B|)
//	"…一页报价说明初稿" / "子任务 A — 一页报价说明初稿"        7/7 = 1.00  duplicate
//	"…48 小时跟进邮件草稿" / "子任务 B — 48 小时跟进邮件草稿"   7/7 = 1.00  duplicate
//	"子任务 C：数字与事实终审（Acme 续约）" / "Acme 续约数字与事实终审" 7/8 = 0.875 duplicate (J9)
//	"逐项核对…退回结论。" / "子任务 C — 出厂质检"            0/5 = 0.00  distinct
//	"Acme 续约报价初稿" / "Acme 续约报价终稿"              4/6 = 0.67  distinct
//	"审阅 JOURN-15 的交付" / "审阅 JOURN-19 的交付"        3/3 = 1.00  → blocked by the floor
//
// Containment (not Jaccard) because the planner pads a lane title with the
// parent id and a lane letter, and those extra shingles must not dilute a
// genuinely shared phrase. The floor keeps short generic titles ("审阅…的交付")
// from collapsing: a match needs at least six shared shingles, i.e. a shared
// phrase of ~7 characters — a real deliverable name, not a verb.
//
// 0.9 was the Journey 8 threshold and Journey 9 walked just under it: the
// planner restated "子任务 C：数字与事实终审（Acme 续约）" as "Acme 续约数字与
// 事实终审" (0.875), renamed the write path by one word to dodge the
// artifact_path rule, and shipped the same review twice. 0.85 catches it while
// staying above the 0.67 draft/final pair; the 0.80 "审阅 JOURN-N 的交付" shape
// is stopped by the shingle floor, not by the threshold.
const (
	lanePhraseContainment = 0.85
	lanePhraseMinShingles = 6
)

// laneTitlesSharePhrase reports whether two lane titles carry the same CJK
// deliverable phrase. Used only by the sibling guard, on top of
// titlesAreSimilar; deliberately NOT folded into it, because the threshold that
// is safe here (containment on shingles) is not safe for the shallower
// restates-parent check.
func laneTitlesSharePhrase(a, b string) bool {
	sa := make(map[string]struct{}, 16)
	for _, s := range cjkBigrams(a) {
		sa[s] = struct{}{}
	}
	sb := make(map[string]struct{}, 16)
	for _, s := range cjkBigrams(b) {
		sb[s] = struct{}{}
	}
	minN := len(sa)
	if len(sb) < minN {
		minN = len(sb)
	}
	if minN < lanePhraseMinShingles {
		return false
	}
	shared := 0
	for s := range sa {
		if _, ok := sb[s]; ok {
			shared++
		}
	}
	return float64(shared)/float64(minN) >= lanePhraseContainment
}
