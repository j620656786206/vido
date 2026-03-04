# Implementation Gap Analysis

## Gap Category 1: PRD Features vs Current Implementation

**Methodology:** Map 94 functional requirements to existing code

### Search & Discovery (FR1-FR10) - 0% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR1: Search by title/keyword | ❌ Missing | No search endpoint, no UI |
| FR2: zh-TW metadata priority | ⚠️ Partial | TMDb client exists (root backend), no database integration |
| FR3: Grid/List view toggle | ❌ Missing | No UI components |
| FR4: Filter by genre/year/rating | ❌ Missing | No filter logic, no UI |
| FR5: Sort options | ❌ Missing | No sort implementation |
| FR6-FR10: Pagination, recommendations, etc. | ❌ Missing | No implementation |

**Blocking Issues:**
- Root backend has TMDb client but NO database to store results
- Apps backend has database but NO TMDb client to fetch metadata
- Frontend has NO search UI components

---

### Filename Parsing & Metadata (FR11-FR26) - 5% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR11: Standard regex parsing | ❌ Missing | No parser implementation |
| FR12: AI-powered parsing | ❌ Missing | No AI provider integration |
| FR13: Manual entry fallback | ❌ Missing | No UI for manual entry |
| FR14: Batch parsing | ❌ Missing | No batch logic |
| FR15-FR20: Multi-source fallback | ⚠️ Partial | TMDb client exists, Douban/Wikipedia/AI missing |
| FR21-FR23: Confidence scoring | ❌ Missing | No scoring logic |
| FR24: Manual verification | ❌ Missing | No UI |
| FR25-FR26: Learning system | ❌ Missing | No `filename_mappings` table |

**Blocking Issues:**
- NO filename parser (regex or AI)
- NO AI provider abstraction layer
- NO multi-source orchestrator
- NO learning system database schema

---

### Download Integration (FR27-FR37) - 0% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR27-FR37: qBittorrent integration | ❌ Missing | No qBittorrent client, no UI, no database schema |

**Blocking Issues:**
- No qBittorrent Web API client implementation
- No download monitoring UI
- No `download_history` table in database schema

---

### Media Library Management (FR38-FR46) - 15% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR38: Browse media library | ⚠️ Partial | Database schema exists, NO backend endpoints, NO UI |
| FR39: Grid/List view | ❌ Missing | No UI |
| FR40-FR46: Batch ops, filters, watch history | ❌ Missing | No implementation |

**What Exists:**
- `movies` and `series` tables in apps/api database
- Repository pattern with CRUD operations

**What's Missing:**
- API endpoints to query repositories
- Frontend UI to display library
- Watch history tracking (no table)
- Filter/sort logic

---

### System Configuration (FR47-FR66) - 10% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR47: Docker deployment | ✅ Exists | Docker Compose configured |
| FR48: Setup wizard | ❌ Missing | No UI |
| FR49-FR66: Settings, cache mgmt, backups, etc. | ⚠️ Partial | `settings` table exists, NO UI, NO cache system, NO backup logic |

**What Exists:**
- Docker Compose configuration
- Settings table in database

**What's Missing:**
- Setup wizard UI
- Settings management UI
- Cache management system (Decision #4 not implemented)
- Backup/restore system
- Performance monitoring

---

### Authentication (FR67-FR74) - 0% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR67-FR74: Auth system | ❌ Missing | No JWT implementation, no `users` table, no login UI |

**Blocking Issues:**
- Decision #3 (JWT auth) not implemented
- No authentication middleware
- No password hashing (bcrypt)
- No login/PIN UI

---

### Subtitle Management (FR75-FR80) - 0% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR75-FR80: Subtitle automation | ❌ Missing | Growth phase feature, deferred |

---

### Automation (FR81-FR86) - 0% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR81-FR86: Watch folder, auto-parsing | ❌ Missing | Growth phase feature, deferred |

---

### External Integration (FR87-FR94) - 5% Implemented

| Requirement | Status | Evidence |
|------------|--------|----------|
| FR87: RESTful API | ⚠️ Partial | Gin routers initialized, NO endpoints implemented |
| FR88: OpenAPI spec | ⚠️ Partial | Swaggo in root backend, NOT integrated with apps backend |
| FR89-FR94: Webhooks, Plex/Jellyfin, mobile | ❌ Missing | Growth phase features |

---

## Gap Category 2: Architectural Decisions vs Implementation

**From Step 4 - Core Architectural Decisions:**

### Decision #1: Tailwind CSS - ❌ Not Implemented

**Required:**
- Install Tailwind CSS v3.x
- Configure `tailwind.config.js`
- Set up PostCSS pipeline

**Current State:**
- No Tailwind installed
- No CSS framework configured
- Basic styles only

**Impact:** Frontend development blocked

---

### Decision #2: Testing Infrastructure - ❌ Not Implemented

**Required:**
- Backend: Go testing + testify
- Frontend: Vitest + React Testing Library
- Coverage gates: Backend >80%, Frontend >70%

**Current State:**
- Zero test files exist
- No testing configuration
- No CI pipeline

**Impact:** Quality gates missing, TDD impossible

---

### Decision #3: JWT Authentication - ❌ Not Implemented

**Required:**
- `golang-jwt/jwt` v5.x integration
- Authentication middleware
- bcrypt password hashing
- `users` table in database

**Current State:**
- No JWT library installed
- No auth middleware
- No `users` table
- No password hashing

**Impact:** Security requirement (NFR-S9-S13) unmet, ALL endpoints unprotected

---

### Decision #4: Caching Strategy - ❌ Not Implemented

**Required:**
- Tiered cache (memory + SQLite)
- `cache_entries` table
- CacheManager implementation
- TTL management

**Current State:**
- No caching system exists
- No `cache_entries` table
- No cache library (bigcache/ristretto)

**Impact:** Performance degradation, AI API cost explosion, NFR-I7/I10 unmet

---

### Decision #5: Background Tasks - ❌ Not Implemented

**Required:**
- Worker pool with goroutines + channels
- Task types: AI parsing, metadata refresh, backups
- Retry logic with exponential backoff

**Current State:**
- No worker pool implementation
- No background task system
- No job queue

**Impact:** AI parsing blocks UI (10s wait), no async operations, NFR violated

---

### Decision #6: Error Handling & Logging - ❌ Partially Implemented

**Required:**
- Go `slog` standard library (NOT zerolog)
- `AppError` unified error types
- Sensitive data filtering

**Current State:**
- Root backend uses **zerolog** (non-compliant with Decision #6)
- Apps backend uses basic logging
- No `AppError` type
- No structured error codes

**Impact:** Inconsistent error messages, security risk (API key logging), debugging difficulty

---

## Gap Summary by Priority

**🔴 Critical Gaps (Block MVP):**

1. **Backend Consolidation** - Dual architecture prevents coherent development
2. **Authentication System** - All endpoints unprotected, security violation
3. **Caching System** - Performance and cost requirements unmet
4. **Filename Parser** - Core differentiator missing (standard regex + AI)
5. **Multi-Source Metadata** - TMDb exists, Douban/Wikipedia/AI missing
6. **Search UI** - Core user feature missing
7. **Media Library UI** - Core user feature missing

**🟡 Important Gaps (Affect Quality):**

8. **Testing Infrastructure** - Quality gates missing
9. **Background Task System** - UI blocking operations
10. **Error Handling Compliance** - Using zerolog instead of slog
11. **Download Monitor UI** - Important feature incomplete
12. **Settings UI** - Configuration management missing

**🟢 Deferred Gaps (Post-MVP):**

13. **Subtitle Automation** - Growth phase
14. **Watch Folder** - Growth phase
15. **Webhooks & External Integrations** - Growth phase

---
