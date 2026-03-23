# Epic 11: Advanced Search & Filter
**Phase:** Phase 2 — Discovery & Browse Experience

Users can filter content by multiple dimensions simultaneously (genre, year, region, rating, streaming platform) using a persistent chip UI that stays visible during browsing, apply complex sorting (e.g., rating descending + year descending), get instant search with debounced suggestions, benefit from zh-TW search priority that boosts Traditional Chinese results, and save filter presets for quick access to frequent browsing patterns.

**v4 Feature IDs covered:** P2-010, P2-011, P2-012, P2-013, P2-014, P2-015

**Dependencies on Completed Work:**
- Epic 2: Basic search interface, TMDB search API
- Epic 5: Library search, sort, filter infrastructure (Story 5-3, 5-4, 5-5)

**Stories (to be created):**
- E-1: Multi-dimensional filter engine — backend support for combining genre + year + region + rating + platform filters
- E-2: Persistent chip UI — visual filter chips that persist across navigation, with add/remove/clear actions
- E-3: Compound sort — multi-key sorting (e.g., rating DESC, year DESC) with drag-to-reorder sort priorities
- E-4: Instant search with suggestions — debounced search input with dropdown suggestions from TMDB + local library
- E-5: zh-TW search priority — boost Traditional Chinese title matches, handle romanization variants
- E-6: Saved filter presets — save/load/delete named filter configurations

**Success Criteria:**
- Search results returned in <500ms for combined multi-dimensional queries
- Filter state persists across navigation (back button restores filters)
- Saved presets sync across browser sessions
