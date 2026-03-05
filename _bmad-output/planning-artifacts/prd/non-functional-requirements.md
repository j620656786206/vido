# Non-Functional Requirements

## Performance

**Page Load Performance:**

- **NFR-P1**: Homepage first contentful paint (FCP) must be <1.5s on first visit
- **NFR-P2**: Homepage largest contentful paint (LCP) must be <2.5s on first visit
- **NFR-P3**: Homepage time to interactive (TTI) must be <3.5s on first visit
- **NFR-P4**: Cumulative layout shift (CLS) must be <0.1 throughout entire session

**API Response Times:**

- **NFR-P5**: Search API must respond within <500ms (p95) for queries
- **NFR-P6**: Media library listing API must respond within <300ms (p95) for up to 1,000 items
- **NFR-P7**: Download status API must respond within <200ms (p95) for real-time updates

**Real-Time Update Performance:**

- **NFR-P8**: qBittorrent download status updates must reflect changes within <5 seconds
- **NFR-P9**: Media library updates must reflect new items within <30 seconds after parsing completion

**UI Responsiveness:**

- **NFR-P10**: Grid scrolling must maintain 60 FPS performance
- **NFR-P11**: Page transitions (routing) must complete within <200ms
- **NFR-P12**: Image lazy loading must complete within <300ms per viewport

**Parsing Performance:**

- **NFR-P13**: Standard regex-based filename parsing must complete within <100ms per file
- **NFR-P14**: AI-powered fansub parsing must complete within <10 seconds per file
- **NFR-P15**: Metadata retrieval from TMDb must complete within <2 seconds per query
- **NFR-P16**: Wikipedia fallback metadata retrieval must complete within <3 seconds per query

**Bundle Size:**

- **NFR-P17**: Initial JavaScript bundle size must be <500 KB (gzipped)
- **NFR-P18**: Route-specific code splitting must be implemented to reduce initial load

---

## Security

**API Key Protection:**

- **NFR-S1**: All API keys (TMDb, Gemini, Claude, qBittorrent password) must support environment variable injection
- **NFR-S2**: API keys stored via UI must be encrypted using AES-256 encryption
- **NFR-S3**: Encryption key must be read from environment variable `ENCRYPTION_KEY` or derived from machine ID as fallback
- **NFR-S4**: API keys must NEVER appear in logs, error messages, or HTTP responses (zero-logging policy)
- **NFR-S5**: API keys must be encrypted in database backup files

**User Data Privacy:**

- **NFR-S6**: Users must be able to completely delete all personal data (media library, download history, AI cache, learned mappings, preferences)
- **NFR-S7**: Media library data must remain local on user's NAS with no automatic external reporting
- **NFR-S8**: System must implement privacy-first approach with no telemetry or analytics by default

**Authentication & Access Control:**

- **NFR-S9**: All Vido endpoints must require user authentication (password/PIN)
- **NFR-S10**: User sessions must be managed with secure, cryptographically-signed tokens
- **NFR-S11**: API endpoints must be protected with authentication tokens
- **NFR-S12**: System must implement rate limiting on API endpoints to prevent abuse (max 100 requests/minute per IP)
- **NFR-S13**: Failed authentication attempts must be logged and rate-limited (max 5 attempts per 15 minutes)

**External Access Security:**

- **NFR-S14**: System must support HTTPS for external access (reverse proxy compatible)
- **NFR-S15**: Sensitive information must never be transmitted in plain text over network
- **NFR-S16**: System must recommend VPN or secure tunnel for external access in documentation

**Dependency Security:**

- **NFR-S17**: All dependencies must be scanned for critical vulnerabilities before each release
- **NFR-S18**: Critical security vulnerabilities must be patched within 7 days of disclosure
- **NFR-S19**: System must maintain zero critical security vulnerabilities in production

---

## Scalability

**Database Scalability:**

- **NFR-SC1**: SQLite database must support up to 10,000 media items with <500ms query latency (p95)
- **NFR-SC2**: System must warn users when media library exceeds 8,000 items and recommend PostgreSQL migration
- **NFR-SC3**: Repository Pattern must be implemented from MVP to enable zero-downtime PostgreSQL migration

**Concurrent Access:**

- **NFR-SC4**: System must support up to 5 concurrent user sessions (for household sharing scenarios)
- **NFR-SC5**: Database write operations must use proper locking to prevent corruption under concurrent access

**Large Library Handling:**

- **NFR-SC6**: Grid view must implement virtual scrolling when library exceeds 1,000 items
- **NFR-SC7**: Image thumbnails must be cached locally to reduce load times
- **NFR-SC8**: Search must use SQLite FTS5 full-text search for <500ms query time on 10,000-item library

**Growth Planning:**

- **NFR-SC9**: System architecture must support horizontal scaling for future multi-user scenarios
- **NFR-SC10**: Database schema must support future user table additions without breaking changes

---

## Reliability

**Uptime & Availability:**

- **NFR-R1**: System must maintain >99.5% uptime for self-hosted deployments (excludes user infrastructure failures)
- **NFR-R2**: System must gracefully handle all external API failures without crashing

**Error Handling & Recovery:**

- **NFR-R3**: When TMDb API fails, system must automatically retry with Douban within <1 second
- **NFR-R4**: When all metadata sources fail, system must gracefully degrade and provide manual search option
- **NFR-R5**: Failed parsing attempts must be queued for automatic retry (with exponential backoff: 1s → 2s → 4s → 8s)
- **NFR-R6**: System must auto-recover from qBittorrent connection failures with reconnection attempts every 30 seconds

**Data Integrity:**

- **NFR-R7**: Database backups must use SQLite `.backup` command (atomic consistency in WAL mode)
- **NFR-R8**: Backup integrity must be verified with checksum validation before marking backup as successful
- **NFR-R9**: System must automatically create snapshot before any restore operation
- **NFR-R10**: All database transactions must be ACID-compliant to prevent data corruption

**Failover & Degradation:**

- **NFR-R11**: When AI API quota is exhausted, system must fallback to regex parsing with user notification
- **NFR-R12**: When qBittorrent is unreachable, homepage must display connection status without blocking other features
- **NFR-R13**: System must maintain core functionality (search, library browsing) even when external APIs are down

---

## Integration

**qBittorrent Integration:**

- **NFR-I1**: System must support qBittorrent Web API v2.x and maintain backward compatibility
- **NFR-I2**: Connection health monitoring must detect failures within <10 seconds
- **NFR-I3**: System must support qBittorrent instances behind reverse proxy (custom base path)
- **NFR-I4**: Authentication credentials must be stored encrypted and never logged

**TMDb API Integration:**

- **NFR-I5**: System must implement TMDb API v3 with zh-TW language priority
- **NFR-I6**: System must respect TMDb API rate limits (40 requests/10 seconds)
- **NFR-I7**: System must cache TMDb responses for 24 hours to minimize API usage
- **NFR-I8**: System must handle TMDb API version upgrades (v3 → v4) with migration plan

**AI API Integration:**

- **NFR-I9**: System must support multiple AI providers (Gemini, Claude) with provider abstraction layer
- **NFR-I10**: AI parsing results must be cached for 30 days to reduce costs
- **NFR-I11**: System must track per-user AI API usage and display cost estimates
- **NFR-I12**: AI API calls must timeout after 15 seconds with fallback to manual search

**Wikipedia Integration:**

- **NFR-I13**: System must use MediaWiki API with proper User-Agent header (compliance)
- **NFR-I14**: System must respect Wikipedia API etiquette (max 1 request/second)
- **NFR-I15**: Wikipedia Infobox parsing must handle multiple template variations gracefully

**Future API Extensibility:**

- **NFR-I16**: RESTful API must be versioned (/api/v1) to support backward compatibility
- **NFR-I17**: API responses must follow OpenAPI/Swagger specification
- **NFR-I18**: Webhook support must allow external systems to subscribe to events (parsing complete, download finished)

---

## Maintainability

**Code Quality:**

- **NFR-M1**: Backend test coverage must be >80%
- **NFR-M2**: Frontend test coverage must be >70%
- **NFR-M3**: All public functions must have clear documentation comments
- **NFR-M4**: Code must follow language-specific linting rules (ESLint for TypeScript, golangci-lint for Go)

**Development Workflow:**

- **NFR-M5**: Hot reload must be supported for both frontend (Vite HMR) and backend (Air)
- **NFR-M6**: Database migrations must be versioned and automated
- **NFR-M7**: All configuration must support environment variable overrides

**Deployment & Updates:**

- **NFR-M8**: Docker image must support version pinning for reproducible deployments
- **NFR-M9**: Database schema changes must include automatic migration scripts
- **NFR-M10**: System must support zero-downtime updates for configuration changes

**Observability:**

- **NFR-M11**: System must log errors with severity levels (ERROR, WARN, INFO, DEBUG)
- **NFR-M12**: Performance metrics must be queryable via system dashboard (query latency, cache hit rate, API usage)
- **NFR-M13**: System health status must be visible on homepage (service connection status)

---

## Usability

**Deployment Ease:**

- **NFR-U1**: First-time deployment must complete within <5 minutes using Docker Compose
- **NFR-U2**: Setup wizard must guide users through configuration with <5 steps
- **NFR-U3**: System must provide sensible defaults requiring zero manual configuration for basic usage

**User Interface:**

- **NFR-U4**: All user actions must provide clear feedback within <200ms (loading indicators, success/error messages)
- **NFR-U5**: Error messages must be actionable (explain what went wrong and how to fix it)
- **NFR-U6**: System must support keyboard navigation for all primary workflows

**Documentation:**

- **NFR-U7**: API documentation must be auto-generated from OpenAPI/Swagger specification
- **NFR-U8**: User-facing documentation must include quick start guide, troubleshooting, and FAQ
- **NFR-U9**: Error logs must include actionable troubleshooting hints
