package team

import (
	"encoding/json"
	"testing"
)

// Real shape of `opencode run --format json` on stdout: one JSON event per
// line. parseSkillJSONFull used to json.Unmarshal the whole blob and fail with
// "invalid character '{' after top-level value", which AskIsSkill then reported
// as "not a skill" — 50 calls per pass, zero verdicts.
const opencodeEventStream = `{"type":"step_start","timestamp":1793979107431,"sessionID":"ses_x","part":{"id":"prt_1","type":"step-start"}}
{"type":"tool_use","timestamp":1793979107500,"sessionID":"ses_x","part":{"type":"tool","tool":"glob","callID":"call-1","state":{"status":"completed","input":{"path":"C:\\work"},"output":"a.md"}}}
{"type":"step_finish","timestamp":1793979107600,"sessionID":"ses_x","part":{"id":"prt_2","type":"step-finish","reason":"tool-calls"}}
{"type":"text","timestamp":1793979107659,"sessionID":"ses_x","part":{"id":"prt_3","type":"text","text":"{\"is_skill\": false}"}}
{"type":"step_finish","timestamp":1793979108090,"sessionID":"ses_x","part":{"id":"prt_4","type":"step-finish","reason":"stop"}}`

func TestParseSkillJSONFull_EventStream(t *testing.T) {
	res := parseSkillJSONFull(opencodeEventStream)
	if res.Err != nil {
		t.Fatalf("event stream must decode, got %v", res.Err)
	}
	if res.IsSkill {
		t.Fatalf("want is_skill=false, got true")
	}
}

func TestParseSkillJSONFull_EventStreamSkillTrue(t *testing.T) {
	verdict := map[string]any{
		"is_skill":    true,
		"name":        "Weekly Recap",
		"description": "Post the weekly recap.",
		"body":        "## Steps\n1. gather",
	}
	answer, err := json.Marshal(verdict)
	if err != nil {
		t.Fatalf("marshal verdict: %v", err)
	}
	// The CLI embeds the answer as a JSON string inside an event, so the
	// answer's own escapes get escaped again. Build the event the same way.
	event, err := json.Marshal(map[string]any{
		"type": "text",
		"part": map[string]any{"type": "text", "text": string(answer)},
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	res := parseSkillJSONFull(string(event))
	if res.Err != nil {
		t.Fatalf("event stream with a skill verdict must decode, got %v", res.Err)
	}
	if !res.IsSkill || res.FM.Name != "weekly-recap" {
		t.Fatalf("got is_skill=%v name=%q", res.IsSkill, res.FM.Name)
	}
}

// An event object whose text field quotes a verdict must not be mistaken for
// the verdict itself when the stream line fails to parse: the wrapper has no
// top-level is_skill key.
func TestLastSkillVerdictObject_SkipsWrapper(t *testing.T) {
	wrapper := `{"type":"text","part":{"type":"text","text":"{\"is_skill\": false}"}}`
	if got := lastSkillVerdictObject(wrapper); got != "" {
		t.Fatalf("wrapper must not be treated as a verdict, got %q", got)
	}
	if got := lastSkillVerdictObject(`noise {"a":1} more {"is_skill": false} tail`); got != `{"is_skill": false}` {
		t.Fatalf("got %q", got)
	}
}

func TestParseSkillJSONFull_TrailingJunk(t *testing.T) {
	raw := "Here is my answer:\n{\"is_skill\": false}\nHope this helps."
	res := parseSkillJSONFull(raw)
	if res.Err != nil || res.IsSkill {
		t.Fatalf("prose around a verdict must decode: err=%v is_skill=%v", res.Err, res.IsSkill)
	}
}

func TestParseSkillJSONFull_LastVerdictWins(t *testing.T) {
	raw := `{"is_skill": false}` + "\n" + `{"is_skill": false, "note": "second"}`
	res := parseSkillJSONFull(raw)
	if res.Err != nil || res.IsSkill {
		t.Fatalf("two concatenated verdicts must decode: err=%v is_skill=%v", res.Err, res.IsSkill)
	}
}

func TestParseSkillJSONFull_StillRejectsGarbage(t *testing.T) {
	if res := parseSkillJSONFull("not json at all"); res.Err == nil {
		t.Fatalf("garbage must stay an error, got %+v", res)
	}
}
