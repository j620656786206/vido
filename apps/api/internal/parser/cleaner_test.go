package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"dot separated", "The.Matrix", "The Matrix"},
		{"underscore separated", "The_Matrix", "The Matrix"},
		{"dash in word preserved", "Spider-Man", "Spider-Man"},
		{"mixed separators", "The.Matrix_Reloaded.2003", "The Matrix Reloaded 2003"},
		{"multiple dots", "The...Matrix", "The Matrix"},
		{"multiple spaces", "The    Matrix", "The Matrix"},
		{"leading/trailing spaces", "  The Matrix  ", "The Matrix"},
		{"already clean", "The Matrix", "The Matrix"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanTitle(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCleanTitleForSearch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"removes year", "The Matrix 1999", "The Matrix"},
		{"removes year in parentheses", "The Matrix (1999)", "The Matrix"},
		{"removes quality", "The Matrix 1080p", "The Matrix"},
		{"removes source", "The Matrix BluRay", "The Matrix"},
		{"removes codec", "The Matrix x264", "The Matrix"},
		{"removes release group", "The Matrix-SPARKS", "The Matrix"},
		{"removes brackets", "The Matrix [1080p]", "The Matrix"},
		{"removes parentheses with quality", "The Matrix (1080p)", "The Matrix"},
		{"complex cleanup", "The.Matrix.1080p.BluRay.x264-SPARKS", "The Matrix"},
		{"preserves title year", "2001 A Space Odyssey", "A Space Odyssey"},
		{"empty after cleanup", "1080p BluRay", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanTitleForSearch(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveReleaseGroup(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard group", "The Matrix-SPARKS", "The Matrix"},
		{"group with extension context", "The.Matrix.1080p-YTS.mkv", "The.Matrix.1080p.mkv"},
		{"no group", "The Matrix", "The Matrix"},
		{"hyphen in title preserved", "Spider-Man Far From Home-SPARKS", "Spider-Man Far From Home"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveReleaseGroup(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveBrackets(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"square brackets", "The Matrix [1080p]", "The Matrix"},
		{"parentheses", "The Matrix (1999)", "The Matrix"},
		{"multiple brackets", "[Group] The Matrix [1080p] (BluRay)", "The Matrix"},
		{"no brackets", "The Matrix", "The Matrix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveBrackets(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveQualityIndicators(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"removes 1080p", "The Matrix 1080p", "The Matrix"},
		{"removes 720p only", "The Matrix 720p Reloaded", "The Matrix Reloaded"},
		{"removes 4K", "The Matrix 4K", "The Matrix"},
		{"removes source", "The Matrix BluRay", "The Matrix"},
		{"removes codec", "The Matrix x264", "The Matrix"},
		{"removes multiple", "The Matrix 1080p BluRay x264", "The Matrix"},
		{"case insensitive", "The Matrix BLURAY HEVC", "The Matrix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveQualityIndicators(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"multiple spaces", "The    Matrix", "The Matrix"},
		{"tabs", "The\t\tMatrix", "The Matrix"},
		{"newlines", "The\n\nMatrix", "The Matrix"},
		{"mixed whitespace", "The  \t\n  Matrix", "The Matrix"},
		{"leading trailing", "  The Matrix  ", "The Matrix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWhitespace(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
