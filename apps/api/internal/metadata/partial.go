package metadata

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/models"
)

// MetadataResult represents a result from a single metadata source.
type MetadataResult struct {
	Source    string
	Title     string
	Year      int
	Overview  string
	PosterURL string
	Genres    []string
	Cast      []string
}

// MergedMetadata represents the merged metadata from multiple sources.
type MergedMetadata struct {
	Title       string   `json:"title"`
	Year        int      `json:"year,omitempty"`
	Overview    string   `json:"overview,omitempty"`
	PosterURL   string   `json:"posterUrl,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Cast        []string `json:"cast,omitempty"`
	PlaceholderFields []string `json:"placeholderFields,omitempty"`
}

// PartialResultHandler merges and fills incomplete metadata.
type PartialResultHandler struct {
	logger *slog.Logger
}

// NewPartialResultHandler creates a new PartialResultHandler.
func NewPartialResultHandler() *PartialResultHandler {
	return &PartialResultHandler{
		logger: slog.Default(),
	}
}

// RequiredFields returns the list of fields required for a complete result.
func RequiredFields() []string {
	return []string{"title", "year", "overview", "posterUrl"}
}

// MergePartialResults merges multiple metadata results into a single result.
// It fills missing fields from subsequent sources in order.
func (h *PartialResultHandler) MergePartialResults(results []*MetadataResult) *models.DegradedResult {
	merged := &MergedMetadata{}
	missing := make([]string, 0)
	sources := make([]string, 0)
	fieldSources := make(map[string]string)

	requiredFields := RequiredFields()

	// Track which fields have been filled
	filled := make(map[string]bool)

	// Merge results in order
	for _, result := range results {
		if result == nil {
			continue
		}

		sources = append(sources, result.Source)

		// Fill title
		if !filled["title"] && result.Title != "" {
			merged.Title = result.Title
			filled["title"] = true
			fieldSources["title"] = result.Source
		}

		// Fill year
		if !filled["year"] && result.Year > 0 {
			merged.Year = result.Year
			filled["year"] = true
			fieldSources["year"] = result.Source
		}

		// Fill overview
		if !filled["overview"] && result.Overview != "" {
			merged.Overview = result.Overview
			filled["overview"] = true
			fieldSources["overview"] = result.Source
		}

		// Fill poster URL
		if !filled["posterUrl"] && result.PosterURL != "" {
			merged.PosterURL = result.PosterURL
			filled["posterUrl"] = true
			fieldSources["posterUrl"] = result.Source
		}

		// Fill genres (merge all unique)
		if len(result.Genres) > 0 {
			merged.Genres = mergeStringSlices(merged.Genres, result.Genres)
			if !filled["genres"] {
				filled["genres"] = true
				fieldSources["genres"] = result.Source
			}
		}

		// Fill cast (merge all unique)
		if len(result.Cast) > 0 {
			merged.Cast = mergeStringSlices(merged.Cast, result.Cast)
			if !filled["cast"] {
				filled["cast"] = true
				fieldSources["cast"] = result.Source
			}
		}
	}

	// Check for missing required fields and set placeholders
	for _, field := range requiredFields {
		if !filled[field] {
			missing = append(missing, field)
			h.SetPlaceholder(merged, field)
			merged.PlaceholderFields = append(merged.PlaceholderFields, field)
		}
	}

	// Determine degradation level
	level := models.DegradationNormal
	if len(missing) > 0 {
		level = models.DegradationPartial
		if len(missing) > len(requiredFields)/2 {
			level = models.DegradationMinimal
		}
	}

	message := h.GenerateMessage(missing)

	h.logger.Debug("Merged partial results",
		"sources", sources,
		"missing_fields", missing,
		"degradation_level", level,
	)

	return &models.DegradedResult{
		Data:             merged,
		DegradationLevel: level,
		MissingFields:    missing,
		FallbackUsed:     sources,
		Message:          message,
	}
}

// GenerateMessage generates a user-friendly message for missing fields.
func (h *PartialResultHandler) GenerateMessage(missing []string) string {
	if len(missing) == 0 {
		return ""
	}

	fieldLabels := map[string]string{
		"title":     "標題",
		"year":      "年份",
		"overview":  "簡介",
		"posterUrl": "海報",
		"genres":    "類型",
		"cast":      "演員",
	}

	var labels []string
	for _, field := range missing {
		if label, ok := fieldLabels[field]; ok {
			labels = append(labels, label)
		}
	}

	if len(labels) == 0 {
		return ""
	}

	return fmt.Sprintf("以下資料暫時無法取得：%s", strings.Join(labels, "、"))
}

// SetPlaceholder sets a placeholder value for a missing field.
func (h *PartialResultHandler) SetPlaceholder(item *MergedMetadata, field string) {
	switch field {
	case "title":
		item.Title = "未知標題"
	case "overview":
		item.Overview = "暫無簡介"
	case "posterUrl":
		item.PosterURL = "/images/placeholder-poster.webp"
	}
}

// mergeStringSlices merges two string slices, removing duplicates.
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}
