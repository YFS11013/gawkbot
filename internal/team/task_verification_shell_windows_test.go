//go:build windows

package team

import (
	"context"
	"strings"
	"testing"
)

// A verification spec that quotes a path is the normal shape on Windows
// (`python "C:\repo\tool.py" "team/prep/x.md"`). When the spec was passed as
// a single argv element, exec re-quoted it and cmd /C handed the child a path
// prefixed with the working directory, so the check never executed
// (ACMER-2920: `python ... Invalid argument`, exit 2 — a correct deliverable
// could not complete). The shell now receives the spec as a literal command
// line; these cases fail if that regresses.
func TestVerificationShellKeepsQuotedSpec(t *testing.T) {
	cases := []struct {
		spec string
		want string
	}{
		{`echo "a b"`, `"a b"`},
		{`echo prefix "a b" suffix`, `prefix "a b" suffix`},
		// A spec whose very first character is a quote: cmd strips the first
		// and last quote of a /C string, so an unwrapped spec starting with
		// `"C:\...\python.exe"` lost its opening quote and the rest of the
		// line was parsed as a broken path — observed as "The filename,
		// directory name, or volume label syntax is incorrect" with the
		// interpreter never running (OFFICE-3012).
		{`"cmd" /C echo quoted-exe`, `quoted-exe`},
		{`"cmd" /C echo "a b"`, `"a b"`},
	}
	for _, c := range cases {
		cmd := verificationShellCommand(context.Background(), c.spec)
		cmd.Dir = t.TempDir()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("spec %q: %v (output %q)", c.spec, err, out)
		}
		if got := strings.TrimSpace(string(out)); got != c.want {
			t.Fatalf("spec %q: got %q, want %q (quotes were mangled)", c.spec, got, c.want)
		}
	}
}
