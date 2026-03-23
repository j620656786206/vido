# Story 8.6: Subtitle Scoring and Ranking

Status: ready-for-dev

## Story

As a **media collector**,
I want **subtitle search results to be scored and ranked by relevance**,
so that **the best-matching Traditional Chinese subtitle is automatically prioritized for download**.

## Acceptance Criteria

1. **Given** a list of SubtitleResult from providers,
   **When** the scorer processes them,
   **Then** each result receives a composite score (0.0–1.0) calculated from:
   - Language match: 40% weight
   - Resolution match: 20% weight
   - Source trust: 20% weight
   - Group match: 10% weight
   - Downloads: 10% weight

2. **Given** a subtitle result with language `zh-Hant` or `zh-TW`,
   **When** the language factor is calculated,
   **Then** it scores 1.0;
   **And** `zh-Hans` or `zh-CN` scores 0.6 (convertible via OpenCC);
   **And** any other language scores 0.0.

3. **Given** media with resolution `1080p` and a subtitle tagged `1080p`,
   **When** the resolution factor is calculated,
   **Then** it scores 1.0;
   **And** if untagged, it scores 0.5;
   **And** if mismatched, it scores 0.2.

4. **Given** subtitle results from different providers,
   **When** the source trust factor is calculated,
   **Then** Assrt scores 0.8, OpenSubtitles scores 0.7, Zimuku scores 0.6;
   **And** trust values are configurable via a map.

5. **Given** subtitle results with fansub group names,
   **When** the group match factor is calculated,
   **Then** a result from a known fansub group scores 1.0;
   **And** an unknown group scores 0.3;
   **And** the known group list is hardcoded with an initial set.

6. **Given** subtitle results with download counts,
   **When** the downloads factor is calculated,
   **Then** counts are normalized to 0.0–1.0 using the max download count in the result set;
   **And** if only one result exists, it scores 1.0.

7. **Given** scored results,
   **When** the scorer returns,
   **Then** results are sorted descending by composite score;
   **And** ties are broken by download count descending.

## Tasks / Subtasks

### Task 1: Define Scoring Types (AC: #1, #7)
- [ ] 1.1 Create `apps/api/internal/subtitle/scorer.go`
- [ ] 1.2 Define `ScoredResult` struct: embeds `SubtitleResult`, adds `Score float64`, `ScoreBreakdown ScoreBreakdown`
- [ ] 1.3 Define `ScoreBreakdown` struct with fields: `Language`, `Resolution`, `SourceTrust`, `Group`, `Downloads` (all `float64`)
- [ ] 1.4 Define `ScorerConfig` struct with configurable weights and provider trust map
- [ ] 1.5 Define `NewDefaultScorerConfig()` returning default weights (0.4, 0.2, 0.2, 0.1, 0.1)

### Task 2: Implement Language Factor (AC: #2)
- [ ] 2.1 Create `scorerLanguage(lang string) float64` method
- [ ] 2.2 Map zh-Hant, zh-TW → 1.0; zh-Hans, zh-CN → 0.6; all other → 0.0
- [ ] 2.3 Handle case-insensitive language tag comparison

### Task 3: Implement Resolution Factor (AC: #3)
- [ ] 3.1 Create `scoreResolution(mediaRes, subtitleRes string) float64` method
- [ ] 3.2 Exact match → 1.0, untagged → 0.5, mismatch → 0.2
- [ ] 3.3 Normalize resolution strings (e.g., "1080p", "1080", "FHD" → "1080p")

### Task 4: Implement Source Trust Factor (AC: #4)
- [ ] 4.1 Create `scoreSourceTrust(providerName string) float64` method
- [ ] 4.2 Default trust map: `{"assrt": 0.8, "opensubtitles": 0.7, "zimuku": 0.6}`
- [ ] 4.3 Unknown provider defaults to 0.5

### Task 5: Implement Group Match Factor (AC: #5)
- [ ] 5.1 Create `scoreGroup(groupName string) float64` method
- [ ] 5.2 Define `knownFansubGroups` set with initial entries (e.g., "CHD", "CMCT", "MySiLU", "FLTth", "HDChina")
- [ ] 5.3 Known group → 1.0, unknown non-empty → 0.3, empty → 0.0

### Task 6: Implement Downloads Factor (AC: #6)
- [ ] 6.1 Create `scoreDownloads(count int, maxCount int) float64` method
- [ ] 6.2 Normalize: `float64(count) / float64(maxCount)`
- [ ] 6.3 Handle maxCount == 0 edge case (return 0.0)

### Task 7: Implement Score Aggregation and Sorting (AC: #1, #7)
- [ ] 7.1 Create `Scorer` struct with `config ScorerConfig`
- [ ] 7.2 Implement `Score(results []SubtitleResult, mediaResolution string) []ScoredResult`
- [ ] 7.3 Calculate composite: `lang*0.4 + res*0.2 + trust*0.2 + group*0.1 + dl*0.1`
- [ ] 7.4 Sort descending by Score, then by DownloadCount for ties
- [ ] 7.5 Populate `ScoreBreakdown` on each result for debugging/UI display

### Task 8: Write Tests (AC: #1–#7)
- [ ] 8.1 Create `apps/api/internal/subtitle/scorer_test.go`
- [ ] 8.2 Test language factor: zh-Hant=1.0, zh-Hans=0.6, en=0.0, empty=0.0
- [ ] 8.3 Test resolution factor: exact match, untagged, mismatch
- [ ] 8.4 Test source trust: known providers, unknown provider
- [ ] 8.5 Test group match: known group, unknown group, empty
- [ ] 8.6 Test downloads normalization: various counts, single result, zero max
- [ ] 8.7 Test composite scoring with full result set
- [ ] 8.8 Test sort order: descending score, tie-breaking by downloads
- [ ] 8.9 Test empty input returns empty slice
- [ ] 8.10 Test custom ScorerConfig overrides defaults
- [ ] 8.11 Ensure >80% coverage on scorer.go

## Dev Notes

### Architecture & Patterns
- Pure scoring logic with no side effects — takes `[]SubtitleResult` in, returns `[]ScoredResult` out
- `Scorer` is instantiated with config, making it testable with different weight configurations
- `ScoreBreakdown` enables transparency — the UI (Story 8-8) can show why a result ranked high
- Language factor intentionally scores zh-Hans > 0 because Story 8-5 provides OpenCC conversion

### Project Structure Notes
- File: `apps/api/internal/subtitle/scorer.go`
- Test: `apps/api/internal/subtitle/scorer_test.go`
- Depends on `SubtitleResult` type which will be defined in the subtitle package by Stories 8-1/8-2/8-3
- Consumed by `engine.go` (Story 8-7) and manual search handler (Story 8-8)

### References
- PRD: P1-016 (Subtitle scoring and ranking)
- Gate 2A decision: Language 40% + Resolution 20% + Source trust 20% + Group 10% + Downloads 10%
- Stories 8-1/8-2/8-3: SubtitleProvider interface and SubtitleResult type
- Story 8-7: Engine consumes `Scorer.Score()` output
- Story 8-8: UI displays `ScoreBreakdown` in results table

## Dev Agent Record

### Agent Model Used
### Completion Notes List
### File List
