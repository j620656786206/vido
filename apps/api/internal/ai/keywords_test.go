package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeywordVariants_GetPrioritizedList(t *testing.T) {
	tests := []struct {
		name     string
		variants KeywordVariants
		want     []string
	}{
		{
			name: "full variants for Chinese anime",
			variants: KeywordVariants{
				Original:           "鬼滅之刃",
				SimplifiedChinese:  "鬼灭之刃",
				TraditionalChinese: "鬼滅之刃",
				English:            "Demon Slayer",
				Romaji:             "Kimetsu no Yaiba",
				AlternativeSpellings: []string{
					"Demon Slayer: Kimetsu no Yaiba",
				},
				CommonAliases: []string{
					"鬼滅",
				},
			},
			want: []string{
				"Demon Slayer",
				"Kimetsu no Yaiba",
				"鬼灭之刃",
				"Demon Slayer: Kimetsu no Yaiba",
				"鬼滅",
			},
		},
		{
			name: "English only",
			variants: KeywordVariants{
				Original: "The Matrix",
				English:  "The Matrix",
			},
			want: nil, // English same as original, no alternatives
		},
		{
			name: "with duplicates removed",
			variants: KeywordVariants{
				Original:             "Test",
				English:              "Test Movie",
				AlternativeSpellings: []string{"Test Movie", "Test Film"},
				CommonAliases:        []string{"Test Film", "TM"},
			},
			want: []string{
				"Test Movie",
				"Test Film",
				"TM",
			},
		},
		{
			name: "traditional chinese same as original is excluded",
			variants: KeywordVariants{
				Original:           "進撃の巨人",
				SimplifiedChinese:  "进击的巨人",
				TraditionalChinese: "進擊的巨人",
				English:            "Attack on Titan",
				Romaji:             "Shingeki no Kyojin",
			},
			want: []string{
				"Attack on Titan",
				"Shingeki no Kyojin",
				"进击的巨人",
				"進擊的巨人",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.variants.GetPrioritizedList()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKeywordVariants_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		variants KeywordVariants
		want     bool
	}{
		{
			name:     "empty variants",
			variants: KeywordVariants{},
			want:     true,
		},
		{
			name: "original only",
			variants: KeywordVariants{
				Original: "Test",
			},
			want: true, // Only original doesn't count as having alternatives
		},
		{
			name: "has english",
			variants: KeywordVariants{
				Original: "鬼滅之刃",
				English:  "Demon Slayer",
			},
			want: false,
		},
		{
			name: "has alternative spellings",
			variants: KeywordVariants{
				Original:             "Test",
				AlternativeSpellings: []string{"Test2"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.variants.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKeywordVariants_Count(t *testing.T) {
	variants := KeywordVariants{
		Original:             "Test",
		English:              "Test English",
		SimplifiedChinese:    "测试",
		TraditionalChinese:   "測試",
		Romaji:               "tesuto",
		AlternativeSpellings: []string{"alt1", "alt2"},
		CommonAliases:        []string{"alias1"},
	}

	// Count should be number of unique alternatives (excluding original)
	count := variants.Count()
	assert.Greater(t, count, 0)
}

func TestNewKeywordVariants(t *testing.T) {
	variants := NewKeywordVariants("鬼滅之刃")

	require.NotNil(t, variants)
	assert.Equal(t, "鬼滅之刃", variants.Original)
}

func TestKeywordVariantsFromJSON(t *testing.T) {
	jsonStr := `{
		"original": "鬼滅之刃",
		"simplified_chinese": "鬼灭之刃",
		"traditional_chinese": "鬼滅之刃",
		"english": "Demon Slayer",
		"romaji": "Kimetsu no Yaiba",
		"alternative_spellings": ["Demon Slayer: Kimetsu no Yaiba"],
		"common_aliases": ["鬼滅"]
	}`

	variants, err := KeywordVariantsFromJSON(jsonStr)

	require.NoError(t, err)
	assert.Equal(t, "鬼滅之刃", variants.Original)
	assert.Equal(t, "鬼灭之刃", variants.SimplifiedChinese)
	assert.Equal(t, "Demon Slayer", variants.English)
	assert.Equal(t, "Kimetsu no Yaiba", variants.Romaji)
	assert.Len(t, variants.AlternativeSpellings, 1)
	assert.Len(t, variants.CommonAliases, 1)
}

func TestKeywordVariantsFromJSON_Invalid(t *testing.T) {
	_, err := KeywordVariantsFromJSON("invalid json")
	assert.Error(t, err)
}

func TestKeywordVariantsFromJSON_MarkdownWrapped(t *testing.T) {
	jsonStr := "```json\n{\"original\": \"Test\", \"english\": \"Test English\"}\n```"

	variants, err := KeywordVariantsFromJSON(jsonStr)

	require.NoError(t, err)
	assert.Equal(t, "Test", variants.Original)
	assert.Equal(t, "Test English", variants.English)
}

// [P1] Tests ToJSON serialization produces valid JSON output
func TestKeywordVariants_ToJSON_Success(t *testing.T) {
	// GIVEN: A KeywordVariants with all fields populated
	variants := KeywordVariants{
		Original:             "鬼滅之刃",
		SimplifiedChinese:    "鬼灭之刃",
		TraditionalChinese:   "鬼滅之刃",
		English:              "Demon Slayer",
		Romaji:               "Kimetsu no Yaiba",
		Pinyin:               "gui mie zhi ren",
		AlternativeSpellings: []string{"Demon Slayer: Kimetsu no Yaiba"},
		CommonAliases:        []string{"鬼滅", "DS"},
	}

	// WHEN: Serializing to JSON
	jsonStr, err := variants.ToJSON()

	// THEN: Should produce valid JSON without error
	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	// AND: JSON should be parseable back to KeywordVariants
	parsed, parseErr := KeywordVariantsFromJSON(jsonStr)
	require.NoError(t, parseErr)
	assert.Equal(t, variants.Original, parsed.Original)
	assert.Equal(t, variants.English, parsed.English)
	assert.Equal(t, variants.Romaji, parsed.Romaji)
	assert.Equal(t, variants.AlternativeSpellings, parsed.AlternativeSpellings)
	assert.Equal(t, variants.CommonAliases, parsed.CommonAliases)
}

// [P2] Tests ToJSON with minimal fields
func TestKeywordVariants_ToJSON_MinimalFields(t *testing.T) {
	// GIVEN: A KeywordVariants with only original field
	variants := KeywordVariants{
		Original: "Test",
	}

	// WHEN: Serializing to JSON
	jsonStr, err := variants.ToJSON()

	// THEN: Should produce valid JSON
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"original":"Test"`)
}

// [P2] Tests GetPrioritizedList with nil slices
func TestKeywordVariants_GetPrioritizedList_NilSlices(t *testing.T) {
	// GIVEN: A KeywordVariants with nil slices
	variants := KeywordVariants{
		Original:             "Test",
		English:              "Test English",
		AlternativeSpellings: nil,
		CommonAliases:        nil,
	}

	// WHEN: Getting prioritized list
	result := variants.GetPrioritizedList()

	// THEN: Should return list without panic
	assert.Equal(t, []string{"Test English"}, result)
}

// [P2] Tests GetPrioritizedList with empty slices
func TestKeywordVariants_GetPrioritizedList_EmptySlices(t *testing.T) {
	// GIVEN: A KeywordVariants with empty slices
	variants := KeywordVariants{
		Original:             "Test",
		English:              "Test English",
		Romaji:               "Tesuto",
		AlternativeSpellings: []string{},
		CommonAliases:        []string{},
	}

	// WHEN: Getting prioritized list
	result := variants.GetPrioritizedList()

	// THEN: Should return only non-empty fields
	assert.Equal(t, []string{"Test English", "Tesuto"}, result)
}
