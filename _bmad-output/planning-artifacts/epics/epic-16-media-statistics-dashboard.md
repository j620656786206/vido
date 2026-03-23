# Epic 16: Media Statistics Dashboard
**Phase:** Phase 4 — Polish & Ecosystem

Users see a dashboard with a media library overview (total movie/TV counts, disk usage, resolution distribution), subtitle coverage rate visualization showing how much of the library has zh-TW subtitles, genre/region/year distribution charts for collection insights, and a recently added media timeline. This gives users a bird's-eye view of their collection health and growth.

**v4 Feature IDs covered:** P4-001, P4-002, P4-003, P4-004

**Dependencies on Completed Work:**
- Epic 2: Media entity storage (counts, metadata fields)
- Epic 5: Library data (genre, year, type fields)
- Epic 6: Performance metrics pattern (Story 6-11, absorbed into this epic)
- Epic 8: Subtitle data (for coverage rate)

**Stories (to be created):**
- J-1: Library overview panel — total counts (movies, TV shows, episodes), disk usage, resolution breakdown (4K/1080p/720p/SD)
- J-2: Subtitle coverage visualization — percentage of library with zh-TW subtitles, breakdown by source
- J-3: Distribution charts — genre, region, year, and rating distribution using lightweight chart library
- J-4: Recently added timeline — chronological list of recently added media with metadata

**Success Criteria:**
- Dashboard loads in <1s with all statistics computed
- Statistics accurate to the latest completed scan
- Charts render without layout shift (CLS <0.1)
