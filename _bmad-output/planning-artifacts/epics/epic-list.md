# Epic List

## Epic 1: Project Foundation & Docker Deployment
**Phase:** MVP (Q1 - March 2026)

Users can deploy Vido on their NAS within 5 minutes using Docker Compose, with zero-configuration startup and sensible defaults. The foundation includes encrypted storage for sensitive data and environment variable configuration support.

**FRs covered:** FR47, FR48, FR49, FR50, FR51

**Implementation Notes:**
- ARCH-1: Repository Pattern mandatory from Day 1
- ARCH-10: Brownfield project with existing Go/Gin + React 19/TanStack stack
- NFR-U1: Docker Compose deployment <5 minutes
- NFR-SC3: Repository Pattern for SQLite → PostgreSQL migration path

---

## Epic 2: Media Search & Traditional Chinese Metadata
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

## Epic 3: AI-Powered Fansub Parsing & Multi-Source Fallback
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

## Epic 4: qBittorrent Download Monitoring
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

## Epic 5: Media Library Management
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

## Epic 6: System Configuration & Backup
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

## Epic 7: User Authentication & Access Control
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

## Epic 8: Advanced Download Control
**Phase:** Growth (Q3+ - September 2026+)

Users can control qBittorrent directly from Vido (pause/resume/delete torrents), adjust download priorities, manage bandwidth settings, and schedule downloads.

**FRs covered:** FR34, FR35, FR36, FR37

**Implementation Notes:**
- Builds upon Epic 4 (qBittorrent Monitoring)
- Extends qBittorrent API integration with write operations

---

## Epic 9: Subtitle Integration
**Phase:** Growth (Q3+ - September 2026+)

Users can search for subtitles (OpenSubtitles and Zimuku), with Traditional Chinese subtitles prioritized. Users can download subtitle files, manually upload subtitles, enable automatic subtitle downloads, and see subtitle availability status.

**FRs covered:** FR75, FR76, FR77, FR78, FR79, FR80

**Implementation Notes:**
- Key pain point from UX research: subtitle timeline matching
- Consider AI-assisted subtitle matching in future iterations

---

## Epic 10: Smart Recommendations & Discovery
**Phase:** Growth (Q3+ - September 2026+)

Users can receive smart recommendations based on genre, cast, and director, and see "similar titles" suggestions to discover new content.

**FRs covered:** FR9, FR10

**Implementation Notes:**
- Builds upon media library data from Epic 5
- Recommendation engine based on user's collection patterns

---

## Epic 11: Watch History & Collections
**Phase:** Growth (Q3+ - September 2026+)

Users can track personal watch history, see watch progress indicators, mark media as watched/unwatched, and create custom collections of media items.

**FRs covered:** FR43, FR44, FR45, FR46

**Implementation Notes:**
- Foundation for future multi-user personal tracking (Epic 13)
- Syncs with Plex/Jellyfin in Epic 14

---

## Epic 12: Automation & Organization
**Phase:** Growth (Q3+ - September 2026+)

The system can monitor watch folders to detect new files, automatically trigger parsing, rename files based on user-configured patterns, move files to organized directory structures, and execute automation tasks in background processing queue.

**FRs covered:** FR81, FR82, FR83, FR84, FR85, FR86

**Implementation Notes:**
- ARCH-4: Background Task Queue
- Builds upon AI parsing from Epic 3

---

## Epic 13: Multi-User Support
**Phase:** Growth (Q3+ - September 2026+)

The system supports multiple user accounts with admin/user permission management. Each user has their own personal watch history and preference settings.

**FRs covered:** FR71, FR72, FR73, FR74

**Implementation Notes:**
- NFR-SC4: Support 5 concurrent user sessions
- NFR-SC10: Database schema supports future user tables
- Extends Epic 7 authentication system

---

## Epic 14: External API & Mobile Application
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
