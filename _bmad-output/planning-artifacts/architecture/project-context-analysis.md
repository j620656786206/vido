# Project Context Analysis

## Requirements Overview

**Functional Requirements:**

Vido has 94 functional requirements organized across three development phases (MVP/1.0/Growth), categorized into nine capability areas:

1. **Media Search & Discovery (FR1-FR10):** Core search functionality for movies/TV shows with Traditional Chinese metadata priority, grid/list views, filtering, and smart recommendations
2. **Filename Parsing & Metadata Retrieval (FR11-FR26):** ⭐ **Core Innovation** - Standard regex parsing (MVP) evolving to AI-powered fansub naming parsing (1.0) with four-layer fallback mechanism (TMDb → Douban → Wikipedia → AI → Manual)
3. **Download Integration & Monitoring (FR27-FR37):** qBittorrent real-time status monitoring, connection health tracking, and download control
4. **Media Library Management (FR38-FR46):** Browsing, batch operations, recently added items, watch history tracking
5. **System Configuration & Management (FR47-FR66):** Docker deployment, setup wizard, cache management, backup/restore, NFO export, performance monitoring
6. **User Authentication & Access Control (FR67-FR74):** Password/PIN authentication, session management, API token protection, multi-user preparation (architecture)
7. **Subtitle Management (FR75-FR80):** Growth phase - Traditional Chinese subtitle priority from OpenSubtitles and Zimuku
8. **Automation & Organization (FR81-FR86):** Growth phase - Watch folder monitoring, auto-parsing, file renaming/organization
9. **External Integration & Extensibility (FR87-FR94):** Growth phase - RESTful API, webhooks, Plex/Jellyfin sync, mobile app

**Architectural Implications of FRs:**
- **Multi-source orchestration required:** Coordinate TMDb API, Douban scraper, Wikipedia MediaWiki API, AI APIs (Gemini/Claude)
- **Caching strategy essential:** 24-hour metadata cache, 30-day AI parsing cache, permanent local storage
- **Background job processing:** AI parsing (10-second async), auto-retry mechanisms, scheduled backups
- **Repository pattern mandatory:** Database abstraction layer from MVP to enable PostgreSQL migration path
- **API-first design:** All frontend operations through versioned RESTful API for future extensibility

**Non-Functional Requirements:**

75+ NFRs across seven critical quality dimensions that will drive architectural decisions:

**Performance (18 NFRs):**
- API response times: <500ms (p95) for search, <300ms for library listing, <200ms for downloads
- Page load: FCP <1.5s, LCP <2.5s, TTI <3.5s, CLS <0.1
- Real-time updates: qBittorrent status <5s, media library updates <30s
- Parsing: Standard regex <100ms, AI parsing <10s
- Frontend: 60 FPS scrolling, <200ms routing, <500KB initial bundle (gzipped)
- **Architectural impact:** Virtual scrolling for large libraries, route-based code splitting, lazy loading, polling-based real-time updates

**Security (19 NFRs):**
- API key protection: Environment variable injection, AES-256 encryption, zero-logging policy
- Authentication: All endpoints require password/PIN, secure session tokens, rate limiting (100 req/min)
- External access: HTTPS support, VPN recommendation, no plain-text transmission
- Dependency scanning: Zero critical vulnerabilities, 7-day patch window
- **Architectural impact:** Encryption layer, authentication middleware, rate limiting middleware, secrets management system

**Scalability (10 NFRs):**
- SQLite supports up to 10,000 media items with <500ms query latency (p95)
- Warning at 8,000 items to recommend PostgreSQL migration
- Repository Pattern from MVP for zero-downtime database migration
- Virtual scrolling when library >1,000 items
- SQLite FTS5 for full-text search
- **Architectural impact:** Database abstraction layer, performance monitoring, graceful degradation thresholds

**Reliability (13 NFRs):**
- Uptime: >99.5% for self-hosted deployments
- Auto-retry with exponential backoff (1s → 2s → 4s → 8s)
- Graceful degradation when external APIs fail
- ACID-compliant transactions, atomic backups with checksums
- **Architectural impact:** Circuit breaker pattern, fallback chains, health check monitoring, automatic recovery mechanisms

**Integration (13 NFRs):**
- qBittorrent Web API v2.x with backward compatibility
- TMDb API v3 with zh-TW language priority, rate limit compliance (40 req/10s)
- AI provider abstraction (Gemini/Claude), 30-day result cache, 15s timeout
- Wikipedia MediaWiki API compliance (1 req/s, proper User-Agent)
- Versioned RESTful API (/api/v1), OpenAPI/Swagger spec
- **Architectural impact:** Adapter pattern for external services, provider abstraction layer, API versioning strategy

**Maintainability (13 NFRs):**
- Test coverage: Backend >80%, Frontend >70%
- Hot reload: Vite HMR (frontend), Air (backend)
- Database migrations: Versioned, automated
- Logging: Severity levels (ERROR/WARN/INFO/DEBUG), no sensitive data
- **Architectural impact:** Test infrastructure, CI/CD pipeline, migration system, observability layer

**Usability (9 NFRs):**
- Docker Compose deployment <5 minutes
- Setup wizard <5 steps with sensible defaults
- Actionable error messages with troubleshooting hints
- Keyboard navigation support
- **Architectural impact:** Configuration management, default values system, user feedback mechanisms

**Scale & Complexity:**

- **Primary domain:** Full-stack Web Application (SPA + RESTful API)
- **Complexity level:** Medium
  - 94 functional requirements across 3 phases
  - Multiple external integrations (qBittorrent, TMDb, Douban, Wikipedia, AI APIs)
  - Real-time monitoring needs
  - Self-learning mechanisms (filename mapping)
  - AI-powered intelligent parsing (market-first innovation)

- **Estimated architectural components:** 12-15 major components
  - **Frontend modules:** Search, Media Library, Download Monitor, Settings, Authentication, UX Components
  - **Backend services:** API Layer, Filename Parser, Multi-Source Metadata Retriever, qBittorrent Integration, Repository Layer, Cache Manager, Background Task Queue, Learning System
  - **Infrastructure:** Database (SQLite → PostgreSQL path), Backup System, Logging/Monitoring

## Technical Constraints & Dependencies

**Known Constraints:**

1. **Self-hosted deployment model:**
   - Users run on their own NAS/servers (no cloud infrastructure)
   - Docker containerization mandatory
   - Zero-config startup required for user-friendliness
   - No telemetry or analytics by default (privacy-first)

2. **Database scalability boundary:**
   - SQLite chosen for 1.0 (simplicity, single-file, no external dependencies)
   - Clear limitation: 10,000 media items maximum
   - Repository Pattern required from Day 1 to enable future PostgreSQL migration
   - WAL mode for concurrent read performance

3. **External API dependencies:**
   - **TMDb API:** Free tier with quota limits, requires user API key for higher quotas
   - **AI APIs (Gemini/Claude):** Users provide their own API keys (cost mitigation), 10-second response time target
   - **Douban:** Web scraping (no official API), anti-scraping countermeasures risk
   - **Wikipedia:** MediaWiki API with rate limits (1 req/s), compliance required
   - **qBittorrent:** Web API v2.x integration, connection health monitoring

4. **User-paid API model:**
   - Users bear AI API costs (own keys)
   - Per-file cost target: <$0.05 USD
   - Per-user monthly cost target: <$2 USD (assuming 50 new files/month)
   - Caching strategy critical to minimize costs

5. **Browser support matrix:**
   - Modern browsers only (Chrome, Firefox, Safari, Edge - latest versions)
   - Mobile browsers: iOS 15+, Android Chrome latest
   - No IE support, no polyfills (keep bundle lightweight)

6. **Technology stack foundations (from existing codebase):**
   - **Backend:** Go 1.21+, Gin framework, Air (hot reload), Swaggo (OpenAPI)
   - **Frontend:** React 19, TanStack Router, TanStack Query
   - **Build system:** Nx monorepo
   - **Deployment:** Docker, Docker Compose

**Critical Dependencies:**

- **qBittorrent:** Download monitoring requires qBittorrent Web API availability
- **TMDb API:** Primary metadata source (fallback to Douban/Wikipedia if unavailable)
- **AI Provider APIs:** Core differentiator (fansub parsing), must gracefully degrade if quota exhausted
- **External network access:** Required for metadata fetching, AI parsing (self-hosted means user's network reliability)

## Cross-Cutting Concerns Identified

These architectural concerns will affect multiple components and require consistent handling across the system:

**1. Error Handling & Resilience:**
- **Scope:** All external integrations (qBittorrent, TMDb, Douban, Wikipedia, AI APIs)
- **Strategy:** Circuit breaker pattern, multi-layer fallback chains, graceful degradation, automatic retry with exponential backoff
- **Impact:** Every external service call needs error handling, timeout management, fallback logic

**2. Caching Strategy:**
- **Scope:** Metadata retrieval, AI parsing results, image thumbnails
- **Requirements:**
  - TMDb metadata: 24-hour cache
  - AI parsing results: 30-day cache
  - Images: Permanent local cache with lazy loading
- **Impact:** Cache layer abstraction, TTL management, cache invalidation strategies, storage optimization

**3. Security & Secrets Management:**
- **Scope:** API keys (TMDb, Gemini, Claude), qBittorrent credentials, encryption keys, user passwords
- **Requirements:**
  - Environment variable priority
  - AES-256 encryption for UI-entered secrets
  - Zero-logging policy (no secrets in logs/errors)
  - Encrypted database backups
- **Impact:** Secrets management service, encryption layer, logging filters, audit trails

**4. Monitoring & Observability:**
- **Scope:** Service connection health, API usage tracking, performance metrics, error logging
- **Requirements:**
  - Real-time health checks (qBittorrent, TMDb, AI APIs)
  - Performance metrics (p95 latency, cache hit rate)
  - Cost tracking (AI API usage)
  - User action logging (for learning system)
- **Impact:** Metrics collection system, health check scheduler, dashboard data aggregation, logging infrastructure

**5. Database Migration Path:**
- **Scope:** All data access patterns
- **Requirements:**
  - Repository Pattern abstraction from MVP
  - Support SQLite (1.0) and PostgreSQL (future) implementations
  - Zero-downtime migration capability
  - Performance monitoring to trigger migration warnings (8,000 items threshold)
- **Impact:** Data access layer design, interface abstraction, migration tooling, performance instrumentation

**6. Internationalization Preparation:**
- **Scope:** All user-facing text, metadata display, error messages
- **Requirements:**
  - 1.0 focuses on Traditional Chinese (zh-TW)
  - Architecture must support i18n for future expansions
  - Metadata language priority (zh-TW → zh-CN → en)
- **Impact:** Localization framework integration, language detection, metadata source selection logic

**7. Background Task Processing:**
- **Scope:** AI parsing, metadata fetching, automatic retries, scheduled backups
- **Requirements:**
  - Non-blocking UI operations
  - Progress tracking and status updates
  - Task prioritization (user-initiated vs automatic)
  - Failure recovery and retry logic
- **Impact:** Job queue system, worker pool management, progress notification mechanisms

**8. API Versioning & Extensibility:**
- **Scope:** All REST API endpoints
- **Requirements:**
  - Versioned endpoints (/api/v1)
  - OpenAPI/Swagger documentation
  - Backward compatibility for configuration changes
  - Future webhook support for external automation
- **Impact:** API design patterns, version management strategy, documentation generation, contract testing

**9. User Experience Consistency:**
- **Scope:** All user interactions, especially wait states and error scenarios
- **Requirements:**
  - AI parsing 10-second wait experience (progress visualization)
  - Unified feedback patterns (loading, success, error states)
  - Actionable error messages with troubleshooting guidance
  - Desktop-optimized information density, mobile-simplified monitoring
- **Impact:** UI component library, state management patterns, progress tracking UI, responsive design system

**10. Cost Optimization:**
- **Scope:** AI API usage, external API calls
- **Requirements:**
  - Intelligent caching to minimize API calls
  - Smart triggering (only use AI when regex fails)
  - Usage monitoring and cost estimates
  - Degradation options (disable AI if costs too high)
- **Impact:** Cache hit optimization, trigger logic design, usage analytics, configurable thresholds
