package workspaces

// appendOrReplace sets key=val in a process environment slice, replacing every
// existing entry for that key and appending when there is none.
//
// It lives outside spawn.go because that file is Unix-only while the behaviour
// is not: the orchestrator tests assert it on every platform.
func appendOrReplace(env []string, key, val string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	found := false
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			out = append(out, prefix+val)
			found = true
		} else {
			out = append(out, e)
		}
	}
	if !found {
		out = append(out, prefix+val)
	}
	return out
}
