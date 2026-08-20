package main

import (
	"context"

	"github.com/vido/api/internal/repository"
)

// autoSubtitlePolicyAdapter answers "which libraries opted in to free subtitle
// auto-generation" from the media_libraries table (Story 9R-10b AC #2).
//
// It lives in cmd/api for the same reason pipelineASRAdapter and
// subtitlePlacerAdapter do: `internal/subtitle` may depend on
// `internal/services`, but nothing below may depend BACK on subtitle (Rule 19),
// so the bridge between a repository and a subtitle-side port is assembled at
// the composition root.
type autoSubtitlePolicyAdapter struct {
	libraries repository.MediaLibraryRepositoryInterface
}

// AutoSubtitleLibraryIDs returns the opted-in library ids as a set.
//
// An error is returned rather than swallowed: on the NAS the realistic failure
// is a locked SQLite file, and reporting that as "nobody opted in" would make
// the feature stop working with nothing in the log to explain why (9R-10a CR M1).
// The caller aborts the round on error.
func (a autoSubtitlePolicyAdapter) AutoSubtitleLibraryIDs(ctx context.Context) (map[string]struct{}, error) {
	libs, err := a.libraries.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	enabled := make(map[string]struct{}, len(libs))
	for _, lib := range libs {
		if lib.AutoSubtitle {
			enabled[lib.ID] = struct{}{}
		}
	}
	return enabled, nil
}
