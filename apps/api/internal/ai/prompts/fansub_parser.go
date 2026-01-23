// Package prompts provides prompt templates for AI-powered filename parsing.
package prompts

import (
	"fmt"
)

// FansubParseRequest represents the input for fansub filename parsing.
type FansubParseRequest struct {
	Filename string
}

// FansubParseResponse represents the expected JSON output from AI.
type FansubParseResponse struct {
	Title          string  `json:"title"`
	TitleRomanized string  `json:"title_romanized,omitempty"`
	Episode        *int    `json:"episode,omitempty"`
	Season         *int    `json:"season,omitempty"`
	Year           *int    `json:"year,omitempty"`
	Quality        string  `json:"quality,omitempty"`
	Source         string  `json:"source,omitempty"`
	Codec          string  `json:"codec,omitempty"`
	FansubGroup    string  `json:"fansub_group,omitempty"`
	Language       string  `json:"language,omitempty"`
	MediaType      string  `json:"media_type"`
	Confidence     float64 `json:"confidence"`
}

// FansubPrompt is the specialized prompt template for parsing fansub filenames.
// It's optimized for Japanese and Chinese fansub naming conventions.
const FansubPrompt = `You are a media filename parser specialized in fansub releases from Japanese and Chinese fansub groups.
Extract structured metadata from the following filename.

Filename: %s

## Common Fansub Naming Patterns

### Japanese Fansub Groups (use square brackets []):
- [Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv
- [SubsPlease] Demon Slayer - 01 (1080p) [ABCD1234].mkv
- [Commie] Steins;Gate 0 - 01 [BD 1080p AAC] [12345678].mkv
- [Erai-raws] Show Title - 01 [1080p][Multiple Subtitle].mkv

### Chinese Fansub Groups (use fullwidth brackets 【】):
- 【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4
- 【极影字幕社】★ 进击的巨人 第01话 HDTV 720P 【简体】.mp4
- 【動漫國字幕組】標題名稱 [01] [1080P] [繁體].mkv
- [ANK-Raws] 標題 - 01 (BDRip 1080p HEVC).mkv

### Pattern Components:
- Square brackets [Group] or fullwidth brackets【字幕組】contain fansub group names
- Episode indicators: "- XX", "第XX話", "第XX集", "[XX]", "EP XX", "S01E01"
- Quality: 1080p, 720p, 2160p, 4K (sometimes as dimensions like 1920x1080)
- Source: BD (Blu-ray), WEB, TV, HDTV, DVDRip, BDRip
- Codec: x264, x265, HEVC, AVC, AAC, FLAC
- Language markers: 繁體, 簡體, 繁中, 簡中, Traditional Chinese, Simplified Chinese

## Extraction Rules:
1. **Title**: Extract the main title, removing group name brackets and episode numbers
2. **Episode**: Extract episode number from various formats (第XX話, - XX, EP XX, [XX])
3. **Season**: Extract season if present (rarely explicit in fansub names)
4. **Quality**: Normalize to standard format (1080p, 720p, 2160p)
5. **Source**: Identify release source (BD, WEB, TV)
6. **Fansub Group**: Extract from brackets (ignore for search purposes)
7. **Language**: Identify subtitle/dub language if indicated
8. **Media Type**: "tv" if episode/season present, "movie" otherwise

## Output JSON Schema:
{
  "title": "extracted title (in original language, cleaned)",
  "title_romanized": "romanized title if the original is in CJK characters",
  "episode": number or null,
  "season": number or null (default 1 if episode exists but no season specified),
  "year": number or null,
  "quality": "1080p" or "720p" or "2160p" or null,
  "source": "BD" or "WEB" or "TV" or null,
  "codec": "x264" or "x265" or "HEVC" or null,
  "fansub_group": "group name without brackets" or null,
  "language": "Traditional Chinese" or "Simplified Chinese" or "Japanese" or null,
  "media_type": "movie" or "tv",
  "confidence": 0.0 to 1.0
}

## Confidence Guidelines:
- 1.0: All major fields extracted with high certainty
- 0.7-0.9: Most fields extracted, some ambiguity
- 0.5-0.7: Title and episode extracted, other fields uncertain
- 0.3-0.5: Only basic info extracted
- 0.0-0.3: Cannot parse meaningfully

Respond ONLY with valid JSON. No markdown, no explanation, no code blocks.`

// BuildFansubPrompt generates the complete prompt for a given filename.
func BuildFansubPrompt(filename string) string {
	return fmt.Sprintf(FansubPrompt, filename)
}

// FansubExamples contains test cases for validating the prompt.
// Used for testing and prompt refinement.
var FansubExamples = []struct {
	Filename string
	Expected FansubParseResponse
}{
	{
		Filename: "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
		Expected: FansubParseResponse{
			Title:       "Kimetsu no Yaiba",
			Episode:     intPtr(26),
			Season:      intPtr(1),
			Quality:     "1080p",
			Source:      "BD",
			Codec:       "x264",
			FansubGroup: "Leopard-Raws",
			MediaType:   "tv",
			Confidence:  0.9,
		},
	},
	{
		Filename: "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4",
		Expected: FansubParseResponse{
			Title:       "我的英雄學院",
			Episode:     intPtr(1),
			Season:      intPtr(1),
			Quality:     "1080p",
			Language:    "Traditional Chinese",
			FansubGroup: "幻櫻字幕組",
			MediaType:   "tv",
			Confidence:  0.9,
		},
	},
	{
		Filename: "[SubsPlease] Demon Slayer - 01 (1080p) [ABCD1234].mkv",
		Expected: FansubParseResponse{
			Title:       "Demon Slayer",
			Episode:     intPtr(1),
			Season:      intPtr(1),
			Quality:     "1080p",
			FansubGroup: "SubsPlease",
			MediaType:   "tv",
			Confidence:  0.9,
		},
	},
	{
		Filename: "【极影字幕社】★ 进击的巨人 第01话 HDTV 720P 【简体】.mp4",
		Expected: FansubParseResponse{
			Title:       "进击的巨人",
			Episode:     intPtr(1),
			Season:      intPtr(1),
			Quality:     "720p",
			Source:      "TV",
			Language:    "Simplified Chinese",
			FansubGroup: "极影字幕社",
			MediaType:   "tv",
			Confidence:  0.9,
		},
	},
	{
		Filename: "[Commie] Steins;Gate 0 - 01 [BD 1080p AAC] [12345678].mkv",
		Expected: FansubParseResponse{
			Title:       "Steins;Gate 0",
			Episode:     intPtr(1),
			Season:      intPtr(1),
			Quality:     "1080p",
			Source:      "BD",
			Codec:       "AAC",
			FansubGroup: "Commie",
			MediaType:   "tv",
			Confidence:  0.9,
		},
	},
}

// intPtr is a helper function to create a pointer to an int.
func intPtr(i int) *int {
	return &i
}
