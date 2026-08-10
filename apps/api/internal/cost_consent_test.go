// Package internal hosts repo-wide invariants enforced as Go tests.
//
// This file guards the COST-CONSENT invariant introduced by story sub-4-1.
package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sweepCall is the library-wide subtitle-generation sweep. Its DEFINITION
// (`func (p *WorkerPool) EnqueueMissing(`) does not contain the leading dot,
// so scanning for the call form finds callers only.
const sweepCall = ".EnqueueMissing("

// TestScanMustNotAutoEnqueueSubtitleGeneration enforces story sub-4-1 AC #1.
//
// WHY THIS TEST EXISTS — this is not hypothetical. On 2026-08-07 the first
// production run of `VIDO_SUBTITLE_PIPELINE_MODE=pipeline` wired this sweep
// into the scan-complete callback. Pressing "掃描媒體庫" enqueued 1026 items
// in one shot; roughly two thirds of them route to PAID speech recognition
// (whisper-1 at $0.006/min — a whole-library run was estimated around US$200)
// and the user was never shown a number, let alone asked. The NAS had to be
// reverted to `legacy` mode to stop the spend.
//
// The ruling (Alexyu, 2026-08-07) is that scanning updates metadata and
// nothing else; paid generation is chosen explicitly on a screen that shows
// the estimate first. This test makes that ruling structural: a library-wide
// sweep must have NO production caller, so re-wiring one is a red test rather
// than a surprise invoice.
//
// Re-enabling it deliberately is still possible — but it must come with a
// consent surface, and whoever does it has to delete this test and explain
// why in the PR.
func TestScanMustNotAutoEnqueueSubtitleGeneration(t *testing.T) {
	callers, scanned, err := findSweepCallers("..")
	if err != nil {
		t.Fatalf("scanning apps/api for %q failed: %v", sweepCall, err)
	}
	if scanned == 0 {
		t.Fatal("no production .go files scanned — wrong working directory?")
	}

	for _, c := range callers {
		t.Errorf("%s calls %s — a library-wide subtitle sweep must not run without cost consent (story sub-4-1 AC #1). "+
			"Paid speech recognition is on the other end of this call.", c, sweepCall)
	}
}

// findSweepCallers walks production Go files under root and returns every file
// containing a call to the sweep. Test files are excluded: they exercise the
// method directly, which is exactly how it stays covered while unused in
// production.
func findSweepCallers(root string) (callers []string, scanned int, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		src, rerr := os.ReadFile(path) //nolint:gosec // walking our own source tree
		if rerr != nil {
			return rerr
		}
		scanned++
		if strings.Contains(string(src), sweepCall) {
			callers = append(callers, filepath.ToSlash(path))
		}
		return nil
	})
	return callers, scanned, err
}
