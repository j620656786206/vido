package douban

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScraper_ParseReviewSummary validates the short-comment (短評) parser against
// the saved subject-page fixture (Story 12-6 Task 1.3/1.6): top comments extracted
// with author + star rating, total count read from the header, and the 5-item cap.
func TestScraper_ParseReviewSummary(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("testdata", "subject_comments.html"))
	require.NoError(t, err)

	client := NewClient(DefaultConfig(), nil)
	scraper := NewScraper(client, nil)

	result, err := scraper.parseReviewSummary("27010768", string(htmlBytes))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "27010768", result.ID)
	assert.Equal(t, 152340, result.TotalComments)

	// Six items in the fixture, capped at maxReviewComments.
	require.Len(t, result.TopComments, maxReviewComments)

	// First comment: author + 5-star rating (allstar50). Text is still Simplified
	// at the parse stage — conversion happens in convertCommentsToTraditional.
	assert.Equal(t, "影评人甲", result.TopComments[0].Author)
	assert.Equal(t, 5, result.TopComments[0].Rating)
	assert.Equal(t, "这部电影太棒了", result.TopComments[0].Text)

	// Star ratings decrease with the allstarNN class on each item.
	assert.Equal(t, 4, result.TopComments[1].Rating)
	assert.Equal(t, 3, result.TopComments[2].Rating)
	assert.Equal(t, 2, result.TopComments[3].Rating)
	assert.Equal(t, 1, result.TopComments[4].Rating)
}

// TestScraper_ConvertCommentsToTraditional asserts the s2twp conversion is applied
// to comment text (AC #3) — Simplified input becomes Traditional Chinese.
func TestScraper_ConvertCommentsToTraditional(t *testing.T) {
	client := NewClient(DefaultConfig(), nil)
	scraper := NewScraper(client, nil)

	result := &ReviewSummaryResult{
		ID: "27010768",
		TopComments: []ReviewComment{
			{Author: "影评人甲", Rating: 5, Text: "这部电影太棒了"},
			{Author: "观众乙", Rating: 4, Text: "导演的叙事很有张力"},
		},
	}

	scraper.convertCommentsToTraditional(result)

	assert.Equal(t, "這部電影太棒了", result.TopComments[0].Text)
	assert.Equal(t, "導演的敘事很有張力", result.TopComments[1].Text)
}

// TestScraper_ParseReviewSummary_NoSection confirms markup drift / a missing
// comments block degrades to an empty summary (not an error) — Rule 27 Pillar 3.
func TestScraper_ParseReviewSummary_NoSection(t *testing.T) {
	client := NewClient(DefaultConfig(), nil)
	scraper := NewScraper(client, nil)

	result, err := scraper.parseReviewSummary("1", `<html><body><div id="content"></div></body></html>`)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.TotalComments)
	assert.Empty(t, result.TopComments)
}

// TestScraper_ScrapeReviewSummary_DisabledClient confirms a disabled client returns
// a BlockedError before any network request (kill-switch — AC #5 / Rule 27 Pillar 3).
func TestScraper_ScrapeReviewSummary_DisabledClient(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	client := NewClient(cfg, nil)
	scraper := NewScraper(client, nil)

	result, err := scraper.ScrapeReviewSummary(context.Background(), "1")
	require.Error(t, err)
	assert.Nil(t, result)

	var blocked *BlockedError
	assert.ErrorAs(t, err, &blocked)
}
