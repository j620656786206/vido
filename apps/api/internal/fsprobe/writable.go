// Package fsprobe answers filesystem questions the pipeline must settle BEFORE
// it spends money (story sub-6-1). It is a Rule 19 leaf package: zero
// internal/ imports, so both `subtitle` (the item flow) and `services` (the
// consent-list analysis) can share one probe without a services→subtitle edge.
package fsprobe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotWritable is the sentinel every ProbeWritable failure wraps. Callers
// classify with errors.Is and re-wrap under their own wire code (the subtitle
// package's SUBTITLE_TARGET_NOT_WRITABLE).
var ErrNotWritable = errors.New("target directory is not writable")

// probePattern is the temp-file name pattern; the leading dot keeps a probe
// that somehow survives (a crash between create and remove) out of media
// scanners, and the suffix names the owner for anyone who finds one.
const probePattern = ".vido-write-probe-*"

// ProbeWritable reports whether the process can create a file in dir, by
// actually creating (and immediately removing) one.
//
// It deliberately does NOT consult mode bits: on NFS/SMB/Unraid shares the
// bits routinely say "group writable" while the container's gid is not in
// that group (eval-1 finding 2 — a full 900-cue translation was paid for and
// then failed at placement with permission denied). A read-only mount
// (finding 1) looks writable to stat too. Creating a file is the only honest
// test, and it costs a few microseconds against the minutes of LLM time it
// guards.
func ProbeWritable(dir string) error {
	if dir == "" {
		return fmt.Errorf("%w: empty directory", ErrNotWritable)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrNotWritable, dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory", ErrNotWritable, dir)
	}
	f, err := os.CreateTemp(dir, probePattern)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrNotWritable, dir, err)
	}
	name := f.Name()
	closeErr := f.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return fmt.Errorf("%w: %s: close probe: %v", ErrNotWritable, dir, closeErr)
	}
	if removeErr != nil {
		// Created but could not delete: the directory IS writable for our
		// purpose (placer writes and renames), but leaving debris is worth a
		// loud error so an operator notices a half-broken mount.
		return fmt.Errorf("%w: %s: could not remove probe %s: %v", ErrNotWritable, dir, filepath.Base(name), removeErr)
	}
	return nil
}
