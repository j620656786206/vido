// Package dbtime is the one place that decides how the app stores a
// timestamp in SQLite. It is a leaf so both the driver wrapper (package
// database) and the migrations that repair old rows can share it without an
// import cycle.
package dbtime

import "time"

// layout is fixed width (always nine fractional digits) so that text order
// equals time order; Format appends a literal "Z" (UTC) — a shape SQLite's
// date()/datetime()/strftime() accept and the modernc driver parses back into
// a time.Time.
const layout = "2006-01-02 15:04:05.000000000"

// Format renders t the way the app stores timestamps: UTC, nanosecond, fixed
// width, "Z"-suffixed, no zone name, no monotonic-clock reading.
//
//	2026-08-24 02:56:26.303854294Z
func Format(t time.Time) string {
	return t.UTC().Format(layout) + "Z"
}
