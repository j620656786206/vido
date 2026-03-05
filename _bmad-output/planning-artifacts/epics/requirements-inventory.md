# Requirements Inventory

## Functional Requirements

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

## NonFunctional Requirements

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

## Additional Requirements

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

## FR Coverage Map

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
