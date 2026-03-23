# Epic 12: Rich Media Detail Page
**Phase:** Phase 2 — Discovery & Browse Experience

Enhanced media detail page that combines TMDB and Douban dual ratings for a more complete quality signal, displays TV show season/episode lists with per-episode subtitle status, shows related content recommendations, indicates streaming platform availability via TMDB Watch Providers API, embeds trailers, and provides direct Douban links for user reviews. This transforms the detail page from a metadata viewer into a comprehensive media hub.

**v4 Feature IDs covered:** P2-020, P2-021, P2-022, P2-023, P2-024, P2-025

**Dependencies on Completed Work:**
- Epic 2: Media detail page (Story 2-4), TMDB metadata
- Epic 3: Douban web scraper (Story 3-4)
- Epic 5: Full media detail page (Story 5-6)

**Stories (to be created):**
- F-1: Dual rating display — TMDB + Douban ratings side-by-side with vote counts
- F-2: Season/episode list — expandable season accordion with episode list, air dates, subtitle status per episode
- F-3: Related content recommendations — TMDB similar/recommended endpoints with "已有" badges
- F-4: Streaming platform availability — TMDB Watch Providers API integration showing where to watch (Netflix, Disney+, etc.)
- F-5: Trailer embeds — YouTube trailer embed with fallback to TMDB video links
- F-6: Douban integration — direct links to Douban page, pull user review summary

**Success Criteria:**
- Detail page loads in <1.5s including dual ratings and streaming data
- Streaming platform data available for >80% of content in TMDB
- Season/episode list correctly reflects subtitle status from Epic 8
