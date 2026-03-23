# Epic 8: Subtitle Engine
**Phase:** Phase 1 — Core Media Pipeline

Users can search for Traditional Chinese subtitles across multiple sources (Assrt, Zimuku, OpenSubtitles), with correct 簡繁 identification to prevent false-positive simplified Chinese subtitles. The engine integrates OpenCC for automatic simplified-to-traditional conversion, scores and ranks subtitle results by quality, supports auto-download for matched media, provides a manual search UI for unmatched content, and enables batch subtitle processing across the entire library. This is the core v4 MVP differentiator — solving the #1 pain point for Traditional Chinese NAS users.

**v4 Feature IDs covered:** P1-010, P1-011, P1-012, P1-013, P1-014, P1-015, P1-016, P1-017, P1-018, P1-019

**Dependencies on Completed Work:**
- Epic 1: Repository pattern, secrets management (for API keys)
- Epic 2: Media entity storage, TMDB metadata (for matching subtitles to media)
- Epic 3: Multi-source fallback pattern (reusable for subtitle sources)
- Epic 5: Library UI (subtitle status indicators in grid/detail views)

**Stories (to be created):**
- 8-1: Assrt API client — search, download, rate limiting, error handling
- 8-2: Zimuku web scraper — search, download, anti-scraping mitigation
- 8-3: OpenSubtitles API client — search, download, auth, hash-based matching
- 8-4: 簡繁 language detection — detect simplified vs traditional Chinese in subtitle files
- 8-5: OpenCC integration — automatic simplified→traditional conversion with configurable profiles
- 8-6: Subtitle scoring and ranking — score results by source reliability, language match, release group match, format
- 8-7: Auto-download service — automatically find and download best-match subtitle for new media
- 8-8: Manual subtitle search UI — search interface with source selection, preview, download
- 8-9: Batch subtitle processing — queue-based processing for library-wide subtitle search
- 8-10: Subtitle file management — store, rename, associate subtitle files with media entries

**Implementation Decisions (Gate 2A — 2026-03-23):**
- Search strategy: All 3 sources (Assrt/Zimuku/OpenSubtitles) searched in parallel, results merged
- Assrt API key: Optional — skip Assrt when not configured, use Zimuku + OpenSubtitles only
- Language detection: Unicode unique character set analysis (~99% accuracy, 3-5ms/file)
- Detection threshold: >70% traditional-unique characters = zh-Hant
- Subtitle extension: .zh-Hant.srt (IETF BCP 47)
- OpenCC profile: s2twp (Simplified → Traditional Taiwan Phrases)
- Scoring weights: Language 40% + Resolution 20% + Source trust 20% + Group 10% + Downloads 10%
- Auto-download failure: Retry next-best result; all exhausted → not_found + UI indicator
- Manual search (P1-018): Single movie/episode scope
- Batch processing (P1-019): Whole season or whole library scope
- P1-011 (Assrt fix): Merged into Story 8-1 (Assrt client)
- P1-013 (extension normalization): Merged into Story 8-10 (subtitle file management)
- Reference: https://github.com/alexyu-tvbs/makeownsrt for Claude-based terminology correction (P1-020)

**Success Criteria:**
- >85% zh-TW subtitle hit rate across Assrt + Zimuku + OpenSubtitles combined
- 0% false-positive simplified Chinese subtitles served as Traditional Chinese
- Batch processing of 100 media items completes in <10 minutes
