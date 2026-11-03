//go:build windows

package team

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// configureHeadlessProcess is empty on Windows on purpose: there is no
// process-group flag to set at spawn time. The equivalent boundary — a kernel
// job object that owns the tool and everything the tool starts — can only be
// attached once a pid exists, so startHeadlessProcess does it.
func configureHeadlessProcess(cmd *exec.Cmd) {}

// startHeadlessProcess starts cmd and puts it in a job object.
//
// Callers that used cmd.Start directly left the tool outside every ownership
// boundary the office has: taskkill /T (terminateHeadlessProcessPID) walks
// parent→child links as they are at kill time, so a descendant whose
// intermediate parent already exited — the normal shape of a .cmd shim that
// hands off to node/bun — is in nobody's tree. The job records membership in
// the kernel instead, so the leftovers are killable no matter who their parent
// was.
func startHeadlessProcess(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		attachHeadlessJob(cmd.Process.Pid)
	}
	return nil
}

// headlessJob is the job object that owns one turn's process tree.
type headlessJob struct {
	mu     sync.Mutex
	handle windows.Handle
	closed bool
}

// terminate kills every process currently in the job.
func (j *headlessJob) terminate() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed || j.handle == 0 {
		return
	}
	_ = windows.TerminateJobObject(j.handle, 1)
}

// close drops the last handle. KILL_ON_JOB_CLOSE is set, so this kills whatever
// is still in the job — which is the point: when the tool a turn started exits,
// the LSP/server/build processes it spawned must not keep running into the next
// turn holding its stdout pipe or its database open.
func (j *headlessJob) close() {
	j.mu.Lock()
	if j.closed || j.handle == 0 {
		j.mu.Unlock()
		return
	}
	j.closed = true
	h := j.handle
	j.handle = 0
	j.mu.Unlock()
	_ = windows.CloseHandle(h)
}

var (
	headlessJobsMu sync.Mutex
	headlessJobs   = map[uint32]*headlessJob{}
)

// attachHeadlessJob creates the job, puts pid in it, and releases both when the
// process exits.
//
// Every failure path is silent on purpose. A machine where job objects are
// denied (a restrictive parent job, an old Windows) loses only the
// escape-proofing: taskkill /T still kills a live tree, so the turn deadline
// keeps working.
func attachHeadlessJob(pid int) {
	if pid <= 0 {
		return
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		_ = windows.CloseHandle(proc)
		return
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		_ = windows.CloseHandle(proc)
		return
	}
	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		_ = windows.CloseHandle(job)
		_ = windows.CloseHandle(proc)
		return
	}

	entry := &headlessJob{handle: job}
	key := uint32(pid)
	headlessJobsMu.Lock()
	// A recycled pid must not keep the previous turn's job reachable.
	if old, ok := headlessJobs[key]; ok {
		old.close()
	}
	headlessJobs[key] = entry
	headlessJobsMu.Unlock()

	go func() {
		_, _ = windows.WaitForSingleObject(proc, windows.INFINITE)
		headlessJobsMu.Lock()
		if cur, ok := headlessJobs[key]; ok && cur == entry {
			delete(headlessJobs, key)
		}
		headlessJobsMu.Unlock()
		entry.close()
		_ = windows.CloseHandle(proc)
	}()
}

func headlessJobFor(pid int) *headlessJob {
	headlessJobsMu.Lock()
	defer headlessJobsMu.Unlock()
	return headlessJobs[uint32(pid)]
}

func terminateHeadlessProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	terminateHeadlessProcessPID(cmd.Process.Pid)
}

// terminateHeadlessProcessPID kills the process AND its tree.
//
// The bots run `opencode` through the npm shim (opencode.cmd -> node/bun), and
// every other headless tool has a similar shim. TerminateProcess on the pid Go
// spawned only kills the shim: the real CLI keeps running, keeps the stdout
// pipe open, and the worker's cmd.Wait blocks until that survivor happens to
// exit. That is what turned a 25-minute turn timeout into an 8.5-hour stall on
// 2026-10-30 (three cos turns: started 23:31, logged "timed out after 25m0s" at
// 08:00 the next morning when the orphaned Bun process finally died of EPIPE).
//
// Two kills, in this order, because they cover different shapes:
//
//  1. taskkill /T walks the live parent→child tree. It is the only thing that
//     reaches a process created in the microsecond window between Start and the
//     job assignment.
//  2. Terminating the job reaches everything the tree walk cannot: descendants
//     whose intermediate parent is already gone, and processes whose pid the
//     walk could no longer resolve. The job handle is found by pid, never by
//     re-walking the tree, so a reused pid cannot point it at another process.
func terminateHeadlessProcessPID(pid int) {
	if pid <= 0 {
		return
	}
	job := headlessJobFor(pid)
	killed := false
	if err := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run(); err == nil {
		killed = true
	}
	if job != nil {
		job.terminate()
		killed = true
	}
	if killed {
		return
	}
	// taskkill unavailable (locked-down PATH) and nothing to terminate by
	// handle: at least end the process Go owns.
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
}
