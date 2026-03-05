# Epic 9: Subtitle Integration

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can search for subtitles from OpenSubtitles and Zimuku with Traditional Chinese subtitle priority, download subtitles, manually upload subtitles, and see subtitle availability status.

## Story 9.1: Subtitle Search Integration

As a **Traditional Chinese user**,
I want to **search for subtitles from multiple sources**,
So that **I can find subtitles for my media**.

**Acceptance Criteria:**

**Given** the user opens a media detail page
**When** subtitles are not attached
**Then** a "Search Subtitles" button is available

**Given** the user clicks "Search Subtitles"
**When** the search executes
**Then** it queries OpenSubtitles API first
**And** then queries Zimuku (web scraping)
**And** results are combined and deduplicated

**Given** search results are displayed
**When** viewing the list
**Then** each result shows: Language, Format, Source, Rating/Downloads
**And** Traditional Chinese subtitles are highlighted

**Given** an error occurs with one source
**When** the other source succeeds
**Then** results from the working source are shown
**And** error message indicates partial results

**Technical Notes:**
- Implements FR75: Search subtitles from OpenSubtitles and Zimuku
- OpenSubtitles API with rate limiting
- Zimuku web scraping with caching

---

## Story 9.2: Traditional Chinese Subtitle Priority

As a **Traditional Chinese user**,
I want **Traditional Chinese subtitles prioritized**,
So that **I see the most relevant results first**.

**Acceptance Criteria:**

**Given** subtitle search results are returned
**When** displaying the list
**Then** Traditional Chinese (zh-TW) subtitles appear first
**And** Simplified Chinese (zh-CN) subtitles second
**And** English subtitles third
**And** Other languages follow

**Given** user preferences are set
**When** configuring subtitle language priority
**Then** users can customize the priority order
**And** preferences persist across sessions

**Given** multiple Traditional Chinese subtitles exist
**When** sorting within the priority group
**Then** higher-rated/more-downloaded subtitles appear first
**And** format preferences (SRT > ASS) can be configured

**Technical Notes:**
- Implements FR76: Prioritize Traditional Chinese subtitles
- Language detection from subtitle metadata
- User-configurable priority system

---

## Story 9.3: Subtitle Download

As a **user**,
I want to **download subtitles directly**,
So that **I can use them with my media player**.

**Acceptance Criteria:**

**Given** subtitle search results are displayed
**When** the user clicks "Download" on a result
**Then** the subtitle file is downloaded
**And** saved to the same folder as the media file

**Given** the subtitle is downloaded
**When** naming the file
**Then** it matches the media filename with language suffix
**And** format: `MediaName.zh-TW.srt` or `MediaName.zh-CN.ass`

**Given** a subtitle already exists for that language
**When** downloading another
**Then** user is prompted: "Replace existing or keep both?"
**And** "Keep both" adds a suffix: `.v2.srt`

**Given** download succeeds
**When** viewing the media detail page
**Then** the subtitle appears in the "Available Subtitles" list
**And** status shows "Downloaded from [Source]"

**Technical Notes:**
- Implements FR77: Download subtitle files
- Automatic filename matching
- Respects media folder write permissions

---

## Story 9.4: Manual Subtitle Upload

As a **user**,
I want to **upload my own subtitle files**,
So that **I can use subtitles from other sources**.

**Acceptance Criteria:**

**Given** the user opens a media detail page
**When** clicking "Upload Subtitle"
**Then** a file picker dialog opens
**And** accepts: .srt, .ass, .ssa, .sub, .vtt formats

**Given** a subtitle file is selected
**When** uploading
**Then** the user can select the language from a dropdown
**And** the file is copied to the media folder

**Given** the upload succeeds
**When** the subtitle is saved
**Then** it appears in the "Available Subtitles" list
**And** status shows "Manually uploaded"

**Given** the subtitle needs editing
**When** the user clicks "Rename/Delete"
**Then** they can change the language tag
**And** delete the subtitle file

**Technical Notes:**
- Implements FR78: Manually upload subtitle files
- Subtitle format validation
- UTF-8 encoding detection and conversion

---

## Story 9.5: Automatic Subtitle Download

As a **user**,
I want **subtitles to download automatically**,
So that **I don't have to search manually every time**.

**Acceptance Criteria:**

**Given** the user enables "Auto-download subtitles" in Settings
**When** configuring preferences
**Then** they can select: Preferred languages (ordered list)
**And** Minimum rating threshold
**And** Preferred format (SRT/ASS/Any)

**Given** auto-download is enabled
**When** new media is added to the library
**Then** the system automatically searches for subtitles
**And** downloads the best match based on preferences

**Given** a subtitle is auto-downloaded
**When** viewing the media detail
**Then** status shows "Auto-downloaded"
**And** the user can reject and search for alternatives

**Given** no suitable subtitle is found
**When** auto-download fails
**Then** the media shows "No subtitle found"
**And** manual search remains available

**Technical Notes:**
- Implements FR79: Automatically download subtitles
- Background task queue (ARCH-4)
- Respects API rate limits

---

## Story 9.6: Subtitle Availability Status

As a **user**,
I want to **see subtitle availability at a glance**,
So that **I know which media has subtitles**.

**Acceptance Criteria:**

**Given** the user views the media library
**When** looking at media cards/list items
**Then** a subtitle indicator shows:
- 🟢 Has Traditional Chinese subtitle
- 🟡 Has Simplified Chinese or English only
- ⚪ No subtitle

**Given** the user opens a media detail page
**When** viewing subtitle section
**Then** all available subtitles are listed with:
- Language flag/code
- Format (SRT, ASS)
- Source (OpenSubtitles, Zimuku, Manual)

**Given** subtitles exist online but not downloaded
**When** viewing the detail page
**Then** "Available online" count is shown
**And** one-click "Download best match" is available

**Technical Notes:**
- Implements FR80: Display subtitle availability status
- Visual indicators on library views
- Cache online availability for performance

---
