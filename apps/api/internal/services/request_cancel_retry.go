// Package services — request cancel / retry (Story 13-7a).
//
// The 想要清單 draws 取消 only on pending rows and 重試 only on failed rows
// (RequestRow-v2 `LkjRd`); this file is the backend half of exactly that
// matrix. Endpoint shapes carry [@contract-v1] (AC #1/#2) — consumer 13-7b.
//
// Contract acks: 13-1a request resource unchanged; 13-3a status derivation
// and request_progress untouched; 13-4a DVRPlugin untouched (queue removal is
// the optional plugins.QueueRemover capability); 13-4b [@contract-v2]
// (selection-aware AddSeries) — a retried terminal row re-runs FulfilRequest,
// which already carries the stored selection, so v2 needs nothing here.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// ErrRequestNotCancellable — the row exists but is no longer pending (*arr
// already has it), so cancel would need an *arr un-add the design does not
// offer (disc-2026-07-cancel-active-requests).
var ErrRequestNotCancellable = errors.New("request is not cancellable")

// ErrRequestNotRetryable — the row exists but has not failed.
var ErrRequestNotRetryable = errors.New("request is not retryable")

// ErrRetryCleanupFailed — a queue-failed retry could not throw away the
// broken download in *arr. The row stays failed: resetting it anyway would
// only have the next poller tick flip it straight back to failed, with no
// hint why (13-7a CR).
var ErrRetryCleanupFailed = errors.New("could not discard the failed download")

// RequestQueueCleaner is the one plugin operation a retry needs (Rule 11):
// discard the downloads that made one *arr movie/series' request fail.
// DVRQueueCleaner is the production adapter.
type RequestQueueCleaner interface {
	DiscardFailedDownloads(ctx context.Context, plugin string, externalID int64) (int, error)
}

// DVRQueueCleaner adapts the plugin manager + qBittorrent to
// RequestQueueCleaner. "Failed" is decided by deriveQueueState — the same
// rule the status poller used to mark the row failed — never by *arr's
// "warning", which also covers torrents that are merely stalled.
type DVRQueueCleaner struct {
	queues   requestQueueSource
	torrents torrentSource
}

// NewDVRQueueCleaner builds the adapter (the poller's two sources).
func NewDVRQueueCleaner(queues requestQueueSource, torrents torrentSource) *DVRQueueCleaner {
	return &DVRQueueCleaner{queues: queues, torrents: torrents}
}

// DiscardFailedDownloads finds this title's queue entries that derive to
// failed and removes them via the client's plugins.QueueRemover capability.
// No failed entry (already gone, or *arr dropped it) is not an error.
func (c *DVRQueueCleaner) DiscardFailedDownloads(ctx context.Context, plugin string, externalID int64) (int, error) {
	client, err := c.queues.GetClient(ctx, plugin)
	if err != nil {
		return 0, err
	}
	items, err := client.GetQueue(ctx)
	if err != nil {
		return 0, err
	}

	// qBT is optional evidence, exactly as in the poller: unavailable means
	// only *arr's own "failed" counts.
	var torrents map[string]qbittorrent.Torrent
	if c.torrents != nil {
		if list, terr := c.torrents.GetAllDownloads(ctx, "all", "added_on", "desc"); terr == nil {
			torrents = make(map[string]qbittorrent.Torrent, len(list))
			for _, t := range list {
				torrents[strings.ToLower(t.Hash)] = t
			}
		}
	}

	var failed []string
	for _, item := range items {
		if item.ExternalID != externalID || item.DownloadID == "" {
			continue
		}
		if state, _ := deriveQueueState(item, torrents); state == queueStateFailed {
			failed = append(failed, item.DownloadID)
		}
	}
	if len(failed) == 0 {
		return 0, nil
	}

	remover, ok := client.(plugins.QueueRemover)
	if !ok {
		return 0, &plugins.PluginError{Code: plugins.ErrCodeNotSupported,
			Message: fmt.Sprintf("%s client cannot remove queue items", plugin)}
	}
	return remover.RemoveQueueItems(ctx, externalID, failed)
}

// SetQueueCleaner wires the optional queue cleaner (nil-safe: without it a
// queue-failed retry just resets the row — test/dev wiring only).
func (s *RequestService) SetQueueCleaner(cleaner RequestQueueCleaner) {
	s.queueCleaner = cleaner
}

// CancelRequest hard-deletes a PENDING request (AC #1).
//
// The delete is conditional on status='pending', so it is atomic against the
// 15s poller. Residual race, documented rather than locked away (AC #5): the
// poller may already be inside FulfilRequest for this row when it is
// deleted. If its AddMovie/AddSeries then succeeds, the follow-up
// UpdateFulfilment hits 0 rows → ErrRequestNotFound, which FulfilmentService
// logs ("Fulfilment transition write failed"). The title stays added in *arr
// with no request row — an orphaned *arr entry; re-requesting that title
// later hits *arr's "already exists" (disc-2026-07-arr-already-exists-loop,
// pre-existing, not fixed here). No row is ever resurrected.
func (s *RequestService) CancelRequest(ctx context.Context, id string) error {
	deleted, err := s.repo.DeleteIfPending(ctx, id)
	if err != nil {
		return fmt.Errorf("cancel request: %w", err)
	}
	if deleted > 0 {
		slog.Info("Media request cancelled", "request_id", id)
		return nil
	}
	row, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err // ErrRequestNotFound passes through typed
	}
	return fmt.Errorf("request %s is %s: %w", id, row.Status, ErrRequestNotCancellable)
}

// RetryRequest moves a FAILED request back into the pipeline (AC #2). Two
// failure flavors, split on external_id:
//
//   - (a) no external id — fulfilment itself failed terminally (e.g. not on
//     TVDB). Reset → pending and run FulfilRequest again, exactly like a
//     create: every degradation stays pending with a zh-TW error_message,
//     and the returned row carries whatever state fulfilment produced.
//   - (b) external id present — *arr has the title but its download broke.
//     First discard the broken download(s) with blocklist + re-search, THEN
//     reset → searching. The order matters: resetting first would let a
//     poller tick see the still-failed queue entry and flip the row back to
//     failed. If the discard fails the row stays failed and the caller gets
//     ErrRetryCleanupFailed — an honest "did nothing" beats a retry that
//     silently reverts 15 seconds later (13-7a CR; this replaces the
//     story's "best effort, still reset" wording).
func (s *RequestService) RetryRequest(ctx context.Context, id string) (*models.Request, error) {
	row, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row.Status != models.RequestStatusFailed {
		return nil, fmt.Errorf("request %s is %s: %w", id, row.Status, ErrRequestNotRetryable)
	}

	if !row.ExternalID.Valid {
		updated, err := s.resetForRetry(ctx, id, models.RequestStatusPending)
		if err != nil {
			return nil, err
		}
		slog.Info("Media request retried — fulfilling again", "request_id", id, "tmdb_id", updated.TMDbID)
		if s.fulfilment != nil {
			s.fulfilment.FulfilRequest(ctx, updated)
		}
		return updated, nil
	}

	// Duplicate check BEFORE touching *arr: a newer active request for the
	// title means the reset will be refused, and the blocklist/re-search
	// must not happen behind a 409.
	if _, err := s.repo.FindActiveByTMDbID(ctx, row.TMDbID, row.MediaType); err == nil {
		return nil, fmt.Errorf("tmdb_id %d (%s): %w", row.TMDbID, row.MediaType, repository.ErrRequestDuplicate)
	} else if !errors.Is(err, repository.ErrRequestNotFound) {
		return nil, fmt.Errorf("duplicate check: %w", err)
	}
	if err := s.discardFailedDownloads(ctx, row); err != nil {
		return nil, err
	}
	updated, err := s.resetForRetry(ctx, id, models.RequestStatusSearching)
	if err != nil {
		return nil, err
	}
	slog.Info("Media request retried — *arr searching again", "request_id", id, "external_id", row.ExternalID.String)
	return updated, nil
}

// resetForRetry wraps the conditional reset: a 0-row result means another
// retry or writer moved the row first, so re-read and report why.
func (s *RequestService) resetForRetry(ctx context.Context, id, status string) (*models.Request, error) {
	updated, err := s.repo.ResetForRetry(ctx, id, status)
	if errors.Is(err, repository.ErrRequestNotFound) {
		row, ferr := s.repo.FindByID(ctx, id)
		if ferr != nil {
			return nil, ferr
		}
		return nil, fmt.Errorf("request %s is %s: %w", id, row.Status, ErrRequestNotRetryable)
	}
	if err != nil {
		return nil, fmt.Errorf("retry request: %w", err) // ErrRequestDuplicate stays matchable
	}
	return updated, nil
}

// discardFailedDownloads is the *arr half of flavor (b).
func (s *RequestService) discardFailedDownloads(ctx context.Context, row *models.Request) error {
	if s.queueCleaner == nil {
		return nil
	}
	externalID, err := strconv.ParseInt(row.ExternalID.String, 10, 64)
	if err != nil {
		return fmt.Errorf("request %s external_id %q: %v: %w", row.ID, row.ExternalID.String, err, ErrRetryCleanupFailed)
	}
	plugin := dvrMoviePlugin
	if row.MediaType == models.RequestMediaTypeTV {
		plugin = dvrSeriesPlugin
	}
	removed, err := s.queueCleaner.DiscardFailedDownloads(ctx, plugin, externalID)
	if err != nil {
		slog.Warn("Retry could not discard the failed download; row stays failed",
			"request_id", row.ID, "plugin", plugin, "external_id", externalID, "removed", removed, "error", err)
		return fmt.Errorf("%s: %v: %w", plugin, err, ErrRetryCleanupFailed)
	}
	slog.Info("Retry discarded failed downloads", "request_id", row.ID, "plugin", plugin, "removed", removed)
	return nil
}
