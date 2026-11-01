//go:build !windows

package team

// finalPath is the identity on platforms where EvalSymlinks already resolved
// every link in the path.
func finalPath(abs string) string {
	return abs
}
