---
stepsCompleted: ['step-01-init', 'step-02-discovery', 'step-03-success', 'step-04-journeys', 'step-05-domain', 'step-06-innovation', 'step-07-project-type', 'step-08-scoping', 'step-09-functional', 'step-10-nonfunctional', 'step-11-polish']
inputDocuments:
  - 'ROADMAP.zh-TW.md'
  - 'docs/README.md'
  - 'docs/AIR_SETUP.md'
  - 'docs/SWAGGO_SETUP.md'
workflowType: 'prd'
briefCount: 0
researchCount: 0
brainstormingCount: 0
projectDocsCount: 4
classification:
  projectType: 'web_app'
  domain: 'general'
  complexity: 'medium'
  projectContext: 'brownfield'
  targetUsers: 'NAS users / Self-hosted server enthusiasts'
  coreDifferentiators:
    - 'Native Traditional Chinese metadata support'
    - 'Complete Chinese subtitle integration'
    - 'AI-powered fansub naming parsing'
  deploymentMode: 'Self-hosted / Local deployment'
  prdScope: 'Complete vision with clear 1.0 version scope'
---

# Product Requirements Document - vido

**Author:** Alexyu
**Date:** 2026-01-11

## Success Criteria

### User Success

**Core "Aha!" Moments:**
- Users experience unified workflow in a single interface:
  - qBittorrent download progress (speed, ETA, completion status)
  - Media library collection (grid/list views)
  - All media items displaying **perfect Traditional Chinese metadata and posters**

**Specific User Success Metrics:**
- First-time users complete their first media search and see Traditional Chinese results within **5 minutes**
- User satisfaction with AI parsing of **fansub naming** (e.g., `[Coalgirls] Show - 01 [BD 1080p].mkv`) >90%
- Users can track the complete "download → parse → manage" workflow without switching applications
- Users discover that Vido can parse naming conventions that other tools (Radarr/Sonarr/FileBot) cannot handle

### Business Success

**Phased Goals:**

**MVP (Q1 - March 2026):**
- Complete core search and metadata functionality
- Obtain feedback from first 50 early adopters

**1.0 Version (Q2 - June 2026):**
- Active users: **500+** (login at least once per week)
- User retention rate: >60% (30-day)
- Average media items per user: >100 items
- User satisfaction: >4.5/5
- **Success metric**: Correctly parse >95% of user files (including fansub naming)

**Growth Phase (Q3 - September 2026):**
- Active users: **1000+** (login at least once per week)
- Total managed media items: 100,000+
- Subtitle success rate: >90%

### Technical Success

**Performance Metrics:**
- Uptime: >99.5%
- API response time: <500ms (p95)
- Homepage load time: <2 seconds
- Build time: <2 minutes

**Parsing Accuracy:**
- **Overall file parsing success rate**: >95% (standard naming + fansub naming combined)
- **AI fansub parsing success rate**: >93% (at least 28 out of 30 correct)
- AI parsing response time: <10 seconds/file
- Metadata fallback success rate: >98% (at least one success in TMDb → Douban → AI chain)

**Quality Metrics:**
- Test coverage: Backend >80%, Frontend >70%
- Zero critical security vulnerabilities
- qBittorrent status update latency: <5 seconds

### Measurable Outcomes

**User Experience Milestones:**
1. First-time user completes media search and sees Traditional Chinese results: <5 minutes
2. AI parsing of fansub naming files completes: <10 seconds
3. Download status real-time updates: <5 seconds latency
4. From download completion to metadata fetched: <30 seconds (after automation)

**Competitive Advantage Validation:**
- Vido correctly parses fansub naming that Radarr/Sonarr cannot handle: >90% success rate
- Traditional Chinese metadata coverage: >95% (vs Jellyfin/Plex's limited support)
- Chinese subtitle auto-download success rate: >90% (vs Bazarr's broken providers)

## User Journeys

### Journey 1: Alex (NAS Media Collector) - The Perfect Day

**Character Background:**
- **Name:** Alex, 32 years old, Software Engineer
- **Situation:** Runs a Synology NAS at home, managing a media collection of over 500 movies and 200 TV shows. 70% is Asian content (Japanese anime, Taiwanese movies, Korean dramas), frequently encountering complex fansub naming conventions.
- **Goal:** Wants the media library to "organize itself" with perfect Traditional Chinese metadata, posters, and automatic metadata fetching after downloads complete.
- **Obstacles:**
  - Radarr/Sonarr cannot parse fansub naming like `[Leopard-Raws] Kimetsu no Yaiba - 01 (BD 1920x1080 x264 FLAC).mkv`
  - Jellyfin/Plex's Traditional Chinese metadata support is poor, often displaying Simplified Chinese or English
  - Constantly switching between qBittorrent, file manager, and Jellyfin
- **Solution:** Vido lets him handle everything in one place

**Journey Narrative:**

**Opening - Saturday Morning, Alex Discovers New Episodes**

Alex opens qBittorrent and sees 3 new anime episodes downloaded overnight. He sighs, preparing for the "weekend organization ritual": manually renaming files, searching for metadata, copying posters...

Suddenly remembers Vido recommended by a friend and decides to give it a try.

**Act One - First Launch of Vido**

1. Alex starts the Vido Docker container on his NAS
2. Browser opens to `http://nas.local:8080`
3. Sees a clean welcome screen: "Connect your qBittorrent"
4. Enters qBittorrent IP and credentials
5. **Aha Moment #1**: Immediately sees his 3 download items showing 100% progress, seeding

**Act Two - The Magic Happens**

6. Alex clicks "Scan completed downloads"
7. Vido detects 3 new files, one being:
   ```
   [Leopard-Raws] Kimetsu no Yaiba - 26 (END) (BD 1920x1080 x264 FLAC).mkv
   ```
8. He thinks: "Radarr is definitely going to fail on this..."
9. Waits 8 seconds... Vido displays:
   - ✅ **鬼滅之刃 (Traditional Chinese title)**
   - Episode 26 (Final)
   - Complete Traditional Chinese plot summary
   - Beautiful anime poster
10. **Aha Moment #2**: "Oh my god! It parsed it correctly!"

**Act Three - Exploring the Media Library**

11. Alex switches to "Library" page
12. Sees grid view with all media showing:
    - Traditional Chinese titles (not Simplified!)
    - High-quality posters
    - Year, genre, ratings
13. He tries searching for "台北物語", finds it immediately with perfect metadata
14. **Aha Moment #3**: "This is exactly what I've always wanted!"

**Act Four - Unified Dashboard Experience**

15. Alex returns to homepage, sees unified dashboard:
    - Left: qBittorrent download list (2 downloading, 5 seeding)
    - Right: Recently added media (including the 3 newly parsed anime)
    - Bottom: Quick TMDb search
16. **Aha Moment #4**: "Finally no more jumping between multiple apps!"

**Resolution - New Life**

17. Alex sits on the couch, opens Vido on his phone
18. Sees download progress: new movie has 30 minutes remaining
19. Switches to library, browses collection, all Chinese titles display perfectly
20. He messages his friend: "Vido is amazing, it can even parse fansub naming!"

**Requirements Revealed by Journey:**
- qBittorrent connection and authentication
- Real-time download status sync (<5 seconds)
- AI filename parsing (handles fansub naming)
- Traditional Chinese metadata priority fetching
- Unified dashboard (downloads + media library)
- Responsive design (mobile/desktop)
- Media search functionality (TMDb)

---

### Journey 2: Alex - When Things Don't Go as Expected (Edge Cases & Error Handling)

**Situation:** Not every file can be parsed perfectly, and metadata sources can fail. Alex encounters some tricky situations.

**Journey Narrative:**

**Opening - Encountering Weird Filenames**

Alex downloads an old anime with the filename:
```
【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4
```

He thinks: "This is even more complex, Chinese fansub group name..."

**Act One - AI Parsing Challenge**

1. Vido starts parsing this file
2. AI processes for 12 seconds (longer than usual)
3. Result shows:
   - ✅ **我的英雄學院** (Correct!)
   - Episode 1
   - But metadata source shows: "TMDb failed → Douban succeeded"
4. **Key Moment**: Alex sees Vido automatically switched to Douban, fetched Traditional Chinese info
5. He thinks: "The fallback mechanism really works!"

**Act Two - Completely Unrecognizable File**

6. Another file is stranger:
   ```
   abc_s01_e05_final_v2_repack.mkv
   ```
7. Vido displays:
   - ⚠️ "Unable to auto-parse"
   - Provides manual edit option
8. Alex clicks "Manual search"
9. Enters "ABC Season 1"
10. Selects correct series
11. Vido saves mapping: will remember for similar filenames next time

**Act Three - All Metadata Sources Fail**

12. One day, TMDb API is down (maintenance)
13. Douban is also inaccessible (network issue)
14. Alex adds a new file
15. Vido displays:
    - ⚠️ "Metadata sources temporarily unavailable"
    - But file info is saved
    - "Will auto-retry in 30 minutes"
16. **Key Moment**: System doesn't crash, gracefully degrades
17. 30 minutes later, TMDb recovers, Vido auto-fills metadata

**Act Four - Manual Correction of AI Error**

18. AI occasionally makes mistakes, misidentifying "Attack on Titan Season 2" as "Season 1"
19. Alex sees "Edit Metadata" button on media detail page
20. Corrects season number, saves
21. Vido asks: "Learn this correction? Future similar filenames will use this rule"
22. Alex selects "Yes"
23. **Key Moment**: System learns from errors, gets smarter with use

**Resolution - Resilience & Trust**

24. Alex realizes Vido isn't "perfect" but "never gives up"
25. Even with the weirdest filenames, API outages, AI errors...
26. There's always a backup, always manual options, always recovery
27. He tells his friend: "Vido's strength isn't 100% accuracy, it's always giving you choices"

**Requirements Revealed by Journey:**
- Multi-source fallback mechanism (TMDb → Douban → AI)
- Manual search and metadata editing
- Filename mapping learning mechanism
- Graceful degradation (handling API failures)
- Auto-retry mechanism (background tasks)
- User feedback learning system
- Parse status indicators (success/failure/processing)

---

### Journey 3: Alex (System Administrator Role) - Initial Setup & Maintenance

**Situation:** Before enjoying Vido, Alex needs to complete setup. He also needs periodic system maintenance.

**Journey Narrative:**

**Opening - Deciding to Try Vido**

Alex sees Vido discussion on Reddit, decides to try it on his Synology NAS.

**Act One - Zero-Config Installation**

1. Alex downloads Vido Docker compose file
2. Runs `docker-compose up -d`
3. Waits 30 seconds... container starts
4. Browser opens to `http://nas.local:8080`
5. **Key Moment**: No complex setup wizard, straight to clean welcome page
6. He thinks: "Really is zero-config!"

**Act Two - Integration Configuration**

7. Vido prompts: "Connect your download tool"
8. Alex enters qBittorrent settings:
   - Host: `192.168.1.100:8080`
   - Username/password
9. Clicks "Test Connection" → ✅ Success
10. Vido prompts: "Configure media folders"
11. Alex enters: `/volume1/media/movies` and `/volume1/media/tv`
12. Vido asks: "Need TMDb API key to increase quota (optional)"
13. Alex skips (use public quota)
14. **Key Moment**: All setup completed within 5 minutes

**Act Three - Daily Maintenance**

15. A month later, Alex notices Vido's disk usage increasing
16. Goes to "Settings → Cache Management"
17. Sees:
    - Image cache: 2.3 GB
    - AI parsing cache: 450 MB
    - Clear cache older than 30 days?
18. Clicks "Clear", reclaims 1.8 GB space

**Act Four - Troubleshooting**

19. One day, qBittorrent connection fails
20. Vido homepage shows: ⚠️ "qBittorrent connection failed - Last success: 2 minutes ago"
21. Alex checks logs: "Settings → System Logs"
22. Sees error: `Connection refused: 192.168.1.100:8080`
23. He realizes qBittorrent restarted, IP unchanged but connection temporarily interrupted
24. Clicks "Reconnect" → ✅ Back to normal

**Act Five - System Upgrade**

25. Vido displays notification: "New version available: v1.2.0 - Subtitle support added"
26. Alex clicks "View changelog"
27. After confirming, clicks "Upgrade"
28. Vido executes:
    - Backup current settings
    - Pull new Docker image
    - Migrate database (if needed)
    - Restart service
29. 5 minutes later, upgrade complete
30. **Key Moment**: Zero-downtime upgrade, all data preserved

**Resolution - Low Maintenance Burden**

31. Alex discovers Vido needs almost no maintenance
32. Occasional cache cleanup, check updates
33. System auto-handles most issues (retry, fallback, error recovery)
34. He tells his friend: "Set it once, forget about it, but it keeps working silently"

**Requirements Revealed by Journey:**
- Docker containerized deployment
- Zero-config startup (sensible defaults)
- Setup wizard (qBittorrent, media folders, API keys)
- Connection test functionality
- Cache management interface
- System log viewing
- Health status monitoring (service connection status)
- Auto-update notifications
- Backup and migration mechanism

---

### Journey 4: Developer David - API Integration (Future Flexibility Consideration)

**Situation:** David is a Python developer who wants to build automation scripts integrating Vido into his media workflow. While 1.0 may not have a complete public API yet, the system architecture needs to consider this extensibility.

**Journey Narrative (Simplified, Focusing on Architecture Needs):**

**Opening**

David wants to build an automation script: when qBittorrent completes a download, automatically trigger Vido parsing and notify him.

**Key Requirements:**

1. **API Authentication**: David needs API token to authenticate requests
2. **Trigger Parsing**: `POST /api/v1/parse` - Manually trigger file parsing
3. **Query Status**: `GET /api/v1/media/{id}` - Query media information
4. **Webhook Callback**: When parsing completes, Vido calls David's webhook
5. **Error Handling**: Clear HTTP status codes and error messages

**Architecture Considerations:**
- RESTful API design
- OpenAPI/Swagger documentation
- Versioning (/api/v1)
- Rate limiting (prevent abuse)
- Webhook subscription mechanism

---

### Journey Requirements Summary

These journeys reveal the following major capability areas:

**Core Capabilities:**
1. qBittorrent integration and real-time sync
2. AI-powered filename parsing (fansub naming)
3. Multi-source metadata fallback (TMDb → Douban → AI)
4. Unified dashboard (downloads + media library)
5. Traditional Chinese priority metadata

**Resilience Mechanisms:**
6. Graceful degradation and error recovery
7. Manual editing and search
8. Auto-retry mechanism
9. Learning and improvement system

**Management & Maintenance:**
10. Zero-config deployment (Docker)
11. Setup wizard and connection testing
12. System monitoring and logging
13. Cache management
14. Auto-update mechanism

**Extensibility (Future):**
15. RESTful API (preserve flexibility)
16. Webhook mechanism
17. External integration support

## Technical Considerations

### Security & Privacy

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

### API Cost Management

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

### Performance & Scalability

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

### Data Integrity & Backup

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

### Risk Assessment & Mitigation

| Risk | Severity | Mitigation |
|------|----------|------------|
| Silent backup failure | HIGH | Backup success/failure notifications + periodic backup verification |
| API key leakage | HIGH | Zero-logging policy + automated test validation |
| Large library performance degradation | MEDIUM | Performance monitoring + graceful degradation (virtual scrolling) |
| SQLite scalability limit | MEDIUM | Repository Pattern + PostgreSQL migration path |
| Docker volume data loss | MEDIUM | Independent backup volume + NFO file dual insurance |

---

### Integration & Extensibility

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

### Operational Requirements

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

## Innovation & Novel Patterns

### Detected Innovation Areas

**1. AI-Powered Fansub Filename Parsing (Market First)**

**Innovation Essence:**
Vido is the first media management tool to use Large Language Models (Gemini/Claude) to parse complex fansub naming conventions.

**Pain Point Addressed:**
Existing tools (Radarr, Sonarr, FileBot) rely on regular expressions and cannot handle:
- Fansub group tags: `[Leopard-Raws]`, `[Coalgirls]`, `【幻櫻字幕組】`
- Mixed-language titles: Japanese original + Chinese translation
- Non-standard episode markers: Absolute numbering, multi-episode packs, OVA/Movie labels
- Quality/encoding information: `[BD 1920x1080 x264 FLAC]`

**Technical Approach:**
```
Complex filename → LLM parsing → Structured information extraction → Search keyword generation
```

Example:
- Input: `【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4`
- AI Extraction:
  - Fansub group: 幻櫻字幕組 (ignore)
  - Title: 我的英雄學院
  - Episode: #1
  - Language: Traditional Chinese
- Search keywords: `"我的英雄學院" episode 1`

**Market Differentiation:**
- Radarr/Sonarr: Complete failure to parse → **Failed**
- FileBot: Relies on community scripts, low coverage → **Partial success**
- Vido: AI contextual understanding → **>93% success rate target**

---

**2. Multi-Source Resilience Architecture (Zero-Failure Design)**

**Innovation Essence:**
Four-layer metadata retrieval strategy ensuring "never complete failure":

```
Layer 1: TMDb API (zh-TW priority)
   ↓ (Failure - API down or not found)

Layer 2: Douban Web Scraper (Traditional Chinese)
   ↓ (Failure - Network issues or not found)

Layer 3: Wikipedia Search (Multilingual)
   - Search Traditional Chinese Wikipedia
   - Extract Infobox metadata (title, director, cast, year, genre)
   - ⚠️ No poster images (display default icon or use Layer 1/2 cache)
   ↓ (Failure - No matching entry)

Layer 4: AI Intelligent Search Assistant
   - Re-parse filename, generate multiple search keywords
   - Retry Layers 1-3 with new keywords
   ↓ (Still fails)

Layer 5: Manual Search Option
```

**Layer 3 Innovation: Wikipedia as Free Fallback**

Not "generating metadata from nothing," but:
1. Free, unlimited API access (MediaWiki API)
2. Multi-language support (zh.wikipedia.org for Traditional Chinese)
3. Rich text metadata from Infobox (director, cast, year, genre, plot)
4. High reliability (community-maintained, timely updates)
5. No poster images (limitation, but better than no metadata)

**Layer 4 Innovation: AI as Intelligent Search Assistant**

Not "generating metadata from nothing," but:
1. AI re-parses filename, extracts keywords from different angles
2. Generates multiple search strategies (original, translated, English, Romaji)
3. Auto-retries TMDb/Douban/Wikipedia searches
4. Selects best match from search results

**Example Scenario:**
- Filename: `鬼滅之刃.S01E26.mkv`
- TMDb search "鬼滅之刃" → No results (Simplified Chinese index)
- AI generates alternative searches:
  - "鬼灭之刃" (Simplified)
  - "Demon Slayer" (English)
  - "Kimetsu no Yaiba" (Japanese Romaji)
- TMDb search "Demon Slayer" → ✅ Found

**Market Differentiation:**
- Jellyfin/Plex: Single source, gives up on failure
- Vido: Multi-source + Wikipedia + AI retry, maximizes success rate

**Coverage Improvement:**
- Three-layer fallback (TMDb → Douban → Wikipedia): >98% metadata coverage
- Four-layer with AI retry: >99% metadata coverage (estimated)
- Even without posters, at least basic information available

---

**3. Traditional Chinese Priority Strategy (Execution Innovation)**

**Innovation Essence:**
Not a technical breakthrough, but market positioning and execution innovation.

**Differentiated Execution:**
- TMDb API default language: `zh-TW` (not `zh-CN`)
- Douban scraper: Traditional Chinese priority display
- Wikipedia: Search zh.wikipedia.org first
- AI prompts: Explicitly request Traditional Chinese output
- UI/UX: All metadata displays Traditional Chinese first

**Market Opportunity:**
Existing tools' Traditional Chinese support:
- Plex/Jellyfin: Mixed Simplified/Traditional, poor experience
- Radarr/Sonarr: Primarily English, very limited Chinese support
- Vido: Traditional Chinese as first-class citizen

---

### Market Context & Competitive Landscape

**Competitor Analysis:**

| Tool | Fansub Parsing | Metadata Sources | Traditional Chinese | Fallback Mechanism |
|------|---------------|------------------|---------------------|-------------------|
| **Radarr/Sonarr** | ❌ Failed | TMDb only | ⚠️ Limited | ❌ None |
| **FileBot** | ⚠️ Community scripts | TMDb/TVDb | ⚠️ Limited | ❌ None |
| **Jellyfin** | ❌ Failed | TMDb only | ⚠️ Mixed | ❌ None |
| **Plex** | ❌ Failed | Proprietary DB | ⚠️ Mixed | ⚠️ Limited |
| **Vido** | ✅ AI | TMDb→Douban→Wikipedia→AI | ✅ Native | ✅ Four-layer |

**Market Gap:**
- NAS user community (especially Taiwan/Hong Kong) has long complained about metadata issues
- Fansub naming is a common complaint on forums/Reddit
- Existing tools' Chinese support seen as "afterthought" rather than core feature

**Vido's Positioning:**
"The first media management tool designed specifically for Traditional Chinese users and fansub content"

---

### Validation Approach

**AI Parsing Accuracy Validation:**

**1. Benchmark Dataset**
- Collect 1,000 real fansub filenames
- Categories:
  - Standard naming (300)
  - Chinese fansub groups (300)
  - Japanese fansub groups (300)
  - Extreme complex cases (100)
- Manually annotate correct parsing results

**2. Accuracy Metrics**
- **Overall accuracy target**: >95%
  - Standard naming: >99%
  - Fansub naming: >93%
  - Extreme cases: >80%
- **Measurement**:
  - Title match: Exact match vs partial match
  - Episode accuracy: Precise to episode number
  - Metadata retrieval success rate

**3. User Satisfaction**
- MVP phase (first 50 users) collect feedback
- Question: "What percentage of your files did Vido successfully parse?"
- Target: >90% users report "most" or "all" successful

**4. Cost Monitoring**
- Track AI API usage
- Per-file average cost target: <$0.05 USD
- Per-user monthly cost target: <$2 USD (assuming 50 new files/month)

**5. Wikipedia Effectiveness**
- Track Layer 3 hit rate
- Measure cases where Wikipedia provides metadata when TMDb/Douban fail
- Monitor user acceptance of "no poster" metadata entries

---

### Risk Mitigation

**Risk 1: AI Cost Too High**

**Scenario**: Users heavily use AI parsing, API costs explode

**Mitigation**:
- **Caching Strategy**: Similar filename parsing results cached for 30 days
- **Smart Triggering**: Only invoke AI when basic regex fails
- **User-Paid**: Users provide their own API keys (1.0 strategy)
- **Degradation Option**: Users can disable AI parsing, fall back to basic mode

**Trigger**: If average cost >$0.10/file, activate cost optimization

---

**Risk 2: AI Accuracy Lower Than Expected**

**Scenario**: AI parsing accuracy <90%, users frequently need manual correction

**Mitigation**:
- **Hybrid Mode**: Regex + AI dual validation
  - If both agree → High confidence
  - If disagree → Prompt user to choose
- **Learning Mechanism**: After user manual correction, system remembers mapping
- **Community Dataset**: Collect user-contributed parsing rules
- **Fallback Option**: If AI consistently fails, suggest manual search

**Trigger**: If benchmark <93% or user satisfaction <85%

---

**Risk 3: External API Dependencies (TMDb/Douban/Wikipedia)**

**Scenario**: TMDb changes API, Douban upgrades anti-scraping, Wikipedia blocks requests

**Mitigation**:
- **Multi-Source Architecture**: One fails, auto-switch to next
- **Data Caching**: Already-fetched metadata permanently saved (local first)
- **NFO Backup**: Metadata exported as NFO files, not dependent on external services
- **API Version Locking**: TMDb API v3 → v4 upgrade prepared in advance
- **Wikipedia Compliance**: Respect MediaWiki API guidelines, set proper User-Agent

**Trigger**: Regular health checks, warn when API failure rate >10%

---

**Risk 4: Traditional Chinese Market Too Small**

**Scenario**: Target market (Taiwan/Hong Kong NAS users) cannot sustain development costs

**Mitigation**:
- **Internationalization Path**: Architecture supports multiple languages (though 1.0 prioritizes Traditional Chinese)
- **Open Source Strategy**: Open source project can gain community contributions
- **Cross-Domain Application**: AI parsing technology applicable to other domains (music, games)

---

**Risk 5: Wikipedia Metadata Quality Issues**

**Scenario**: Wikipedia entries have inconsistent Infobox formats or missing information

**Mitigation**:
- **Robust Parsing**: Handle multiple Infobox template variations
- **Validation**: Cross-check extracted data for consistency
- **Graceful Degradation**: If Infobox parsing fails, still mark as "searched but no data"
- **User Feedback**: Allow users to report Wikipedia metadata issues
- **Fallback to AI**: If Wikipedia data quality poor, proceed to Layer 4

**Trigger**: If Wikipedia success rate <50%, consider adjusting priority order

## Web Application Specific Requirements

### Project-Type Overview

**Architecture Pattern: Single Page Application (SPA)**

Vido adopts modern SPA architecture:
- **Frontend**: React 19 + TanStack Router + TanStack Query
- **Backend**: Go/Gin RESTful API
- **Communication**: JSON over HTTP/HTTPS
- **State Management**: TanStack Query for server state, React hooks for UI state

**SPA Benefits for Vido:**
- Smooth user experience (no full page refreshes)
- Suitable for real-time updates (download progress, library changes)
- Offline caching strategy (PWA potential)
- Reduced server burden (static files + API)

---

### Browser Support Matrix

**Supported Browsers (1.0):**

| Browser | Minimum Version | Test Priority |
|---------|----------------|---------------|
| **Chrome** | Latest | P0 (Primary testing) |
| **Firefox** | Latest | P0 (Primary testing) |
| **Safari** | Latest | P1 (macOS NAS users) |
| **Edge** | Latest | P1 (Windows NAS users) |
| **iOS Safari** | iOS 15+ | P0 (Mobile access) |
| **Android Chrome** | Latest | P0 (Mobile access) |

**Explicitly NOT Supported:**
- ❌ Internet Explorer (any version)
- ❌ Legacy browsers (>2 years old)

**Browser Feature Requirements:**
- ES6+ JavaScript support
- CSS Grid & Flexbox
- Fetch API
- LocalStorage
- Intersection Observer (lazy loading)
- Modern CSS (CSS Variables, calc())

**Polyfills Strategy:**
- No polyfills provided (keep bundle lightweight)
- Unsupported browsers show upgrade prompt

---

### Responsive Design

**Breakpoints:**

```css
/* Mobile First Approach */
- Mobile: 320px - 767px (iPhone SE → iPhone 14 Pro Max)
- Tablet: 768px - 1023px (iPad)
- Desktop: 1024px+ (Desktop browsers, NAS admin interface)
```

**Layout Adaptations:**

**Mobile (320-767px):**
- Single column layout
- Grid view: 2 column posters
- Hide secondary information
- Bottom navigation bar
- Touch optimized (buttons >44px)

**Tablet (768-1023px):**
- Two column layout (sidebar + main content)
- Grid view: 3-4 column posters
- Show some secondary information

**Desktop (1024px+):**
- Three column layout (sidebar + main content + detail panel)
- Grid view: 4-6 column posters
- Full information display
- Mouse hover effects

**Touch vs Mouse Optimization:**
- Mobile devices: Large touch targets, swipe gestures, pull-to-refresh
- Desktop: Mouse hover, right-click menus, keyboard shortcuts

---

### Performance Targets

**Page Load Performance:**

| Metric | Target | Measurement Condition |
|--------|--------|---------------------|
| **First Contentful Paint (FCP)** | <1.5s | First visit |
| **Largest Contentful Paint (LCP)** | <2.5s | First visit |
| **Time to Interactive (TTI)** | <3.5s | First visit |
| **Cumulative Layout Shift (CLS)** | <0.1 | Entire session |

**Runtime Performance:**

| Operation | Target Latency | Notes |
|-----------|---------------|-------|
| Page transition (routing) | <200ms | SPA navigation |
| Search response | <500ms | Local + API |
| Download status update | <5s | Polling interval |
| Grid scrolling | 60 FPS | Virtual scrolling |
| Image lazy load | <300ms | Intersection Observer |

**Bundle Size Targets:**

- Initial bundle: <500 KB (gzipped)
- Route-based code splitting
- Dynamic imports for heavy components
- Image optimization (WebP, responsive images)

---

### Real-Time Update Mechanism

**Strategy: Polling (1.0 Approach)**

**Why Polling for 1.0:**
- ✅ Simple implementation (TanStack Query built-in support)
- ✅ No WebSocket server maintenance needed
- ✅ Firewall/proxy friendly
- ✅ Automatic reconnection mechanism
- ⚠️ More resource-intensive (acceptable for self-hosted environment)

**Polling Configuration:**

```typescript
// qBittorrent download status
const { data: downloads } = useQuery({
  queryKey: ['downloads'],
  queryFn: fetchDownloads,
  refetchInterval: 5000, // 5 second polling
  refetchIntervalInBackground: false, // No polling in background
});

// Library updates (less frequent)
const { data: library } = useQuery({
  queryKey: ['library'],
  queryFn: fetchLibrary,
  refetchInterval: 30000, // 30 second polling
  staleTime: 10000, // Consider fresh for 10 seconds
});
```

**Smart Polling Optimization:**
- Stop polling when user not on current page (`refetchIntervalInBackground: false`)
- Immediately refresh when user returns (`refetchOnWindowFocus: true`)
- Auto-trigger library refresh after download completion
- Exponential backoff after errors (1s → 2s → 4s → 8s)

**Future Consideration (Post-1.0):**
- WebSocket or SSE for true real-time updates
- Reduce server load
- Better battery efficiency (mobile devices)

---

### SEO Strategy

**SEO Requirement: None**

Vido is a self-hosted tool deployed on private networks:
- ❌ No search engine indexing needed
- ❌ No Open Graph tags needed
- ❌ No sitemap.xml needed
- ❌ No SSR (Server-Side Rendering) needed

**Minimal Meta Tags (Good Practice):**
```html
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>Vido - Media Management</title>
```

---

### Accessibility

**1.0 Priority: Low (But Architecture Ready)**

**Minimum Requirements (1.0):**
- ✅ Semantic HTML (`<main>`, `<nav>`, `<article>`)
- ✅ Basic ARIA labels (`role`, `aria-label`)
- ✅ Keyboard navigation (Tab, Enter, Escape)
- ✅ Focus visibility (outline styles)
- ✅ Alt text (images, posters)

**Future Enhancements (Post-1.0):**
- WCAG 2.1 Level AA compliance
- Full screen reader support
- High contrast mode
- Font size adjustment
- Keyboard shortcut system

**Testing Approach:**
- Manual keyboard navigation testing
- Lighthouse Accessibility Score >70 (1.0 target)
- axe DevTools automated detection (during development)

---

### Frontend Tech Stack Detailed Specification

**Core Framework:**
- React 19 (latest stable)
- TypeScript (strict mode)
- Vite (build tool)

**Routing:**
- TanStack Router v1
- Type-safe routes
- Code splitting per route

**Data Fetching:**
- TanStack Query v5
- Optimistic updates
- Cache management

**Styling:**
- CSS Modules or Tailwind CSS (TBD)
- Responsive design utilities
- Dark mode support (architecture ready, 1.0 optional)

**Form Handling:**
- React Hook Form
- Zod schema validation

**State Management:**
- TanStack Query (server state)
- React Context + hooks (UI state)
- LocalStorage (user preferences)

**Testing:**
- Vitest (unit tests)
- React Testing Library (component tests)
- Playwright (E2E tests, post-1.0)

---

### Implementation Considerations

**Development Workflow:**
- Nx monorepo architecture (already established)
- Hot reload with Air (backend) and Vite HMR (frontend)
- OpenAPI/Swagger for API documentation
- Git-based version control

**Deployment:**
- Docker containerization
- Single Docker Compose deployment
- Frontend: Static files served by Nginx or backend
- Backend: Go binary with embedded static files option

**Progressive Enhancement:**
- Core functionality works without JavaScript (limited)
- Enhanced experience with JavaScript enabled
- Graceful degradation for unsupported browsers

**Internationalization (Future):**
- Architecture supports i18n (react-i18next ready)
- 1.0: Traditional Chinese + English only
- Post-1.0: Community translations

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-Solving MVP

Vido's MVP focuses on solving a single, well-defined pain point: **inability to parse fansub naming conventions** that other tools (Radarr/Sonarr/FileBot) cannot handle.

**Core Philosophy:**
- Minimum viable feature set that makes users say "this is useful"
- Fast path to validated learning with 50 early adopters
- Extreme simplification of initial scope to reduce risk
- Foundation for iterative improvement

**Why Problem-Solving MVP:**
- ✅ Addresses a real, measurable pain point (fansub parsing failures)
- ✅ Clear success metrics (parsing accuracy, user satisfaction)
- ✅ Enables rapid validation with target user community
- ❌ Not experience-focused (UX polish comes later)
- ❌ Not platform-focused (API/extensibility comes later)
- ❌ Not revenue-focused (self-hosted tool, not SaaS)

**Resource Requirements:**
- **Team Size:** Single full-stack developer
- **Skills Required:**
  - Frontend: React, TypeScript, TanStack Router/Query
  - Backend: Go, Gin framework, SQLite
  - DevOps: Docker, basic networking
- **Time Estimate:** 6-8 weeks for MVP (Q1 target)
- **Infrastructure:** Development machine + test NAS environment

---

### MVP Feature Set (Phase 1 - Q1 March 2026)

**Core User Journeys Supported:**
- **Journey 1 (Partial):**
  - Act One-Two: Search movies/TV shows and view Traditional Chinese metadata
  - Basic filename parsing (regex-based, standard naming only)
- **Journey 3 (Partial):**
  - Act One-Two: Zero-config Docker installation and basic setup

**Must-Have Capabilities:**

**1. Search & Metadata Display**
- TMDb API integration with zh-TW language priority
- Search by title (Chinese or English)
- Display Traditional Chinese metadata:
  - Title, description, release year
  - Poster images
  - Genre, director, cast (basic info)

**2. Basic Filename Parsing**
- Regex-based parser for standard naming conventions:
  - `Movie.Name.2024.1080p.BluRay.mkv`
  - `Show.Name.S01E05.1080p.WEB-DL.mkv`
- Extract: Title, Year, Season/Episode
- **Explicitly NOT included in MVP:**
  - ❌ AI-powered parsing (deferred to 1.0)
  - ❌ Fansub naming support (deferred to 1.0)
  - ❌ Chinese fansub group tags

**3. Data Storage**
- SQLite database with WAL mode
- **Repository Pattern implementation** (critical for future PostgreSQL migration)
- Store: Media metadata, search history, user preferences
- Basic CRUD operations

**4. Minimal Web UI**
- Single-page search interface
- Grid view for search results
- Media detail page (read-only)
- Responsive design (mobile + desktop)
- **Explicitly NOT included in MVP:**
  - ❌ Media library management
  - ❌ Batch operations
  - ❌ Advanced filtering/sorting

**5. Deployment**
- Docker containerization
- Docker Compose setup
- Volume mounts for data persistence
- Environment variable configuration

**MVP Success Criteria:**
- 50 early adopters complete installation
- Users can search and see Traditional Chinese metadata within 5 minutes
- Basic filename parsing >90% success rate (standard naming only)
- Positive feedback: "This solves my problem" from early users

**What's Explicitly NOT in MVP:**
- ❌ AI-powered fansub parsing (core 1.0 feature)
- ❌ qBittorrent integration
- ❌ Multi-source metadata fallback
- ❌ Media library management UI
- ❌ Download monitoring
- ❌ Subtitle integration
- ❌ Multi-user support

---

### Post-MVP Features

#### Phase 2: 1.0 Core Platform (Q2 - June 2026)

**Core User Journeys Supported:**
- **Journey 1 (Complete):** Full fansub parsing workflow with AI
- **Journey 2 (Complete):** Multi-source fallback and error handling
- **Journey 3 (Complete):** System maintenance and upgrades

**Planned 1.0 Features:**

**1. AI-Powered Fansub Parsing** ⭐ Core Innovation
- Gemini/Claude API integration
- Parse complex fansub naming:
  - `[Leopard-Raws] Show - 01 (BD 1920x1080 x264 FLAC).mkv`
  - `【幻櫻字幕組】【4月新番】Title 第01話 1080P【繁體】.mp4`
- Target accuracy: >93% for fansub naming
- Caching mechanism to reduce API costs
- User-provided API keys (cost mitigation)

**2. Multi-Source Metadata Fallback** ⭐ Core Innovation
- Four-layer resilience architecture:
  - Layer 1: TMDb API (zh-TW priority)
  - Layer 2: Douban Web Scraper (Traditional Chinese)
  - Layer 3: Wikipedia Search (MediaWiki API)
  - Layer 4: AI Intelligent Search Assistant
  - Layer 5: Manual search option
- Zero-failure design: Always provide metadata or manual option
- Graceful degradation and auto-retry

**3. qBittorrent Integration**
- Real-time download monitoring
- Display: Progress, speed, ETA, status
- Connection health monitoring
- Polling-based updates (5-second interval)

**4. Media Library Management UI**
- Grid/list view toggle
- Sorting: Date added, title, year, rating
- Filtering: Genre, year, media type
- Batch operations: Delete, re-parse, manual edit
- Search within library

**5. Media Detail Pages**
- Complete metadata display
- Cast & crew information
- Trailer integration (YouTube embeds)
- Manual metadata editing
- Parse status indicators

**6. Enhanced Error Handling**
- Manual search fallback
- Filename mapping learning
- Parse retry mechanism
- User feedback collection

**1.0 Success Criteria:**
- 500+ active users (weekly login)
- 95%+ overall parsing success rate
- Zero downtime when metadata sources fail
- User satisfaction >4.5/5

---

#### Phase 3: Growth & Expansion (Q3+ - September 2026+)

**Core User Journeys Supported:**
- Advanced automation workflows
- Mobile device access
- External system integrations
- Multi-user households

**Planned Growth Features:**

**1. Subtitle Integration**
- OpenSubtitles API integration
- Zimuku web scraper (Traditional Chinese subtitles)
- Automatic subtitle download
- Traditional Chinese subtitle priority
- Manual upload option

**2. Automatic Media Organization**
- Watch folder monitoring
- Auto-parse on file detection
- Automatic file renaming (user-configured patterns)
- Automatic file moving to organized structure
- Background processing queue

**3. Download Queue Management**
- Direct qBittorrent control from Vido
- Pause/resume/delete torrents
- Priority adjustment
- Bandwidth management
- Download scheduling

**4. Smart Recommendations**
- Genre-based recommendations
- Cast/director-based suggestions
- "Similar titles" engine
- Personalized recommendations (based on library)

**5. Performance Optimization**
- Image CDN and caching
- Lazy loading optimizations
- Virtual scrolling for large libraries
- Database query optimization
- Bundle size reduction

**6. Multi-User Support**
- User authentication system
- Permission management (admin/user roles)
- Personal watch history tracking
- Personal preferences and settings
- Shared vs personal libraries

**7. Mobile Application**
- React Native or Flutter app
- Browse media library on mobile
- Remote download control
- Push notifications (download complete)
- Offline mode (cached metadata)

**8. External Integrations**
- Plex/Jellyfin metadata export
- NFO file generation (Kodi compatible)
- Watch status sync
- Public API (beta)
- Webhook support

**Growth Phase Success Criteria:**
- 1,000+ active users
- Subtitle success rate >90%
- API response time <500ms (p95)
- Community contributions (plugins, translations)

---

### Risk Mitigation Strategy

#### Technical Risks

| Risk | Severity | Mitigation Strategy | Trigger Point |
|------|----------|-------------------|--------------|
| **AI parsing accuracy <90%** | HIGH | - Hybrid mode: Regex + AI dual validation<br>- Learning mechanism from user corrections<br>- Community dataset collection<br>- Fallback to manual search | If benchmark <93% or user satisfaction <85% |
| **AI API costs too high** | HIGH | - Caching strategy (30-day cache)<br>- Smart triggering (only when regex fails)<br>- User-paid model (own API keys)<br>- Degradation option (disable AI) | If average cost >$0.10/file |
| **SQLite scalability limits** | MEDIUM | - Repository Pattern from MVP<br>- Performance monitoring (p95 latency)<br>- PostgreSQL migration path prepared<br>- System warning at 8,000 items | When library >10,000 items or query p95 >500ms |
| **External API dependencies** | MEDIUM | - Multi-source architecture<br>- Metadata caching (permanent local storage)<br>- NFO file backup<br>- API version locking | When API failure rate >10% |
| **Large library performance** | MEDIUM | - Virtual scrolling implementation<br>- Pagination and lazy loading<br>- Database indexing<br>- Query optimization | When initial load >2s or grid scroll <30 FPS |
| **Wikipedia metadata quality** | LOW | - Robust Infobox parsing<br>- Data validation and consistency checks<br>- Graceful degradation<br>- User feedback mechanism | If Wikipedia success rate <50% |

#### Market Risks

| Risk | Severity | Validation Strategy | Learning Objective |
|------|----------|-------------------|-------------------|
| **Traditional Chinese market too small** | MEDIUM | - MVP with 50 early adopters<br>- User feedback collection<br>- Community engagement (Reddit, forums)<br>- Internationalization architecture ready | Validate actual market demand and willingness to adopt |
| **Fansub parsing need unclear** | HIGH | - Benchmark dataset (1,000 real filenames)<br>- User-contributed test cases<br>- Parse success rate tracking<br>- Competitor comparison testing | Measure actual parsing success rate vs user expectations |
| **Users unwilling to provide API keys** | MEDIUM | - MVP without AI (observe demand)<br>- Clear API key setup guide<br>- Cost transparency dashboard<br>- Alternative free parsing options | Assess user willingness to pay API costs |
| **Competition from existing tools** | LOW | - Focus on unique value (fansub parsing)<br>- Traditional Chinese positioning<br>- Community-driven development<br>- Open source advantage | Identify sustainable competitive advantages |

#### Resource Risks

| Risk | Severity | Contingency Strategy |
|------|----------|---------------------|
| **Development time exceeds estimate** | MEDIUM | - MVP scope extremely minimal (search + metadata only)<br>- Defer AI and qBittorrent to 1.0<br>- Single developer sustainable pace |
| **Team size reduction** | LOW | - MVP designed for single developer<br>- Existing infrastructure (Nx, Docker) ready<br>- Clear phased roadmap allows flexible timeline |
| **API cost explosion** | MEDIUM | - User-paid model (own API keys)<br>- Usage monitoring and alerts<br>- Degradation options (disable AI)<br>- Community-funded pool (future) |
| **Infrastructure costs** | LOW | - Self-hosted deployment model<br>- No server hosting costs<br>- Users bear their own infrastructure costs |

**Risk Monitoring Cadence:**
- Weekly: API usage and costs
- Bi-weekly: User feedback and satisfaction scores
- Monthly: Performance metrics and scalability indicators
- Per release: Security vulnerabilities and dependency updates

**Escalation Triggers:**
- Any HIGH severity risk materializes → Immediate scope adjustment
- Two MEDIUM severity risks materialize → Re-evaluate roadmap
- User satisfaction drops below 4.0/5 → User research sprint
- API costs exceed $0.10/file → Mandatory cost optimization

## Functional Requirements

### Media Search & Discovery

**MVP:**
- FR1: Users can search for movies and TV shows by title (Traditional Chinese or English)
- FR2: Users can view search results with Traditional Chinese metadata (title, description, release year, poster, genre, director, cast)
- FR3: Users can browse search results in grid view
- FR4: Users can view media item detail pages (read-only)

**1.0:**
- FR5: Users can search within their saved media library
- FR6: Users can sort media library by date added, title, year, rating
- FR7: Users can filter media library by genre, year, media type
- FR8: Users can toggle between grid view and list view

**Growth:**
- FR9: Users can receive smart recommendations based on genre, cast, director
- FR10: Users can see "similar titles" suggestions

---

### Filename Parsing & Metadata Retrieval

**MVP:**
- FR11: System can parse standard naming convention filenames (e.g., `Movie.Name.2024.1080p.BluRay.mkv`)
- FR12: System can extract title, year, season/episode from filenames
- FR13: System can retrieve Traditional Chinese priority metadata from TMDb API
- FR14: System can store metadata to local database

**1.0:**
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

---

### Download Integration & Monitoring

**1.0:**
- FR27: Users can connect to qBittorrent instance (enter host, username, password)
- FR28: Users can test qBittorrent connection
- FR29: System can monitor qBittorrent download status in real-time (progress, speed, ETA, status)
- FR30: Users can view download list in unified dashboard
- FR31: Users can filter downloads by status (downloading, paused, completed, seeding)
- FR32: System can detect completed downloads and trigger parsing
- FR33: System can display qBittorrent connection health status

**Growth:**
- FR34: Users can control qBittorrent directly from Vido (pause/resume/delete torrents)
- FR35: Users can adjust download priority
- FR36: Users can manage bandwidth settings
- FR37: Users can schedule downloads

---

### Media Library Management

**1.0:**
- FR38: Users can browse complete media library collection
- FR39: Users can view media detail pages (cast info, trailers, complete metadata)
- FR40: Users can perform batch operations on media items (delete, re-parse)
- FR41: Users can view recently added media items
- FR42: System can display metadata source indicators (TMDb/Douban/Wikipedia/AI/Manual)

**Growth:**
- FR43: Users can track personal watch history
- FR44: System can display watch progress indicators
- FR45: Users can mark media as watched/unwatched
- FR46: Users can create custom collections of media items

---

### System Configuration & Management

**MVP:**
- FR47: Users can deploy Vido via Docker container
- FR48: System can provide zero-config startup (sensible defaults)
- FR49: Users can configure media folder locations
- FR50: Users can configure API keys via environment variables
- FR51: System can store sensitive data in encrypted format (AES-256)

**1.0:**
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

---

### User Authentication & Access Control

**1.0:**
- FR67: Users must authenticate via password/PIN to access Vido
- FR68: System can manage user sessions with secure tokens
- FR69: API endpoints must be protected with authentication tokens
- FR70: System can implement rate limiting to prevent abuse

**Growth:**
- FR71: System can support multiple user accounts
- FR72: Administrators can manage user permissions (admin/user roles)
- FR73: Users can have personal watch history
- FR74: Users can have personal preference settings

---

### Subtitle Management (Growth - Post-1.0)

- FR75: Users can search for subtitles (OpenSubtitles and Zimuku)
- FR76: System can prioritize Traditional Chinese subtitles
- FR77: Users can download subtitle files
- FR78: Users can manually upload subtitle files
- FR79: System can automatically download subtitles (based on user preferences)
- FR80: System can display subtitle availability status

---

### Automation & Organization (Growth - Post-1.0)

- FR81: System can monitor watch folders to detect new files
- FR82: System can automatically trigger parsing when files are detected
- FR83: System can automatically rename files based on user-configured patterns
- FR84: System can automatically move files to organized directory structure
- FR85: System can execute automation tasks in background processing queue
- FR86: Users can configure automation rules (watch folders, naming patterns, target folders)

---

### External Integration & Extensibility (Growth - Post-1.0)

- FR87: System can provide RESTful API (versioned /api/v1)
- FR88: Developers can authenticate API requests with API tokens
- FR89: System can provide OpenAPI/Swagger documentation
- FR90: System can support webhook subscriptions for external automation
- FR91: Users can export metadata to Plex/Jellyfin
- FR92: System can sync watch status with Plex/Jellyfin
- FR93: Users can access Vido via mobile application (React Native/Flutter)
- FR94: Users can remotely control downloads from mobile device

## Non-Functional Requirements

### Performance

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

### Security

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

### Scalability

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

### Reliability

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

### Integration

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

### Maintainability

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

### Usability

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
