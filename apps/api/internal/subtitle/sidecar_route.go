package subtitle

import "context"

// SidecarFinder finds an English subtitle file beside a video that matches the
// video's timing (disc-2026-10-generate-keeps-user-en-srt-b2).
// sidecarcheck.Finder implements it; it lives there because that package
// needs this one (mine → subtitle), not the other way round.
type SidecarFinder interface {
	// FindAligned checks the English sidecars in priority order and returns
	// the first whose timing matches. ok is false when there is none, none
	// matches, or the check could not run — the caller then routes as if
	// there were no sidecar.
	FindAligned(ctx context.Context, mediaPath string) (path string, ok bool)
}

// SetSidecarFinder lets the router translate a matching English sidecar when
// the file has no usable embedded track. ⚖️ 2026-10-19 Alexyu overturned the
// M1 "sidecars are the search engine's domain" scope line (sub-1-4 AC #8):
// a subtitle the user placed is not a search result, and translating it is
// cheaper than speech recognition and keeps its timing. nil = embedded only.
func (r *Router) SetSidecarFinder(f SidecarFinder) {
	r.sidecars = f
}

type noSidecarKey struct{}

// WithoutSidecars marks a routing run that must not check sidecars: the
// FreeOnly auto-generation lane parks a translate verdict and a no-text-source
// verdict alike, so a sidecar check (seconds of ffmpeg per film, during a
// scan) could not change what it does.
func WithoutSidecars(ctx context.Context) context.Context {
	return context.WithValue(ctx, noSidecarKey{}, true)
}

func sidecarsDisabled(ctx context.Context) bool {
	off, _ := ctx.Value(noSidecarKey{}).(bool)
	return off
}
