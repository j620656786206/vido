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

**Authentication & Access Control:**
- **1.0 Requirement**: Authentication required even for single-user deployment
- Simple password/PIN protection (prepare for future multi-user)
- Session management with secure tokens
- API endpoints protected with authentication tokens
- **External Access Security**:
  - Support HTTPS for external access (reverse proxy compatible)
  - Avoid transmitting sensitive information in plain text over network
  - Recommend VPN or secure tunnel for external access
  - Rate limiting on API endpoints to prevent abuse

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

**External API Access (Future-Ready Architecture):**
- RESTful API with versioning (`/api/v1`)
- OpenAPI/Swagger documentation
- API authentication with tokens
- Rate limiting to prevent abuse
- Webhook support for external automation

**Multi-User Preparation (Architecture Design):**
- Database schema supports future user tables
- Permission model foundation (even if 1.0 has single user)
- Authentication layer abstracted for future expansion

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
