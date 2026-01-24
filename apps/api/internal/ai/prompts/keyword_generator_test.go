// Package prompts provides prompt templates for AI-powered operations.
package prompts

import (
	"strings"
	"testing"
)

func TestBuildKeywordPrompt(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		language string
		wantContains []string
	}{
		{
			name:     "Chinese title",
			title:    "鬼滅之刃",
			language: "Traditional Chinese",
			wantContains: []string{
				"鬼滅之刃",
				"Traditional Chinese",
				"JSON",
			},
		},
		{
			name:     "English title",
			title:    "The Matrix",
			language: "English",
			wantContains: []string{
				"The Matrix",
				"English",
				"JSON",
			},
		},
		{
			name:     "Japanese title",
			title:    "進撃の巨人",
			language: "Japanese",
			wantContains: []string{
				"進撃の巨人",
				"Japanese",
				"romaji",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildKeywordPrompt(tt.title, tt.language)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("BuildKeywordPrompt(%q, %q) missing expected content %q", tt.title, tt.language, want)
				}
			}
		})
	}
}

func TestKeywordPrompt_HasRequiredOutputFields(t *testing.T) {
	prompt := BuildKeywordPrompt("Test Title", "English")

	requiredFields := []string{
		"original",
		"simplified_chinese",
		"traditional_chinese",
		"english",
		"romaji",
		"alternative_spellings",
		"common_aliases",
	}

	for _, field := range requiredFields {
		if !strings.Contains(prompt, field) {
			t.Errorf("KeywordPrompt missing required output field: %s", field)
		}
	}
}

func TestKeywordPrompt_HasExamples(t *testing.T) {
	prompt := BuildKeywordPrompt("Test", "English")

	// Should contain at least one example from the story spec
	examples := []string{
		"鬼滅之刃",
		"Demon Slayer",
		"Kimetsu no Yaiba",
	}

	foundExample := false
	for _, example := range examples {
		if strings.Contains(prompt, example) {
			foundExample = true
			break
		}
	}

	if !foundExample {
		t.Error("KeywordPrompt should contain examples from the story specification")
	}
}

func TestKeywordGeneratorRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     KeywordGeneratorRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: KeywordGeneratorRequest{
				Title:    "鬼滅之刃",
				Language: "Traditional Chinese",
			},
			wantErr: false,
		},
		{
			name: "empty title",
			req: KeywordGeneratorRequest{
				Title:    "",
				Language: "English",
			},
			wantErr: true,
		},
		{
			name: "empty language defaults to auto-detect",
			req: KeywordGeneratorRequest{
				Title:    "Test",
				Language: "",
			},
			wantErr: false, // Language is optional, will auto-detect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeywordExamples(t *testing.T) {
	// Verify example test cases are properly structured
	if len(KeywordExamples) == 0 {
		t.Error("KeywordExamples should not be empty")
	}

	for i, ex := range KeywordExamples {
		if ex.Title == "" {
			t.Errorf("KeywordExamples[%d].Title should not be empty", i)
		}
		if ex.Expected.Original == "" {
			t.Errorf("KeywordExamples[%d].Expected.Original should not be empty", i)
		}
	}
}
