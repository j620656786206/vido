# Epic 15: Indexer Integration
**Phase:** Phase 3 — Automation & Integration

Users can connect to Prowlarr for centralized indexer management, leveraging their existing indexer configurations for search and download. For users without Prowlarr, Vido provides a built-in basic public tracker search as a fallback, enabling the request-to-download pipeline without external dependencies.

**v4 Feature IDs covered:** P3-020, P3-021

**Dependencies on Completed Work:**
- Epic 4: External service connection pattern (reusable for Prowlarr)
- Epic 13: Request system (indexer search is triggered by requests)

**Stories (to be created):**
- I-1: Prowlarr plugin — connection config, search API integration, indexer list sync, result normalization
- I-2: Built-in basic indexer — public tracker search for users without Prowlarr (limited scope, best-effort)

**Success Criteria:**
- Prowlarr search results returned in <5s
- Built-in indexer provides results for >70% of popular content searches
