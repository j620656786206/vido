# Epic 10: Homepage TV Wall
**Phase:** Phase 2 — Discovery & Browse Experience

Users see a visually rich homepage with a Hero Banner showcasing trending content, customizable explore blocks (e.g., "近期台灣院線", "熱門日韓劇"), smart trending with server-side TMDB language/region filtering for zh-TW relevance, auto-hiding of far-future and low-quality content, and "已有/已請求" badges showing which content is already in the library or has been requested. The homepage transforms Vido from a management tool into a discovery platform.

**v4 Feature IDs covered:** P2-001, P2-002, P2-003, P2-004, P2-005, P2-006

**Dependencies on Completed Work:**
- Epic 2: TMDB API integration (trending/discover endpoints)
- Epic 5: Grid view components, media card design

**Stories (to be created):**
- D-1: Hero Banner component — auto-rotating backdrop images with trending content, play trailer action
- D-1a: Discover year-range input validation (follow-up to Story 10-1) — reject `year_gte > year_lte` with HTTP 400 `INVALID_YEAR_RANGE` at the handler layer; preserves zero-value "unlimited" semantics
- D-2: Custom explore blocks CRUD — admin can create/edit/delete themed content blocks with TMDB discover filters
- D-3: Server-side TMDB filtering — language/region/date filtering to ensure zh-TW relevant content
- D-4: Content quality filtering — auto-hide unreleased (>6 months future), low-vote-count, and adult content
- D-5: Availability badges — "已有" (owned) and "已請求" (requested) badges overlaid on media cards
- D-6: Homepage layout engine — responsive grid layout with configurable block ordering and sizing

**Success Criteria:**
- Homepage loads in <2s (LCP) with all blocks populated
- Trending content respects zh-TW region (>90% of results have zh-TW metadata)
- Users can customize homepage blocks without page reload
