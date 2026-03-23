# Story 8.5: OpenCC Integration

Status: ready-for-dev

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

- [ ] Task 1: Evaluate and implement OpenCC invocation strategy (AC: 1, 5, 7)
  - [ ] 1.1: Create `apps/api/internal/subtitle/converter.go` with `Converter` struct
  - [ ] 1.2: Evaluate Go binding options: `github.com/longbridgeapp/opencc` (pure Go) vs subprocess `opencc` CLI — prefer pure Go binding for deployment simplicity
  - [ ] 1.3: If using Go binding: implement `NewConverter() (*Converter, error)` — initialize with "s2twp" config
  - [ ] 1.4: If using subprocess: implement `NewConverter() (*Converter, error)` — verify `opencc` binary exists in PATH
  - [ ] 1.5: Implement `IsAvailable() bool` — returns true if OpenCC can perform conversions

- [ ] Task 2: Implement Convert method (AC: 1, 2, 4, 5, 6)
  - [ ] 2.1: Implement `Convert(content []byte, profile string) ([]byte, error)` — primary conversion method
  - [ ] 2.2: Decode input bytes to UTF-8 string (handle BOM if present)
  - [ ] 2.3: If using Go binding: call `opencc.Convert(input)` with s2twp config
  - [ ] 2.4: If using subprocess: pipe content to `opencc -c s2twp` via stdin/stdout
  - [ ] 2.5: Re-encode result to []byte preserving original encoding characteristics
  - [ ] 2.6: On any error: return original `content` unchanged + the error (AC: 5 graceful degradation)

- [ ] Task 3: Implement convenience methods (AC: 1, 3)
  - [ ] 3.1: Implement `ConvertS2TWP(content []byte) ([]byte, error)` — shorthand for `Convert(content, "s2twp")`
  - [ ] 3.2: Implement `NeedsConversion(language string) bool` — returns true only for "zh-Hans" (simplified); returns false for "zh-Hant", "zh", "und", or any non-Chinese language
  - [ ] 3.3: Document that calling Convert on already-traditional text is safe (idempotent)

- [ ] Task 4: Handle SRT format preservation (AC: 4)
  - [ ] 4.1: Verify that SRT timing codes (00:01:23,456 --> 00:01:25,789) pass through unchanged
  - [ ] 4.2: Verify that SRT sequence numbers pass through unchanged
  - [ ] 4.3: Verify that HTML-style tags in SRT (<i>, <b>, <font>) pass through unchanged
  - [ ] 4.4: Verify that line breaks (\r\n, \n) are preserved
  - [ ] 4.5: Add specific test cases for each of these format elements

- [ ] Task 5: Write unit tests (AC: all)
  - [ ] 5.1: Create `apps/api/internal/subtitle/converter_test.go`
  - [ ] 5.2: Test basic s2twp conversion: 简体字 → 簡體字 (character conversion)
  - [ ] 5.3: Test Taiwan phrase substitution: 软件 → 軟體, 内存 → 記憶體, 网络 → 網路, 信息 → 訊息, 程序 → 程式
  - [ ] 5.4: Test idempotent conversion: Traditional Chinese input → identical output
  - [ ] 5.5: Test mixed content: Chinese + English + numbers → only Chinese converted
  - [ ] 5.6: Test SRT format preservation: timing codes, sequence numbers, tags intact
  - [ ] 5.7: Test graceful degradation: simulate failure → original content returned with error
  - [ ] 5.8: Test IsAvailable returns correct status
  - [ ] 5.9: Test NeedsConversion for various language codes
  - [ ] 5.10: Test empty input → empty output (no error)
  - [ ] 5.11: Test content with BOM → conversion works correctly
  - [ ] 5.12: Verify ≥80% code coverage

- [ ] Task 6: Write benchmark test (AC: 6)
  - [ ] 6.1: Create benchmark in `converter_test.go`: `BenchmarkConvertS2TWP`
  - [ ] 6.2: Use a realistic 100KB Simplified Chinese subtitle sample
  - [ ] 6.3: Verify ≤100ms per conversion

- [ ] Task 7: Build and integration verification (AC: all)
  - [ ] 7.1: Run `go build ./...` — verify no compilation errors
  - [ ] 7.2: Run `go test ./internal/subtitle/...` — verify all tests pass
  - [ ] 7.3: Run `go test -bench=. ./internal/subtitle/...` — verify benchmark completes
  - [ ] 7.4: Run `go vet ./internal/subtitle/...` — verify no vet issues
  - [ ] 7.5: If using CGo or subprocess, document any build/deployment prerequisites in dev notes

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
### Completion Notes List
### File List
