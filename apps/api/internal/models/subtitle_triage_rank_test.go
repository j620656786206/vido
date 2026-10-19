package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story disc-2026-10-library-list-triage-a AC #2: the library's 字幕狀態 sort
// ranks the badge the list shows. This is the BACKEND half of the contract —
// every (verdict × subtitle_status) combination and the rank it must land in.
// The FRONTEND half (deriveSubtitleStatus in apps/web/src/utils/libraryStatus.ts
// produces the same badge for the same pair) is pinned by B2's table test
// (disc-2026-10-library-list-triage-b2); if either side changes, both tables
// must change together.
//
//	0 缺中文 → 1 無字幕源／未翻譯／已略過 → 2 不知道 → 3 簡中 → 4 中文 → 5 繁中
func TestSubtitleTriageRank_EveryCombination(t *testing.T) {
	statuses := append([]SubtitleStatus{""}, AllSubtitleStatuses()...)
	require.Len(t, statuses, 11, "10 subtitle_status values + empty (NULL) — a new status needs a row below")

	// want[verdict][status] — written out by hand, not derived from the rule.
	want := map[ChineseSubtitle]map[SubtitleStatus]int{
		ChineseSubtitleZhHant: {
			"": 5, "not_searched": 5, "searching": 5, "found": 5, "not_found": 5,
			"probing": 5, "extracting": 5, "translating": 5,
			"no_text_source": 5, "skipped": 5, "untranslated": 5,
		},
		ChineseSubtitleZhHans: {
			"": 3, "not_searched": 3, "searching": 3, "found": 3, "not_found": 3,
			"probing": 3, "extracting": 3, "translating": 3,
			"no_text_source": 3, "skipped": 3, "untranslated": 3,
		},
		ChineseSubtitleZh: {
			"": 4, "not_searched": 4, "searching": 4, "found": 4, "not_found": 4,
			"probing": 4, "extracting": 4, "translating": 4,
			"no_text_source": 4, "skipped": 4, "untranslated": 4,
		},
		ChineseSubtitleNone: {
			"": 0, "not_searched": 0, "searching": 0, "found": 0, "not_found": 0,
			"probing": 2, "extracting": 2, "translating": 2,
			"no_text_source": 1, "skipped": 1, "untranslated": 1,
		},
		ChineseSubtitleUnknown: {
			"": 2, "not_searched": 2, "searching": 2, "found": 2, "not_found": 2,
			"probing": 2, "extracting": 2, "translating": 2,
			"no_text_source": 1, "skipped": 1, "untranslated": 1,
		},
	}
	require.Len(t, want, len(AllChineseSubtitleVerdicts()), "a new verdict needs a row above")

	for verdict, byStatus := range want {
		for _, st := range statuses {
			exp, ok := byStatus[st]
			require.True(t, ok, "missing expectation for %q × %q", verdict, st)
			assert.Equal(t, exp, SubtitleTriageRank(verdict, st), "verdict=%q status=%q", verdict, st)
		}
	}

	// An unreadable verdict ranks as unknown (the badge shows nothing for it).
	for _, bad := range []ChineseSubtitle{"", "garbage", "ZH_HANT"} {
		for _, st := range statuses {
			assert.Equal(t, want[ChineseSubtitleUnknown][st], SubtitleTriageRank(bad, st), "verdict=%q status=%q", bad, st)
		}
	}
}

func TestSubtitleTriageRank_NamedConstants(t *testing.T) {
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, []int{
		SubtitleRankMissing, SubtitleRankNoSourceOrStopped, SubtitleRankUnknown,
		SubtitleRankHans, SubtitleRankZh, SubtitleRankHant,
	})
}

// An unknown subtitle_status (a value a future migration adds before this
// table learns it) must not be mistaken for a pipeline reason.
func TestSubtitleTriageRank_UnknownStatus(t *testing.T) {
	assert.Equal(t, SubtitleRankMissing, SubtitleTriageRank(ChineseSubtitleNone, "some_future_state"))
	assert.Equal(t, SubtitleRankUnknown, SubtitleTriageRank(ChineseSubtitleUnknown, "some_future_state"))
}
