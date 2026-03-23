> **STATUS: COMPLETED (v3 PRD)**
> This epic was completed under the v3 PRD structure. Its work is fully integrated
> into the v4 codebase. See [Completed Work Registry](./completed-work-registry.md)
> for the mapping to v4 feature IDs.

# Epic 2: Media Search & Traditional Chinese Metadata

**Phase:** MVP (Q1 - March 2026)

**Goal:** Users can search for movies and TV shows by title (Traditional Chinese or English), view search results with beautiful Traditional Chinese metadata, and browse results in a responsive grid view.

## Story 2.1: TMDb API Integration with zh-TW Priority

As a **media collector**,
I want to **search TMDb with Traditional Chinese as the priority language**,
So that **I see movie and TV show information in my preferred language**.

**Acceptance Criteria:**

**Given** a user searches for a movie or TV show
**When** the search request is sent to TMDb API
**Then** the API is called with `language=zh-TW` parameter
**And** fallback to `zh-CN` if zh-TW not available
**And** fallback to `en` if no Chinese available

**Given** the TMDb API returns results
**When** the response is processed
**Then** the results are cached for 24 hours (NFR-I7)
**And** duplicate requests within cache period use cached data

**Given** the TMDb API rate limit is 40 requests per 10 seconds
**When** multiple requests are made rapidly
**Then** the system respects the rate limit (NFR-I6)
**And** queues excess requests with appropriate delays

**Technical Notes:**
- Implements FR13: Retrieve zh-TW metadata from TMDb
- Implements NFR-I5, NFR-I6, NFR-I7
- API key can be user-provided or use default quota

---

## Story 2.2: Media Search Interface

As a **media collector**,
I want to **search for movies and TV shows by typing a title**,
So that **I can quickly find the content I'm looking for**.

**Acceptance Criteria:**

**Given** the user is on the search page
**When** they type a search query (minimum 2 characters)
**Then** search results appear within 500ms (NFR-P5)
**And** results show poster, title (zh-TW), year, and media type

**Given** search results are displayed
**When** results exceed 20 items
**Then** pagination is provided
**And** the user can navigate between pages

**Given** the user searches in Traditional Chinese (e.g., "鬼滅之刃")
**When** results are returned
**Then** Traditional Chinese titles are displayed prominently
**And** English/original titles are shown as secondary information

**Given** the user searches in English (e.g., "Demon Slayer")
**When** results are returned
**Then** the system still displays Traditional Chinese metadata when available

**Technical Notes:**
- Implements FR1: Search movies/TV shows by title
- Implements NFR-P5: Search API <500ms
- Desktop-first with hover interactions (UX-1)

---

## Story 2.3: Search Results Grid View

As a **media collector**,
I want to **browse search results in a responsive grid view**,
So that **I can quickly scan through multiple results visually**.

**Acceptance Criteria:**

**Given** search results are displayed
**When** viewed on desktop (>1024px)
**Then** results display in a 4-6 column grid
**And** each card shows poster, title, year, rating

**Given** search results are displayed
**When** viewed on tablet (768-1023px)
**Then** results display in a 3-4 column grid

**Given** search results are displayed
**When** viewed on mobile (<768px)
**Then** results display in a 2 column grid
**And** touch targets are at least 44px

**Given** the user hovers over a result card (desktop)
**When** the mouse is over the card
**Then** additional information appears (genre, description preview)
**And** the card has a subtle highlight effect

**Technical Notes:**
- Implements FR3: Browse search results in grid view
- Implements UX-1: Desktop-first design with hover interactions
- Implements UX-8: Hover over Click principle

---

## Story 2.4: Media Detail Page

As a **media collector**,
I want to **view detailed information about a movie or TV show**,
So that **I can learn more before adding it to my library**.

**Acceptance Criteria:**

**Given** the user clicks on a search result
**When** the detail page loads
**Then** it displays:
- Full Traditional Chinese title and original title
- High-resolution poster
- Release year and runtime
- Genre tags
- Director and main cast
- Plot summary in Traditional Chinese
- TMDb rating

**Given** the media is a TV show
**When** viewing the detail page
**Then** it also displays:
- Number of seasons and episodes
- Air date information
- Network/streaming platform

**Given** the detail page is loading
**When** data is being fetched
**Then** a loading skeleton is displayed
**And** the page transition completes within 200ms (NFR-P11)

**Technical Notes:**
- Implements FR4: View media item detail pages
- Page opens in new tab (desktop) or modal (mobile) per UX principles

---

## Story 2.5: Standard Filename Parser (Regex-based)

As a **media collector**,
I want the **system to parse standard naming convention filenames**,
So that **most of my files are automatically identified without AI**.

**Acceptance Criteria:**

**Given** a file with standard naming like `Movie.Name.2024.1080p.BluRay.mkv`
**When** the parser processes the filename
**Then** it extracts:
- Title: "Movie Name"
- Year: 2024
- Quality: 1080p
- Source: BluRay

**Given** a TV show file like `Show.Name.S01E05.720p.WEB-DL.mkv`
**When** the parser processes the filename
**Then** it extracts:
- Title: "Show Name"
- Season: 1
- Episode: 5
- Quality: 720p
- Source: WEB-DL

**Given** parsing completes
**When** measuring performance
**Then** standard regex parsing completes within 100ms per file (NFR-P13)

**Given** the filename cannot be parsed by regex
**When** parsing fails
**Then** the file is flagged for AI parsing (Epic 3)
**And** a clear status indicator shows "Pending AI parsing"

**Technical Notes:**
- Implements FR11, FR12: Parse standard naming, extract metadata
- Regex patterns cover 80%+ of standard naming conventions
- Foundation for AI parsing fallback in Epic 3

---

## Story 2.6: Media Entity and Database Storage

As a **developer**,
I want to **store parsed media metadata in the database**,
So that **users can access their library without re-fetching from APIs**.

**Acceptance Criteria:**

**Given** media metadata is retrieved from TMDb
**When** the user adds it to their library
**Then** the metadata is stored in the local database
**And** the original filename is preserved for reference

**Given** a media entity is stored
**When** querying by title
**Then** full-text search finds matches within 500ms (NFR-SC8)
**And** both Chinese and English titles are searchable

**Given** the database contains media entries
**When** the application restarts
**Then** all stored metadata is preserved
**And** cached images are still available

**Technical Notes:**
- Implements FR14: Store metadata to local database
- Uses Repository Pattern from Story 1.1
- Creates Media entity table with proper indexes
- SQLite FTS5 for full-text search

---
