# Epic 3: AI-Powered Fansub Parsing & Multi-Source Fallback

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can parse complex fansub naming conventions using AI, with a four-layer fallback mechanism ensuring metadata is always found or manual options provided.

## Story 3.1: AI Provider Abstraction Layer

As a **developer**,
I want an **abstraction layer for AI providers**,
So that **we can switch between Gemini and Claude without code changes**.

**Acceptance Criteria:**

**Given** the system needs AI parsing capabilities
**When** configuring the AI provider
**Then** users can select Gemini or Claude via environment variable `AI_PROVIDER`
**And** the same interface is used regardless of provider

**Given** an AI provider is configured
**When** making API calls
**Then** the system uses the appropriate API format for that provider
**And** responses are normalized to a common format

**Given** AI parsing results are returned
**When** caching the results
**Then** results are cached for 30 days (NFR-I10)
**And** cache key is based on filename hash

**Given** AI API calls are made
**When** the call exceeds 15 seconds
**Then** it times out and falls back to next option (NFR-I12)

**Technical Notes:**
- Implements ARCH-3: AI Provider Abstraction Layer
- Implements NFR-I9, NFR-I10, NFR-I12
- Strategy pattern for provider switching

---

## Story 3.2: AI Fansub Filename Parsing

As a **media collector with fansub content**,
I want the **system to parse complex fansub naming using AI**,
So that **files like `[Leopard-Raws] Show - 01 (BD 1080p).mkv` are correctly identified**.

**Acceptance Criteria:**

**Given** a fansub filename like `[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv`
**When** AI parsing is triggered
**Then** it extracts:
- Fansub group: Leopard-Raws (ignored for search)
- Title: Kimetsu no Yaiba / 鬼滅之刃
- Episode: 26
- Quality: 1080p
- Source: BD (Blu-ray)

**Given** a Chinese fansub filename like `【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4`
**When** AI parsing is triggered
**Then** it extracts:
- Title: 我的英雄學院
- Episode: 1
- Quality: 1080P
- Language: Traditional Chinese

**Given** AI parsing is in progress
**When** the user views the status
**Then** they see a progress indicator showing current step (UX-3)
**And** parsing completes within 10 seconds (NFR-P14)

**Technical Notes:**
- Implements FR15: Parse fansub naming with AI
- AI prompt engineered for fansub pattern recognition
- Uses Provider Abstraction from Story 3.1

---

## Story 3.3: Multi-Source Metadata Fallback Chain

As a **media collector**,
I want the **system to try multiple metadata sources automatically**,
So that **I always get metadata even when one source fails**.

**Acceptance Criteria:**

**Given** a search query is made
**When** TMDb returns no results
**Then** the system automatically tries Douban within 1 second (NFR-R3)
**And** the user sees "TMDb ❌ → Searching Douban..." status

**Given** both TMDb and Douban fail
**When** fallback continues
**Then** the system tries Wikipedia MediaWiki API
**And** respects Wikipedia rate limit (1 req/s, NFR-I14)

**Given** all automated sources fail
**When** the fallback chain completes
**Then** the user is offered manual search option
**And** the status shows all attempted sources: "TMDb ❌ → Douban ❌ → Wikipedia ❌ → Manual search"

**Given** any source succeeds
**When** metadata is found
**Then** the source is indicated (FR42): "Source: Douban"
**And** the fallback chain stops

**Technical Notes:**
- Implements FR16, FR17, FR18: Multi-source fallback
- Implements ARCH-2: Multi-source metadata orchestrator
- Implements ARCH-7: Circuit Breaker Pattern

---

## Story 3.4: Douban Web Scraper

As a **media collector with Asian content**,
I want **Douban as a fallback metadata source**,
So that **Chinese movies and shows not on TMDb can still be identified**.

**Acceptance Criteria:**

**Given** TMDb search fails
**When** Douban scraper is triggered
**Then** it searches Douban for the title
**And** extracts: Chinese title, year, director, cast, rating, poster URL

**Given** Douban anti-scraping measures are encountered
**When** the scraper detects blocking
**Then** it implements exponential backoff
**And** falls back to Wikipedia if blocked

**Given** Douban returns results
**When** displaying metadata
**Then** Traditional Chinese is prioritized
**And** the source is clearly labeled

**Technical Notes:**
- Implements FR17: Auto-switch to Douban
- Web scraping with proper rate limiting
- Respect robots.txt and implement polite scraping

---

## Story 3.5: Wikipedia Metadata Fallback

As a **media collector with obscure content**,
I want **Wikipedia as a third fallback source**,
So that **even rare titles can have basic metadata**.

**Acceptance Criteria:**

**Given** TMDb and Douban both fail
**When** Wikipedia fallback is triggered
**Then** it searches zh.wikipedia.org for the title
**And** extracts data from Infobox templates

**Given** Wikipedia article has an Infobox
**When** parsing the page
**Then** it extracts: title, director, cast, year, genre, plot summary
**And** handles multiple Infobox template variations (NFR-I15)

**Given** Wikipedia has no poster images
**When** displaying the media
**Then** a default placeholder icon is shown
**And** the user is notified: "No poster available from Wikipedia"

**Given** Wikipedia API is called
**When** making requests
**Then** proper User-Agent is set (NFR-I13)
**And** rate limit of 1 request/second is respected (NFR-I14)

**Technical Notes:**
- Implements FR18: Retrieve metadata from Wikipedia
- Implements NFR-I13, NFR-I14, NFR-I15
- Uses MediaWiki API for structured data access

---

## Story 3.6: AI Search Keyword Generation

As a **media collector**,
I want the **AI to generate alternative search keywords**,
So that **hard-to-find titles can be located through different search terms**.

**Acceptance Criteria:**

**Given** initial search fails on all sources
**When** AI keyword generation is triggered
**Then** AI analyzes the filename and generates:
- Original title
- English translation
- Japanese romaji (if applicable)
- Alternative spellings

**Given** alternative keywords are generated
**When** retrying metadata sources
**Then** each keyword variant is tried
**And** the first successful match is used

**Given** the filename is `鬼滅之刃.S01E26.mkv`
**When** TMDb search "鬼滅之刃" fails
**Then** AI generates alternatives:
- "鬼灭之刃" (Simplified)
- "Demon Slayer" (English)
- "Kimetsu no Yaiba" (Romaji)

**Technical Notes:**
- Implements FR19: AI re-parse and generate alternative keywords
- Layer 4 of the fallback architecture
- Increases metadata coverage from 98% to 99%+

---

## Story 3.7: Manual Metadata Search and Selection

As a **media collector**,
I want to **manually search and select the correct metadata**,
So that **I can fix misidentified or unfound titles**.

**Acceptance Criteria:**

**Given** automatic parsing fails
**When** the user clicks "Manual Search"
**Then** a search dialog opens
**And** they can enter a custom search query

**Given** manual search returns results
**When** the user views the results
**Then** they see poster, title, year, and description preview
**And** they can select the correct match

**Given** the user selects a match
**When** confirming the selection
**Then** the metadata is applied to the file
**And** the mapping is saved for learning (Story 3.9)

**Technical Notes:**
- Implements FR20: Manual search and select metadata
- Implements UX-4: Always show next step
- Part of the graceful degradation chain

---

## Story 3.8: Metadata Editor

As a **media collector**,
I want to **manually edit metadata for any media item**,
So that **I can correct errors or add missing information**.

**Acceptance Criteria:**

**Given** a media item in the library
**When** the user clicks "Edit Metadata"
**Then** an edit form opens with all editable fields:
- Title (Chinese/English)
- Year
- Genre
- Director
- Cast
- Description
- Poster (upload or URL)

**Given** the user modifies metadata
**When** saving changes
**Then** the changes are persisted to the database
**And** the source is updated to "Manual"

**Given** the user uploads a custom poster
**When** the upload completes
**Then** the image is resized and optimized
**And** stored in local cache

**Technical Notes:**
- Implements FR21: Manually edit media metadata
- Form validation for required fields
- Image processing for poster optimization

---

## Story 3.9: Filename Mapping Learning System

As a **power user**,
I want the **system to learn from my corrections**,
So that **similar filenames are automatically matched in the future**.

**Acceptance Criteria:**

**Given** the user manually corrects a filename match
**When** the correction is saved
**Then** the system asks: "Learn this pattern for future files?"
**And** if confirmed, stores the pattern-to-metadata mapping

**Given** a learned pattern exists
**When** a new file matches the pattern
**Then** the system automatically applies the learned mapping
**And** shows: "✓ 已套用你之前的設定" (UX-5)

**Given** the user views settings
**When** checking learned patterns
**Then** they see: "已記住 15 個自訂規則"
**And** can view, edit, or delete learned patterns

**Technical Notes:**
- Implements FR24: Learn from user corrections
- Implements UX-5: Learning system feedback
- Pattern matching with fuzzy matching support

---

## Story 3.10: Parse Status Indicators

As a **media collector**,
I want to **see clear status indicators for parsing progress**,
So that **I know what's happening with each file**.

**Acceptance Criteria:**

**Given** a file is being parsed
**When** viewing the file list
**Then** status icons indicate:
- ⏳ Parsing in progress
- ✅ Successfully parsed
- ⚠️ Parsed with warnings (manual selection needed)
- ❌ Parse failed

**Given** parsing is in progress
**When** viewing the progress
**Then** step indicators show: "解析檔名中..." → "搜尋 TMDb..." → "下載海報..."
**And** the current step is highlighted (UX-3)

**Given** parsing fails
**When** viewing the error
**Then** the failure reason is explained
**And** clear next steps are provided (UX-4)

**Technical Notes:**
- Implements FR22: View parse status indicators
- Implements UX-3, UX-4: Wait experience and failure handling

---

## Story 3.11: Auto-Retry Mechanism

As a **media collector**,
I want the **system to automatically retry when sources are temporarily unavailable**,
So that **temporary failures don't require my intervention**.

**Acceptance Criteria:**

**Given** a metadata source returns a temporary error
**When** the error is detected
**Then** the system automatically queues a retry
**And** uses exponential backoff: 1s → 2s → 4s → 8s (NFR-R5)

**Given** all retries fail
**When** the maximum attempts are reached
**Then** the file is marked for manual intervention
**And** the user is notified via the activity monitor

**Given** the source recovers
**When** the retry succeeds
**Then** the metadata is applied
**And** the file status updates automatically

**Technical Notes:**
- Implements FR25: Auto-retry when sources unavailable
- Implements NFR-R5: Exponential backoff
- Maximum 4 retry attempts before manual fallback

---

## Story 3.12: Graceful Degradation

As a **media collector**,
I want the **system to never completely fail**,
So that **I always have options even in worst-case scenarios**.

**Acceptance Criteria:**

**Given** all metadata sources fail
**When** the user views the file
**Then** they see:
- The original filename
- "Unable to auto-identify" message
- Three clear options: Manual search, Edit filename, Skip for now

**Given** the AI service is down
**When** parsing a fansub filename
**Then** the system falls back to regex parsing
**And** notifies: "AI 服務暫時無法使用，使用基本解析" (NFR-R11)

**Given** core functionality is needed
**When** all external APIs are unavailable
**Then** the library browsing and search still work (NFR-R13)
**And** only new metadata fetching is affected

**Technical Notes:**
- Implements FR26: Graceful degradation with manual option
- Implements NFR-R11, NFR-R13: Fallback behaviors
- Core principle: Never leave user stuck

---
