package teammcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Journey 11 (core-loop-runs/2026-11-10-journey11, JOURN-26): the QA lane called
// team_wiki_search with the argument name its sibling tools use (`query`), got
// `pattern is required`, and reported to the human that both deliverables
// "exist nowhere". The alias closes that gap at the tool boundary instead of
// teaching every prompt the one right spelling.
func TestTeamWikiSearchAcceptsQueryAlias(t *testing.T) {
	var seenQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hits": []map[string]any{{
				"path":    "team/accounts/acme-renewal-quote.md",
				"line":    1,
				"snippet": "# Acme — Renewal Quote",
			}},
		})
	}))
	defer srv.Close()
	withBrokerURL(t, srv.URL)
	t.Setenv("WUPHF_AGENT_SLUG", "reviewer")

	res, _, err := handleTeamWikiSearch(context.Background(), nil, TeamWikiSearchArgs{Query: "Acme"})
	if err != nil {
		t.Fatalf("search with query alias: %v", err)
	}
	if isToolError(res) {
		t.Fatalf("query must be accepted as an alias for pattern, got error %q", toolErrorText(res))
	}
	if !strings.Contains(seenQuery, "pattern=Acme") {
		t.Fatalf("the alias must reach the broker as pattern=, got query string %q", seenQuery)
	}
	if !strings.Contains(seenQuery, "reader=reviewer") {
		t.Fatalf("the reader identity must still be attached, got %q", seenQuery)
	}
	if text := toolErrorText(res); !strings.Contains(text, "acme-renewal-quote.md") {
		t.Fatalf("hits must be returned verbatim, got %q", text)
	}
}

// `pattern` still wins when both names are present, so a caller that learns the
// real field does not get silently overridden by a stale alias.
func TestTeamWikiSearchPrefersPatternOverQuery(t *testing.T) {
	var seenQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"hits": []map[string]any{}})
	}))
	defer srv.Close()
	withBrokerURL(t, srv.URL)

	if _, _, err := handleTeamWikiSearch(context.Background(), nil, TeamWikiSearchArgs{Pattern: "Renewal", Query: "Acme"}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(seenQuery, "pattern=Renewal") {
		t.Fatalf("pattern must win over query, got %q", seenQuery)
	}
}

// Zero hits used to be `[]`, which reads as "this does not exist" to anyone who
// never opened the path. Say what was searched.
func TestTeamWikiSearchZeroHitsSaysWhatItSearched(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"hits": []map[string]any{}})
	}))
	defer srv.Close()
	withBrokerURL(t, srv.URL)
	t.Setenv("WUPHF_AGENT_SLUG", "reviewer")

	res, _, err := handleTeamWikiSearch(context.Background(), nil, TeamWikiSearchArgs{Pattern: "renewal quote"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	text := toolErrorText(res)
	for _, want := range []string{`"renewal quote"`, "reviewer", "team_wiki_read"} {
		if !strings.Contains(text, want) {
			t.Fatalf("zero-hit answer must name %q, got %q", want, text)
		}
	}
	if strings.TrimSpace(text) == "[]" || strings.TrimSpace(text) == "null" {
		t.Fatalf("zero hits must not be reported as an empty JSON list, got %q", text)
	}
}

// The remaining error is unreachable-by-accident: if a caller sends neither
// name, the message has to say both are valid.
func TestTeamWikiSearchEmptyArgsNameBothFields(t *testing.T) {
	withBrokerURL(t, "http://127.0.0.1:1")

	res, _, err := handleTeamWikiSearch(context.Background(), nil, TeamWikiSearchArgs{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !isToolError(res) {
		t.Fatal("a search with no substring must be an error")
	}
	text := toolErrorText(res)
	for _, want := range []string{"pattern", "query"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the error must name %q as accepted, got %q", want, text)
		}
	}
}
