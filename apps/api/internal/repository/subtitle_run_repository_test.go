package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupSubtitleRunDB applies the REAL migration chain (incl. 030) so these tests
// track the shipped schema. Rule 15 / bugfix-20-1: a mocked repository cannot
// catch a column that exists in the table but is missing from the SELECT list —
// that is precisely how series.seasons silently returned [] for every series.
func setupSubtitleRunDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func fullyPopulatedRun() *models.SubtitleRun {
	tmdbID := int64(1399)
	completed := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
	return &models.SubtitleRun{
		ID:              "run-full",
		MediaID:         "media-42",
		MediaType:       models.SubtitleRunMediaEpisode,
		TMDbID:          &tmdbID,
		MetadataHash:    "meta-hash-abc",
		GlossaryVersion: "glossary-v1",
		PromptVersion:   "prompt-v3",
		ModelID:         "claude-haiku-4-5",
		Status:          models.SubtitleRunCompleted,
		SourceLanguage:  "eng",
		OutputPath:      "/media/tv/show/S01E01.zh-Hant.srt",
		CueCount:        842,
		CacheEnabled:    true,
		ErrorMessage:    "",
		StartedAt:       time.Now().Add(-2 * time.Minute).UTC().Truncate(time.Second),
		CompletedAt:     &completed,
	}
}

// TestSubtitleRunRepository_RoundTripsAllSixteenColumns is the Rule 15 DB Column
// Sync guard: every column of migration 030 must survive INSERT → SELECT →
// Scan. A column present in the table but missing from subtitleRunColumns reads
// back as its zero value, which this test fails on.
func TestSubtitleRunRepository_RoundTripsAllSixteenColumns(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()

	want := fullyPopulatedRun()
	require.NoError(t, repo.Create(ctx, want))

	got, err := repo.FindByID(ctx, want.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	// 16 columns, asserted one by one — a loop over a struct would hide which
	// field was dropped.
	assert.Equal(t, want.ID, got.ID)                                           // 1
	assert.Equal(t, want.MediaID, got.MediaID)                                 // 2
	assert.Equal(t, want.MediaType, got.MediaType)                             // 3
	require.NotNil(t, got.TMDbID, "tmdb_id must survive")                      // 4
	assert.Equal(t, *want.TMDbID, *got.TMDbID)                                 //
	assert.Equal(t, want.MetadataHash, got.MetadataHash)                       // 5
	assert.Equal(t, want.GlossaryVersion, got.GlossaryVersion)                 // 6
	assert.Equal(t, want.PromptVersion, got.PromptVersion)                     // 7
	assert.Equal(t, want.ModelID, got.ModelID)                                 // 8
	assert.Equal(t, want.Status, got.Status)                                   // 9
	assert.Equal(t, want.SourceLanguage, got.SourceLanguage)                   // 10
	assert.Equal(t, want.OutputPath, got.OutputPath)                           // 11
	assert.Equal(t, want.CueCount, got.CueCount)                               // 12
	assert.True(t, got.CacheEnabled, "cache_enabled must survive as true")     // 13
	assert.Equal(t, want.ErrorMessage, got.ErrorMessage)                       // 14
	assert.WithinDuration(t, want.StartedAt, got.StartedAt, time.Second)       // 15
	require.NotNil(t, got.CompletedAt, "completed_at must survive")            // 16
	assert.WithinDuration(t, *want.CompletedAt, *got.CompletedAt, time.Second) //

	// The version tuple must reassemble from the persisted columns — this is
	// what sub-1-5b hashes into the cue-grain cache key.
	assert.Equal(t, want.Version(), got.Version())
}

func TestSubtitleRunRepository_NullableColumnsRoundTripAsUnset(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()

	// An unmatched, still-running item: no tmdb id, no completion time, and
	// none of the optional text columns set.
	run := &models.SubtitleRun{MediaID: "media-1", MediaType: models.SubtitleRunMediaMovie}
	require.NoError(t, repo.Create(ctx, run))

	assert.NotEmpty(t, run.ID, "Create must assign an id")
	assert.Equal(t, models.SubtitleRunPending, run.Status, "Create must default the status")
	assert.False(t, run.StartedAt.IsZero(), "Create must stamp started_at")

	got, err := repo.FindByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Nil(t, got.TMDbID, "an absent tmdb_id must stay nil, not become 0")
	assert.Nil(t, got.CompletedAt, "an unfinished run must have no completion time")
	assert.Equal(t, "", got.SourceLanguage)
	assert.Equal(t, "", got.OutputPath)
	assert.Equal(t, 0, got.CueCount)
	assert.False(t, got.CacheEnabled, "cache_enabled is reserved for sub-1-5 and defaults false")
	assert.Equal(t, "", got.ErrorMessage)
}

// TestSubtitleRunRepository_ScanToleratesRawNulls covers rows written outside
// the repository (a bare INSERT applying column defaults): the nullable columns
// are genuinely NULL there, not empty strings.
func TestSubtitleRunRepository_ScanToleratesRawNulls(t *testing.T) {
	db := setupSubtitleRunDB(t)
	repo := NewSubtitleRunRepository(db)

	_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type) VALUES ('raw', 'm-raw', 'series')`)
	require.NoError(t, err)

	got, err := repo.FindByID(context.Background(), "raw")
	require.NoError(t, err, "scanning genuine NULLs must not error")
	assert.Equal(t, models.SubtitleRunPending, got.Status)
	assert.Nil(t, got.TMDbID)
	assert.Equal(t, "", got.SourceLanguage)
	assert.Equal(t, 0, got.CueCount)
}

func TestSubtitleRunRepository_Create_Validates(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()

	t.Run("rejects a nil run", func(t *testing.T) {
		assert.Error(t, repo.Create(ctx, nil))
	})

	t.Run("rejects a blank media_id before hitting the DB", func(t *testing.T) {
		err := repo.Create(ctx, &models.SubtitleRun{MediaType: models.SubtitleRunMediaMovie})
		require.Error(t, err)
		var ve *models.ValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "media_id", ve.Field)
	})

	t.Run("rejects the TMDB media_type vocabulary", func(t *testing.T) {
		err := repo.Create(ctx, &models.SubtitleRun{MediaID: "m1", MediaType: "tv"})
		require.Error(t, err)
		var ve *models.ValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "media_type", ve.Field)
	})
}

func TestSubtitleRunRepository_Update(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()

	run := &models.SubtitleRun{MediaID: "media-9", MediaType: models.SubtitleRunMediaMovie}
	require.NoError(t, repo.Create(ctx, run))

	t.Run("overwrites every mutable column", func(t *testing.T) {
		tmdbID := int64(550)
		done := time.Now().UTC().Truncate(time.Second)
		run.TMDbID = &tmdbID
		run.MetadataHash = "meta-1"
		run.GlossaryVersion = "g-1"
		run.PromptVersion = "p-1"
		run.ModelID = "claude-haiku-4-5"
		run.Status = models.SubtitleRunCompleted
		run.SourceLanguage = "eng"
		run.OutputPath = "/media/movies/x.zh-Hant.srt"
		run.CueCount = 120
		run.CacheEnabled = true
		run.ErrorMessage = "none"
		run.CompletedAt = &done
		require.NoError(t, repo.Update(ctx, run))

		got, err := repo.FindByID(ctx, run.ID)
		require.NoError(t, err)
		require.NotNil(t, got.TMDbID)
		assert.Equal(t, int64(550), *got.TMDbID)
		assert.Equal(t, models.SubtitleRunCompleted, got.Status)
		assert.Equal(t, "eng", got.SourceLanguage)
		assert.Equal(t, "/media/movies/x.zh-Hant.srt", got.OutputPath)
		assert.Equal(t, 120, got.CueCount)
		assert.True(t, got.CacheEnabled)
		assert.Equal(t, "none", got.ErrorMessage)
		require.NotNil(t, got.CompletedAt)
		assert.Equal(t, run.Version(), got.Version())
	})

	t.Run("clears a previously-set nullable back to unset", func(t *testing.T) {
		run.TMDbID = nil
		run.CompletedAt = nil
		run.Status = models.SubtitleRunRunning
		require.NoError(t, repo.Update(ctx, run))

		got, err := repo.FindByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Nil(t, got.TMDbID, "Update must be able to clear a column, not only set it")
		assert.Nil(t, got.CompletedAt)
	})

	t.Run("requires an id", func(t *testing.T) {
		err := repo.Update(ctx, &models.SubtitleRun{MediaID: "m", MediaType: models.SubtitleRunMediaMovie})
		require.Error(t, err)
		var ve *models.ValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "id", ve.Field)
	})

	t.Run("rejects an empty status with a typed error, not a raw CHECK failure", func(t *testing.T) {
		bad := &models.SubtitleRun{
			ID: run.ID, MediaID: run.MediaID, MediaType: run.MediaType,
			StartedAt: run.StartedAt,
		}
		err := repo.Update(ctx, bad)
		require.Error(t, err)
		var ve *models.ValidationError
		require.ErrorAs(t, err, &ve,
			"an empty status must be caught before the DB — the CHECK constraint would surface as an opaque driver error")
		assert.Equal(t, "status", ve.Field)
	})

	t.Run("rejects a zero started_at instead of silently zeroing the column", func(t *testing.T) {
		bad := &models.SubtitleRun{
			ID: run.ID, MediaID: run.MediaID, MediaType: run.MediaType,
			Status: models.SubtitleRunRunning,
		}
		err := repo.Update(ctx, bad)
		require.Error(t, err)
		var ve *models.ValidationError
		require.ErrorAs(t, err, &ve,
			"Update overwrites every column — a zero started_at would corrupt ORDER BY started_at semantics")
		assert.Equal(t, "started_at", ve.Field)
	})

	t.Run("reports a missing row", func(t *testing.T) {
		err := repo.Update(ctx, &models.SubtitleRun{
			ID: "nope", MediaID: "m", MediaType: models.SubtitleRunMediaMovie,
			Status: models.SubtitleRunPending, StartedAt: time.Now().UTC(),
		})
		assert.ErrorIs(t, err, ErrSubtitleRunNotFound)
	})
}

func TestSubtitleRunRepository_FindByID_NotFound(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))

	got, err := repo.FindByID(context.Background(), "missing")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrSubtitleRunNotFound)
}

// --- the resume predicate (AC #5) ---

func seedCompletedRun(t *testing.T, repo *SubtitleRunRepository, v models.RunVersion) *models.SubtitleRun {
	t.Helper()
	run := &models.SubtitleRun{
		MediaID:         "media-r",
		MediaType:       models.SubtitleRunMediaEpisode,
		MetadataHash:    v.MetadataHash,
		GlossaryVersion: v.GlossaryVersion,
		PromptVersion:   v.PromptVersion,
		ModelID:         v.ModelID,
		Status:          models.SubtitleRunCompleted,
		OutputPath:      "/media/tv/x.zh-Hant.srt",
	}
	require.NoError(t, repo.Create(context.Background(), run))
	return run
}

func baseVersion() models.RunVersion {
	return models.RunVersion{
		MetadataHash:    "meta-1",
		GlossaryVersion: "",
		PromptVersion:   "p-1",
		ModelID:         "claude-haiku-4-5",
	}
}

func TestSubtitleRunRepository_FindCompletedRun_ExactTupleMatches(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()

	seeded := seedCompletedRun(t, repo, baseVersion())

	got, err := repo.FindCompletedRun(ctx, "media-r", models.SubtitleRunMediaEpisode, baseVersion())
	require.NoError(t, err)
	require.NotNil(t, got, "an exact tuple match is the resume case")
	assert.Equal(t, seeded.ID, got.ID)
	assert.Equal(t, "/media/tv/x.zh-Hant.srt", got.OutputPath)
}

// TestSubtitleRunRepository_FindCompletedRun_AnySingleTupleFieldMisses mutates
// ONE field at a time. A predicate that compares only three of the four columns
// would pass a test that changes them all together — and would silently reuse a
// prior translation after a prompt bump, invalidating the M1 pilot's comparison.
func TestSubtitleRunRepository_FindCompletedRun_AnySingleTupleFieldMisses(t *testing.T) {
	mutations := map[string]func(v models.RunVersion) models.RunVersion{
		"MetadataHash": func(v models.RunVersion) models.RunVersion {
			v.MetadataHash = "meta-2"
			return v
		},
		"GlossaryVersion": func(v models.RunVersion) models.RunVersion {
			v.GlossaryVersion = "g-1"
			return v
		},
		"PromptVersion": func(v models.RunVersion) models.RunVersion {
			v.PromptVersion = "p-2"
			return v
		},
		"ModelID": func(v models.RunVersion) models.RunVersion {
			v.ModelID = "claude-sonnet-5"
			return v
		},
	}

	for field, mutate := range mutations {
		t.Run("a changed "+field+" yields no match", func(t *testing.T) {
			repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
			seedCompletedRun(t, repo, baseVersion())

			got, err := repo.FindCompletedRun(context.Background(), "media-r", models.SubtitleRunMediaEpisode, mutate(baseVersion()))
			require.NoError(t, err, "a non-match is not an error")
			assert.Nilf(t, got, "a changed %s must make the prior run non-matching", field)
		})
	}
}

func TestSubtitleRunRepository_FindCompletedRun_IgnoresNonCompletedStatuses(t *testing.T) {
	ctx := context.Background()

	for _, status := range []models.SubtitleRunStatus{
		models.SubtitleRunPending,
		models.SubtitleRunRunning,
		models.SubtitleRunFailed,
		models.SubtitleRunSkipped,
	} {
		t.Run("a "+string(status)+" run with a matching tuple is not resumable", func(t *testing.T) {
			repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
			v := baseVersion()
			run := &models.SubtitleRun{
				MediaID: "media-r", MediaType: models.SubtitleRunMediaEpisode,
				MetadataHash: v.MetadataHash, GlossaryVersion: v.GlossaryVersion,
				PromptVersion: v.PromptVersion, ModelID: v.ModelID,
				Status: status,
			}
			require.NoError(t, repo.Create(ctx, run))

			got, err := repo.FindCompletedRun(ctx, "media-r", models.SubtitleRunMediaEpisode, v)
			require.NoError(t, err)
			assert.Nil(t, got, "only a completed run produced a subtitle worth resuming from")
		})
	}
}

func TestSubtitleRunRepository_FindCompletedRun_MediaScoped(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()
	seedCompletedRun(t, repo, baseVersion())

	t.Run("a different media id does not match", func(t *testing.T) {
		got, err := repo.FindCompletedRun(ctx, "media-other", models.SubtitleRunMediaEpisode, baseVersion())
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("the same id under a different grain does not match", func(t *testing.T) {
		got, err := repo.FindCompletedRun(ctx, "media-r", models.SubtitleRunMediaMovie, baseVersion())
		require.NoError(t, err)
		assert.Nil(t, got, "media_id is only unique within a grain — an episode run must not answer for a movie")
	})
}

func TestSubtitleRunRepository_FindCompletedRun_NoRowsIsNotAnError(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))

	got, err := repo.FindCompletedRun(context.Background(), "never-run", models.SubtitleRunMediaMovie, baseVersion())
	assert.NoError(t, err, "the first run of any item hits this path — it must not error")
	assert.Nil(t, got)
}

func TestSubtitleRunRepository_FindCompletedRun_ReturnsMostRecent(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()
	v := baseVersion()

	older := &models.SubtitleRun{
		MediaID: "media-r", MediaType: models.SubtitleRunMediaEpisode,
		MetadataHash: v.MetadataHash, GlossaryVersion: v.GlossaryVersion,
		PromptVersion: v.PromptVersion, ModelID: v.ModelID,
		Status: models.SubtitleRunCompleted, OutputPath: "/old.srt",
		StartedAt: time.Now().Add(-time.Hour).UTC(),
	}
	newer := &models.SubtitleRun{
		MediaID: "media-r", MediaType: models.SubtitleRunMediaEpisode,
		MetadataHash: v.MetadataHash, GlossaryVersion: v.GlossaryVersion,
		PromptVersion: v.PromptVersion, ModelID: v.ModelID,
		Status: models.SubtitleRunCompleted, OutputPath: "/new.srt",
		StartedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, older))
	require.NoError(t, repo.Create(ctx, newer))

	got, err := repo.FindCompletedRun(ctx, "media-r", models.SubtitleRunMediaEpisode, v)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "/new.srt", got.OutputPath, "the newest matching run wins")
}

// TestSubtitleRunRepository_TimesStoredAsUTC locks the started_at storage
// format. The driver stores a time.Time as text and the resume predicate /
// ListByStatus ORDER BY compares that text, so a local-zone value ("… 18:00:00
// +0800 CST") would out-sort a genuinely later UTC row by wall-clock digits.
// Normalizing to UTC at the value layer keeps the textual order equal to the
// chronological order.
func TestSubtitleRunRepository_TimesStoredAsUTC(t *testing.T) {
	db := setupSubtitleRunDB(t)
	repo := NewSubtitleRunRepository(db)
	ctx := context.Background()
	v := baseVersion()

	newRun := func(id string, started time.Time, output string) *models.SubtitleRun {
		return &models.SubtitleRun{
			ID: id, MediaID: "media-tz", MediaType: models.SubtitleRunMediaEpisode,
			MetadataHash: v.MetadataHash, GlossaryVersion: v.GlossaryVersion,
			PromptVersion: v.PromptVersion, ModelID: v.ModelID,
			Status: models.SubtitleRunCompleted, OutputPath: output,
			StartedAt: started,
		}
	}

	// 18:00 at +08:00 == 10:00 UTC — chronologically OLDER than 12:00 UTC, but
	// its unnormalized text ("… 18:00:00 +0800 CST") would sort as newer.
	local := time.Date(2026, 7, 28, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
	utc := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Create(ctx, newRun("run-local", local, "/older-local.srt")))
	require.NoError(t, repo.Create(ctx, newRun("run-utc", utc, "/newer-utc.srt")))

	t.Run("started_at is persisted as the UTC instant", func(t *testing.T) {
		var raw string
		require.NoError(t, db.QueryRow(
			`SELECT CAST(started_at AS TEXT) FROM subtitle_runs WHERE id = 'run-local'`).Scan(&raw))
		assert.Truef(t, strings.HasPrefix(raw, "2026-07-28 10:00:00"),
			"a +08:00 local time must be stored as its UTC instant, got %q", raw)
	})

	t.Run("the chronologically newest run wins ORDER BY across source zones", func(t *testing.T) {
		got, err := repo.FindCompletedRun(ctx, "media-tz", models.SubtitleRunMediaEpisode, v)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "/newer-utc.srt", got.OutputPath,
			"12:00 UTC is later than 18:00+08:00 (=10:00 UTC) — text order must agree")
	})

	t.Run("a scanned run reads back as the same instant", func(t *testing.T) {
		got, err := repo.FindByID(ctx, "run-local")
		require.NoError(t, err)
		assert.True(t, got.StartedAt.Equal(local), "the instant survives even though the zone is normalized")
	})
}

func TestSubtitleRunRepository_ListByStatus(t *testing.T) {
	repo := NewSubtitleRunRepository(setupSubtitleRunDB(t))
	ctx := context.Background()

	for i, status := range []models.SubtitleRunStatus{
		models.SubtitleRunFailed, models.SubtitleRunFailed, models.SubtitleRunCompleted,
	} {
		require.NoError(t, repo.Create(ctx, &models.SubtitleRun{
			MediaID:   "m" + string(rune('a'+i)),
			MediaType: models.SubtitleRunMediaMovie,
			Status:    status,
			StartedAt: time.Now().Add(time.Duration(i) * time.Minute).UTC(),
		}))
	}

	t.Run("filters by status, newest first", func(t *testing.T) {
		failed, err := repo.ListByStatus(ctx, models.SubtitleRunFailed, 0)
		require.NoError(t, err)
		require.Len(t, failed, 2)
		for _, run := range failed {
			assert.Equal(t, models.SubtitleRunFailed, run.Status)
		}
		assert.Equal(t, "mb", failed[0].MediaID, "the interface promises newest first — mb started after ma")
		assert.Equal(t, "ma", failed[1].MediaID)
	})

	t.Run("honours the limit, keeping the newest", func(t *testing.T) {
		failed, err := repo.ListByStatus(ctx, models.SubtitleRunFailed, 1)
		require.NoError(t, err)
		require.Len(t, failed, 1)
		assert.Equal(t, "mb", failed[0].MediaID, "LIMIT must trim the tail, not the head")
	})

	t.Run("an empty result is not an error", func(t *testing.T) {
		runs, err := repo.ListByStatus(ctx, models.SubtitleRunSkipped, 0)
		require.NoError(t, err)
		assert.Empty(t, runs)
	})
}

// TestSubtitleRunRepository_RegisteredInBothConstructors guards the classic
// half-wiring bug: a repository added to NewRepositories but forgotten in
// NewRepositoriesWithCache (which is the constructor main.go actually calls).
func TestSubtitleRunRepository_RegisteredInBothConstructors(t *testing.T) {
	db := setupSubtitleRunDB(t)

	assert.NotNil(t, NewRepositories(db).SubtitleRuns, "NewRepositories must wire SubtitleRuns")
	assert.NotNil(t, NewRepositoriesWithCache(db).SubtitleRuns, "NewRepositoriesWithCache is what main.go calls — it must wire SubtitleRuns too")
}
