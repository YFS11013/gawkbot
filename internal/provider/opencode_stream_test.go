package provider

import (
	"strings"
	"testing"
)

func TestReadOpencodeJSONStreamCollectsTextChunks(t *testing.T) {
	stream := strings.Join([]string{
		`{"type":"step_start"}`,
		`{"type":"text","part":{"text":"Hello "}}`,
		`{"type":"text","part":{"text":"world."}}`,
		`{"type":"step_finish"}`,
	}, "\n")

	var events []OpencodeStreamEvent
	res, err := ReadOpencodeJSONStream(strings.NewReader(stream), func(ev OpencodeStreamEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("ReadOpencodeJSONStream: %v", err)
	}
	if got, want := res.FinalMessage, "Hello world."; got != want {
		t.Fatalf("FinalMessage = %q, want %q", got, want)
	}
	if res.LastError != "" {
		t.Fatalf("unexpected LastError: %q", res.LastError)
	}

	textCount := 0
	for _, ev := range events {
		if ev.Type == "text" {
			textCount++
		}
	}
	if textCount != 2 {
		t.Fatalf("expected 2 text events, got %d (%+v)", textCount, events)
	}
}

// TestReadOpencodeJSONStreamOversizedLine is the wedge regression. A
// single text-event line larger than the prior 4 MiB scanner buffer
// must drain cleanly, the oversized chunk must be delivered to onEvent,
// and a normal step_finish event after it must still parse. Under the
// prior bufio.Scanner this would have aborted at the oversized line.
func TestReadOpencodeJSONStreamOversizedLine(t *testing.T) {
	const huge = 5*1024*1024 + 11
	body := strings.Repeat("y", huge)

	stream := strings.Join([]string{
		`{"type":"text","part":{"text":"` + body + `"}}`,
		`{"type":"step_finish"}`,
	}, "\n")

	var sawHuge, sawFinish bool
	res, err := ReadOpencodeJSONStream(strings.NewReader(stream), func(ev OpencodeStreamEvent) {
		if ev.Type == "text" && len(ev.Text) == huge {
			sawHuge = true
		}
		if ev.Type == "step_finish" {
			sawFinish = true
		}
	})
	if err != nil {
		t.Fatalf("oversized-line ReadOpencodeJSONStream: %v", err)
	}
	if !sawHuge {
		t.Fatal("oversized text chunk was not delivered to onEvent")
	}
	if !sawFinish {
		t.Fatal("step_finish after oversized line was not delivered")
	}
	if len(res.FinalMessage) != huge {
		t.Fatalf("FinalMessage length: got %d, want %d", len(res.FinalMessage), huge)
	}
}

func TestReadOpencodeJSONStreamSurfacesToolUseAndResult(t *testing.T) {
	stream := strings.Join([]string{
		`{"type":"tool_use","part":{"toolName":"team_wiki_write","callID":"abc123"}}`,
		`{"type":"tool_result","part":{"callID":"abc123","output":"wrote 1 file"}}`,
	}, "\n")

	var toolUse, toolResult *OpencodeStreamEvent
	if _, err := ReadOpencodeJSONStream(strings.NewReader(stream), func(ev OpencodeStreamEvent) {
		captured := ev
		switch ev.Type {
		case "tool_use":
			toolUse = &captured
		case "tool_result":
			toolResult = &captured
		}
	}); err != nil {
		t.Fatalf("ReadOpencodeJSONStream: %v", err)
	}
	if toolUse == nil || toolUse.ToolName != "team_wiki_write" || toolUse.ToolID != "abc123" {
		t.Fatalf("missing/wrong tool_use event: %+v", toolUse)
	}
	if toolResult == nil || toolResult.ToolID != "abc123" {
		t.Fatalf("missing/wrong tool_result event: %+v", toolResult)
	}
	if !strings.Contains(toolResult.Detail, "wrote 1 file") {
		t.Fatalf("tool_result detail %q does not include output", toolResult.Detail)
	}
}

func TestReadOpencodeJSONStreamCapturesError(t *testing.T) {
	stream := `{"type":"error","message":"rate limited (429)"}`
	res, err := ReadOpencodeJSONStream(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatalf("ReadOpencodeJSONStream: %v", err)
	}
	if !strings.Contains(res.LastError, "rate limited") {
		t.Fatalf("LastError = %q, want substring 'rate limited'", res.LastError)
	}
}

func TestReadOpencodeJSONStreamFallsBackToPlainTextOnNonJSON(t *testing.T) {
	stream := strings.Join([]string{
		"opencode v1.14.25 — running",
		`{"type":"text","part":{"text":"ack"}}`,
	}, "\n")

	res, err := ReadOpencodeJSONStream(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatalf("ReadOpencodeJSONStream: %v", err)
	}
	if !strings.Contains(res.FinalMessage, "opencode v1.14.25") {
		t.Fatalf("plain-text banner missing from FinalMessage: %q", res.FinalMessage)
	}
	if !strings.Contains(res.FinalMessage, "ack") {
		t.Fatalf("text event missing from FinalMessage: %q", res.FinalMessage)
	}
}

func TestReadOpencodeJSONStreamForwardsUnknownEventTypes(t *testing.T) {
	stream := `{"type":"future-event-from-opencode-2","part":{"text":"ignored"}}`
	var typesSeen []string
	if _, err := ReadOpencodeJSONStream(strings.NewReader(stream), func(ev OpencodeStreamEvent) {
		typesSeen = append(typesSeen, ev.Type)
	}); err != nil {
		t.Fatalf("ReadOpencodeJSONStream: %v", err)
	}
	if len(typesSeen) != 1 || typesSeen[0] != "future-event-from-opencode-2" {
		t.Fatalf("unknown event not forwarded as-is, got types=%v", typesSeen)
	}
}

func TestReadOpencodeJSONStreamSkipsMalformedJSON(t *testing.T) {
	// First line is malformed JSON (missing closing brace) — should be
	// surfaced as plain text rather than aborting the stream.
	stream := strings.Join([]string{
		`{"type":"text","part":{"text":"first"`, // truncated
		`{"type":"text","part":{"text":"second"}}`,
	}, "\n")

	res, err := ReadOpencodeJSONStream(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatalf("ReadOpencodeJSONStream: %v", err)
	}
	if !strings.Contains(res.FinalMessage, "second") {
		t.Fatalf("second valid text event missing: %q", res.FinalMessage)
	}
}

func TestReadOpencodeJSONStreamStepFinishUsage(t *testing.T) {
	stream := strings.Join([]string{
		`{"type":"text","part":{"text":"OK"}}`,
		`{"type":"step_finish","part":{"reason":"stop","tokens":{"total":14886,"input":14862,"output":3,"reasoning":21,"cache":{"write":0,"read":5}},"cost":0.25}}`,
	}, "\n")
	var finishes []OpencodeStreamEvent
	res, err := ReadOpencodeJSONStream(strings.NewReader(stream), func(ev OpencodeStreamEvent) {
		if ev.Type == "step_finish" {
			finishes = append(finishes, ev)
		}
	})
	if err != nil {
		t.Fatalf("ReadOpencodeJSONStream: %v", err)
	}
	if len(finishes) != 1 || finishes[0].Tokens == nil {
		t.Fatalf("step_finish token payload missing: %+v", finishes)
	}
	if got := finishes[0].Tokens.InputTokens; got != 14862 {
		t.Fatalf("event input tokens = %d, want 14862", got)
	}
	if got := res.Usage.InputTokens; got != 14862 {
		t.Fatalf("result input tokens = %d, want 14862", got)
	}
	if got := res.Usage.OutputTokens; got != 3 {
		t.Fatalf("result output tokens = %d, want 3", got)
	}
	if got := res.Usage.CacheReadTokens; got != 5 {
		t.Fatalf("result cache read tokens = %d, want 5", got)
	}
	if got := res.Usage.CostUSD; got != 0.25 {
		t.Fatalf("result cost = %v, want 0.25", got)
	}
}
