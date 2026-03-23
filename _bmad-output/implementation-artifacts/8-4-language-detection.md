# Story 8.4: 簡繁 Language Detection

Status: ready-for-dev

## Story

As a **NAS media collector**,
I want **Vido to detect whether a subtitle file contains Simplified Chinese (簡體) or Traditional Chinese (繁體) by analyzing the file content**,
so that **only verified Traditional Chinese subtitles are served to my media player, eliminating false-positive simplified subtitles**.

## Acceptance Criteria

1. **Given** a subtitle file containing >70% traditional-unique characters (out of all uniquely-classifiable characters), **When** `Detect()` is called, **Then** it returns `"zh-Hant"` (Traditional Chinese)
2. **Given** a subtitle file containing ≤30% traditional-unique characters, **When** `Detect()` is called, **Then** it returns `"zh-Hans"` (Simplified Chinese)
3. **Given** a subtitle file where the traditional-unique ratio is between 30% and 70%, **When** `Detect()` is called, **Then** it returns `"zh"` (ambiguous/mixed)
4. **Given** a subtitle file with no CJK characters (e.g., English-only), **When** `Detect()` is called, **Then** it returns `"und"` (undetermined) without error
5. **Given** the detector uses Unicode unique character set analysis, **When** classifying characters, **Then** it references ~2000 simplified-only codepoints and ~2000 traditional-only codepoints (characters that exist in ONLY one variant)
6. **Given** a typical subtitle file (50-200KB), **When** `Detect()` is called, **Then** it completes in ≤5ms (benchmark verified)
7. **Given** a corpus of known Traditional Chinese subtitles, **When** running detection on the corpus, **Then** accuracy is ≥99% (0% false-positive rate for simplified being classified as traditional)
8. **Given** subtitle content as `[]byte`, **When** `Detect()` is called, **Then** the detection is based ONLY on content analysis — filename, metadata, or other heuristics are NOT used

## Tasks / Subtasks

- [ ] Task 1: Build Unicode character sets (AC: 5)
  - [ ] 1.1: Create `apps/api/internal/subtitle/detector.go` with package-level `var` declarations for character sets
  - [ ] 1.2: Define `simplifiedOnly` set: ~2000 Unicode codepoints that exist only in Simplified Chinese (not in Traditional) — sourced from Unicode Unihan database analysis
  - [ ] 1.3: Define `traditionalOnly` set: ~2000 Unicode codepoints that exist only in Traditional Chinese (not in Simplified) — sourced from Unicode Unihan database analysis
  - [ ] 1.4: Use `map[rune]struct{}` for O(1) lookup performance
  - [ ] 1.5: Add comments explaining the source and methodology for character set derivation

- [ ] Task 2: Implement DetectionResult type (AC: 1, 2, 3, 4)
  - [ ] 2.1: Define `DetectionResult` struct with fields: Language (string), TraditionalRatio (float64), SimplifiedCount (int), TraditionalCount (int), TotalCJK (int), Confidence (float64)
  - [ ] 2.2: Define language constants: `LangTraditional = "zh-Hant"`, `LangSimplified = "zh-Hans"`, `LangAmbiguous = "zh"`, `LangUndetermined = "und"`

- [ ] Task 3: Implement Detect function (AC: 1, 2, 3, 4, 5, 6, 8)
  - [ ] 3.1: Implement `Detect(content []byte) DetectionResult` — accepts raw subtitle content
  - [ ] 3.2: Decode content to UTF-8 string (handle BOM if present)
  - [ ] 3.3: Iterate over each rune: count occurrences in `simplifiedOnly` and `traditionalOnly` sets
  - [ ] 3.4: Calculate traditional ratio: `traditionalCount / (traditionalCount + simplifiedCount)`
  - [ ] 3.5: Apply threshold: >70% → zh-Hant, <30% → zh-Hans, 30-70% → zh, no CJK → und
  - [ ] 3.6: Return `DetectionResult` with all metrics populated
  - [ ] 3.7: Handle edge case: if `simplifiedCount + traditionalCount == 0` but CJK exists (all shared characters), return "zh" with note

- [ ] Task 4: Implement batch detection helper (AC: 6)
  - [ ] 4.1: Implement `DetectFromFile(filePath string) (DetectionResult, error)` — reads file and calls Detect
  - [ ] 4.2: Use buffered I/O for large files
  - [ ] 4.3: Limit reading to first 100KB for very large files (sufficient for detection)

- [ ] Task 5: Write unit tests (AC: 1, 2, 3, 4, 7, 8)
  - [ ] 5.1: Create `apps/api/internal/subtitle/detector_test.go`
  - [ ] 5.2: Test pure Traditional Chinese content → returns "zh-Hant" (ratio >0.95)
  - [ ] 5.3: Test pure Simplified Chinese content → returns "zh-Hans" (ratio <0.05)
  - [ ] 5.4: Test mixed content (50/50) → returns "zh" (ambiguous)
  - [ ] 5.5: Test content at boundaries: 69% traditional → "zh", 71% traditional → "zh-Hant"
  - [ ] 5.6: Test English-only content → returns "und"
  - [ ] 5.7: Test empty content → returns "und"
  - [ ] 5.8: Test content with BOM marker → correctly decoded
  - [ ] 5.9: Test that shared CJK characters (common to both) are ignored in ratio calculation
  - [ ] 5.10: Create test fixtures with known Traditional and Simplified subtitle excerpts
  - [ ] 5.11: Verify ≥80% code coverage

- [ ] Task 6: Write benchmark tests (AC: 6)
  - [ ] 6.1: Create benchmark in `detector_test.go`: `BenchmarkDetect`
  - [ ] 6.2: Use a realistic 100KB subtitle content sample
  - [ ] 6.3: Verify ≤5ms per detection (assert in CI-friendly way: log warning if exceeded)

- [ ] Task 7: Build verification (AC: all)
  - [ ] 7.1: Run `go build ./...` — verify no compilation errors
  - [ ] 7.2: Run `go test ./internal/subtitle/...` — verify all tests pass
  - [ ] 7.3: Run `go test -bench=. ./internal/subtitle/...` — verify benchmark completes
  - [ ] 7.4: Run `go vet ./internal/subtitle/...` — verify no vet issues

## Dev Notes

### Architecture & Patterns
- The detector is a pure function with no external dependencies — it takes bytes in and returns a result. This makes it highly testable and fast
- Content-only analysis (AC: 8) is a firm requirement: filename-based detection is unreliable (fansub groups often mislabel)
- The character sets are derived from Unicode Unihan database: characters that have kSimplifiedVariant but no kTraditionalVariant are simplified-only, and vice versa
- The 70% threshold was chosen to handle real-world subtitles that may contain a small number of cross-variant characters (e.g., proper nouns, technical terms)

### Project Structure Notes
- Detector file: `apps/api/internal/subtitle/detector.go`
- Test file: `apps/api/internal/subtitle/detector_test.go`
- The detector will be called by the engine (Story 8-6+) after downloading a subtitle and before scoring
- Character sets should be package-level variables (initialized once, read-only) for performance

### Unicode Unique Character Set Methodology
- Simplified-only characters: codepoints in the CJK Unified Ideographs block that have a `kTraditionalVariant` mapping in Unihan (meaning they ARE the simplified form of a different traditional character)
- Traditional-only characters: codepoints that have a `kSimplifiedVariant` mapping (meaning they ARE the traditional form of a different simplified character)
- Characters without either mapping are shared between both systems and are NOT counted in the ratio
- Common examples: 簡(trad-only: 簡, simp: 简), 體(trad-only: 體, simp: 体), 軟(trad-only: 軟, simp: 软)

### Performance Considerations
- Use `map[rune]struct{}` for character set lookup (O(1) per character)
- Single pass through content: iterate runes once, count both sets simultaneously
- For very large files, cap analysis at first 100KB — language distribution is consistent throughout a subtitle file

### References
- PRD feature: P1-012 (簡繁 detection via content analysis)
- Detection threshold decision: >70% traditional-unique = zh-Hant (Gate 2A)
- Will be used by: scorer (Story 8-6+) for language scoring factor (40% weight)

## Dev Agent Record

### Agent Model Used
### Completion Notes List
### File List
