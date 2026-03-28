# Technical Considerations

## Security & Privacy

**API Key Protection:**
- **Environment Variable Priority**: Support injecting all sensitive information from environment variables
  ```
  TMDB_API_KEY, GEMINI_API_KEY, QBITTORRENT_PASSWORD, ENCRYPTION_KEY
  ```
- **Encrypted Storage**: If users input API keys via UI, encrypt using AES-256
  - Encryption key itself read from environment variable `ENCRYPTION_KEY`
  - If environment variable not set, use machine-ID-derived key (fallback)
- **Zero-Logging Policy**: API keys never appear in logs, error messages, or HTTP responses

**Testing Requirements:**
- [ ] API keys do not appear in any logs (including debug mode)
- [ ] API keys masked in error messages
- [ ] Keys encrypted in database backup files

**User Data Privacy:**
- **Data Deletion**: Users must be able to completely clear all data:
  - Media library metadata
  - Download history
  - AI parsing cache
  - Learned filename mappings
  - User preferences
- **Sensitive Content Privacy**:
  - Media library may contain adult content or other sensitive materials
  - No automatic sharing or external reporting of user's media collection
  - Privacy-first approach: user data stays on their NAS

**Network Security:**
- **v4.0**: Single-user deployment, no authentication required
- **External Access Security**:
  - Support HTTPS for external access (reverse proxy compatible)
  - Avoid transmitting sensitive information in plain text over network
  - Recommend VPN or secure tunnel for external access
  - Rate limiting on API endpoints to prevent abuse (defense in depth)

---

## API Cost Management

**AI API Strategy:**
- **1.0 Approach**: Users provide their own API keys
  - Gemini/Claude API keys required for fansub parsing
  - Setup wizard guides users through API key acquisition
  - Zero cost to project maintainer
  - Trade-off: Higher barrier to entry for users

**TMDb API:**
- Free tier with quota limits
- Optional: Users can provide their own TMDb API key for higher quota
- Fallback to Douban if TMDb quota exhausted

**Future Consideration:**
- Post-1.0: Explore hybrid model with limited free AI parsing quota
- Possible premium tier or pooled API key sharing

---

## Performance & Scalability

**Database Architecture:**
- **1.0 Clear Boundary**: SQLite supports up to 10,000 media items
  - Covers 95% of NAS user scenarios
  - System recommends PostgreSQL when exceeding this limit

- **Repository Pattern (Must Implement in 1.0)**:
  - Database abstraction layer (`MediaRepository` interface)
  - Implementations: `SQLiteRepository` (1.0) + `PostgresRepository` (architecture preparation)
  - Future migration with zero breaking changes

- **Performance Monitoring from Day One**:
  - Track p95 latency for all database queries
  - System warning when homepage query p95 >100ms
  - System warning when search query p95 >500ms

**Large Library Handling:**

**Homepage Loading:**
- Load only 20-30 media items initially
- Pagination or infinite scroll for additional items
- Prioritize recently added/modified media
- Performance target: <2 seconds initial load

**Grid View Image Handling:**
- **Lazy loading**: Intersection Observer API implementation
- **Virtual scrolling**: Enabled when >1,000 items
- Image thumbnails cached locally
- Progressive loading: low-res placeholder → high-res on demand

**Search Performance:**
- **SQLite FTS5** (Full-Text Search) for Traditional Chinese/English search
- Indexed fields: titles (Chinese/English), director, cast, genre
- Search result pagination (20 items per page)
- Performance target: Full-text search <500ms (p95) for 10,000-item library

---

## Data Integrity & Backup

**Backup Strategy:**

**1. Atomic Backup Implementation:**
- Use SQLite `.backup` command (not file copy)
- Ensures consistency in WAL mode
- Backup includes: database snapshot + config files + AI learned mappings

**2. Backup Storage Architecture:**
- **Independent Docker Volume**: `/vido-backups` (separated from main data)
- Backup file format: `vido-backup-YYYYMMDD-HHMMSS-v{schema_version}.tar.gz`
- Schema versioning: Ensures forward compatibility

**3. Automated Backup Schedule:**
- Configurable frequency (daily/weekly)
- Retention policy: Last 7 daily backups + last 4 weekly backups
- Automatic cleanup of old backups

**4. Restore Functionality:**
- Backup integrity verification (checksum)
- One-click restore
- Auto-snapshot current state before restore

**Testing Requirements:**
- **CI/CD Quality Gate**: Every release must pass backup → restore tests
- **Test Scenarios**:
  - 10,000-item library backup <5 minutes
  - 100% data integrity after restore (checksum verification)
  - Cross-version restore testing (v1.0 backup → v1.2 restore)

**Export/Import Functionality:**
- **Export to JSON/YAML**: Complete library metadata export
- **Import from JSON/YAML**: Restore from backup or migrate between instances
- Export format human-readable and portable

**NFO File Support:**
- **Export metadata alongside media files** as `.nfo` files
- Kodi/Plex/Jellyfin compatible format
- Ensures metadata survives even if Vido database is lost
- Can be re-imported or used by other media tools

---

## Risk Assessment & Mitigation

| Risk | Severity | Mitigation |
|------|----------|------------|
| Silent backup failure | HIGH | Backup success/failure notifications + periodic backup verification |
| API key leakage | HIGH | Zero-logging policy + automated test validation |
| Large library performance degradation | MEDIUM | Performance monitoring + graceful degradation (virtual scrolling) |
| SQLite scalability limit | MEDIUM | Repository Pattern + PostgreSQL migration path |
| Docker volume data loss | MEDIUM | Independent backup volume + NFO file dual insurance |

---

## Integration & Extensibility

**qBittorrent Integration:**
- Secure credential storage (encrypted)
- Connection health monitoring
- Retry logic for temporary network failures
- Support for qBittorrent behind reverse proxy

**Plugin Architecture (v4 — Pluggable Integration Layer):**
- Go interfaces for external service integration: `MediaServerPlugin`, `DownloaderPlugin`, `DVRPlugin`
- Plugin Manager with registration, health check, configuration
- Each external service (Sonarr, Radarr, Plex, Jellyfin, qBittorrent, Prowlarr) implemented as a plugin
- Plugin configuration stored in SQLite
- Connection testing via `TestConnection(config)` before saving

**Subtitle Engine Integration:**
- Multi-source subtitle search: Assrt API, Zimuku scraper, OpenSubtitles API
- Subtitle scoring algorithm (language detection + release group match + format)
- OpenCC integration for 簡繁轉換 with terminology correction
- Pipeline: search → score → download → post-process → place

**SSE (Server-Sent Events) Hub:**
- Real-time progress push for download status, scan progress, subtitle status
- Replaces polling-based updates for download monitoring
- Native Go `http.Flusher` implementation, zero external dependencies
- Frontend `EventSource` integration

**Server-Side TMDB Filtering:**
- Post-filtering of TMDB trending/discover results in Go backend
- Filter criteria: language, region, release date range, vote threshold
- In-memory cache with 1-hour TTL to reduce API calls
- Eliminates Seerr's client-side filtering limitations

---

## Operational Requirements

**Deployment:**
- Docker containerization (primary deployment method)
- Docker Compose for easy setup
- Volume mounts for persistent data:
  - `/vido-data` - Main database and cache
  - `/vido-backups` - Backup storage (independent)
  - `/media` - User's media library (read-only)
- Environment variable configuration

**Monitoring & Logging:**
- System health dashboard
- Service connection status (qBittorrent, TMDb, AI APIs)
- Error logging with severity levels
- No sensitive data in logs (API keys, passwords masked)
- Performance metrics (query latency, cache hit rate)

**Updates:**
- Docker image versioning
- Automatic update notifications
- Database migration automation
- Backward compatibility for configuration

---

## TestSprite Journey Test Strategy

**Role in Test Pyramid:**
TestSprite provides journey-level integration tests that fill the gap between unit tests and Playwright E2E tests. While unit tests validate individual functions and Playwright verifies feature-point interactions and API contracts, TestSprite validates complete user journeys against PRD acceptance criteria on the live deployed application.

| Layer | Tool | Scope | Runs Against |
|-------|------|-------|-------------|
| Unit | Vitest / Go test | Functions, components | CI (in-process) |
| Journey Integration | TestSprite | PRD AC-level complete flows | Live NAS deployment |
| E2E | Playwright | Feature-point + API verification | CI or staging |

**Target URL & Execution Environment:**
- Target: `http://192.168.50.52:8088` (Unraid NAS direct LAN access)
- No tunnel or external access required — TestSprite connects directly to the NAS
- Tests execute against the actual production deployment with real data

**Division of Responsibility with Playwright E2E:**
- **TestSprite**: PRD acceptance-criteria-level complete journeys (e.g., "user scans library and sees metadata populated", "user searches for a movie and views detail page")
- **Playwright**: Feature-point verification, API contract assertions, and regression checks that run in CI against ephemeral environments

**Credits Budget & Regeneration Cadence:**
- Free plan: 150 credits/month (resets monthly)
- Budget sufficient for monthly full regeneration + 2–3 iteration cycles
- Regenerate all test cases from scratch when:
  - PRD acceptance criteria change
  - Significant UI or API changes are deployed
  - After a major bugfix sprint lands on NAS
- Strategy: discard and regenerate rather than maintain stale test cases

**Baseline Strategy:**
1. Snapshot the current app state as the baseline after deployment
2. After bugfix deployments, any intentional behavioral changes are marked as `intentional-change` in TestSprite
3. Update the baseline to reflect the corrected behavior
4. Unintentional regressions (not marked) surface as test failures

**Manual Trigger Workflow:**
- TestSprite runs are triggered manually after each bugfix deploy to NAS
- No CI integration — tests validate the live NAS environment post-deployment
- Workflow: deploy to NAS → trigger TestSprite run → review results → mark intentional changes → update baseline
