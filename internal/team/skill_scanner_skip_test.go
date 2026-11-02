package team

import "testing"

// One-off deliverables must not become skill candidates. The first two prep
// packets were registered as skills by the compile cron (`fund-invest-
// 2026-11-02`, `forex-watch-2026-11-03`), so a pack per meeting polluted the
// library that every bot reads. Guard the directory and the neighbours that
// DO carry reusable method.
func TestShouldSkipDirPrepDeliverables(t *testing.T) {
	cases := []struct {
		rel  string
		want bool
	}{
		{"team/prep", true},
		{"team/prep/fund-invest-2026-11-02", true},
		{"team/candidates", true},
		{"team/candidates/entity-ced07fb4-2026-11-05", true},
		{"team/style", true},
		{"team/style/diffs", true},
		{"team/styles", false}, // prefix must not match a sibling name
		{"team/routines", true},
		{"team/routines/scorecard-2026-11-02", true},
		{"team/skills", true},
		{"team/skills/.system", true},
		{"team/playbooks/.compiled", true},
		{"team/playbooks", false},
		{"team/reviews", false},
		{"team/learnings", false},
	}
	for _, c := range cases {
		if got := shouldSkipDir(c.rel); got != c.want {
			t.Errorf("shouldSkipDir(%q) = %v, want %v", c.rel, got, c.want)
		}
	}
}
