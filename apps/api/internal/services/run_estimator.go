package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// RunEstimator answers "what would the 管理字幕 dialog quote for this item right
// now" for the run ledger (disc-2026-11-spend-history-a-ledger). Both legs that
// open a subtitle_runs row consume it: the subtitle pipeline (through its own
// identically-shaped port — subtitle → services is the legal Rule 19
// direction) and TranscriptionService's own ledger (solo / legacy batch).
type RunEstimator interface {
	EstimateRunUSD(ctx context.Context, mediaType, mediaID string) (float64, error)
}

// runEstimateMovies / runEstimateEpisodes are the row reads the estimate needs
// — the same rows the estimate handler resolves before calling Estimate.
type runEstimateMovies interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
}

type runEstimateEpisodes interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
}

// runEstimateQuoter is the one method of TranscriptionEstimateService this
// adapter calls — narrow so a test can pin the target it is handed.
type runEstimateQuoter interface {
	Estimate(ctx context.Context, target TranscriptionEstimateTarget) TranscriptionEstimate
}

// TranscriptionRunEstimator recomputes the dialog's quote at run start with the
// SAME TranscriptionEstimateService.Estimate the dialog calls, building the
// target exactly as transcription_handler.go's EstimateMovie / EstimateEpisode
// do, so the recorded figure is the figure the user would have seen.
type TranscriptionRunEstimator struct {
	quoter   runEstimateQuoter
	movies   runEstimateMovies
	episodes runEstimateEpisodes
}

// NewTranscriptionRunEstimator wires the adapter. Production passes the
// TranscriptionEstimateService the estimate handler uses and repos.Movies /
// repos.Episodes.
func NewTranscriptionRunEstimator(quoter runEstimateQuoter, movies runEstimateMovies, episodes runEstimateEpisodes) *TranscriptionRunEstimator {
	return &TranscriptionRunEstimator{quoter: quoter, movies: movies, episodes: episodes}
}

// runEstimateTimeout bounds the quote so a cold probe (ffprobe on a large
// file the dialog never priced) cannot hold up the start of the run it only
// annotates. On a warm path — the dialog or the batch sweep priced the item a
// moment ago — the probes are cached and this is never reached.
const runEstimateTimeout = 20 * time.Second

// EstimateRunUSD returns the dialog's EstimatedUSD for one movie or episode.
// An error means "no figure" — the caller records NULL; it never fails a run.
func (e *TranscriptionRunEstimator) EstimateRunUSD(ctx context.Context, mediaType, mediaID string) (float64, error) {
	if e == nil || e.quoter == nil {
		return 0, fmt.Errorf("run estimator not wired")
	}
	ctx, cancel := context.WithTimeout(ctx, runEstimateTimeout)
	defer cancel()
	target := TranscriptionEstimateTarget{MediaID: mediaID, MediaType: mediaType}
	switch mediaType {
	case models.SubtitleRunMediaMovie:
		if e.movies == nil {
			return 0, fmt.Errorf("run estimator: no movie reader")
		}
		movie, err := e.movies.FindByID(ctx, mediaID)
		if err != nil {
			return 0, fmt.Errorf("run estimator: load movie %s: %w", mediaID, err)
		}
		if movie == nil {
			return 0, fmt.Errorf("run estimator: movie %s not found", mediaID)
		}
		target.FilePath = movie.FilePath.String
		target.DurationSeconds = movie.DurationSeconds
		target.Runtime = movie.Runtime
		target.SubtitleTracksJSON = movie.SubtitleTracks.String
	case models.SubtitleRunMediaEpisode:
		if e.episodes == nil {
			return 0, fmt.Errorf("run estimator: no episode reader")
		}
		episode, err := e.episodes.FindByID(ctx, mediaID)
		if err != nil {
			return 0, fmt.Errorf("run estimator: load episode %s: %w", mediaID, err)
		}
		if episode == nil {
			return 0, fmt.Errorf("run estimator: episode %s not found", mediaID)
		}
		target.FilePath = episode.FilePath.String
		target.DurationSeconds = episode.DurationSeconds
		target.Runtime = episode.Runtime
	default:
		// A series row is a container, not a media file — the dialog has no
		// quote for it either.
		return 0, fmt.Errorf("run estimator: media type %q has no estimate", mediaType)
	}
	if target.FilePath == "" {
		return 0, fmt.Errorf("run estimator: %s %s has no file path", mediaType, mediaID)
	}
	return e.quoter.Estimate(ctx, target).EstimatedUSD, nil
}

// estimateRunUSD is the nil-safe, never-failing read both ledger writers use:
// a nil estimator or an error yields nil (NULL in the row) plus one Warn.
func estimateRunUSD(ctx context.Context, est RunEstimator, mediaType, mediaID string, warn func(msg string, args ...any)) *float64 {
	if est == nil {
		warn("subtitle run ledger: no run estimator wired — estimate_usd stays NULL",
			"media_id", mediaID, "media_type", mediaType)
		return nil
	}
	usd, err := est.EstimateRunUSD(ctx, mediaType, mediaID)
	if err != nil {
		warn("subtitle run ledger: estimate failed — estimate_usd stays NULL",
			"media_id", mediaID, "media_type", mediaType, "error", err)
		return nil
	}
	return &usd
}
