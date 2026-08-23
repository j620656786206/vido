//go:build linux

package subtitle

import (
	"os"
	"syscall"
	"time"
)

// inodeChangeTime returns the inode change time (ctime) when the platform
// exposes it. See fileChangedAt for why mtime alone is not enough.
func inodeChangeTime(info os.FileInfo) (time.Time, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(st.Ctim.Sec, st.Ctim.Nsec), true
}
