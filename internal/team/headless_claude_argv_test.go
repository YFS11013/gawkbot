package team

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunHeadlessClaudeTurn_PromptViaFile pins the Windows fix (upstream
// issue #1240): the office system prompt must travel via
// --append-system-prompt-file, never as an inlined --append-system-prompt
// argument — Windows CreateProcess caps the command line at 32,767 chars
// and the prompt alone exceeds it, so every turn died at process start
// with "The filename or extension is too long".
func TestRunHeadlessClaudeTurn_PromptViaFile(t *testing.T) {
	tmpDir := t.TempDir()

	origCommandContext := headlessClaudeCommandContext
	origLookPath := headlessClaudeLookPath
	defer func() {
		headlessClaudeCommandContext = origCommandContext
		headlessClaudeLookPath = origLookPath
	}()

	var capturedArgs []string
	headlessClaudeLookPath = func(file string) (string, error) { return "/bin/true", nil }
	headlessClaudeCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = append(capturedArgs, args...)
		return exec.CommandContext(ctx, "/bin/true")
	}

	b := newBrokerWithTeamRoom(filepath.Join(tmpDir, "broker-state.json"))
	l := minimalLauncher(false)
	l.broker = b
	l.cwd = tmpDir

	mcpPath := filepath.Join(tmpDir, "mcp.json")
	_ = os.WriteFile(mcpPath, []byte(`{"mcpServers":{}}`), 0o600)
	l.mcpConfig = mcpPath

	// Expected to fail at JSON parsing (/bin/true emits nothing); only argv matters.
	_ = l.runHeadlessClaudeTurn(t.Context(), "eng", "do the thing")

	if len(capturedArgs) == 0 {
		t.Fatal("no args captured; headlessClaudeCommandContext hook was not called")
	}
	for _, arg := range capturedArgs {
		if arg == "--append-system-prompt" {
			t.Fatalf("must not inline --append-system-prompt (Windows argv overflow #1240), got args: %v", capturedArgs)
		}
	}
	fileIdx := -1
	for i, arg := range capturedArgs {
		if arg == "--append-system-prompt-file" {
			fileIdx = i
		}
	}
	if fileIdx == -1 || fileIdx+1 >= len(capturedArgs) {
		t.Fatalf("expected --append-system-prompt-file <path> in args, got: %v", capturedArgs)
	}
	promptPath := capturedArgs[fileIdx+1]
	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("prompt file unreadable %q: %v", promptPath, err)
	}
	if len(content) == 0 {
		t.Fatalf("prompt file %q is empty", promptPath)
	}

	// Total argv must stay far under the 32,767-char Windows limit even with
	// a giant prompt: only the file path travels on the command line.
	total := 0
	for _, arg := range capturedArgs {
		total += len(arg) + 1
	}
	if total > 8191 {
		t.Fatalf("argv too long (%d chars); prompt may be leaking onto the command line", total)
	}
	if strings.Contains(strings.Join(capturedArgs, "\x00"), string(content)) && len(content) > 1024 {
		t.Fatalf("prompt file content appears inlined in argv")
	}
}
