---
stepsCompleted: ['step-01-validate-prerequisites', 'step-02-design-epics', 'step-03-create-stories', 'step-04-final-validation']
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/architecture.md'
  - '_bmad-output/planning-artifacts/ux-design-specification.md'
---

# vido - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for vido, decomposing the requirements from the PRD, UX Design, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

**Media Search & Discovery (MVP/1.0/Growth):**
- FR1: Users can search for movies and TV shows by title (Traditional Chinese or English)
- FR2: Users can view search results with Traditional Chinese metadata (title, description, release year, poster, genre, director, cast)
- FR3: Users can browse search results in grid view
- FR4: Users can view media item detail pages (read-only)
- FR5: Users can search within their saved media library
- FR6: Users can sort media library by date added, title, year, rating
- FR7: Users can filter media library by genre, year, media type
- FR8: Users can toggle between grid view and list view
- FR9: Users can receive smart recommendations based on genre, cast, director
- FR10: Users can see "similar titles" suggestions

**Filename Parsing & Metadata Retrieval (MVP/1.0):**
- FR11: System can parse standard naming convention filenames (e.g., `Movie.Name.2024.1080p.BluRay.mkv`)
- FR12: System can extract title, year, season/episode from filenames
- FR13: System can retrieve Traditional Chinese priority metadata from TMDb API
- FR14: System can store metadata to local database
- FR15: System can parse fansub naming conventions using AI (Gemini/Claude) (e.g., `[Leopard-Raws] Show - 01 (BD 1920x1080 x264 FLAC).mkv`)
- FR16: System can implement multi-source metadata fallback (TMDb → Douban → Wikipedia → AI → Manual)
- FR17: System can automatically switch to Douban web scraper when TMDb fails
- FR18: System can retrieve metadata from Wikipedia when TMDb and Douban fail
- FR19: System can use AI to re-parse filenames and generate alternative search keywords
- FR20: Users can manually search and select correct metadata
- FR21: Users can manually edit media item metadata
- FR22: Users can view parse status indicators (success/failure/processing)
- FR23: System can cache AI parsing results to reduce API costs
- FR24: System can learn from user manual corrections and remember filename mapping rules
- FR25: System can automatically retry when metadata sources are temporarily unavailable
- FR26: System can gracefully degrade when all sources fail and provide manual option

**Download Integration & Monitoring (1.0/Growth):**
- FR27: Users can connect to qBittorrent instance (enter host, username, password)
- FR28: Users can test qBittorrent connection
- FR29: System can monitor qBittorrent download status in real-time (progress, speed, ETA, status)
- FR30: Users can view download list in unified dashboard
- FR31: Users can filter downloads by status (downloading, paused, completed, seeding)
- FR32: System can detect completed downloads and trigger parsing
- FR33: System can display qBittorrent connection health status
- FR34: Users can control qBittorrent directly from Vido (pause/resume/delete torrents)
- FR35: Users can adjust download priority
- FR36: Users can manage bandwidth settings
- FR37: Users can schedule downloads

**Media Library Management (1.0/Growth):**
- FR38: Users can browse complete media library collection
- FR39: Users can view media detail pages (cast info, trailers, complete metadata)
- FR40: Users can perform batch operations on media items (delete, re-parse)
- FR41: Users can view recently added media items
- FR42: System can display metadata source indicators (TMDb/Douban/Wikipedia/AI/Manual)
- FR43: Users can track personal watch history
- FR44: System can display watch progress indicators
- FR45: Users can mark media as watched/unwatched
- FR46: Users can create custom collections of media items

**System Configuration & Management (MVP/1.0):**
- FR47: Users can deploy Vido via Docker container
- FR48: System can provide zero-config startup (sensible defaults)
- FR49: Users can configure media folder locations
- FR50: Users can configure API keys via environment variables
- FR51: System can store sensitive data in encrypted format (AES-256)
- FR52: Users can complete initial setup via setup wizard
- FR53: Users can manage cache (view cache size, clear old cache)
- FR54: Users can view system logs
- FR55: System can display service connection status (qBittorrent, TMDb, AI APIs)
- FR56: Users can receive automatic update notifications
- FR57: System can backup database and configuration
- FR58: Users can restore data from backup
- FR59: System can verify backup integrity (checksum)
- FR60: System can export metadata to JSON/YAML format
- FR61: System can import metadata from JSON/YAML
- FR62: System can export metadata as NFO files (Kodi/Plex/Jellyfin compatible)
- FR63: Users can configure backup schedule (daily/weekly)
- FR64: System can automatically cleanup old backups (retention policy)
- FR65: System can display performance metrics (query latency, cache hit rate)
- FR66: System can warn when approaching scalability limits (e.g., SQLite items >8,000)

**User Authentication & Access Control (1.0/Growth):**
- FR67: Users must authenticate via password/PIN to access Vido
- FR68: System can manage user sessions with secure tokens
- FR69: API endpoints must be protected with authentication tokens
- FR70: System can implement rate limiting to prevent abuse
- FR71: System can support multiple user accounts
- FR72: Administrators can manage user permissions (admin/user roles)
- FR73: Users can have personal watch history
- FR74: Users can have personal preference settings

**Subtitle Management (Growth):**
- FR75: Users can search for subtitles (OpenSubtitles and Zimuku)
- FR76: System can prioritize Traditional Chinese subtitles
- FR77: Users can download subtitle files
- FR78: Users can manually upload subtitle files
- FR79: System can automatically download subtitles (based on user preferences)
- FR80: System can display subtitle availability status

**Automation & Organization (Growth):**
- FR81: System can monitor watch folders to detect new files
- FR82: System can automatically trigger parsing when files are detected
- FR83: System can automatically rename files based on user-configured patterns
- FR84: System can automatically move files to organized directory structure
- FR85: System can execute automation tasks in background processing queue
- FR86: Users can configure automation rules (watch folders, naming patterns, target folders)

**External Integration & Extensibility (Growth):**
- FR87: System can provide RESTful API (versioned /api/v1)
- FR88: Developers can authenticate API requests with API tokens
- FR89: System can provide OpenAPI/Swagger documentation
- FR90: System can support webhook subscriptions for external automation
- FR91: Users can export metadata to Plex/Jellyfin
- FR92: System can sync watch status with Plex/Jellyfin
- FR93: Users can access Vido via mobile application (React Native/Flutter)
- FR94: Users can remotely control downloads from mobile device

### NonFunctional Requirements

**Performance (NFR-P1 to NFR-P18):**
- NFR-P1: Homepage FCP <1.5s on first visit
- NFR-P2: Homepage LCP <2.5s on first visit
- NFR-P3: Homepage TTI <3.5s on first visit
- NFR-P4: CLS <0.1 throughout session
- NFR-P5: Search API <500ms (p95)
- NFR-P6: Media library listing API <300ms (p95) for up to 1,000 items
- NFR-P7: Download status API <200ms (p95)
- NFR-P8: qBittorrent status updates <5 seconds
- NFR-P9: Media library updates <30 seconds after parsing
- NFR-P10: Grid scrolling 60 FPS
- NFR-P11: Page transitions <200ms
- NFR-P12: Image lazy loading <300ms per viewport
- NFR-P13: Standard regex parsing <100ms per file
- NFR-P14: AI fansub parsing <10 seconds per file
- NFR-P15: TMDb metadata retrieval <2 seconds per query
- NFR-P16: Wikipedia fallback retrieval <3 seconds per query
- NFR-P17: Initial bundle <500 KB (gzipped)
- NFR-P18: Route-based code splitting implemented

**Security (NFR-S1 to NFR-S19):**
- NFR-S1: API keys support environment variable injection
- NFR-S2: UI-stored API keys encrypted with AES-256
- NFR-S3: Encryption key from ENCRYPTION_KEY env var or machine-ID fallback
- NFR-S4: API keys never in logs/errors/HTTP responses (zero-logging)
- NFR-S5: API keys encrypted in database backups
- NFR-S6: Users can completely delete all personal data
- NFR-S7: Media library data remains local (no external reporting)
- NFR-S8: Privacy-first (no telemetry by default)
- NFR-S9: All endpoints require authentication (password/PIN)
- NFR-S10: Secure, cryptographically-signed session tokens
- NFR-S11: API endpoints protected with auth tokens
- NFR-S12: Rate limiting (100 requests/minute per IP)
- NFR-S13: Failed auth attempts logged and rate-limited (5 per 15 min)
- NFR-S14: HTTPS support for external access (reverse proxy compatible)
- NFR-S15: No plain text transmission of sensitive info
- NFR-S16: VPN recommendation in documentation
- NFR-S17: Dependency vulnerability scanning before releases
- NFR-S18: Critical vulnerabilities patched within 7 days
- NFR-S19: Zero critical security vulnerabilities in production

**Scalability (NFR-SC1 to NFR-SC10):**
- NFR-SC1: SQLite supports 10,000 media items with <500ms query latency (p95)
- NFR-SC2: Warning at 8,000 items recommending PostgreSQL
- NFR-SC3: Repository Pattern from MVP for zero-downtime migration
- NFR-SC4: Support 5 concurrent user sessions
- NFR-SC5: Proper database locking for concurrent access
- NFR-SC6: Virtual scrolling when library >1,000 items
- NFR-SC7: Image thumbnails cached locally
- NFR-SC8: SQLite FTS5 for full-text search <500ms
- NFR-SC9: Architecture supports horizontal scaling
- NFR-SC10: Database schema supports future user tables

**Reliability (NFR-R1 to NFR-R13):**
- NFR-R1: >99.5% uptime for self-hosted deployments
- NFR-R2: Graceful handling of all external API failures
- NFR-R3: Auto-retry TMDb → Douban within <1 second
- NFR-R4: Graceful degradation with manual search option
- NFR-R5: Exponential backoff retry (1s → 2s → 4s → 8s)
- NFR-R6: Auto-recover from qBittorrent failures (30s reconnection)
- NFR-R7: SQLite atomic backups (.backup command)
- NFR-R8: Backup integrity verification (checksum)
- NFR-R9: Auto-snapshot before restore
- NFR-R10: ACID-compliant transactions
- NFR-R11: AI quota exhausted → regex fallback with notification
- NFR-R12: qBittorrent unreachable → show status without blocking
- NFR-R13: Core functionality works when external APIs down

**Integration (NFR-I1 to NFR-I18):**
- NFR-I1: qBittorrent Web API v2.x with backward compatibility
- NFR-I2: Connection health detection within <10 seconds
- NFR-I3: Support qBittorrent behind reverse proxy
- NFR-I4: Encrypted credential storage, never logged
- NFR-I5: TMDb API v3 with zh-TW language priority
- NFR-I6: Respect TMDb rate limits (40 req/10s)
- NFR-I7: TMDb response cache 24 hours
- NFR-I8: TMDb v3 → v4 migration plan
- NFR-I9: AI provider abstraction (Gemini/Claude)
- NFR-I10: AI parsing cache 30 days
- NFR-I11: Per-user AI API usage tracking
- NFR-I12: AI API 15s timeout with fallback
- NFR-I13: Wikipedia MediaWiki API compliance (User-Agent)
- NFR-I14: Wikipedia rate limit (1 req/s)
- NFR-I15: Robust Infobox template parsing
- NFR-I16: Versioned API (/api/v1)
- NFR-I17: OpenAPI/Swagger spec
- NFR-I18: Webhook support for events

**Maintainability (NFR-M1 to NFR-M13):**
- NFR-M1: Backend test coverage >80%
- NFR-M2: Frontend test coverage >70%
- NFR-M3: Public functions documented
- NFR-M4: Linting rules enforced (ESLint, golangci-lint)
- NFR-M5: Hot reload (Vite HMR, Air)
- NFR-M6: Versioned, automated database migrations
- NFR-M7: Environment variable configuration support
- NFR-M8: Docker image version pinning
- NFR-M9: Automatic migration scripts
- NFR-M10: Zero-downtime config updates
- NFR-M11: Severity-level logging (ERROR/WARN/INFO/DEBUG)
- NFR-M12: Performance metrics queryable
- NFR-M13: Health status visible on homepage

**Usability (NFR-U1 to NFR-U9):**
- NFR-U1: Docker Compose deployment <5 minutes
- NFR-U2: Setup wizard <5 steps
- NFR-U3: Sensible defaults (zero manual config for basic usage)
- NFR-U4: Clear feedback within <200ms
- NFR-U5: Actionable error messages
- NFR-U6: Keyboard navigation support
- NFR-U7: Auto-generated API documentation
- NFR-U8: Quick start guide, troubleshooting, FAQ
- NFR-U9: Error logs with troubleshooting hints

### Additional Requirements

**From Architecture:**
- ARCH-1: Repository Pattern mandatory from MVP (database abstraction for SQLite → PostgreSQL migration)
- ARCH-2: Multi-source metadata orchestrator (TMDb → Douban → Wikipedia → AI → Manual fallback chain)
- ARCH-3: AI Provider Abstraction Layer (support Gemini/Claude switching, 30-day caching)
- ARCH-4: Background Task Queue (AI parsing 10s non-blocking, auto-retry, scheduled backups)
- ARCH-5: Cache Management System (multi-tier: metadata 24h, AI results 30d, images permanent)
- ARCH-6: Secrets Management Service (env var priority, AES-256 encryption, zero-logging)
- ARCH-7: Circuit Breaker Pattern (protect external service calls, implement fallback logic)
- ARCH-8: Health Check Scheduler (monitor qBittorrent, TMDb, AI APIs connection health)
- ARCH-9: API Versioning Strategy (versioned endpoints /api/v1 for backward compatibility)
- ARCH-10: Brownfield project with existing stack: Go 1.21+/Gin, React 19/TanStack Router/TanStack Query, Nx monorepo, Docker

**From UX Design:**
- UX-1: Desktop-first design with multi-column layout, hover interactions, keyboard shortcuts
- UX-2: Mobile simplified monitoring (single column, quick actions, push notifications)
- UX-3: AI parsing wait experience (progress visualization, step indicators, non-blocking UI)
- UX-4: Failure handling friendliness (always show next step, explain reasons, multi-layer fallback visible)
- UX-5: Learning system feedback (show when system applies learned rules: "已套用你之前的設定")
- UX-6: Activity Monitor Center (background task visibility, pause/cancel options, detailed logs)
- UX-7: Minimal onboarding (3-5 steps, visual guides, skip non-essential settings)
- UX-8: Hover over Click principle (information preview on hover, click for deep dive)
- UX-9: Dual-core experience loops: Discovery Loop (search → download → parse → display) and Appreciation Loop (browse → see zh-TW posters → select content)
- UX-10: Emotional design goals: Empowered/In Control, Delighted/Surprised, Efficient/Productive, Understood/Supported

### FR Coverage Map

| FR | Epic | Description |
|----|------|-------------|
| FR1 | Epic 2 | Search movies/TV shows by title |
| FR2 | Epic 2 | View search results with zh-TW metadata |
| FR3 | Epic 2 | Browse search results in grid view |
| FR4 | Epic 2 | View media item detail pages |
| FR5 | Epic 5 | Search within saved media library |
| FR6 | Epic 5 | Sort media library |
| FR7 | Epic 5 | Filter media library |
| FR8 | Epic 5 | Toggle grid/list view |
| FR9 | Epic 10 | Smart recommendations |
| FR10 | Epic 10 | Similar titles suggestions |
| FR11 | Epic 2 | Parse standard naming filenames |
| FR12 | Epic 2 | Extract title/year/episode from filenames |
| FR13 | Epic 2 | Retrieve zh-TW metadata from TMDb |
| FR14 | Epic 2 | Store metadata to local database |
| FR15 | Epic 3 | Parse fansub naming with AI |
| FR16 | Epic 3 | Multi-source metadata fallback |
| FR17 | Epic 3 | Auto-switch to Douban when TMDb fails |
| FR18 | Epic 3 | Retrieve metadata from Wikipedia |
| FR19 | Epic 3 | AI re-parse and alternative keywords |
| FR20 | Epic 3 | Manual search and select metadata |
| FR21 | Epic 3 | Manual edit media metadata |
| FR22 | Epic 3 | View parse status indicators |
| FR23 | Epic 3 | Cache AI parsing results |
| FR24 | Epic 3 | Learn from user corrections |
| FR25 | Epic 3 | Auto-retry when sources unavailable |
| FR26 | Epic 3 | Graceful degradation with manual option |
| FR27 | Epic 4 | Connect to qBittorrent instance |
| FR28 | Epic 4 | Test qBittorrent connection |
| FR29 | Epic 4 | Monitor download status real-time |
| FR30 | Epic 4 | View download list in dashboard |
| FR31 | Epic 4 | Filter downloads by status |
| FR32 | Epic 4 | Detect completed downloads and trigger parsing |
| FR33 | Epic 4 | Display connection health status |
| FR34 | Epic 8 | Control qBittorrent (pause/resume/delete) |
| FR35 | Epic 8 | Adjust download priority |
| FR36 | Epic 8 | Manage bandwidth settings |
| FR37 | Epic 8 | Schedule downloads |
| FR38 | Epic 5 | Browse complete media library |
| FR39 | Epic 5 | View media detail pages with cast/trailers |
| FR40 | Epic 5 | Batch operations on media items |
| FR41 | Epic 5 | View recently added media |
| FR42 | Epic 5 | Display metadata source indicators |
| FR43 | Epic 11 | Track personal watch history |
| FR44 | Epic 11 | Display watch progress indicators |
| FR45 | Epic 11 | Mark media watched/unwatched |
| FR46 | Epic 11 | Create custom collections |
| FR47 | Epic 1 | Deploy via Docker container |
| FR48 | Epic 1 | Zero-config startup |
| FR49 | Epic 1 | Configure media folder locations |
| FR50 | Epic 1 | Configure API keys via env vars |
| FR51 | Epic 1 | Store sensitive data encrypted |
| FR52 | Epic 6 | Complete initial setup via wizard |
| FR53 | Epic 6 | Manage cache |
| FR54 | Epic 6 | View system logs |
| FR55 | Epic 6 | Display service connection status |
| FR56 | Epic 6 | Receive update notifications |
| FR57 | Epic 6 | Backup database and config |
| FR58 | Epic 6 | Restore data from backup |
| FR59 | Epic 6 | Verify backup integrity |
| FR60 | Epic 6 | Export metadata to JSON/YAML |
| FR61 | Epic 6 | Import metadata from JSON/YAML |
| FR62 | Epic 6 | Export metadata as NFO files |
| FR63 | Epic 6 | Configure backup schedule |
| FR64 | Epic 6 | Auto-cleanup old backups |
| FR65 | Epic 6 | Display performance metrics |
| FR66 | Epic 6 | Warn on scalability limits |
| FR67 | Epic 7 | Authenticate via password/PIN |
| FR68 | Epic 7 | Manage user sessions |
| FR69 | Epic 7 | Protect API endpoints |
| FR70 | Epic 7 | Implement rate limiting |
| FR71 | Epic 13 | Support multiple user accounts |
| FR72 | Epic 13 | Manage user permissions |
| FR73 | Epic 13 | Personal watch history per user |
| FR74 | Epic 13 | Personal preference settings |
| FR75 | Epic 9 | Search for subtitles |
| FR76 | Epic 9 | Prioritize zh-TW subtitles |
| FR77 | Epic 9 | Download subtitle files |
| FR78 | Epic 9 | Manually upload subtitles |
| FR79 | Epic 9 | Auto-download subtitles |
| FR80 | Epic 9 | Display subtitle availability |
| FR81 | Epic 12 | Monitor watch folders |
| FR82 | Epic 12 | Auto-trigger parsing on new files |
| FR83 | Epic 12 | Auto-rename files |
| FR84 | Epic 12 | Auto-move files to organized structure |
| FR85 | Epic 12 | Background task queue |
| FR86 | Epic 12 | Configure automation rules |
| FR87 | Epic 14 | Provide RESTful API |
| FR88 | Epic 14 | Authenticate API requests |
| FR89 | Epic 14 | Provide OpenAPI/Swagger docs |
| FR90 | Epic 14 | Support webhook subscriptions |
| FR91 | Epic 14 | Export to Plex/Jellyfin |
| FR92 | Epic 14 | Sync watch status |
| FR93 | Epic 14 | Mobile application |
| FR94 | Epic 14 | Remote download control |

## Epic List

### Epic 1: Project Foundation & Docker Deployment
**Phase:** MVP (Q1 - March 2026)

Users can deploy Vido on their NAS within 5 minutes using Docker Compose, with zero-configuration startup and sensible defaults. The foundation includes encrypted storage for sensitive data and environment variable configuration support.

**FRs covered:** FR47, FR48, FR49, FR50, FR51

**Implementation Notes:**
- ARCH-1: Repository Pattern mandatory from Day 1
- ARCH-10: Brownfield project with existing Go/Gin + React 19/TanStack stack
- NFR-U1: Docker Compose deployment <5 minutes
- NFR-SC3: Repository Pattern for SQLite → PostgreSQL migration path

---

### Epic 2: Media Search & Traditional Chinese Metadata
**Phase:** MVP (Q1 - March 2026)

Users can search for movies and TV shows by title (Traditional Chinese or English), view search results with beautiful Traditional Chinese metadata (title, poster, description, genre, cast), and browse results in a responsive grid view. The system can parse standard naming convention filenames and retrieve metadata from TMDb with zh-TW priority.

**FRs covered:** FR1, FR2, FR3, FR4, FR11, FR12, FR13, FR14

**Implementation Notes:**
- NFR-I5: TMDb API v3 with zh-TW language priority
- NFR-I6: Respect TMDb rate limits (40 req/10s)
- NFR-I7: TMDb response cache 24 hours
- NFR-P5: Search API <500ms (p95)
- UX-1: Desktop-first design with hover interactions
- UX-8: Hover over Click principle

---

### Epic 3: AI-Powered Fansub Parsing & Multi-Source Fallback
**Phase:** 1.0 (Q2 - June 2026)

Users can parse complex fansub naming conventions (e.g., `[Leopard-Raws]`, `【幻櫻字幕組】`) using AI (Gemini/Claude). The system implements a four-layer fallback mechanism (TMDb → Douban → Wikipedia → AI → Manual), automatically retries when sources fail, learns from user corrections, and provides graceful degradation with clear next steps.

**FRs covered:** FR15, FR16, FR17, FR18, FR19, FR20, FR21, FR22, FR23, FR24, FR25, FR26

**Implementation Notes:**
- ARCH-2: Multi-source metadata orchestrator
- ARCH-3: AI Provider Abstraction Layer (Gemini/Claude switching, 30-day cache)
- ARCH-7: Circuit Breaker Pattern for external services
- NFR-P14: AI fansub parsing <10 seconds per file
- NFR-R3: Auto-retry TMDb → Douban within <1 second
- UX-3: AI parsing wait experience (progress visualization)
- UX-4: Failure handling friendliness
- UX-5: Learning system feedback

---

### Epic 4: qBittorrent Download Monitoring
**Phase:** 1.0 (Q2 - June 2026)

Users can connect to their qBittorrent instance, monitor download status in real-time (progress, speed, ETA, status), view a unified download dashboard, filter downloads by status, and see connection health indicators. The system automatically detects completed downloads and triggers parsing.

**FRs covered:** FR27, FR28, FR29, FR30, FR31, FR32, FR33

**Implementation Notes:**
- NFR-I1: qBittorrent Web API v2.x with backward compatibility
- NFR-I2: Connection health detection within <10 seconds
- NFR-I3: Support qBittorrent behind reverse proxy
- NFR-P8: qBittorrent status updates <5 seconds
- NFR-R6: Auto-recover from qBittorrent failures (30s reconnection)
- ARCH-8: Health Check Scheduler

---

### Epic 5: Media Library Management
**Phase:** 1.0 (Q2 - June 2026)

Users can browse their complete media library collection, search within the library, sort by date/title/year/rating, filter by genre/year/type, toggle between grid and list views, view detailed media pages with cast info and trailers, perform batch operations, and see metadata source indicators.

**FRs covered:** FR5, FR6, FR7, FR8, FR38, FR39, FR40, FR41, FR42

**Implementation Notes:**
- NFR-SC6: Virtual scrolling when library >1,000 items
- NFR-SC8: SQLite FTS5 for full-text search <500ms
- NFR-P6: Media library listing API <300ms (p95)
- NFR-P10: Grid scrolling 60 FPS
- UX-9: Appreciation Loop (browse → see zh-TW posters → select content)

---

### Epic 6: System Configuration & Backup
**Phase:** 1.0 (Q2 - June 2026)

Users can complete initial setup via a guided wizard (<5 steps), manage cache, view system logs, see service connection status, receive update notifications, backup/restore data with integrity verification, export/import metadata (JSON/YAML/NFO), configure backup schedules, and view performance metrics with scalability warnings.

**FRs covered:** FR52, FR53, FR54, FR55, FR56, FR57, FR58, FR59, FR60, FR61, FR62, FR63, FR64, FR65, FR66

**Implementation Notes:**
- ARCH-4: Background Task Queue for scheduled backups
- ARCH-5: Cache Management System (multi-tier)
- ARCH-6: Secrets Management Service
- NFR-U2: Setup wizard <5 steps
- NFR-R7: SQLite atomic backups (.backup command)
- NFR-R8: Backup integrity verification (checksum)
- UX-6: Activity Monitor Center
- UX-7: Minimal onboarding

---

### Epic 7: User Authentication & Access Control
**Phase:** 1.0 (Q2 - June 2026)

Users must authenticate via password/PIN to access Vido. The system manages secure sessions with cryptographically-signed tokens, protects all API endpoints, and implements rate limiting to prevent abuse.

**FRs covered:** FR67, FR68, FR69, FR70

**Implementation Notes:**
- NFR-S9: All endpoints require authentication
- NFR-S10: Secure, cryptographically-signed session tokens
- NFR-S11: API endpoints protected with auth tokens
- NFR-S12: Rate limiting (100 requests/minute per IP)
- NFR-S13: Failed auth attempts rate-limited (5 per 15 min)

---

### Epic 8: Advanced Download Control
**Phase:** Growth (Q3+ - September 2026+)

Users can control qBittorrent directly from Vido (pause/resume/delete torrents), adjust download priorities, manage bandwidth settings, and schedule downloads.

**FRs covered:** FR34, FR35, FR36, FR37

**Implementation Notes:**
- Builds upon Epic 4 (qBittorrent Monitoring)
- Extends qBittorrent API integration with write operations

---

### Epic 9: Subtitle Integration
**Phase:** Growth (Q3+ - September 2026+)

Users can search for subtitles (OpenSubtitles and Zimuku), with Traditional Chinese subtitles prioritized. Users can download subtitle files, manually upload subtitles, enable automatic subtitle downloads, and see subtitle availability status.

**FRs covered:** FR75, FR76, FR77, FR78, FR79, FR80

**Implementation Notes:**
- Key pain point from UX research: subtitle timeline matching
- Consider AI-assisted subtitle matching in future iterations

---

### Epic 10: Smart Recommendations & Discovery
**Phase:** Growth (Q3+ - September 2026+)

Users can receive smart recommendations based on genre, cast, and director, and see "similar titles" suggestions to discover new content.

**FRs covered:** FR9, FR10

**Implementation Notes:**
- Builds upon media library data from Epic 5
- Recommendation engine based on user's collection patterns

---

### Epic 11: Watch History & Collections
**Phase:** Growth (Q3+ - September 2026+)

Users can track personal watch history, see watch progress indicators, mark media as watched/unwatched, and create custom collections of media items.

**FRs covered:** FR43, FR44, FR45, FR46

**Implementation Notes:**
- Foundation for future multi-user personal tracking (Epic 13)
- Syncs with Plex/Jellyfin in Epic 14

---

### Epic 12: Automation & Organization
**Phase:** Growth (Q3+ - September 2026+)

The system can monitor watch folders to detect new files, automatically trigger parsing, rename files based on user-configured patterns, move files to organized directory structures, and execute automation tasks in background processing queue.

**FRs covered:** FR81, FR82, FR83, FR84, FR85, FR86

**Implementation Notes:**
- ARCH-4: Background Task Queue
- Builds upon AI parsing from Epic 3

---

### Epic 13: Multi-User Support
**Phase:** Growth (Q3+ - September 2026+)

The system supports multiple user accounts with admin/user permission management. Each user has their own personal watch history and preference settings.

**FRs covered:** FR71, FR72, FR73, FR74

**Implementation Notes:**
- NFR-SC4: Support 5 concurrent user sessions
- NFR-SC10: Database schema supports future user tables
- Extends Epic 7 authentication system

---

### Epic 14: External API & Mobile Application
**Phase:** Growth (Q3+ - September 2026+)

The system provides a versioned RESTful API (/api/v1) with OpenAPI/Swagger documentation, supports webhook subscriptions for external automation, enables metadata export to Plex/Jellyfin with watch status sync. Users can access Vido via mobile application and remotely control downloads.

**FRs covered:** FR87, FR88, FR89, FR90, FR91, FR92, FR93, FR94

**Implementation Notes:**
- ARCH-9: API Versioning Strategy
- NFR-I16: Versioned API (/api/v1)
- NFR-I17: OpenAPI/Swagger spec
- NFR-I18: Webhook support for events
- UX-2: Mobile simplified monitoring

---

# Epic Details with Stories

## Epic 1: Project Foundation & Docker Deployment

**Phase:** MVP (Q1 - March 2026)

**Goal:** Users can deploy Vido on their NAS within 5 minutes using Docker Compose, with zero-configuration startup and sensible defaults.

### Story 1.1: Repository Pattern Database Abstraction Layer

As a **developer**,
I want a **Repository Pattern abstraction layer for database operations**,
So that **we can migrate from SQLite to PostgreSQL in the future without changing business logic**.

**Acceptance Criteria:**

**Given** the application needs database operations
**When** the developer implements data access code
**Then** all database operations go through repository interfaces (MediaRepository, ConfigRepository)
**And** the SQLite implementation is provided as the default
**And** the interface design supports future PostgreSQL implementation
**And** WAL mode is enabled for SQLite concurrent read performance

**Given** a new entity needs to be persisted
**When** the developer creates the repository method
**Then** the method signature is database-agnostic
**And** only the SQLite-specific implementation contains SQL syntax

**Technical Notes:**
- Implements ARCH-1: Repository Pattern mandatory from MVP
- Implements NFR-SC3: Zero-downtime migration path
- Database schema versioning with migrations table

---

### Story 1.2: Docker Compose Production Configuration

As a **NAS user**,
I want to **deploy Vido using a single docker-compose command**,
So that **I can have the application running within 5 minutes without complex setup**.

**Acceptance Criteria:**

**Given** a user has Docker and Docker Compose installed
**When** they run `docker-compose up -d`
**Then** the Vido container starts successfully within 60 seconds
**And** the web interface is accessible at `http://localhost:8080`
**And** data persists across container restarts via volume mounts

**Given** the container is running
**When** the user checks container health
**Then** a health check endpoint returns status 200
**And** the container reports as "healthy" in Docker

**Given** no environment variables are set
**When** the container starts
**Then** it uses sensible defaults for all configuration
**And** the application is functional without any manual configuration

**Technical Notes:**
- Volume mounts: `/vido-data` (database, cache), `/vido-backups` (backups), `/media` (read-only)
- Implements NFR-U1: Docker Compose deployment <5 minutes
- Implements FR47, FR48

---

### Story 1.3: Environment Variable Configuration System

As a **system administrator**,
I want to **configure Vido using environment variables**,
So that **I can customize the application without modifying files inside the container**.

**Acceptance Criteria:**

**Given** environment variables are set in docker-compose.yml
**When** the container starts
**Then** the application reads and applies all configuration from environment variables
**And** environment variables take precedence over config file values

**Given** the following environment variables are supported:
- `VIDO_PORT` (default: 8080)
- `VIDO_DATA_DIR` (default: /vido-data)
- `VIDO_MEDIA_DIRS` (comma-separated paths)
- `TMDB_API_KEY` (optional)
- `GEMINI_API_KEY` (optional)
- `ENCRYPTION_KEY` (optional, for secrets encryption)
**When** any variable is not set
**Then** the application uses the documented default value
**And** the application logs which configuration source is being used (env var vs default)

**Given** an invalid configuration value is provided
**When** the application starts
**Then** it logs a clear error message indicating the problem
**And** it exits with a non-zero status code (fail fast)

**Technical Notes:**
- Implements FR50: Configure API keys via environment variables
- Implements NFR-S1: API keys support environment variable injection
- Implements NFR-U3: Sensible defaults

---

### Story 1.4: Secrets Management with AES-256 Encryption

As a **security-conscious user**,
I want **my API keys and credentials encrypted when stored**,
So that **my sensitive data is protected even if the database file is accessed**.

**Acceptance Criteria:**

**Given** an API key is saved through the UI
**When** it is stored in the database
**Then** it is encrypted using AES-256-GCM encryption
**And** the plaintext key never appears in database files

**Given** the `ENCRYPTION_KEY` environment variable is set
**When** the application encrypts/decrypts secrets
**Then** it uses this key for encryption operations

**Given** the `ENCRYPTION_KEY` environment variable is NOT set
**When** the application needs an encryption key
**Then** it derives a key from the machine ID as fallback
**And** logs a warning recommending setting ENCRYPTION_KEY for better security

**Given** any application component logs data
**When** the log contains API keys or credentials
**Then** the sensitive values are masked (e.g., `TMDB_****1234`)
**And** the full value never appears in logs, errors, or HTTP responses

**Technical Notes:**
- Implements FR51: Store sensitive data in encrypted format
- Implements NFR-S2: AES-256 encryption for UI-stored keys
- Implements NFR-S3: Encryption key from env var or machine-ID
- Implements NFR-S4: Zero-logging policy for secrets

---

### Story 1.5: Media Folder Configuration

As a **NAS user**,
I want to **configure which folders contain my media files**,
So that **Vido knows where to scan for movies and TV shows**.

**Acceptance Criteria:**

**Given** the user sets `VIDO_MEDIA_DIRS=/movies,/tv,/anime`
**When** the application starts
**Then** it validates that each path exists and is accessible
**And** it stores the configured paths for future scanning operations

**Given** a configured media path does not exist
**When** the application starts
**Then** it logs a warning about the inaccessible path
**And** it continues starting with the valid paths (graceful degradation)

**Given** no media directories are configured
**When** the application starts
**Then** it logs a notice that no media directories are set
**And** the application starts successfully (search-only mode)

**Given** media directories are configured
**When** a user views the settings page
**Then** they see the list of configured media directories
**And** they see the accessibility status of each directory (accessible/not found)

**Technical Notes:**
- Implements FR49: Configure media folder locations
- Paths are read-only mounted in Docker (`/media:ro`)
- Supports multiple directories for different media types

---

## Epic 2: Media Search & Traditional Chinese Metadata

**Phase:** MVP (Q1 - March 2026)

**Goal:** Users can search for movies and TV shows by title (Traditional Chinese or English), view search results with beautiful Traditional Chinese metadata, and browse results in a responsive grid view.

### Story 2.1: TMDb API Integration with zh-TW Priority

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

### Story 2.2: Media Search Interface

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

### Story 2.3: Search Results Grid View

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

### Story 2.4: Media Detail Page

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

### Story 2.5: Standard Filename Parser (Regex-based)

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

### Story 2.6: Media Entity and Database Storage

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

## Epic 3: AI-Powered Fansub Parsing & Multi-Source Fallback

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can parse complex fansub naming conventions using AI, with a four-layer fallback mechanism ensuring metadata is always found or manual options provided.

### Story 3.1: AI Provider Abstraction Layer

As a **developer**,
I want an **abstraction layer for AI providers**,
So that **we can switch between Gemini and Claude without code changes**.

**Acceptance Criteria:**

**Given** the system needs AI parsing capabilities
**When** configuring the AI provider
**Then** users can select Gemini or Claude via environment variable `AI_PROVIDER`
**And** the same interface is used regardless of provider

**Given** an AI provider is configured
**When** making API calls
**Then** the system uses the appropriate API format for that provider
**And** responses are normalized to a common format

**Given** AI parsing results are returned
**When** caching the results
**Then** results are cached for 30 days (NFR-I10)
**And** cache key is based on filename hash

**Given** AI API calls are made
**When** the call exceeds 15 seconds
**Then** it times out and falls back to next option (NFR-I12)

**Technical Notes:**
- Implements ARCH-3: AI Provider Abstraction Layer
- Implements NFR-I9, NFR-I10, NFR-I12
- Strategy pattern for provider switching

---

### Story 3.2: AI Fansub Filename Parsing

As a **media collector with fansub content**,
I want the **system to parse complex fansub naming using AI**,
So that **files like `[Leopard-Raws] Show - 01 (BD 1080p).mkv` are correctly identified**.

**Acceptance Criteria:**

**Given** a fansub filename like `[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv`
**When** AI parsing is triggered
**Then** it extracts:
- Fansub group: Leopard-Raws (ignored for search)
- Title: Kimetsu no Yaiba / 鬼滅之刃
- Episode: 26
- Quality: 1080p
- Source: BD (Blu-ray)

**Given** a Chinese fansub filename like `【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4`
**When** AI parsing is triggered
**Then** it extracts:
- Title: 我的英雄學院
- Episode: 1
- Quality: 1080P
- Language: Traditional Chinese

**Given** AI parsing is in progress
**When** the user views the status
**Then** they see a progress indicator showing current step (UX-3)
**And** parsing completes within 10 seconds (NFR-P14)

**Technical Notes:**
- Implements FR15: Parse fansub naming with AI
- AI prompt engineered for fansub pattern recognition
- Uses Provider Abstraction from Story 3.1

---

### Story 3.3: Multi-Source Metadata Fallback Chain

As a **media collector**,
I want the **system to try multiple metadata sources automatically**,
So that **I always get metadata even when one source fails**.

**Acceptance Criteria:**

**Given** a search query is made
**When** TMDb returns no results
**Then** the system automatically tries Douban within 1 second (NFR-R3)
**And** the user sees "TMDb ❌ → Searching Douban..." status

**Given** both TMDb and Douban fail
**When** fallback continues
**Then** the system tries Wikipedia MediaWiki API
**And** respects Wikipedia rate limit (1 req/s, NFR-I14)

**Given** all automated sources fail
**When** the fallback chain completes
**Then** the user is offered manual search option
**And** the status shows all attempted sources: "TMDb ❌ → Douban ❌ → Wikipedia ❌ → Manual search"

**Given** any source succeeds
**When** metadata is found
**Then** the source is indicated (FR42): "Source: Douban"
**And** the fallback chain stops

**Technical Notes:**
- Implements FR16, FR17, FR18: Multi-source fallback
- Implements ARCH-2: Multi-source metadata orchestrator
- Implements ARCH-7: Circuit Breaker Pattern

---

### Story 3.4: Douban Web Scraper

As a **media collector with Asian content**,
I want **Douban as a fallback metadata source**,
So that **Chinese movies and shows not on TMDb can still be identified**.

**Acceptance Criteria:**

**Given** TMDb search fails
**When** Douban scraper is triggered
**Then** it searches Douban for the title
**And** extracts: Chinese title, year, director, cast, rating, poster URL

**Given** Douban anti-scraping measures are encountered
**When** the scraper detects blocking
**Then** it implements exponential backoff
**And** falls back to Wikipedia if blocked

**Given** Douban returns results
**When** displaying metadata
**Then** Traditional Chinese is prioritized
**And** the source is clearly labeled

**Technical Notes:**
- Implements FR17: Auto-switch to Douban
- Web scraping with proper rate limiting
- Respect robots.txt and implement polite scraping

---

### Story 3.5: Wikipedia Metadata Fallback

As a **media collector with obscure content**,
I want **Wikipedia as a third fallback source**,
So that **even rare titles can have basic metadata**.

**Acceptance Criteria:**

**Given** TMDb and Douban both fail
**When** Wikipedia fallback is triggered
**Then** it searches zh.wikipedia.org for the title
**And** extracts data from Infobox templates

**Given** Wikipedia article has an Infobox
**When** parsing the page
**Then** it extracts: title, director, cast, year, genre, plot summary
**And** handles multiple Infobox template variations (NFR-I15)

**Given** Wikipedia has no poster images
**When** displaying the media
**Then** a default placeholder icon is shown
**And** the user is notified: "No poster available from Wikipedia"

**Given** Wikipedia API is called
**When** making requests
**Then** proper User-Agent is set (NFR-I13)
**And** rate limit of 1 request/second is respected (NFR-I14)

**Technical Notes:**
- Implements FR18: Retrieve metadata from Wikipedia
- Implements NFR-I13, NFR-I14, NFR-I15
- Uses MediaWiki API for structured data access

---

### Story 3.6: AI Search Keyword Generation

As a **media collector**,
I want the **AI to generate alternative search keywords**,
So that **hard-to-find titles can be located through different search terms**.

**Acceptance Criteria:**

**Given** initial search fails on all sources
**When** AI keyword generation is triggered
**Then** AI analyzes the filename and generates:
- Original title
- English translation
- Japanese romaji (if applicable)
- Alternative spellings

**Given** alternative keywords are generated
**When** retrying metadata sources
**Then** each keyword variant is tried
**And** the first successful match is used

**Given** the filename is `鬼滅之刃.S01E26.mkv`
**When** TMDb search "鬼滅之刃" fails
**Then** AI generates alternatives:
- "鬼灭之刃" (Simplified)
- "Demon Slayer" (English)
- "Kimetsu no Yaiba" (Romaji)

**Technical Notes:**
- Implements FR19: AI re-parse and generate alternative keywords
- Layer 4 of the fallback architecture
- Increases metadata coverage from 98% to 99%+

---

### Story 3.7: Manual Metadata Search and Selection

As a **media collector**,
I want to **manually search and select the correct metadata**,
So that **I can fix misidentified or unfound titles**.

**Acceptance Criteria:**

**Given** automatic parsing fails
**When** the user clicks "Manual Search"
**Then** a search dialog opens
**And** they can enter a custom search query

**Given** manual search returns results
**When** the user views the results
**Then** they see poster, title, year, and description preview
**And** they can select the correct match

**Given** the user selects a match
**When** confirming the selection
**Then** the metadata is applied to the file
**And** the mapping is saved for learning (Story 3.9)

**Technical Notes:**
- Implements FR20: Manual search and select metadata
- Implements UX-4: Always show next step
- Part of the graceful degradation chain

---

### Story 3.8: Metadata Editor

As a **media collector**,
I want to **manually edit metadata for any media item**,
So that **I can correct errors or add missing information**.

**Acceptance Criteria:**

**Given** a media item in the library
**When** the user clicks "Edit Metadata"
**Then** an edit form opens with all editable fields:
- Title (Chinese/English)
- Year
- Genre
- Director
- Cast
- Description
- Poster (upload or URL)

**Given** the user modifies metadata
**When** saving changes
**Then** the changes are persisted to the database
**And** the source is updated to "Manual"

**Given** the user uploads a custom poster
**When** the upload completes
**Then** the image is resized and optimized
**And** stored in local cache

**Technical Notes:**
- Implements FR21: Manually edit media metadata
- Form validation for required fields
- Image processing for poster optimization

---

### Story 3.9: Filename Mapping Learning System

As a **power user**,
I want the **system to learn from my corrections**,
So that **similar filenames are automatically matched in the future**.

**Acceptance Criteria:**

**Given** the user manually corrects a filename match
**When** the correction is saved
**Then** the system asks: "Learn this pattern for future files?"
**And** if confirmed, stores the pattern-to-metadata mapping

**Given** a learned pattern exists
**When** a new file matches the pattern
**Then** the system automatically applies the learned mapping
**And** shows: "✓ 已套用你之前的設定" (UX-5)

**Given** the user views settings
**When** checking learned patterns
**Then** they see: "已記住 15 個自訂規則"
**And** can view, edit, or delete learned patterns

**Technical Notes:**
- Implements FR24: Learn from user corrections
- Implements UX-5: Learning system feedback
- Pattern matching with fuzzy matching support

---

### Story 3.10: Parse Status Indicators

As a **media collector**,
I want to **see clear status indicators for parsing progress**,
So that **I know what's happening with each file**.

**Acceptance Criteria:**

**Given** a file is being parsed
**When** viewing the file list
**Then** status icons indicate:
- ⏳ Parsing in progress
- ✅ Successfully parsed
- ⚠️ Parsed with warnings (manual selection needed)
- ❌ Parse failed

**Given** parsing is in progress
**When** viewing the progress
**Then** step indicators show: "解析檔名中..." → "搜尋 TMDb..." → "下載海報..."
**And** the current step is highlighted (UX-3)

**Given** parsing fails
**When** viewing the error
**Then** the failure reason is explained
**And** clear next steps are provided (UX-4)

**Technical Notes:**
- Implements FR22: View parse status indicators
- Implements UX-3, UX-4: Wait experience and failure handling

---

### Story 3.11: Auto-Retry Mechanism

As a **media collector**,
I want the **system to automatically retry when sources are temporarily unavailable**,
So that **temporary failures don't require my intervention**.

**Acceptance Criteria:**

**Given** a metadata source returns a temporary error
**When** the error is detected
**Then** the system automatically queues a retry
**And** uses exponential backoff: 1s → 2s → 4s → 8s (NFR-R5)

**Given** all retries fail
**When** the maximum attempts are reached
**Then** the file is marked for manual intervention
**And** the user is notified via the activity monitor

**Given** the source recovers
**When** the retry succeeds
**Then** the metadata is applied
**And** the file status updates automatically

**Technical Notes:**
- Implements FR25: Auto-retry when sources unavailable
- Implements NFR-R5: Exponential backoff
- Maximum 4 retry attempts before manual fallback

---

### Story 3.12: Graceful Degradation

As a **media collector**,
I want the **system to never completely fail**,
So that **I always have options even in worst-case scenarios**.

**Acceptance Criteria:**

**Given** all metadata sources fail
**When** the user views the file
**Then** they see:
- The original filename
- "Unable to auto-identify" message
- Three clear options: Manual search, Edit filename, Skip for now

**Given** the AI service is down
**When** parsing a fansub filename
**Then** the system falls back to regex parsing
**And** notifies: "AI 服務暫時無法使用，使用基本解析" (NFR-R11)

**Given** core functionality is needed
**When** all external APIs are unavailable
**Then** the library browsing and search still work (NFR-R13)
**And** only new metadata fetching is affected

**Technical Notes:**
- Implements FR26: Graceful degradation with manual option
- Implements NFR-R11, NFR-R13: Fallback behaviors
- Core principle: Never leave user stuck

---

## Epic 4: qBittorrent Download Monitoring

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can connect to their qBittorrent instance and monitor downloads in real-time from a unified dashboard.

### Story 4.1: qBittorrent Connection Configuration

As a **NAS user**,
I want to **connect Vido to my qBittorrent instance**,
So that **I can monitor downloads from within Vido**.

**Acceptance Criteria:**

**Given** the user navigates to qBittorrent settings
**When** they enter:
- Host URL (e.g., `http://192.168.1.100:8080`)
- Username
- Password
**Then** credentials are encrypted before storage (FR51)
**And** credentials never appear in logs (NFR-I4)

**Given** credentials are entered
**When** the user clicks "Test Connection"
**Then** the system verifies connectivity within 10 seconds (NFR-I2)
**And** shows success or detailed error message

**Given** qBittorrent is behind a reverse proxy
**When** configuring the connection
**Then** custom base paths are supported (NFR-I3)
**And** HTTPS connections work properly

**Technical Notes:**
- Implements FR27, FR28: Connect and test qBittorrent
- Implements NFR-I1, NFR-I2, NFR-I3, NFR-I4
- Uses qBittorrent Web API v2.x

---

### Story 4.2: Real-Time Download Status Monitoring

As a **media collector**,
I want to **see real-time download status**,
So that **I can monitor progress without opening qBittorrent**.

**Acceptance Criteria:**

**Given** qBittorrent is connected
**When** viewing the downloads dashboard
**Then** all torrents are displayed with:
- Name
- Progress percentage
- Download/upload speed
- ETA
- Status (downloading, paused, seeding, completed)

**Given** a torrent is active
**When** 5 seconds pass (NFR-P8)
**Then** the status updates automatically
**And** the UI updates without full page refresh

**Given** polling is active
**When** the user navigates away from the downloads page
**Then** polling stops to conserve resources
**And** resumes when they return

**Technical Notes:**
- Implements FR29: Real-time download status
- Implements NFR-P8: Updates within 5 seconds
- Uses TanStack Query polling with refetchInterval

---

### Story 4.3: Unified Download Dashboard

As a **media collector**,
I want a **unified dashboard showing downloads and recent media**,
So that **I can see my complete workflow in one place**.

**Acceptance Criteria:**

**Given** the user opens the homepage
**When** the dashboard loads
**Then** they see:
- Left panel: qBittorrent download list
- Right panel: Recently added media
- Bottom: Quick search bar

**Given** downloads and media are displayed
**When** a download completes
**Then** the completed item moves from "Downloads" to "Recent Media" after parsing
**And** a notification indicates successful addition

**Given** qBittorrent is disconnected
**When** viewing the dashboard
**Then** the download panel shows connection status
**And** other panels remain functional (NFR-R12)

**Technical Notes:**
- Implements FR30: View download list in unified dashboard
- Implements NFR-R12: Partial functionality when disconnected
- Desktop multi-column layout (UX-1)

---

### Story 4.4: Download Status Filtering

As a **media collector**,
I want to **filter downloads by status**,
So that **I can focus on specific download states**.

**Acceptance Criteria:**

**Given** the download list is displayed
**When** filter buttons are shown
**Then** options include: All, Downloading, Paused, Completed, Seeding

**Given** the user selects "Downloading" filter
**When** the filter is applied
**Then** only actively downloading torrents are shown
**And** the count updates in the filter button

**Given** filters are applied
**When** the list updates (polling)
**Then** new items matching the filter appear
**And** items no longer matching disappear

**Technical Notes:**
- Implements FR31: Filter downloads by status
- Client-side filtering for responsiveness
- Filter state persisted in URL for bookmarking

---

### Story 4.5: Completed Download Detection and Parsing Trigger

As a **media collector**,
I want **completed downloads to automatically trigger parsing**,
So that **new media appears in my library without manual action**.

**Acceptance Criteria:**

**Given** qBittorrent reports a torrent as complete
**When** the next polling cycle detects the completion
**Then** the system automatically queues the file for parsing
**And** the download shows status: "Parsing..."

**Given** parsing completes successfully
**When** metadata is retrieved
**Then** the media appears in "Recently Added"
**And** a success notification is shown

**Given** parsing fails
**When** errors occur
**Then** the download shows: "Parsing failed - Manual action needed"
**And** links to manual search options

**Technical Notes:**
- Implements FR32: Detect completed downloads and trigger parsing
- Integrates with Epic 3 parsing pipeline
- Non-blocking: user can continue browsing

---

### Story 4.6: Connection Health Monitoring

As a **system administrator**,
I want to **see qBittorrent connection health status**,
So that **I know immediately when there are connectivity issues**.

**Acceptance Criteria:**

**Given** qBittorrent is connected
**When** viewing the dashboard header
**Then** a status indicator shows: 🟢 Connected

**Given** qBittorrent becomes unreachable
**When** the health check fails
**Then** the indicator changes to: 🔴 Disconnected
**And** shows: "Last success: 2 minutes ago"

**Given** connection is lost
**When** automatic recovery is attempted
**Then** the system retries every 30 seconds (NFR-R6)
**And** reconnects automatically when available

**Given** the user clicks on the connection status
**When** viewing details
**Then** they see connection history and error logs

**Technical Notes:**
- Implements FR33: Display connection health status
- Implements NFR-R6: Auto-recover (30s reconnection)
- Implements ARCH-8: Health Check Scheduler

---

## Epic 5: Media Library Management

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can browse, search, filter, and manage their complete media library collection.

### Story 5.1: Media Library Grid View

As a **media collector**,
I want to **browse my media library in a visual grid**,
So that **I can enjoy seeing my collection with beautiful posters**.

**Acceptance Criteria:**

**Given** the user opens the Library page
**When** the page loads
**Then** media items display in a responsive grid
**And** each card shows: poster, title (zh-TW), year, rating

**Given** the library has more than 1,000 items
**When** scrolling through the grid
**Then** virtual scrolling is enabled (NFR-SC6)
**And** scrolling maintains 60 FPS (NFR-P10)

**Given** the grid is displayed
**When** hovering over a card (desktop)
**Then** additional info appears: genre, description preview, metadata source
**And** the card has a subtle animation effect

**Technical Notes:**
- Implements FR38: Browse complete media library
- Implements NFR-SC6, NFR-P10: Virtual scrolling, 60 FPS
- Implements UX-9: Appreciation Loop

---

### Story 5.2: Library List View Toggle

As a **media collector**,
I want to **switch between grid and list views**,
So that **I can choose the display format that suits my preference**.

**Acceptance Criteria:**

**Given** the library is displayed in grid view
**When** the user clicks the "List View" toggle
**Then** the display switches to a table/list format
**And** columns include: poster thumbnail, title, year, genre, rating, date added

**Given** list view is active
**When** the user clicks a column header
**Then** the list sorts by that column
**And** ascending/descending toggle is available

**Given** the user's view preference
**When** they return to the library later
**Then** their preferred view (grid/list) is remembered

**Technical Notes:**
- Implements FR8: Toggle between grid and list view
- View preference stored in localStorage
- Table component with sortable columns

---

### Story 5.3: Library Search

As a **media collector**,
I want to **search within my saved media library**,
So that **I can quickly find specific titles in my collection**.

**Acceptance Criteria:**

**Given** the user is on the Library page
**When** they type in the search box
**Then** results filter in real-time as they type
**And** both Chinese and English titles are searched

**Given** a search query is entered
**When** results are displayed
**Then** matching terms are highlighted
**And** search completes within 500ms (NFR-SC8)

**Given** no results match the query
**When** the search completes
**Then** a friendly message suggests: "No results found. Try a different search term or add new media."

**Technical Notes:**
- Implements FR5: Search within saved media library
- Implements NFR-SC8: SQLite FTS5 full-text search
- Debounced input for performance

---

### Story 5.4: Library Sorting

As a **media collector**,
I want to **sort my library by different criteria**,
So that **I can organize my view based on what I'm looking for**.

**Acceptance Criteria:**

**Given** the library is displayed
**When** the user opens the sort dropdown
**Then** options include:
- Date Added (newest/oldest)
- Title (A-Z / Z-A)
- Year (newest/oldest)
- Rating (highest/lowest)

**Given** a sort option is selected
**When** the sort is applied
**Then** the library reorders immediately
**And** the current sort is indicated in the UI

**Given** the user's sort preference
**When** they return to the library
**Then** their last used sort is applied

**Technical Notes:**
- Implements FR6: Sort media library
- Sort state persisted in URL and localStorage
- Efficient sorting with database indexes

---

### Story 5.5: Library Filtering

As a **media collector**,
I want to **filter my library by genre, year, and media type**,
So that **I can narrow down to specific categories**.

**Acceptance Criteria:**

**Given** the library is displayed
**When** the user opens the filter panel
**Then** filter options include:
- Genre (multi-select)
- Year range (slider or inputs)
- Media Type (Movie, TV Show, Anime)

**Given** filters are applied
**When** the library updates
**Then** only matching items are displayed
**And** the filter count is shown: "Showing 45 of 500 items"

**Given** multiple filters are active
**When** viewing the filter status
**Then** active filters are shown as removable chips
**And** a "Clear all filters" option is available

**Technical Notes:**
- Implements FR7: Filter media library
- Filters work in combination (AND logic)
- Filter state persisted in URL for sharing

---

### Story 5.6: Media Detail Page (Full Version)

As a **media collector**,
I want to **view comprehensive details about media in my library**,
So that **I can access all information including cast, trailers, and metadata source**.

**Acceptance Criteria:**

**Given** the user clicks on a library item
**When** the detail page opens
**Then** it displays all information from Story 2.4 plus:
- Full cast list with roles
- Embedded trailer (YouTube)
- Metadata source indicator (TMDb/Douban/Wikipedia/AI/Manual)
- File information (filename, size, quality)
- Date added to library

**Given** trailers are available
**When** the user clicks "Watch Trailer"
**Then** the YouTube video plays in an embedded player
**And** doesn't navigate away from the page

**Given** the metadata source is displayed
**When** hovering over the source badge
**Then** details show: "Fetched from TMDb on 2026-01-10"

**Technical Notes:**
- Implements FR39: View media detail pages with cast/trailers
- Implements FR42: Display metadata source indicators
- YouTube embed with privacy-enhanced mode

---

### Story 5.7: Batch Operations

As a **power user**,
I want to **perform batch operations on multiple media items**,
So that **I can efficiently manage large numbers of files**.

**Acceptance Criteria:**

**Given** the library is displayed
**When** the user enters "selection mode"
**Then** checkboxes appear on each item
**And** a toolbar shows available batch actions

**Given** multiple items are selected
**When** batch actions are available
**Then** options include:
- Delete selected
- Re-parse selected
- Export metadata

**Given** the user selects "Delete selected"
**When** confirming the action
**Then** a confirmation dialog shows item count
**And** upon confirmation, items are removed from library

**Given** a batch operation is in progress
**When** processing multiple items
**Then** a progress indicator shows: "Processing 5 of 20..."
**And** errors are collected and shown at the end

**Technical Notes:**
- Implements FR40: Batch operations (delete, re-parse)
- Confirmation required for destructive operations
- Progress tracking for large batches

---

### Story 5.8: Recently Added Section

As a **media collector**,
I want to **see recently added media prominently**,
So that **I can quickly access my newest additions**.

**Acceptance Criteria:**

**Given** the user opens the Library page
**When** the page loads
**Then** a "Recently Added" section shows the newest 10-20 items
**And** items are sorted by date added (newest first)

**Given** new media is added
**When** the library updates within 30 seconds (NFR-P9)
**Then** the new item appears at the top of "Recently Added"
**And** a subtle animation highlights the new addition

**Given** the user clicks "See All"
**When** navigating to the full library
**Then** the sort is set to "Date Added (newest)"
**And** all items are visible

**Technical Notes:**
- Implements FR41: View recently added media items
- Implements NFR-P9: Library updates <30 seconds
- Section appears on Library page and Dashboard

---

## Epic 6: System Configuration & Backup

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can configure the system through a setup wizard, manage cache, view logs, and backup/restore their data.

### Story 6.1: Setup Wizard

As a **first-time user**,
I want a **guided setup wizard**,
So that **I can configure Vido quickly without confusion**.

**Acceptance Criteria:**

**Given** the user opens Vido for the first time
**When** no configuration exists
**Then** the setup wizard launches automatically
**And** shows progress: Step 1 of 5

**Given** the wizard is running
**When** completing each step:
1. Welcome & language selection
2. qBittorrent connection (skip option available)
3. Media folder configuration
4. API keys (optional, can skip)
5. Complete
**Then** each step validates before proceeding
**And** back navigation is available

**Given** the user completes the wizard
**When** clicking "Finish"
**Then** settings are saved
**And** the user is taken to the main dashboard

**Technical Notes:**
- Implements FR52: Initial setup via wizard
- Implements NFR-U2: Setup wizard <5 steps
- Implements UX-7: Minimal onboarding

---

### Story 6.2: Cache Management

As a **system administrator**,
I want to **view and manage cached data**,
So that **I can reclaim disk space when needed**.

**Acceptance Criteria:**

**Given** the user opens Settings > Cache
**When** viewing cache information
**Then** they see:
- Image cache: X.X GB
- AI parsing cache: X MB
- Metadata cache: X MB
- Total: X.X GB

**Given** cache information is displayed
**When** the user clicks "Clear cache older than 30 days"
**Then** old cache is removed
**And** space reclaimed is shown

**Given** individual cache types are shown
**When** the user clicks "Clear" on a specific type
**Then** only that cache type is cleared
**And** a confirmation is required

**Technical Notes:**
- Implements FR53: Manage cache
- Implements ARCH-5: Cache Management System
- Shows cache by type (images, AI, metadata)

---

### Story 6.3: System Logs Viewer

As a **system administrator**,
I want to **view system logs**,
So that **I can troubleshoot issues and monitor system health**.

**Acceptance Criteria:**

**Given** the user opens Settings > Logs
**When** logs are displayed
**Then** entries show: timestamp, level (ERROR/WARN/INFO/DEBUG), message
**And** logs are color-coded by level

**Given** many log entries exist
**When** viewing the log list
**Then** pagination or infinite scroll is available
**And** newest logs are shown first

**Given** logs are displayed
**When** the user filters by level (e.g., "ERROR only")
**Then** only matching entries are shown
**And** search by keyword is available

**Given** any log entry
**When** it contains sensitive information
**Then** API keys are masked (NFR-S4)
**And** error hints are provided (NFR-U9)

**Technical Notes:**
- Implements FR54: View system logs
- Implements NFR-M11: Severity-level logging
- Implements NFR-U9: Error logs with troubleshooting hints

---

### Story 6.4: Service Connection Status Dashboard

As a **system administrator**,
I want to **see connection status for all external services**,
So that **I can identify integration issues at a glance**.

**Acceptance Criteria:**

**Given** the user opens Settings > Status
**When** the status page loads
**Then** it shows connection status for:
- qBittorrent: 🟢 Connected / 🔴 Disconnected
- TMDb API: 🟢 Available / 🟡 Rate Limited / 🔴 Error
- AI Service: 🟢 Available / 🔴 Error

**Given** a service shows an error
**When** hovering or clicking on the status
**Then** detailed error message is shown
**And** last successful connection time is displayed

**Given** the status page is open
**When** service status changes
**Then** the status updates in real-time
**And** a notification indicates the change

**Technical Notes:**
- Implements FR55: Display service connection status
- Implements NFR-M13: Health status visible
- Implements ARCH-8: Health Check Scheduler

---

### Story 6.5: Database Backup

As a **system administrator**,
I want to **backup my Vido database and configuration**,
So that **I can restore my data if something goes wrong**.

**Acceptance Criteria:**

**Given** the user opens Settings > Backup
**When** they click "Create Backup Now"
**Then** an atomic backup is created using SQLite .backup (NFR-R7)
**And** backup includes: database, configuration, learned mappings

**Given** a backup is created
**When** the backup completes
**Then** it is saved to `/vido-backups` volume
**And** filename format: `vido-backup-YYYYMMDD-HHMMSS-v{schema}.tar.gz`

**Given** backup is in progress
**When** viewing the progress
**Then** a progress indicator is shown
**And** backup for 10,000 items completes in <5 minutes

**Technical Notes:**
- Implements FR57: Backup database and configuration
- Implements NFR-R7: SQLite atomic backups
- Implements ARCH-4: Background Task Queue

---

### Story 6.6: Backup Integrity Verification

As a **system administrator**,
I want **backup integrity to be verified**,
So that **I know my backups are reliable**.

**Acceptance Criteria:**

**Given** a backup is created
**When** the backup completes
**Then** a SHA-256 checksum is calculated
**And** stored alongside the backup file

**Given** a backup file exists
**When** the user clicks "Verify Backup"
**Then** the checksum is recalculated
**And** compared against the stored checksum

**Given** verification fails
**When** the checksum doesn't match
**Then** the backup is marked as "Potentially Corrupted"
**And** the user is warned before attempting restore

**Technical Notes:**
- Implements FR59: Verify backup integrity
- Implements NFR-R8: Backup integrity verification
- Checksum stored in separate .sha256 file

---

### Story 6.7: Data Restore

As a **system administrator**,
I want to **restore data from a backup**,
So that **I can recover from data loss or migration**.

**Acceptance Criteria:**

**Given** backup files exist
**When** the user opens Settings > Backup > Restore
**Then** available backups are listed with date, size, and version

**Given** the user selects a backup
**When** they click "Restore"
**Then** a confirmation dialog warns: "This will replace current data"
**And** an auto-snapshot of current state is created first (NFR-R9)

**Given** restore is confirmed
**When** the restore process runs
**Then** progress is shown
**And** the application restarts with restored data

**Given** restore fails
**When** an error occurs
**Then** the auto-snapshot is used to recover
**And** an error message explains what happened

**Technical Notes:**
- Implements FR58: Restore data from backup
- Implements NFR-R9: Auto-snapshot before restore
- Schema version compatibility checked

---

### Story 6.8: Scheduled Backups

As a **system administrator**,
I want to **schedule automatic backups**,
So that **I don't have to remember to backup manually**.

**Acceptance Criteria:**

**Given** the user opens backup settings
**When** configuring schedule
**Then** options include: Daily, Weekly, or Disabled
**And** time of day can be selected

**Given** scheduled backup is enabled
**When** the scheduled time arrives
**Then** backup runs automatically
**And** runs in background without UI disruption

**Given** backups accumulate
**When** retention policy is active
**Then** keeps last 7 daily + last 4 weekly backups
**And** older backups are automatically deleted (FR64)

**Technical Notes:**
- Implements FR63: Configure backup schedule
- Implements FR64: Auto-cleanup old backups
- Uses ARCH-4: Background Task Queue

---

### Story 6.9: Metadata Export (JSON/YAML/NFO)

As a **power user**,
I want to **export my library metadata in various formats**,
So that **I can use it with other tools or for backup purposes**.

**Acceptance Criteria:**

**Given** the user opens Export options
**When** selecting export format
**Then** options include: JSON, YAML, NFO (Kodi/Plex compatible)

**Given** JSON/YAML export is selected
**When** export completes
**Then** a single file contains all library metadata
**And** the format is human-readable and documented

**Given** NFO export is selected
**When** export completes
**Then** .nfo files are created alongside each media file
**And** format is compatible with Kodi/Plex/Jellyfin

**Given** export is in progress
**When** processing large library
**Then** progress is shown
**And** can be run in background

**Technical Notes:**
- Implements FR60, FR62: Export to JSON/YAML/NFO
- NFO follows Kodi standard format
- Export runs asynchronously

---

### Story 6.10: Metadata Import

As a **power user**,
I want to **import metadata from JSON/YAML files**,
So that **I can restore or migrate my library data**.

**Acceptance Criteria:**

**Given** the user has a JSON/YAML export file
**When** they select "Import Metadata"
**Then** they can upload or provide path to the file

**Given** an import file is provided
**When** import runs
**Then** metadata is merged with existing library
**And** conflicts are handled: Skip / Overwrite / Ask

**Given** import completes
**When** viewing results
**Then** summary shows: Added X, Updated Y, Skipped Z items

**Technical Notes:**
- Implements FR61: Import metadata from JSON/YAML
- Supports incremental import (merge)
- Validates file format before processing

---

### Story 6.11: Performance Metrics Dashboard

As a **system administrator**,
I want to **view performance metrics**,
So that **I can monitor system health and identify issues**.

**Acceptance Criteria:**

**Given** the user opens Settings > Performance
**When** metrics are displayed
**Then** they see:
- Query latency (p50, p95)
- Cache hit rate
- API response times
- Library item count

**Given** metrics show concerning values
**When** p95 latency > 500ms or items > 8,000
**Then** a warning is displayed (NFR-SC2)
**And** recommendation: "Consider PostgreSQL migration"

**Given** the metrics page is open
**When** viewing trends
**Then** charts show 24-hour and 7-day trends

**Technical Notes:**
- Implements FR65: Display performance metrics
- Implements FR66, NFR-SC2: Scalability warnings
- Implements NFR-M12: Performance metrics queryable

---

## Epic 7: User Authentication & Access Control

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users must authenticate to access Vido, with secure session management and API protection.

### Story 7.1: Password/PIN Setup

As a **first-time user**,
I want to **set up a password or PIN for Vido**,
So that **my media library is protected from unauthorized access**.

**Acceptance Criteria:**

**Given** the user completes the setup wizard
**When** reaching the security step
**Then** they must set a password or PIN
**And** minimum requirements: 6+ characters (password) or 4+ digits (PIN)

**Given** a password is set
**When** stored in the database
**Then** it is hashed using bcrypt
**And** the plaintext is never stored

**Given** the user wants to change their password
**When** accessing Settings > Security
**Then** they must enter current password first
**And** can then set a new password

**Technical Notes:**
- Implements FR67: Authenticate via password/PIN
- Bcrypt with appropriate cost factor
- Password strength indicator in UI

---

### Story 7.2: Login Page

As a **returning user**,
I want to **log in with my password or PIN**,
So that **I can access my media library**.

**Acceptance Criteria:**

**Given** the user is not authenticated
**When** accessing any Vido page
**Then** they are redirected to the login page

**Given** the login page is displayed
**When** the user enters correct credentials
**Then** they are authenticated
**And** redirected to their intended destination

**Given** incorrect credentials are entered
**When** login fails
**Then** an error message is shown: "Invalid password"
**And** failed attempt is logged (NFR-S13)

**Given** 5 failed attempts in 15 minutes
**When** the limit is reached
**Then** login is temporarily blocked
**And** message: "Too many attempts. Try again in X minutes."

**Technical Notes:**
- Implements NFR-S13: Failed auth attempts rate-limited
- No username required (single-user system)
- Rate limiting per IP address

---

### Story 7.3: Session Management

As an **authenticated user**,
I want **my session to be secure and persistent**,
So that **I don't have to log in repeatedly but remain protected**.

**Acceptance Criteria:**

**Given** the user logs in successfully
**When** a session is created
**Then** a cryptographically-signed JWT token is issued (NFR-S10)
**And** stored in httpOnly cookie

**Given** a session is active
**When** the user makes requests
**Then** the session token is validated
**And** refreshed automatically before expiry

**Given** the session expires (after 7 days)
**When** the user tries to access Vido
**Then** they are redirected to login
**And** a message indicates session expiration

**Given** the user clicks "Logout"
**When** logout is processed
**Then** the session is invalidated
**And** the user is redirected to login

**Technical Notes:**
- Implements FR68: Manage user sessions
- Implements NFR-S10: Cryptographically-signed tokens
- JWT with RS256 or HS256 signing

---

### Story 7.4: API Authentication

As a **developer integrating with Vido**,
I want **API endpoints to be authenticated**,
So that **only authorized requests can access data**.

**Acceptance Criteria:**

**Given** an API request is made
**When** no authentication token is provided
**Then** the request is rejected with 401 Unauthorized

**Given** a valid session cookie exists
**When** making API requests from the browser
**Then** the session cookie authenticates the request
**And** CSRF protection is enforced

**Given** an API token is generated
**When** used in Authorization header
**Then** the request is authenticated
**And** the token has configurable expiration

**Technical Notes:**
- Implements FR69: Protect API endpoints
- Implements NFR-S11: API endpoints protected
- Support both session and API token auth

---

### Story 7.5: Rate Limiting

As a **system administrator**,
I want **API rate limiting**,
So that **the system is protected from abuse**.

**Acceptance Criteria:**

**Given** API requests are made
**When** rate exceeds 100 requests/minute from same IP
**Then** subsequent requests return 429 Too Many Requests
**And** Retry-After header indicates when to retry

**Given** rate limit is hit
**When** the user sees the error
**Then** a friendly message explains the limit
**And** suggests waiting before retrying

**Given** different endpoints
**When** rate limits are applied
**Then** more restrictive limits for sensitive operations (login)
**And** relaxed limits for read-only operations

**Technical Notes:**
- Implements FR70: Implement rate limiting
- Implements NFR-S12: Rate limiting (100 req/min per IP)
- Token bucket or sliding window algorithm

---

## Epic 8: Advanced Download Control

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can control qBittorrent directly from Vido, including pause/resume/delete torrents, adjust download priority, manage bandwidth settings, and schedule downloads.

### Story 8.1: Torrent Control Operations

As a **media collector**,
I want to **control my torrents directly from Vido**,
So that **I don't need to switch to the qBittorrent interface for basic operations**.

**Acceptance Criteria:**

**Given** the user views the download list
**When** they select a torrent
**Then** control buttons are available: Pause, Resume, Delete

**Given** a running torrent
**When** the user clicks "Pause"
**Then** the torrent pauses immediately via qBittorrent API
**And** status updates within 2 seconds

**Given** a paused torrent
**When** the user clicks "Resume"
**Then** the torrent resumes
**And** status shows "Downloading"

**Given** the user clicks "Delete"
**When** confirming the action
**Then** a dialog asks: "Delete torrent only" or "Delete with files"
**And** the selected action is executed
**And** confirmation shows success

**Technical Notes:**
- Implements FR34: Control qBittorrent directly
- Uses qBittorrent Web API v2.x
- Requires confirmed connection from Epic 4

---

### Story 8.2: Download Priority Management

As a **media collector**,
I want to **adjust download priority**,
So that **important downloads complete first**.

**Acceptance Criteria:**

**Given** multiple torrents are downloading
**When** the user views the download list
**Then** each torrent shows its current priority level

**Given** a torrent is selected
**When** the user clicks "Set Priority"
**Then** options are available: High, Normal, Low
**And** the change is applied immediately

**Given** priority is changed
**When** qBittorrent processes the change
**Then** bandwidth allocation adjusts accordingly
**And** higher priority torrents get more bandwidth

**Given** file priority within a torrent
**When** the user expands torrent details
**Then** individual files can be set to High/Normal/Low/Skip
**And** Skip means the file won't download

**Technical Notes:**
- Implements FR35: Adjust download priority
- Maps to qBittorrent's priority system (0-7)
- File-level priority for selective downloading

---

### Story 8.3: Bandwidth Settings Control

As a **NAS user**,
I want to **manage bandwidth settings**,
So that **downloads don't saturate my network**.

**Acceptance Criteria:**

**Given** the user opens Settings > Downloads
**When** viewing bandwidth settings
**Then** they see:
- Global download limit (KB/s)
- Global upload limit (KB/s)
- Alternative speed limits (for scheduled mode)

**Given** bandwidth limits are set
**When** the user saves changes
**Then** qBittorrent applies the limits immediately
**And** current speeds adjust within 5 seconds

**Given** alternative speed mode
**When** the user toggles "Alternative Speed"
**Then** the preset slower limits are applied
**And** status bar shows alternative mode is active

**Given** per-torrent limits are needed
**When** the user selects a specific torrent
**Then** individual download/upload limits can be set
**And** these override global settings

**Technical Notes:**
- Implements FR36: Manage bandwidth settings
- Maps to qBittorrent preferences API
- Alternative speed mode for peak hours

---

### Story 8.4: Download Scheduling

As a **NAS user**,
I want to **schedule downloads for specific times**,
So that **downloads run during off-peak hours**.

**Acceptance Criteria:**

**Given** the user opens Settings > Schedule
**When** configuring the schedule
**Then** a weekly time grid is available (7 days × 24 hours)
**And** users can select time blocks for each mode

**Given** schedule modes are:
- Full Speed: No limits
- Alternative Speed: Use alternative limits
- Pause All: No downloading
**When** the user selects time blocks
**Then** each block is assigned a mode
**And** visual color coding shows the schedule

**Given** a schedule is configured
**When** the scheduled time arrives
**Then** qBittorrent automatically switches modes
**And** Vido displays current schedule status

**Given** the user wants a simple schedule
**When** using "Quick Schedule"
**Then** presets are available: "Night Only (00:00-06:00)", "Off-Peak (20:00-08:00)"
**And** one-click applies the schedule

**Technical Notes:**
- Implements FR37: Schedule downloads
- Uses qBittorrent's built-in scheduler
- Visual weekly grid interface

---

## Epic 9: Subtitle Integration

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can search for subtitles from OpenSubtitles and Zimuku with Traditional Chinese subtitle priority, download subtitles, manually upload subtitles, and see subtitle availability status.

### Story 9.1: Subtitle Search Integration

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

### Story 9.2: Traditional Chinese Subtitle Priority

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

### Story 9.3: Subtitle Download

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

### Story 9.4: Manual Subtitle Upload

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

### Story 9.5: Automatic Subtitle Download

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

### Story 9.6: Subtitle Availability Status

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

## Epic 10: Smart Recommendations & Discovery

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can receive smart recommendations based on genre, cast, and director, and see "similar titles" suggestions to discover new content.

### Story 10.1: Genre-Based Recommendations

As a **media collector**,
I want to **receive recommendations based on my genres**,
So that **I can discover similar content I might enjoy**.

**Acceptance Criteria:**

**Given** the user has media in their library
**When** the system analyzes their collection
**Then** it identifies the top genres by count
**And** generates recommendations based on genre overlap

**Given** recommendations are generated
**When** viewing the Dashboard
**Then** a "Recommended for You" section appears
**And** shows 6-12 recommendations with posters

**Given** a recommendation is displayed
**When** hovering over it
**Then** the reason is shown: "Because you like [Genre]"
**And** clicking opens the detail page

**Given** the user dismisses a recommendation
**When** clicking "Not Interested"
**Then** that title is hidden
**And** similar content is de-prioritized

**Technical Notes:**
- Implements FR9: Smart recommendations based on genre
- Uses TMDb similar/recommendations API
- Local caching for recommendations

---

### Story 10.2: Cast and Director Based Recommendations

As a **media collector**,
I want **recommendations based on actors and directors I follow**,
So that **I discover their other works**.

**Acceptance Criteria:**

**Given** the user's library has multiple works by the same director
**When** generating recommendations
**Then** other works by that director are suggested
**And** reason shows: "From director [Name]"

**Given** the user's library has multiple works with the same actor
**When** generating recommendations
**Then** other works featuring that actor are suggested
**And** reason shows: "[Actor Name] is in this"

**Given** recommendations are personalized
**When** viewing a specific media detail
**Then** "More from this director" section appears
**And** "More with [Lead Actor]" section appears

**Given** the user explicitly "follows" an actor/director
**When** new content becomes available
**Then** it's highlighted in recommendations
**And** optional notification is sent

**Technical Notes:**
- Implements FR9: Smart recommendations based on cast, director
- TMDb person credits API
- Follow feature stored in user preferences

---

### Story 10.3: Similar Titles Suggestions

As a **media collector**,
I want to **see similar titles for media I'm viewing**,
So that **I can find related content**.

**Acceptance Criteria:**

**Given** the user views a media detail page
**When** scrolling to the bottom
**Then** "Similar Titles" section shows 6-10 related items
**And** items are sourced from TMDb similar/recommendations API

**Given** similar titles are displayed
**When** one is already in the user's library
**Then** it shows "In Your Library" badge
**And** clicking goes to the library entry, not external

**Given** similar titles are displayed
**When** one is not in the library
**Then** it shows basic info (title, year, poster)
**And** clicking shows a mini-detail modal

**Given** the user is browsing similar titles
**When** they want to add one
**Then** "Add to Wishlist" button is available
**And** the wishlist can be exported for future reference

**Technical Notes:**
- Implements FR10: Similar titles suggestions
- TMDb similar movies/TV shows endpoint
- Local library cross-reference

---

## Epic 11: Watch History & Collections

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can track personal watch history, see watch progress indicators, mark media as watched/unwatched, and create custom collections of media items.

### Story 11.1: Watch History Tracking

As a **user**,
I want to **track what I've watched**,
So that **I can remember my viewing history**.

**Acceptance Criteria:**

**Given** the user marks a movie as "Watched"
**When** saving the status
**Then** the watch date is recorded
**And** the movie appears in "Watch History"

**Given** the user marks a TV show episode as "Watched"
**When** saving the status
**Then** the episode is marked
**And** show progress percentage updates

**Given** the user opens Watch History page
**When** viewing the list
**Then** items are sorted by watch date (newest first)
**And** filtering by type (Movies/TV) is available

**Given** the user wants to track rewatches
**When** marking an already-watched item
**Then** option appears: "Mark as rewatched"
**And** rewatch count is incremented

**Technical Notes:**
- Implements FR43: Track personal watch history
- Watch events stored with timestamps
- Foundation for multi-user history (Epic 13)

---

### Story 11.2: Watch Progress Indicators

As a **user**,
I want to **see my watch progress**,
So that **I know what I've finished and what's remaining**.

**Acceptance Criteria:**

**Given** a TV series in the library
**When** viewing the library card
**Then** progress bar shows: X of Y episodes watched
**And** percentage is displayed: "60% complete"

**Given** a movie in the library
**When** viewing the library card
**Then** watched status shows: ✓ (checkmark) or empty
**And** watch date shows if watched

**Given** a series with unwatched episodes
**When** viewing the detail page
**Then** "Continue Watching" shows the next unwatched episode
**And** season progress is displayed per season

**Given** the user filters the library
**When** selecting "In Progress"
**Then** only partially-watched series are shown
**And** sorted by most recently watched

**Technical Notes:**
- Implements FR44: Display watch progress indicators
- Episode-level tracking for TV series
- Visual progress bars in UI

---

### Story 11.3: Mark as Watched/Unwatched

As a **user**,
I want to **easily mark items as watched or unwatched**,
So that **I can manage my watch status**.

**Acceptance Criteria:**

**Given** the user views a movie detail page
**When** clicking the "Mark as Watched" button
**Then** the movie status changes to watched
**And** current date is recorded
**And** button changes to "Mark as Unwatched"

**Given** a TV series detail page
**When** the user right-clicks an episode
**Then** context menu shows: "Mark Watched" / "Mark Unwatched"
**And** bulk options: "Mark season as watched", "Mark all as watched"

**Given** the user accidentally marks something
**When** clicking "Undo" within 5 seconds
**Then** the action is reversed
**And** history is corrected

**Given** the library list view
**When** the user selects multiple items
**Then** batch action: "Mark selected as watched"
**And** confirmation shows count affected

**Technical Notes:**
- Implements FR45: Mark media as watched/unwatched
- Undo support with 5-second window
- Batch operations for efficiency

---

### Story 11.4: Custom Collections

As a **media collector**,
I want to **create custom collections**,
So that **I can organize media by my own categories**.

**Acceptance Criteria:**

**Given** the user opens Collections page
**When** clicking "Create Collection"
**Then** a dialog asks for: Name, Description (optional), Cover image (optional)
**And** the collection is created empty

**Given** a collection exists
**When** the user views a media detail page
**Then** "Add to Collection" button is available
**And** clicking shows list of collections to choose from

**Given** the user views a collection
**When** opening the collection page
**Then** all items in the collection are displayed
**And** custom ordering is supported (drag-drop)

**Given** the user wants to organize collections
**When** editing a collection
**Then** they can: Rename, Change cover, Delete, Export as list
**And** "Smart Collections" option creates auto-updating rules

**Technical Notes:**
- Implements FR46: Create custom collections
- Manual ordering with drag-drop
- Smart collections with filter rules

---

## Epic 12: Automation & Organization

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** The system can monitor watch folders to detect new files, automatically trigger parsing, rename files based on patterns, move files to organized structure, and execute automation tasks in background queue.

### Story 12.1: Watch Folder Monitoring

As a **media collector**,
I want **folders monitored for new files**,
So that **new downloads are automatically detected**.

**Acceptance Criteria:**

**Given** the user opens Settings > Automation
**When** adding a watch folder
**Then** they can browse/enter a folder path
**And** select file extensions to watch (e.g., .mkv, .mp4, .avi)

**Given** a watch folder is configured
**When** a new file appears in that folder
**Then** it is detected within 30 seconds
**And** appears in "Pending Processing" queue

**Given** file detection occurs
**When** the file is still being written
**Then** the system waits until file size is stable
**And** processes only after write is complete

**Given** multiple watch folders are configured
**When** viewing the automation dashboard
**Then** all watched folders are listed with status
**And** file counts per folder are shown

**Technical Notes:**
- Implements FR81: Monitor watch folders
- inotify/fsnotify for file detection
- File stability check before processing

---

### Story 12.2: Automatic Parsing Trigger

As a **media collector**,
I want **new files automatically parsed**,
So that **metadata is retrieved without manual intervention**.

**Acceptance Criteria:**

**Given** a new file is detected in a watch folder
**When** the file is stable and ready
**Then** parsing is automatically triggered
**And** uses the same logic as manual parsing (Epic 3)

**Given** automatic parsing succeeds
**When** metadata is retrieved
**Then** the file is added to the library
**And** status shows "Auto-processed"

**Given** automatic parsing fails
**When** no metadata is found
**Then** the file is marked as "Needs Review"
**And** notification alerts the user (if enabled)

**Given** many files arrive simultaneously
**When** queue builds up
**Then** files are processed in order (oldest first)
**And** processing rate respects API limits

**Technical Notes:**
- Implements FR82: Auto-trigger parsing
- Reuses parsing logic from Epic 3
- Queue management (ARCH-4)

---

### Story 12.3: Automatic File Renaming

As a **media collector**,
I want **files renamed based on patterns**,
So that **my library has consistent naming**.

**Acceptance Criteria:**

**Given** the user configures rename patterns in Settings
**When** setting up the pattern
**Then** variables are available:
- `{title}`, `{year}`, `{quality}`, `{codec}`
- `{season}`, `{episode}` (for TV)
- Example: `{title} ({year}) - {quality}.{ext}`

**Given** a file is successfully parsed
**When** rename automation is enabled
**Then** the file is renamed according to the pattern
**And** original name is logged for reference

**Given** rename would cause conflict
**When** target filename exists
**Then** a suffix is added: `(1)`, `(2)`
**And** user is notified of the conflict

**Given** the user wants to preview
**When** configuring patterns
**Then** a "Preview" shows example renames
**And** dry-run mode tests without actual changes

**Technical Notes:**
- Implements FR83: Auto-rename files
- Pattern template system
- Conflict resolution with suffixes

---

### Story 12.4: Automatic File Organization

As a **media collector**,
I want **files moved to organized folders**,
So that **my library structure is consistent**.

**Acceptance Criteria:**

**Given** the user configures organization in Settings
**When** setting up folder structure
**Then** patterns are available:
- Movies: `/media/Movies/{title} ({year})/`
- TV: `/media/TV/{title}/Season {season}/`

**Given** a file is successfully parsed
**When** organization automation is enabled
**Then** the file is moved to the target folder
**And** parent folders are created if needed

**Given** the source and target are different drives
**When** moving the file
**Then** file is copied first, then source deleted
**And** integrity is verified before deleting source

**Given** the user has custom requirements
**When** configuring advanced rules
**Then** genre-based folders are supported
**And** year-based folders: `/media/Movies/2024/`

**Technical Notes:**
- Implements FR84: Auto-move files to organized structure
- Cross-filesystem move support
- Verify integrity before deleting source

---

### Story 12.5: Background Task Queue

As a **system administrator**,
I want **automation tasks processed in background**,
So that **the UI remains responsive**.

**Acceptance Criteria:**

**Given** automation tasks are triggered
**When** they enter the queue
**Then** each task has: ID, Type, Status, Progress, Created time

**Given** the user opens Automation > Queue
**When** viewing the queue
**Then** all pending and running tasks are listed
**And** completed tasks show for 24 hours

**Given** a task is running
**When** viewing its details
**Then** progress percentage is shown
**And** current step is described

**Given** a task fails
**When** viewing the queue
**Then** status shows "Failed" with error message
**And** "Retry" button is available

**Technical Notes:**
- Implements FR85: Background processing queue
- Implements ARCH-4: Background Task Queue
- Persistent queue survives restarts

---

### Story 12.6: Automation Rules Configuration

As a **media collector**,
I want to **configure automation rules**,
So that **I can customize how files are processed**.

**Acceptance Criteria:**

**Given** the user opens Settings > Automation Rules
**When** creating a new rule
**Then** they configure:
- Watch folder(s)
- File filters (extension, size, name pattern)
- Actions: Parse, Rename, Move, Notify

**Given** multiple rules exist
**When** a file matches multiple rules
**Then** rules are applied in priority order
**And** first match wins (or all match option)

**Given** a rule is configured
**When** the user wants to test
**Then** "Test Rule" shows what would happen
**And** no actual changes are made

**Given** the user wants presets
**When** selecting from templates
**Then** common patterns are available:
- "Movies to /media/Movies/"
- "TV Shows by Season"
- "Anime with fansub naming"

**Technical Notes:**
- Implements FR86: Configure automation rules
- Rule priority system
- Dry-run test mode

---

## Epic 13: Multi-User Support

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** The system supports multiple user accounts with admin/user permission management, and each user has their own personal watch history and preference settings.

### Story 13.1: Multiple User Accounts

As a **household admin**,
I want to **create multiple user accounts**,
So that **each family member has their own profile**.

**Acceptance Criteria:**

**Given** the admin opens Settings > Users
**When** clicking "Add User"
**Then** they enter: Username, Password, Display Name
**And** the new user account is created

**Given** multiple users exist
**When** opening the login page
**Then** user selection is available
**And** each user logs in with their own password

**Given** a user is created
**When** they first log in
**Then** they see an empty watch history
**And** default preferences are applied

**Given** user limit is reached (5 users, NFR-SC4)
**When** trying to add another
**Then** message shows: "Maximum users reached"
**And** suggests removing inactive users

**Technical Notes:**
- Implements FR71: Support multiple user accounts
- Implements NFR-SC4: Support 5 concurrent sessions
- Each user gets separate data tables

---

### Story 13.2: Admin Permission Management

As a **system admin**,
I want to **manage user permissions**,
So that **I control who can modify system settings**.

**Acceptance Criteria:**

**Given** user roles are: Admin, User
**When** an Admin views Settings > Users
**Then** they see all users and their roles
**And** can change roles (except last admin)

**Given** a User role account
**When** accessing settings
**Then** they cannot see: Users, Backup, Automation Rules
**And** they can see: Their profile, Display preferences

**Given** Admin role account
**When** accessing any setting
**Then** full access is granted
**And** system-wide changes are allowed

**Given** the last admin account
**When** trying to change role to User
**Then** action is blocked
**And** message: "At least one admin required"

**Technical Notes:**
- Implements FR72: Manage user permissions
- Role-based access control (RBAC)
- Admin-only sections enforced in UI and API

---

### Story 13.3: Personal Watch History Per User

As a **household member**,
I want **my own watch history**,
So that **my viewing doesn't affect others**.

**Acceptance Criteria:**

**Given** User A marks a movie as watched
**When** User B logs in
**Then** that movie shows as unwatched for User B
**And** watch histories are completely separate

**Given** the user views their dashboard
**When** seeing "Continue Watching" section
**Then** only their own in-progress shows appear
**And** recommendations are based on their history

**Given** the admin wants to view all history
**When** accessing admin dashboard
**Then** aggregate statistics are available
**And** individual user histories remain private

**Given** a user is deleted
**When** admin removes the account
**Then** that user's watch history is deleted
**And** library content is unaffected

**Technical Notes:**
- Implements FR73: Personal watch history per user
- User-scoped database queries
- Privacy maintained between users

---

### Story 13.4: Personal Preference Settings

As a **household member**,
I want **my own preferences**,
So that **my settings don't affect others**.

**Acceptance Criteria:**

**Given** User A sets "Dark Mode"
**When** User B logs in with "Light Mode" preference
**Then** User B sees Light Mode
**And** preferences are user-specific

**Given** personal preferences include:
- Theme (Light/Dark)
- Default view (Grid/List)
- Subtitle language priority
- Dashboard layout
**When** the user changes any setting
**Then** it saves to their profile only

**Given** a new user is created
**When** they first access preferences
**Then** system defaults are applied
**And** they can customize as needed

**Given** the user wants to reset
**When** clicking "Reset to Defaults"
**Then** all personal preferences revert
**And** confirmation is required

**Technical Notes:**
- Implements FR74: Personal preference settings
- User preferences JSON in user table
- System defaults as fallback

---

## Epic 14: External API & Mobile Application

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** The system provides a versioned RESTful API (/api/v1) with OpenAPI/Swagger documentation, supports webhook subscriptions, enables Plex/Jellyfin integration, and users can access via mobile application.

### Story 14.1: Versioned RESTful API

As a **developer**,
I want a **versioned RESTful API**,
So that **I can build integrations that won't break on updates**.

**Acceptance Criteria:**

**Given** the API is accessed
**When** making requests
**Then** endpoints follow pattern: `/api/v1/{resource}`
**And** standard HTTP methods are used (GET, POST, PUT, DELETE)

**Given** API versioning
**When** a breaking change is needed
**Then** new version `/api/v2/` is created
**And** v1 remains available with deprecation notice

**Given** API response format
**When** receiving data
**Then** responses are JSON with consistent structure:
```json
{
  "data": {...},
  "meta": {"page": 1, "total": 100},
  "errors": []
}
```

**Given** pagination is needed
**When** listing resources
**Then** `?page=1&limit=20` parameters are supported
**And** Link headers provide next/prev URLs

**Technical Notes:**
- Implements FR87: RESTful API (/api/v1)
- Implements ARCH-9: API Versioning Strategy
- Implements NFR-I16: Versioned API

---

### Story 14.2: API Token Authentication

As a **developer**,
I want to **authenticate with API tokens**,
So that **my integrations can access the API securely**.

**Acceptance Criteria:**

**Given** the user opens Settings > API Tokens
**When** clicking "Generate Token"
**Then** a new token is created with:
- Name (user-provided)
- Permissions (read-only, read-write)
- Expiration (optional)

**Given** an API token is generated
**When** using it in requests
**Then** Authorization header: `Bearer {token}`
**And** request is authenticated

**Given** token permissions are set
**When** a read-only token attempts write
**Then** request is rejected with 403 Forbidden
**And** error explains insufficient permissions

**Given** a token is compromised
**When** the user revokes it
**Then** token immediately stops working
**And** cannot be reactivated

**Technical Notes:**
- Implements FR88: Authenticate API with tokens
- Implements NFR-S11: API endpoints protected
- Token stored as hash, never plaintext

---

### Story 14.3: OpenAPI/Swagger Documentation

As a **developer**,
I want **interactive API documentation**,
So that **I can explore and test the API easily**.

**Acceptance Criteria:**

**Given** the developer accesses `/api/docs`
**When** the page loads
**Then** Swagger UI displays all endpoints
**And** each endpoint shows: Method, Path, Description, Parameters

**Given** an endpoint is viewed
**When** clicking "Try it out"
**Then** interactive testing is available
**And** responses are shown in real-time

**Given** the OpenAPI spec is needed
**When** accessing `/api/v1/openapi.json`
**Then** full OpenAPI 3.0 spec is returned
**And** can be imported into Postman/Insomnia

**Given** authentication is required
**When** using Swagger UI
**Then** "Authorize" button accepts API token
**And** subsequent requests include the token

**Technical Notes:**
- Implements FR89: OpenAPI/Swagger documentation
- Implements NFR-I17: OpenAPI/Swagger spec
- Auto-generated from code annotations

---

### Story 14.4: Webhook Subscriptions

As a **developer**,
I want to **subscribe to webhooks**,
So that **my systems are notified of events**.

**Acceptance Criteria:**

**Given** the user opens Settings > Webhooks
**When** creating a new webhook
**Then** they configure:
- URL to call
- Events to subscribe (new_media, parse_complete, download_complete)
- Secret for signature verification

**Given** a subscribed event occurs
**When** the system sends the webhook
**Then** POST request to configured URL
**And** payload includes event type and data
**And** `X-Vido-Signature` header for verification

**Given** webhook delivery fails
**When** target URL returns error
**Then** retry with exponential backoff (3 attempts)
**And** failure is logged in webhook history

**Given** webhook history is needed
**When** viewing webhook details
**Then** last 100 deliveries are shown
**And** each shows: timestamp, status, response

**Technical Notes:**
- Implements FR90: Webhook subscriptions
- Implements NFR-I18: Webhook support for events
- HMAC-SHA256 signature verification

---

### Story 14.5: Plex/Jellyfin Metadata Export

As a **media center user**,
I want to **export metadata to Plex/Jellyfin**,
So that **my media center shows correct information**.

**Acceptance Criteria:**

**Given** the user opens Settings > Integrations
**When** configuring Plex/Jellyfin export
**Then** they select export format: NFO (Kodi/Plex), Jellyfin API

**Given** NFO export is configured
**When** exporting media
**Then** NFO files are created alongside media files
**And** format is Kodi-compatible XML

**Given** the user clicks "Export All"
**When** export runs
**Then** all library items get NFO files
**And** existing NFO files can be overwritten or skipped

**Given** auto-export is enabled
**When** new media is added
**Then** NFO is automatically created
**And** Plex/Jellyfin can scan and import

**Technical Notes:**
- Implements FR91: Export metadata to Plex/Jellyfin
- Implements FR62: Export as NFO files
- NFO format: tvshow.nfo, movie.nfo

---

### Story 14.6: Watch Status Sync with Plex/Jellyfin

As a **media center user**,
I want **watch status synced with Plex/Jellyfin**,
So that **progress is consistent across platforms**.

**Acceptance Criteria:**

**Given** the user configures Plex integration
**When** entering Plex server details
**Then** connection is tested
**And** library matching is configured

**Given** sync is enabled
**When** user marks watched in Vido
**Then** Plex/Jellyfin is updated (if online)
**And** vice versa: Plex changes sync to Vido

**Given** conflict occurs
**When** both systems changed
**Then** user is prompted to resolve
**And** options: "Use Vido", "Use Plex", "Keep both"

**Given** sync fails
**When** Plex/Jellyfin is unreachable
**Then** changes are queued
**And** synced when connection restored

**Technical Notes:**
- Implements FR92: Sync watch status with Plex/Jellyfin
- Plex API and Jellyfin API clients
- Conflict resolution UI

---

### Story 14.7: Mobile Application Core

As a **mobile user**,
I want to **access Vido from my phone**,
So that **I can manage my library on the go**.

**Acceptance Criteria:**

**Given** the mobile app is installed
**When** the user opens the app
**Then** they connect to their Vido server URL
**And** authenticate with password/PIN

**Given** the user is authenticated
**When** using the mobile app
**Then** they can:
- Browse library
- View media details
- See download status
- Search for new content

**Given** mobile-optimized UI
**When** viewing content
**Then** interface adapts to phone screen
**And** touch gestures are supported

**Given** the server is unreachable
**When** the app loses connection
**Then** offline cached data is available
**And** message indicates limited mode

**Technical Notes:**
- Implements FR93: Mobile application access
- Implements UX-2: Mobile simplified monitoring
- React Native or Flutter (PWA as MVP)

---

### Story 14.8: Remote Download Control from Mobile

As a **mobile user**,
I want to **control downloads remotely**,
So that **I can manage my NAS when away from home**.

**Acceptance Criteria:**

**Given** the mobile app is connected
**When** viewing Downloads section
**Then** current download status is shown
**And** refreshes every 10 seconds

**Given** downloads are listed
**When** tapping a download
**Then** control options appear: Pause, Resume, Delete
**And** actions are sent to server

**Given** the user wants to add downloads
**When** searching from mobile
**Then** they can initiate new downloads
**And** downloads start on the NAS

**Given** notifications are enabled
**When** a download completes
**Then** push notification is sent
**And** tapping opens the completed item

**Technical Notes:**
- Implements FR94: Remote download control from mobile
- Push notifications via Firebase/APNs
- Requires VPN or exposed server for remote access
