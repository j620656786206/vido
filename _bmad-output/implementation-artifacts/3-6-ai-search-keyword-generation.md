# Story 3.6: AI Search Keyword Generation

Status: done

## Story

As a **media collector**,
I want the **AI to generate alternative search keywords**,
So that **hard-to-find titles can be located through different search terms**.

## Acceptance Criteria

1. **AC1: Generate Alternative Keywords**
   - Given initial search fails on all sources
   - When AI keyword generation is triggered
   - Then AI analyzes the filename and generates:
     - Original title
     - English translation
     - Japanese romaji (if applicable)
     - Alternative spellings

2. **AC2: Retry with Alternatives**
   - Given alternative keywords are generated
   - When retrying metadata sources
   - Then each keyword variant is tried
   - And the first successful match is used

3. **AC3: Specific Example**
   - Given the filename is `鬼滅之刃.S01E26.mkv`
   - When TMDb search "鬼滅之刃" fails
   - Then AI generates alternatives:
     - "鬼灭之刃" (Simplified Chinese)
     - "Demon Slayer" (English)
     - "Kimetsu no Yaiba" (Romaji)

4. **AC4: Caching Generated Keywords**
   - Given AI generates alternative keywords
   - When the same title is searched again
   - Then cached keywords are used (30-day cache from Story 3.1)

## Tasks / Subtasks

- [x] Task 1: Design AI Prompt for Keyword Generation (AC: 1, 3)
  - [x] 1.1: Create prompt template for generating alternative search terms
  - [x] 1.2: Include examples for various languages (Chinese, Japanese, Korean, English)
  - [x] 1.3: Define JSON output schema for keyword variants
  - [x] 1.4: Store prompt in `/apps/api/internal/ai/prompts/keyword_generator.go`
  - [x] 1.5: Write prompt tests

- [x] Task 2: Extend AI Service with Keyword Generation (AC: 1, 4)
  - [x] 2.1: Add `GenerateKeywords()` method to AIService interface
  - [x] 2.2: Implement keyword generation using AI provider
  - [x] 2.3: Parse AI response to `KeywordVariants` struct
  - [x] 2.4: Leverage existing 30-day cache from Story 3.1
  - [x] 2.5: Write unit tests with mocked AI responses

- [x] Task 3: Create Keyword Variants Types (AC: 1, 2)
  - [x] 3.1: Create `/apps/api/internal/ai/keywords.go`
  - [x] 3.2: Define `KeywordVariants` struct
  - [x] 3.3: Define priority order for keyword retry
  - [x] 3.4: Write type tests

- [x] Task 4: Integrate with Fallback Orchestrator (AC: 2)
  - [x] 4.1: Update orchestrator to accept AI keyword generator
  - [x] 4.2: Add "AI keyword retry" phase after primary sources fail
  - [x] 4.3: Iterate through keyword variants with each provider
  - [x] 4.4: Stop on first successful match
  - [x] 4.5: Write integration tests

- [x] Task 5: Create Keyword Generation Service (AC: 1, 2, 3, 4)
  - [x] 5.1: Create `/apps/api/internal/services/keyword_service.go`
  - [x] 5.2: Define `KeywordServiceInterface` in services package
  - [x] 5.3: Implement `GetAlternativeKeywords()` method
  - [x] 5.4: Wire up in `main.go`
  - [x] 5.5: Write service tests

- [x] Task 6: Language Detection and Conversion (AC: 1, 3)
  - [x] 6.1: Detect source language (Chinese, Japanese, Korean, English)
  - [x] 6.2: Generate Simplified ↔ Traditional Chinese variants
  - [x] 6.3: Generate romaji for Japanese titles
  - [x] 6.4: Generate common English translations
  - [x] 6.5: Write conversion tests

## Dev Notes

### Architecture Requirements

**FR19: AI re-parse and generate alternative keywords**
- Layer 4 of the fallback architecture
- Increases metadata coverage from 98% to 99%+

### Keyword Generation Flow

```
Original Search: "鬼滅之刃" → TMDb ❌ → Douban ❌ → Wikipedia ❌
                              ↓
                    AI Keyword Generation
                              ↓
            ┌─────────────────┼─────────────────┐
            ↓                 ↓                 ↓
       "鬼灭之刃"      "Demon Slayer"    "Kimetsu no Yaiba"
       (Simplified)      (English)         (Romaji)
            ↓                 ↓                 ↓
         TMDb ❌           TMDb ✅           (skip)
                              ↓
                        Return Result
```

### AI Prompt Design

**Recommended Prompt Structure:**
```
You are a media title translator and search keyword generator.

Given a media title, generate alternative search keywords that could help find the same media on different databases.

Title: {{title}}
Detected Language: {{language}}

Generate the following variations in JSON format:
{
  "original": "original title as-is",
  "simplified_chinese": "Simplified Chinese version (if applicable)",
  "traditional_chinese": "Traditional Chinese version (if applicable)",
  "english": "English title or translation",
  "romaji": "Japanese romaji (if Japanese title)",
  "pinyin": "Pinyin (if Chinese title)",
  "alternative_spellings": ["variant1", "variant2"],
  "common_aliases": ["alias1", "alias2"]
}

Rules:
1. Only include fields that are applicable
2. For anime/manga, include both Japanese and localized titles
3. For Chinese content, include both Simplified and Traditional
4. Include common fan translations or unofficial names if known
5. For Korean content, include romanization

Example:
Input: "鬼滅之刃"
Output: {
  "original": "鬼滅之刃",
  "simplified_chinese": "鬼灭之刃",
  "traditional_chinese": "鬼滅之刃",
  "english": "Demon Slayer",
  "romaji": "Kimetsu no Yaiba",
  "alternative_spellings": ["Demon Slayer: Kimetsu no Yaiba"],
  "common_aliases": ["鬼滅", "Demon Slayer"]
}
```

### Keyword Variants Types

```go
// KeywordVariants holds alternative search terms
type KeywordVariants struct {
    Original            string   `json:"original"`
    SimplifiedChinese   string   `json:"simplified_chinese,omitempty"`
    TraditionalChinese  string   `json:"traditional_chinese,omitempty"`
    English             string   `json:"english,omitempty"`
    Romaji              string   `json:"romaji,omitempty"`
    Pinyin              string   `json:"pinyin,omitempty"`
    AlternativeSpellings []string `json:"alternative_spellings,omitempty"`
    CommonAliases       []string `json:"common_aliases,omitempty"`
}

// GetPrioritizedList returns keywords in search priority order
func (k *KeywordVariants) GetPrioritizedList() []string {
    var keywords []string

    // Priority order for search
    if k.English != "" {
        keywords = append(keywords, k.English)
    }
    if k.Romaji != "" {
        keywords = append(keywords, k.Romaji)
    }
    if k.SimplifiedChinese != "" && k.SimplifiedChinese != k.Original {
        keywords = append(keywords, k.SimplifiedChinese)
    }
    if k.TraditionalChinese != "" && k.TraditionalChinese != k.Original {
        keywords = append(keywords, k.TraditionalChinese)
    }
    keywords = append(keywords, k.AlternativeSpellings...)
    keywords = append(keywords, k.CommonAliases...)

    return unique(keywords)
}
```

### Orchestrator Integration

```go
// Updated Search method with AI keyword retry
func (o *Orchestrator) Search(ctx context.Context, req *SearchRequest) (*SearchResult, *FallbackStatus) {
    // Phase 1: Try original query with all providers
    result, status := o.searchWithProviders(ctx, req)
    if result != nil && len(result.Items) > 0 {
        return result, status
    }

    // Phase 2: Generate alternative keywords with AI
    if o.keywordGenerator != nil {
        variants, err := o.keywordGenerator.GenerateKeywords(ctx, req.Query)
        if err != nil {
            slog.Warn("AI keyword generation failed", "error", err)
        } else {
            // Phase 3: Retry with each keyword variant
            for _, keyword := range variants.GetPrioritizedList() {
                altReq := &SearchRequest{
                    Query:     keyword,
                    Year:      req.Year,
                    MediaType: req.MediaType,
                    Language:  req.Language,
                }

                result, altStatus := o.searchWithProviders(ctx, altReq)
                status.KeywordAttempts = append(status.KeywordAttempts, KeywordAttempt{
                    Keyword: keyword,
                    Success: result != nil && len(result.Items) > 0,
                })

                if result != nil && len(result.Items) > 0 {
                    status.SuccessfulKeyword = keyword
                    return result, status
                }
            }
        }
    }

    return nil, status
}
```

### Language Detection

```go
// DetectLanguage determines the primary language of a title
func DetectLanguage(title string) Language {
    // Check for CJK characters
    hasChinese := containsChineseChars(title)
    hasJapanese := containsJapaneseChars(title)
    hasKorean := containsKoreanChars(title)

    if hasJapanese && !hasChinese {
        return LanguageJapanese
    }
    if hasKorean {
        return LanguageKorean
    }
    if hasChinese {
        // Detect Traditional vs Simplified
        if isTraditionalChinese(title) {
            return LanguageTraditionalChinese
        }
        return LanguageSimplifiedChinese
    }

    return LanguageEnglish
}

func containsChineseChars(s string) bool {
    for _, r := range s {
        if unicode.Is(unicode.Han, r) {
            return true
        }
    }
    return false
}

func containsJapaneseChars(s string) bool {
    for _, r := range s {
        if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
            return true
        }
    }
    return false
}
```

### Project Structure Notes

**Files to Create:**
```
/apps/api/internal/ai/
├── prompts/
│   └── keyword_generator.go  # Keyword generation prompt
├── keywords.go               # KeywordVariants types
└── keywords_test.go

/apps/api/internal/services/
├── keyword_service.go        # Keyword service
└── keyword_service_test.go
```

**Files to Modify:**
- `/apps/api/internal/metadata/orchestrator.go` - Add AI keyword retry phase
- `/apps/api/internal/services/ai_service.go` - Add `GenerateKeywords()` method
- `/apps/api/cmd/api/main.go` - Wire up keyword service

### Common Title Mappings (Reference)

| Original | Simplified | English | Romaji |
|----------|-----------|---------|--------|
| 鬼滅之刃 | 鬼灭之刃 | Demon Slayer | Kimetsu no Yaiba |
| 進撃の巨人 | 进击的巨人 | Attack on Titan | Shingeki no Kyojin |
| 我的英雄學院 | 我的英雄学院 | My Hero Academia | Boku no Hero Academia |
| 咒術迴戰 | 咒术回战 | Jujutsu Kaisen | Jujutsu Kaisen |
| 寄生上流 | 寄生虫 | Parasite | Gisaengchung |

### Testing Strategy

**Unit Tests:**
1. AI prompt generation tests
2. Keyword variants parsing tests
3. Language detection tests
4. S2T/T2S conversion tests

**Integration Tests:**
1. Full keyword generation → retry flow
2. Cache hit for generated keywords
3. Fallback success with alternative keyword

**Test Cases:**
```go
var keywordTestCases = []struct {
    title    string
    expected KeywordVariants
}{
    {
        title: "鬼滅之刃",
        expected: KeywordVariants{
            Original:           "鬼滅之刃",
            SimplifiedChinese:  "鬼灭之刃",
            TraditionalChinese: "鬼滅之刃",
            English:            "Demon Slayer",
            Romaji:             "Kimetsu no Yaiba",
        },
    },
    {
        title: "Parasite",
        expected: KeywordVariants{
            Original:           "Parasite",
            TraditionalChinese: "寄生上流",
            SimplifiedChinese:  "寄生虫",
        },
    },
}
```

**Coverage Targets:**
- Keyword service: ≥80%
- AI keywords: ≥80%

### Error Codes

Following project-context.md Rule 7:
- `KEYWORD_GENERATION_FAILED` - AI failed to generate keywords
- `KEYWORD_NO_ALTERNATIVES` - No alternative keywords generated
- `KEYWORD_ALL_FAILED` - All keyword variants failed to find results

### Dependencies

**Story Dependencies:**
- Story 3.1 (AI Provider Abstraction) - Provides AI client
- Story 3.3 (Fallback Chain) - Provides orchestrator to integrate with

### Performance Considerations

- AI keyword generation adds latency (~2-5s)
- Only trigger after all primary sources fail
- Cache generated keywords for 30 days (reuse Story 3.1 cache)
- Limit to 5-8 keyword variants to avoid excessive retries

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.6]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR19]
- [Source: _bmad-output/implementation-artifacts/3-1-ai-provider-abstraction-layer.md]
- [Source: _bmad-output/implementation-artifacts/3-3-multi-source-metadata-fallback-chain.md]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.1 (AI Provider Abstraction):**
- `AIProvider` interface and 30-day caching available
- Prompt can be added to prompts directory
- Error codes established

**From Story 3.3 (Fallback Chain):**
- Orchestrator pattern ready for extension
- `SearchRequest` and `SearchResult` types defined

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

N/A - All tests pass, no errors encountered.

### Completion Notes List

1. **Task 1**: Created comprehensive AI prompt template in `keyword_generator.go` with examples for Chinese, Japanese, Korean, and English titles.

2. **Task 2**: Extended `AIServiceInterface` with `GenerateKeywords()` method. Uses existing 30-day cache with `keywords:` prefix for cache keys.

3. **Task 3**: Created `KeywordVariants` type in `keywords.go` with `GetPrioritizedList()` method that returns keywords in optimal search order: English > Romaji > Simplified Chinese > Traditional Chinese > Alternative Spellings > Aliases.

4. **Task 4**: Integrated keyword retry phase into orchestrator's `Search()` method. After all primary providers fail, orchestrator calls `KeywordGenerator.GenerateKeywords()` and retries with each variant until success.

5. **Task 5**: Created `KeywordService` that wraps AIService and implements `metadata.KeywordGenerator` interface. Wired up in `main.go` to automatically enable AI keyword retry when AI service is configured.

6. **Task 6**: Implemented language detection using Unicode character ranges (hiragana/katakana for Japanese, hangul for Korean, CJK ideographs for Chinese). Simplified/Traditional Chinese detection uses character frequency analysis.

### File List

**New Files:**
- `apps/api/internal/ai/prompts/keyword_generator.go` - AI prompt template
- `apps/api/internal/ai/prompts/keyword_generator_test.go` - Prompt tests
- `apps/api/internal/ai/keywords.go` - KeywordVariants type
- `apps/api/internal/ai/keywords_test.go` - KeywordVariants tests
- `apps/api/internal/ai/language.go` - Language detection utilities
- `apps/api/internal/ai/language_test.go` - Language detection tests
- `apps/api/internal/services/keyword_service.go` - Keyword generation service
- `apps/api/internal/services/keyword_service_test.go` - Service tests

**Modified Files:**
- `apps/api/internal/services/ai_service.go` - Added `GenerateKeywords()` method
- `apps/api/internal/services/ai_service_test.go` - Added GenerateKeywords tests
- `apps/api/internal/services/parser_service_test.go` - Updated mock to implement new interface
- `apps/api/internal/metadata/orchestrator.go` - Added AI keyword retry phase
- `apps/api/internal/metadata/orchestrator_test.go` - Added keyword retry tests
- `apps/api/internal/services/metadata_service.go` - Added `SetKeywordGenerator()` method
- `apps/api/cmd/api/main.go` - Wired up KeywordService
