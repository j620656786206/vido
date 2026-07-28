package models

import (
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubtitleRunStatus_IsValid(t *testing.T) {
	for _, s := range AllSubtitleRunStatuses() {
		assert.Truef(t, s.IsValid(), "%q is a declared run status and must be valid", s)
	}

	// 'translating' is a SubtitleStatus, never a run status — the two
	// vocabularies overlap by design and must not be confused.
	for _, junk := range []SubtitleRunStatus{"", "translating", "not_searched", "COMPLETED", "cancelled"} {
		assert.Falsef(t, junk.IsValid(), "%q must not be accepted as a run status", junk)
	}
}

func TestSubtitleRunStatus_MirrorsMigrationCheck(t *testing.T) {
	all := AllSubtitleRunStatuses()

	t.Run("holds exactly the 5 CHECK values, in migration order", func(t *testing.T) {
		assert.Equal(t, []SubtitleRunStatus{
			SubtitleRunPending,
			SubtitleRunRunning,
			SubtitleRunCompleted,
			SubtitleRunFailed,
			SubtitleRunSkipped,
		}, all, "the Go enum must mirror migration 030's status CHECK exactly")
	})

	// The same source-scanning guard AllSubtitleStatuses carries: a hand-written
	// mirror list alone cannot catch "added a const, forgot the slice", because
	// the author who forgets the slice forgets the list too.
	t.Run("every declared SubtitleRunStatus constant appears in the slice", func(t *testing.T) {
		src, err := os.ReadFile("subtitle_run.go")
		require.NoError(t, err)

		declared := regexp.MustCompile(`SubtitleRunStatus\s*=\s*"([a-z_]+)"`).FindAllStringSubmatch(string(src), -1)
		require.NotEmpty(t, declared, "regex must find the const declarations — update it if the declaration style changed")

		inSlice := map[SubtitleRunStatus]bool{}
		for _, s := range all {
			inSlice[s] = true
		}
		for _, m := range declared {
			assert.Truef(t, inSlice[SubtitleRunStatus(m[1])],
				"constant %q is declared but missing from AllSubtitleRunStatuses", m[1])
		}
		assert.Len(t, all, len(declared), "AllSubtitleRunStatuses must have one entry per declared constant")
	})
}

// TestRunVersion_Equal is the resume predicate's guard. Each field is mutated
// INDEPENDENTLY: a naive Equal that compares only three fields would pass a test
// that mutates them all at once, and the missed field would silently let a
// re-run reuse a stale translation.
func TestRunVersion_Equal(t *testing.T) {
	base := RunVersion{
		MetadataHash:    "meta-abc",
		GlossaryVersion: "",
		PromptVersion:   "p1",
		ModelID:         "claude-haiku-4-5",
	}

	t.Run("identical tuples are equal", func(t *testing.T) {
		assert.True(t, base.Equal(base))
		assert.True(t, base.Equal(RunVersion{
			MetadataHash:    "meta-abc",
			GlossaryVersion: "",
			PromptVersion:   "p1",
			ModelID:         "claude-haiku-4-5",
		}))
	})

	mutations := map[string]func(v RunVersion) RunVersion{
		"MetadataHash": func(v RunVersion) RunVersion {
			v.MetadataHash = "meta-xyz"
			return v
		},
		"GlossaryVersion": func(v RunVersion) RunVersion {
			// Always "" in M1 — but it MUST participate, or the P2 glossary
			// rollout would silently reuse pre-glossary translations.
			v.GlossaryVersion = "g1"
			return v
		},
		"PromptVersion": func(v RunVersion) RunVersion {
			v.PromptVersion = "p2"
			return v
		},
		"ModelID": func(v RunVersion) RunVersion {
			v.ModelID = "claude-sonnet-5"
			return v
		},
	}

	for field, mutate := range mutations {
		t.Run("differs when only "+field+" changes", func(t *testing.T) {
			other := mutate(base)
			assert.Falsef(t, base.Equal(other), "a changed %s must make the tuple non-matching", field)
			assert.Falsef(t, other.Equal(base), "%s inequality must be symmetric", field)
		})
	}

	t.Run("the zero tuple equals itself", func(t *testing.T) {
		assert.True(t, RunVersion{}.Equal(RunVersion{}))
		assert.False(t, RunVersion{}.Equal(base))
	})
}

func TestSubtitleRun_Version(t *testing.T) {
	run := &SubtitleRun{
		MetadataHash:    "meta-1",
		GlossaryVersion: "g-1",
		PromptVersion:   "p-1",
		ModelID:         "claude-haiku-4-5",
	}

	assert.Equal(t, RunVersion{
		MetadataHash:    "meta-1",
		GlossaryVersion: "g-1",
		PromptVersion:   "p-1",
		ModelID:         "claude-haiku-4-5",
	}, run.Version())
}

func TestSubtitleRun_Validate(t *testing.T) {
	valid := func() *SubtitleRun {
		return &SubtitleRun{MediaID: "m1", MediaType: SubtitleRunMediaMovie, Status: SubtitleRunPending}
	}

	t.Run("accepts a valid run", func(t *testing.T) {
		require.NoError(t, valid().Validate())
	})

	t.Run("accepts all three internal media grains", func(t *testing.T) {
		for _, grain := range []string{SubtitleRunMediaMovie, SubtitleRunMediaSeries, SubtitleRunMediaEpisode} {
			run := valid()
			run.MediaType = grain
			assert.NoErrorf(t, run.Validate(), "%q is an internal media grain", grain)
		}
	})

	t.Run("accepts an empty status (the DB default applies)", func(t *testing.T) {
		run := valid()
		run.Status = ""
		assert.NoError(t, run.Validate())
	})

	t.Run("rejects a blank media_id", func(t *testing.T) {
		for _, blank := range []string{"", "   "} {
			run := valid()
			run.MediaID = blank
			err := run.Validate()
			require.Error(t, err)
			var ve *ValidationError
			require.ErrorAs(t, err, &ve)
			assert.Equal(t, "media_id", ve.Field)
		}
	})

	t.Run("rejects the TMDB media_type vocabulary", func(t *testing.T) {
		run := valid()
		run.MediaType = "tv"
		err := run.Validate()
		require.Error(t, err)
		var ve *ValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "media_type", ve.Field, "'tv' is requests' vocabulary — subtitle_runs uses 'series'")
	})

	t.Run("rejects an unknown status", func(t *testing.T) {
		run := valid()
		run.Status = "translating" // a SubtitleStatus, not a run status
		err := run.Validate()
		require.Error(t, err)
		var ve *ValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "status", ve.Field)
	})
}

func TestSubtitleRun_OptionalPointerFields(t *testing.T) {
	// tmdb_id and completed_at are the two nullable columns modelled as
	// pointers — nil must survive as "unset", not as a zero value.
	run := &SubtitleRun{MediaID: "m1", MediaType: SubtitleRunMediaEpisode}
	assert.Nil(t, run.TMDbID, "an unmatched item carries no tmdb id")
	assert.Nil(t, run.CompletedAt, "an unfinished run has no completion time")

	id := int64(550)
	done := time.Now()
	run.TMDbID = &id
	run.CompletedAt = &done
	require.NotNil(t, run.TMDbID)
	assert.Equal(t, int64(550), *run.TMDbID)
	require.NotNil(t, run.CompletedAt)
}
