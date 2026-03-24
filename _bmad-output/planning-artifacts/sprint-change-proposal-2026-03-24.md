# Sprint Change Proposal — CN Content Subtitle Conversion Policy

**Date:** 2026-03-24
**Epic:** Epic 8 — Subtitle Engine
**Scope:** Minor (Direct Adjustment)
**Status:** Approved by Alexyu

---

## 1. Issue Summary

During UX design review for Story 8-8/8-9, a product requirement gap was identified: the current subtitle pipeline unconditionally converts all Simplified Chinese subtitles to Traditional Chinese via OpenCC s2twp. This is incorrect for Chinese mainland content (大陸劇/電影) where the dialogue itself uses mainland Chinese expressions — converting the subtitles to Traditional Chinese creates a mismatch between audio and text.

**Example:** "老铁没毛病" → "老鐵沒毛病" — the Traditional Chinese characters don't match the mainland colloquial expression heard in the audio.

## 2. Impact Analysis

### Epic Impact
- Epic 8 can still be completed as planned
- No new Epics needed
- No priority changes

### Story Impact

| Story | Status | Change Required |
|-------|--------|----------------|
| 8-5 OpenCC Integration | done | Add `ConversionPolicy` type + skip logic to `convertIfNeeded()` |
| 8-7 Auto-Download Service | done | `Process()` accepts `productionCountry` param, passes policy |
| 8-8 Manual Subtitle Search UI | review | Handler passes production_countries; Dialog adds conversion toggle |
| 8-9 Batch Subtitle Processing | ready-for-dev | BatchProcessor fetches production_countries per media item |

### Artifact Impact
- `project-context.md` Rule 9 — Update conversion rules
- `ux-design.pen` — Design note already added for conversion toggle

## 3. Recommended Approach

**Direct Adjustment** — Modify existing stories within current Epic 8 structure.

- **Effort:** Low — 3-4 files to modify, logic is straightforward
- **Risk:** Low — conversion skip is additive, existing behavior preserved for non-CN content
- **Timeline:** No impact — can be integrated into 8-8 code review fixes and 8-9 development

## 4. Detailed Change Proposals

### 4.1 converter.go — Add ConversionPolicy type

```
NEW:
type ConversionPolicy string
const (
    ConvertAlways    ConversionPolicy = "always"     // default: always s2twp
    ConvertNever     ConversionPolicy = "never"       // skip conversion entirely
    ConvertAuto      ConversionPolicy = "auto"        // skip if production_country is CN
)
```

### 4.2 engine.go — convertIfNeeded() add policy check

```
OLD:
func (e *Engine) convertIfNeeded(data []byte) ([]byte, string) {
    lang := e.detector.Detect(data)
    if lang == LangSimplified || lang == LangAmbiguous {
        converted, err := e.converter.ConvertS2TWP(data)
        ...

NEW:
func (e *Engine) convertIfNeeded(data []byte, policy ConversionPolicy) ([]byte, string) {
    if policy == ConvertNever {
        lang := e.detector.Detect(data)
        return data, langToTag(lang)  // preserve original
    }
    lang := e.detector.Detect(data)
    if lang == LangSimplified || lang == LangAmbiguous {
        converted, err := e.converter.ConvertS2TWP(data)
        ...
```

### 4.3 engine.go — Process() derive policy from production country

```
OLD:
func (e *Engine) Process(ctx, mediaID, mediaType, mediaFilePath, query, mediaResolution)

NEW:
func (e *Engine) Process(ctx, mediaID, mediaType, mediaFilePath, query, mediaResolution, productionCountry string)
// if productionCountry == "CN" && global setting == "auto" → policy = ConvertNever
```

### 4.4 subtitle_handler.go — Pass production country from TMDb data

```
NEW:
// In search/download handler, look up media's production_countries from DB/TMDb
// Pass to Engine.Process() as productionCountry parameter
```

### 4.5 SubtitleSearchDialog.tsx — Add conversion toggle

```
NEW:
// Toggle: "繁體轉換" (Traditional Chinese conversion)
// Default: ON for non-CN content, OFF for CN content
// User can override per-search
```

### 4.6 project-context.md Rule 9 — Update conversion rules

```
OLD:
- OpenCC conversion direction: s2twp (Simplified → Traditional with Taiwan phrases)

NEW:
- OpenCC conversion direction: s2twp (Simplified → Traditional with Taiwan phrases)
- CN content policy: Skip conversion when production_countries contains "CN" (mainland content keeps simplified subtitles)
- Conversion toggle: Users can override per-search or set global preference
- Edge cases: Co-productions default to convert (conservative); already-traditional subtitles pass through unchanged
```

## 5. Implementation Handoff

| Agent | Responsibility |
|-------|---------------|
| **Bob (SM)** | Update Story 8-8/8-9 ACs and Tasks with conversion policy requirements |
| **Amelia (Dev)** | Implement converter policy + engine changes + UI toggle |
| **Murat (QA)** | Verify CN skip, non-CN convert, toggle override, co-production handling |
| **Sally (UX)** | Verify Dialog toggle matches design spec |

**Success Criteria:**
- CN content (production_countries contains "CN") → subtitles remain Simplified Chinese
- Non-CN content → subtitles converted to Traditional Chinese (existing behavior)
- User can override via toggle in search dialog
- All existing converter tests still pass
