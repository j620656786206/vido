package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// BenchmarkMovieParser benchmarks movie filename parsing.
func BenchmarkMovieParser(b *testing.B) {
	parser := NewMovieParser()
	filename := "The.Matrix.1999.1080p.BluRay.x264-SPARKS.mkv"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(filename)
	}
}

// BenchmarkTVParser benchmarks TV show filename parsing.
func BenchmarkTVParser(b *testing.B) {
	parser := NewTVParser()
	filename := "Breaking.Bad.S01E05.720p.BluRay.x264-DEMAND.mkv"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(filename)
	}
}

// BenchmarkQualityDetection benchmarks quality detection.
func BenchmarkQualityDetection(b *testing.B) {
	filename := "Movie.2020.1080p.BluRay.x264-GROUP.mkv"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectQuality(filename)
		DetectSource(filename)
		DetectVideoCodec(filename)
		DetectAudioCodec(filename)
		DetectReleaseGroup(filename)
	}
}

// BenchmarkCleanTitle benchmarks title cleaning.
func BenchmarkCleanTitle(b *testing.B) {
	title := "The.Matrix.1999.1080p.BluRay.x264-SPARKS"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanTitle(title)
	}
}

// BenchmarkCleanTitleForSearch benchmarks search title cleaning.
func BenchmarkCleanTitleForSearch(b *testing.B) {
	title := "The.Matrix.1999.1080p.BluRay.x264-SPARKS"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanTitleForSearch(title)
	}
}

// TestParsePerformance_Under100ms verifies that parsing completes within 100ms.
// This is a critical NFR requirement (NFR-P13).
func TestParsePerformance_Under100ms(t *testing.T) {
	movieParser := NewMovieParser()
	tvParser := NewTVParser()

	testCases := []string{
		// Movies
		"The.Matrix.1999.1080p.BluRay.x264-SPARKS.mkv",
		"Inception.2010.2160p.UHD.BluRay.HEVC-YTS.mkv",
		"Parasite.2019.720p.WEB-DL.AAC-RARBG.mkv",
		// TV Shows
		"Breaking.Bad.S01E05.720p.BluRay.x264-DEMAND.mkv",
		"Game.of.Thrones.S08E06.1080p.WEB-DL.x265-GROUP.mkv",
		"The.Mandalorian.S02E01.720p.WEBRip.mkv",
		// Edge cases
		"2001.A.Space.Odyssey.1968.1080p.BluRay.mkv",
		"Blade.Runner.2049.2017.2160p.UHD.BluRay.mkv",
	}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			start := time.Now()

			// Try both parsers
			movieParser.Parse(filename)
			tvParser.Parse(filename)

			duration := time.Since(start)

			// Must complete within 100ms per NFR-P13
			assert.Less(t, duration.Milliseconds(), int64(100),
				"Parsing took %v, expected < 100ms", duration)
		})
	}
}

// TestParseBatchPerformance tests performance with many files.
func TestParseBatchPerformance(t *testing.T) {
	movieParser := NewMovieParser()
	tvParser := NewTVParser()

	// Generate 1000 filenames with alternating movie/TV patterns
	filenames := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			filenames[i] = "Movie.Title." + string(rune('A'+i%26)) + ".1999.1080p.BluRay.mkv"
		} else {
			filenames[i] = "TV.Show.S01E" + string(rune('0'+i%10)) + "1.720p.WEB-DL.mkv"
		}
	}

	start := time.Now()

	for _, filename := range filenames {
		if movieParser.CanParse(filename) {
			movieParser.Parse(filename)
		} else if tvParser.CanParse(filename) {
			tvParser.Parse(filename)
		}
	}

	duration := time.Since(start)

	// 1000 files should complete in under 1 second (1ms per file average)
	assert.Less(t, duration.Seconds(), 1.0,
		"Batch parsing 1000 files took %v, expected < 1s", duration)

	t.Logf("Parsed 1000 files in %v (avg: %v per file)", duration, duration/1000)
}
