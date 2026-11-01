package action

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The fake One CLI is the test binary itself rather than a shell script.
//
// The scripts this package used were `#!/bin/sh` with `[ "$1" = ... ]`
// branches, which Windows cannot run at all: no shebang, no /bin/sh, and a
// file without an extension is not a program there. Standing in as the test
// binary keeps one fake for every platform and removes the shell from the
// picture entirely.
const fakeOneEnv = "WUPHF_FAKE_ONE"

// fakeOneTraceEnv makes the fake write its working directory to a file, so a
// test can assert which directory the CLI was invoked from.
const fakeOneTraceEnv = "WUPHF_FAKE_ONE_TRACE"

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeOneEnv); mode != "" {
		os.Exit(runFakeOne(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// installFakeOne copies the test binary into dir under name and returns the
// path to run. On Windows the copy has to keep the .exe suffix.
func installFakeOne(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		name += ".exe"
	}
	dst := filepath.Join(dir, name)
	self, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	if err := os.WriteFile(dst, self, 0o755); err != nil {
		t.Fatalf("install fake one: %v", err)
	}
	return dst
}

// fakeOneCLI installs the fake in a fresh directory, tells the child which
// mode to run, and returns the path to put in OneCLI.Bin.
func fakeOneCLI(t *testing.T, mode string) string {
	t.Helper()
	t.Setenv(fakeOneEnv, mode)
	return installFakeOne(t, t.TempDir(), "one")
}

func runFakeOne(mode string, args []string) int {
	if len(args) > 0 && args[0] == "--agent" {
		args = args[1:]
	}
	if mode == "npx" {
		if len(args) < 4 || args[0] != "-y" || args[1] != "@withone/cli" || args[2] != "--agent" {
			fmt.Fprintf(os.Stderr, "unexpected prefix: %v\n", args)
			return 1
		}
		args = args[3:]
	}
	if trace := os.Getenv(fakeOneTraceEnv); trace != "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := os.WriteFile(trace, []byte(wd+"\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	joined := strings.Join(args, " ")
	emit := func(s string) int {
		fmt.Println(s)
		return 0
	}
	// The real CLI takes trailing flags ("actions search gmail send email
	// --type execute"), so the branches match on their command prefix and
	// ignore the rest, exactly as the shell stand-in did with $1 $2 $3.
	switch {
	case joined == "list":
		return emit(`{"total":1,"showing":1,"connections":[{"platform":"gmail","state":"operational","key":"live::gmail::default::abc123"}]}`)
	case strings.HasPrefix(joined, "actions search gmail"):
		return emit(`{"actions":[{"actionId":"act-send","title":"Send Email","method":"POST","path":"/gmail/send"}]}`)
	case strings.HasPrefix(joined, "actions knowledge gmail"):
		return emit(`{"knowledge":"Needs to, subject, body","method":"POST"}`)
	case strings.HasPrefix(joined, "actions execute gmail"):
		return emit(`{"dryRun":true,"request":{"method":"POST","url":"https://api.withone.ai/send","headers":{"x-test":"1"},"data":{"to":"a@example.com"}}}`)
	case strings.HasPrefix(joined, "relay event-types gmail"):
		return emit(`{"platform":"gmail","eventTypes":["message.received"]}`)
	case strings.HasPrefix(joined, "relay activate relay-1"):
		return emit(`{"id":"relay-1","active":true,"actions":[{"type":"passthrough"}]}`)
	case strings.HasPrefix(joined, "relay events"):
		return emit(`{"total":1,"showing":1,"events":[{"id":"evt-1","platform":"gmail","eventType":"message.received","timestamp":"2026-03-29T10:00:00Z"}]}`)
	case strings.HasPrefix(joined, "relay event evt-1"):
		return emit(`{"id":"evt-1","platform":"gmail","eventType":"message.received","timestamp":"2026-03-29T10:00:00Z","payload":{"from":"a@example.com"}}`)
	}
	// flow create carries the definition inline ("--definition {...}"), so the
	// key is argv[2] and never the joined tail.
	switch {
	case len(args) >= 3 && args[0] == "flow" && args[1] == "create":
		key := args[2]
		return emit(`{"created":true,"key":"` + key + `","path":"/tmp/.one/flows/` + key + `/flow.json"}`)
	case strings.HasPrefix(joined, "flow execute "):
		return emit(`{"event":"step:start","stepId":"execute"}` + "\n" +
			`{"event":"workflow:result","runId":"run-1","logFile":"/tmp/run.log","status":"success","steps":{"execute":{"status":"success","response":{"ok":true,"posted":true,"channel":"#ops"}}}}`)
	case strings.HasPrefix(joined, "relay create"):
		return emit(`{"id":"relay-1","url":"https://relay.example","active":false,"description":"mail relay","eventFilters":["message.received"]}`)
	}
	fmt.Fprintf(os.Stderr, "unexpected args: %v\n", args)
	return 1
}
