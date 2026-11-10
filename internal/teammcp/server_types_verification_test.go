package teammcp

import (
	"reflect"
	"strings"
	"testing"
)

// Journey 14 (core-loop-runs/2026-11-10-journey14): both lanes declared
// verification_kind=artifact with a wiki-relative spec, because the tool schema
// never advertised wiki_artifact. The artifact check globs the task worktree /
// bot scratch dir, so a file published to the team wiki can never satisfy it —
// the human's approve on the lane's own decision card 409'd with
// "no non-empty artifact matches \"team/accounts/acme-renewal-quote.md\""
// while the file was committed to the wiki. The schema must keep naming the
// wiki-aware kind, or the model picks the worktree kind again.
func TestTeamTaskSchemaAdvertisesWikiArtifactVerification(t *testing.T) {
	typ := reflect.TypeOf(TeamTaskArgs{})
	kind, ok := typ.FieldByName("VerificationKind")
	if !ok {
		t.Fatal("TeamTaskArgs has no VerificationKind field")
	}
	tag := kind.Tag.Get("jsonschema")
	if !strings.Contains(tag, "wiki_artifact") {
		t.Fatalf("the model-facing schema must advertise wiki_artifact, got %q", tag)
	}
	if !strings.Contains(tag, "team wiki") {
		t.Fatalf("wiki_artifact must say where the file is checked, got %q", tag)
	}
	spec, ok := typ.FieldByName("VerificationSpec")
	if !ok {
		t.Fatal("TeamTaskArgs has no VerificationSpec field")
	}
	if !strings.Contains(spec.Tag.Get("jsonschema"), "wiki-relative") {
		t.Fatalf("VerificationSpec must mention the wiki-relative path, got %q", spec.Tag.Get("jsonschema"))
	}
}
