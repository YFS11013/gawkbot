package team

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeCLIEnv selects what the test binary does when it is executed as a fake
// external command.
//
// These fakes used to be /bin/sh scripts written into a temp dir. Windows
// cannot start a file with a shebang and no extension at all — exec reports
// "executable file not found" — so the CLI-parsing tests passed only on Unix
// while the parsers themselves are platform-independent. The test binary stands
// in for the script instead: TestMain exits before running any test when this
// variable is set, so the same assertions run everywhere.
const fakeCLIEnv = "WUPHF_FAKE_CLI"

// fakeCLIPath makes the current test binary behave like the named fake and
// returns the path a caller should exec.
func fakeCLIPath(t *testing.T, mode string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test binary for fake CLI: %v", err)
	}
	t.Setenv(fakeCLIEnv, mode)
	return exe
}

// runFakeCLIHelper runs the fake selected by fakeCLIEnv. handled is false for
// an ordinary test run, where the caller continues into m.Run.
func runFakeCLIHelper() (handled bool, code int) {
	mode := strings.TrimSpace(os.Getenv(fakeCLIEnv))
	if mode == "" {
		return false, 0
	}
	switch mode {
	case "one-dry-run":
		return true, fakeOneDryRun(os.Args[1:])
	case "one-rate-limited":
		return true, fakeOneRateLimited(os.Args[1:])
	case "box":
		return true, fakeBox(os.Args[1:])
	}
	fmt.Fprintf(os.Stderr, "fake cli: unknown %s=%q\n", fakeCLIEnv, mode)
	return true, 2
}

// fakeOneDryRun mirrors the dry-run workflow fake: create succeeds, execute
// streams one step and then a successful result.
func fakeOneDryRun(args []string) int {
	args = trimAgentFlag(args)
	switch strings.Join(args[:min(3, len(args))], " ") {
	case "flow create sponsor-outreach-dry-run":
		fmt.Println(`{"created":true,"key":"sponsor-outreach-dry-run","path":"/tmp/.one/flows/sponsor-outreach-dry-run.flow.json"}`)
	case "flow execute sponsor-outreach-dry-run":
		fmt.Println(`{"event":"step:start","stepId":"compose-email"}`)
		fmt.Println(`{"event":"workflow:result","runId":"run-123","logFile":"/tmp/run.log","status":"success","steps":{"gmail-preview":{"status":"success"}}}`)
	default:
		fmt.Fprintf(os.Stderr, "unexpected args: %s\n", strings.Join(args, " "))
		return 1
	}
	return 0
}

// fakeOneRateLimited mirrors the live-workflow fake: the run starts and then
// the provider answers 429 with a retry timestamp.
func fakeOneRateLimited(args []string) int {
	args = trimAgentFlag(args)
	switch strings.Join(args[:min(3, len(args))], " ") {
	case "flow create consulting-live-gmail-read":
		fmt.Println(`{"created":true,"key":"consulting-live-gmail-read","path":"/tmp/.one/flows/consulting-live-gmail-read.flow.json"}`)
	case "flow execute consulting-live-gmail-read":
		fmt.Println(`{"event":"flow:start","runId":"run-live-1"}`)
		fmt.Fprintln(os.Stderr, `{"event":"step:error","stepId":"fetchEmails","error":"429 User-rate limit exceeded. Retry after 2026-04-14T22:55:02.178Z"}`)
		return 1
	default:
		fmt.Fprintf(os.Stderr, "unexpected args: %s\n", strings.Join(args, " "))
		return 1
	}
	return 0
}

func trimAgentFlag(args []string) []string {
	if len(args) > 0 && args[0] == "--agent" {
		return args[1:]
	}
	return args
}

// fakeBox mirrors the box CLI surface the sign-in flow drives. The marker file
// is how the real CLI records "signed in", so the fake keeps that on disk too:
// the state machine is what the test is about, not the CLI.
func fakeBox(args []string) int {
	marker := os.Getenv("WUPHF_FAKE_BOX_MARKER")
	revoked := os.Getenv("WUPHF_FAKE_BOX_REVOKED")
	if len(args) == 0 {
		return 2
	}
	switch args[0] {
	case "login":
		fmt.Println(`{"event":"login_url","url":"https://ascii.dev/api/box/auth/github?state=abc"}`)
	case "onboard":
		time.Sleep(200 * time.Millisecond)
		fmt.Println(`{"event":"login_complete"}`)
		if marker != "" {
			_ = os.WriteFile(marker, nil, 0o600)
		}
	case "status":
		if marker != "" {
			if _, err := os.Stat(marker); err == nil {
				fmt.Println(`{"account":{"loginState":"signed in"}}`)
				return 0
			}
		}
		fmt.Println(`{"account":{"loginState":"signed out"}}`)
	case "api-key":
		if len(args) < 2 {
			return 2
		}
		switch args[1] {
		case "list":
			fmt.Println(`{"apiKeys":[{"id":"k1","name":"gawkbot"},{"id":"k2","name":"other"}]}`)
		case "revoke":
			if revoked != "" && len(args) > 2 {
				f, err := os.OpenFile(revoked, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
				if err == nil {
					_, _ = fmt.Fprintln(f, args[2])
					_ = f.Close()
				}
			}
		default:
			fmt.Println(`{"id":"k1","secret":"box_minted"}`)
		}
	case "logout":
		if marker != "" {
			_ = os.Remove(marker)
		}
		fmt.Println(`{"event":"logged_out"}`)
	case "limits":
		fmt.Println(`{"event":"limits","limits":{"canStart":false,"blockedReason":"subscription_required","planName":null,"accessTier":"trial","trialLine":"2 boxes, 25h compute"}}`)
	default:
		return 2
	}
	return 0
}
