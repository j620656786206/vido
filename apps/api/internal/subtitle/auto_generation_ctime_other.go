//go:build !linux && !darwin

package subtitle

import (
	"os"
	"time"
)

// inodeChangeTime: no portable ctime here; fileChangedAt falls back to mtime.
func inodeChangeTime(os.FileInfo) (time.Time, bool) { return time.Time{}, false }
