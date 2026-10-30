package team

import (
	"fmt"
	"net/http"
	"testing"
)

// getAppsFetch issues a GET /apps/fetch with the given token (empty = none)
// and returns the status code. The URL is passed verbatim so each case can
// probe one guard.
func getAppsFetch(t *testing.T, base, token, raw string) int {
	t.Helper()
	target := base + "/apps/fetch"
	if raw != "" {
		target += "?url=" + raw
	}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// The bridge proxy is a sandboxed app's only network path, so every guard is
// load-bearing: no token = no proxy (it would otherwise be an open SSRF
// endpoint on localhost), and the host allowlist must reject everything that
// is not an explicit public data endpoint. These cases never hit the network —
// each one is refused before the upstream request is built.
func TestAppsFetchGuards(t *testing.T) {
	t.Setenv("WUPHF_RUNTIME_HOME", t.TempDir())
	b := newTestBroker(t)
	if err := b.StartOnPort(0); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	defer b.Stop()
	base := fmt.Sprintf("http://%s", b.Addr())

	cases := []struct {
		name  string
		token string
		url   string
		want  int
	}{
		{"missing token", "", "https%3A%2F%2Fapi.github.com%2Fzen", http.StatusUnauthorized},
		{"missing url", b.Token(), "", http.StatusBadRequest},
		{"plain http", b.Token(), "http%3A%2F%2Fapi.github.com%2Fzen", http.StatusBadRequest},
		{"host not allowlisted", b.Token(), "https%3A%2F%2Fexample.com%2F", http.StatusForbidden},
		{"allowlist lookalike suffix", b.Token(), "https%3A%2F%2Fnotopencode.ai%2F", http.StatusForbidden},
		{"embedded credentials", b.Token(), "https%3A%2F%2Fuser%3Apass%40api.github.com%2Fzen", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getAppsFetch(t, base, tc.token, tc.url); got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}
