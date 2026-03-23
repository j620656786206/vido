# Story 8.5: OpenCC Integration

Status: review

## Story

As a **NAS media collector in Taiwan**,
I want **Vido to convert Simplified Chinese subtitles to Traditional Chinese (Taiwan variant) using OpenCC with the s2twp profile**,
so that **subtitles use correct Taiwan terminology and phrasing (e.g., 軟體 not 軟件, 記憶體 not 內存) for a natural reading experience**.

## Acceptance Criteria

1. **Given** a Simplified Chinese subtitle content ([]byte), **When** `Convert()` is called with profile "s2twp", **Then** it returns the content converted to Traditional Chinese with Taiwan phrase substitutions
2. **Given** the s2twp profile is used, **When** converting cross-strait terminology, **Then** mainland terms are replaced with Taiwan equivalents (e.g., 軟件→軟體, 內存→記憶體, 網絡→網路, 信息→訊息, 程序→程式)
3. **Given** a subtitle that is already Traditional Chinese, **When** `Convert()` is called, **Then** it returns the content unchanged (idempotent — no corruption of existing Traditional text)
4. **Given** the subtitle content contains mixed Chinese and non-Chinese text (English, punctuation, SRT timing codes), **When** `Convert()` is called, **Then** only Chinese text is converted; non-Chinese content is preserved exactly
5. **Given** OpenCC conversion fails (binary not found, invalid input, process crash), **When** the error occurs, **Then** `Convert()` returns the original unconverted content as fallback along with a non-nil error (graceful degradation)
6. **Given** a subtitle file of typical size (50-200KB), **When** `Convert()` is called, **Then** conversion completes within 100ms
7. **Given** the converter is initialized, **When** checking OpenCC availability, **Then** `IsAvailable() bool` returns whether OpenCC can be invoked successfully

## Tasks / Subtasks

- [x] Task 1: Evaluate and implement OpenCC invocation strategy (AC: 1, 5, 7)
  - [x] 1.1: Create `apps/api/internal/subtitle/converter.go` with `Converter` struct
  - [x] 1.2: Evaluate Go binding options: chose `github.com/longbridgeapp/opencc` (pure Go, already in go.mod) — no CGo, no external binary needed
  - [x] 1.3: Implement `NewConverter() (*Converter, error)` — initialize with "s2twp" config via `opencc.New("s2twp")`
  - [x] 1.4: N/A — using Go binding, not subprocess
  - [x] 1.5: Implement `IsAvailable() bool` — returns true if OpenCC was initialized successfully

- [x] Task 2: Implement Convert method (AC: 1, 2, 4, 5, 6)
  - [x] 2.1: Implement `Convert(content []byte, profile string) ([]byte, error)` — primary conversion method
  - [x] 2.2: Decode input bytes to UTF-8 string (strip BOM before conversion, restore after)
  - [x] 2.3: Call `cc.Convert(input)` with s2twp config (Go binding)
  - [x] 2.4: N/A — using Go binding, not subprocess
  - [x] 2.5: Re-encode result to []byte, restore BOM if original had one
  - [x] 2.6: On any error: return original `content` unchanged + the error (graceful degradation)

- [x] Task 3: Implement convenience methods (AC: 1, 3)
  - [x] 3.1: Implement `ConvertS2TWP(content []byte) ([]byte, error)` — shorthand for `Convert(content, "s2twp")`
  - [x] 3.2: Implement `NeedsConversion(language string) bool` — returns true only for "zh-Hans"; false for "zh-Hant", "zh", "und", etc.
  - [x] 3.3: Document idempotency in ConvertS2TWP godoc comment

- [x] Task 4: Handle SRT format preservation (AC: 4)
  - [x] 4.1: Verify SRT timing codes (00:01:23,456 --> 00:01:25,789) pass through unchanged — tested
  - [x] 4.2: Verify SRT sequence numbers pass through unchanged — tested
  - [x] 4.3: Verify HTML-style tags (<i>, <b>, </i>, </b>) pass through unchanged — tested
  - [x] 4.4: Verify line breaks (\r\n, \n) are preserved — tested (TestConverter_WindowsLineEndings)
  - [x] 4.5: Test cases added in TestConverter_SRTFormatPreservation and TestConverter_WindowsLineEndings

- [x] Task 5: Write unit tests (AC: all)
  - [x] 5.1: Create `apps/api/internal/subtitle/converter_test.go`
  - [x] 5.2: Test basic s2twp conversion: 简体字 → 簡體字 — TestConverter_BasicConversion
  - [x] 5.3: Test Taiwan phrase substitution: 软件→軟體, 内存→記憶體, 网络→網路, 信息→資訊, 程序→程式, 打印机→印表機, 硬盘→硬碟 — TestConverter_TaiwanPhraseSubstitution (Note: OpenCC maps 信息→資訊, not 訊息)
  - [x] 5.4: Test idempotent conversion: Traditional Chinese → identical output — TestConverter_Idempotent_TraditionalInput
  - [x] 5.5: Test mixed content: Chinese + English + numbers → only Chinese converted — TestConverter_MixedContent
  - [x] 5.6: Test SRT format preservation — TestConverter_SRTFormatPreservation
  - [x] 5.7: Test graceful degradation: unavailable converter → original content + error — TestConverter_GracefulDegradation
  - [x] 5.8: Test IsAvailable returns correct status — TestConverter_IsAvailable
  - [x] 5.9: Test NeedsConversion for various language codes — TestNeedsConversion (7 cases)
  - [x] 5.10: Test empty input → empty output (no error) — TestConverter_EmptyInput
  - [x] 5.11: Test content with BOM → BOM preserved, conversion works — TestConverter_BOMHandling
  - [x] 5.12: Verify ≥80% code coverage — 88.2%

- [x] Task 6: Write benchmark test (AC: 6)
  - [x] 6.1: Create benchmark `BenchmarkConvertS2TWP` in converter_test.go
  - [x] 6.2: Use ~100KB simplified Chinese subtitle sample
  - [x] 6.3: ~50ms per 100KB conversion (well under 100ms target)

- [x] Task 7: Build and integration verification (AC: all)
  - [x] 7.1: Run `go build ./...` — no compilation errors
  - [x] 7.2: Run `go test ./internal/subtitle/...` — all tests pass (32 subtitle + 48 providers)
  - [x] 7.3: Run `go test -bench=. ./internal/subtitle/...` — benchmark completes (~50ms/100KB)
  - [x] 7.4: Run `go vet ./internal/subtitle/...` — no vet issues
  - [x] 7.5: Pure Go binding — no CGo or subprocess dependency, no build prerequisites

## Dev Notes

### Architecture & Patterns
- The converter is used by the engine pipeline AFTER a subtitle is downloaded and detected as Simplified Chinese (detector returns "zh-Hans")
- Pipeline flow: Download → Detect (Story 8-4) → if zh-Hans → Convert (this story) → Save as .zh-Hant.srt
- Graceful degradation (AC: 5) is critical: if conversion fails, the user still gets a subtitle (unconverted simplified is better than no subtitle) — the engine should log a warning but not fail the entire operation
- Idempotence (AC: 3) is important because some subtitles may be mislabeled: a subtitle tagged as simplified might actually be traditional

### Project Structure Notes
- Converter file: `apps/api/internal/subtitle/converter.go`
- Test file: `apps/api/internal/subtitle/converter_test.go`
- The converter will be called by the engine (Story 8-6+) after detection
- Depends on detector (Story 8-4) for the `NeedsConversion()` decision

### OpenCC Profile: s2twp
- `s2t` = Simplified to Traditional (character-level only)
- `s2tw` = Simplified to Traditional (Taiwan standard characters)
- `s2twp` = Simplified to Traditional (Taiwan standard characters + Taiwan phrases) ← THIS ONE
- The "p" suffix adds phrase-level substitution, which is critical for Taiwan users:
  - 軟件 → 軟體 (software)
  - 內存 → 記憶體 (memory/RAM)
  - 網絡 → 網路 (network)
  - 信息 → 訊息 (message/information)
  - 程序 → 程式 (program)
  - 打印機 → 印表機 (printer)
  - 硬盤 → 硬碟 (hard disk)

### Go Binding Options
1. **`github.com/longbridgeapp/opencc` (recommended)**: Pure Go implementation, no CGo dependency, bundles dictionary data. Easier to deploy, no external binary needed.
2. **Subprocess `opencc` CLI**: Requires OpenCC installed on the system. More reliable conversion but adds deployment complexity.
3. Decision: prefer pure Go binding unless testing reveals accuracy gaps vs CLI tool.

### References
- PRD feature: P1-014 (OpenCC s2twp conversion)
- OpenCC profile decision: s2twp (Gate 2A)
- Extension normalization: .zh-Hant.srt (P1-013, used by engine in later stories)
- Detector: `apps/api/internal/subtitle/detector.go` (Story 8-4)

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Completion Notes List
- Used pure Go binding `github.com/longbridgeapp/opencc` (already in go.mod) — no CGo, no external binary
- Profile: s2twp (Simplified → Traditional Taiwan + phrases)
- Taiwan phrase substitutions verified: 軟件→軟體, 內存→記憶體, 網絡→網路, 信息→資訊, 程序→程式, 打印機→印表機, 硬盤→硬碟
- Note: OpenCC maps 信息→資訊 (not 訊息 as listed in epic) — this is the correct s2twp mapping
- Idempotent: Traditional Chinese passes through unchanged
- SRT format preserved: timing codes, sequence numbers, HTML tags, line endings
- BOM handling: strip before conversion, restore after
- Graceful degradation: unavailable converter returns original content + error
- NeedsConversion() only returns true for "zh-Hans"
- Benchmark: ~50ms per 100KB (target: ≤100ms)
- 16 converter tests + 1 benchmark, 88.2% coverage
- 🎨 UX Verification: SKIPPED — no UI changes

### File List
- apps/api/internal/subtitle/converter.go (NEW)
- apps/api/internal/subtitle/converter_test.go (NEW)
