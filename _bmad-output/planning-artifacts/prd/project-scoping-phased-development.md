# Project Scoping & Phased Development

## MVP Strategy & Philosophy

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

## MVP Feature Set (Phase 1 - Q1 March 2026)

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

## Post-MVP Features

### Phase 2: 1.0 Core Platform (Q2 - June 2026)

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

### Phase 3: Growth & Expansion (Q3+ - September 2026+)

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

## Risk Mitigation Strategy

### Technical Risks

| Risk | Severity | Mitigation Strategy | Trigger Point |
|------|----------|-------------------|--------------|
| **AI parsing accuracy <90%** | HIGH | - Hybrid mode: Regex + AI dual validation<br>- Learning mechanism from user corrections<br>- Community dataset collection<br>- Fallback to manual search | If benchmark <93% or user satisfaction <85% |
| **AI API costs too high** | HIGH | - Caching strategy (30-day cache)<br>- Smart triggering (only when regex fails)<br>- User-paid model (own API keys)<br>- Degradation option (disable AI) | If average cost >$0.10/file |
| **SQLite scalability limits** | MEDIUM | - Repository Pattern from MVP<br>- Performance monitoring (p95 latency)<br>- PostgreSQL migration path prepared<br>- System warning at 8,000 items | When library >10,000 items or query p95 >500ms |
| **External API dependencies** | MEDIUM | - Multi-source architecture<br>- Metadata caching (permanent local storage)<br>- NFO file backup<br>- API version locking | When API failure rate >10% |
| **Large library performance** | MEDIUM | - Virtual scrolling implementation<br>- Pagination and lazy loading<br>- Database indexing<br>- Query optimization | When initial load >2s or grid scroll <30 FPS |
| **Wikipedia metadata quality** | LOW | - Robust Infobox parsing<br>- Data validation and consistency checks<br>- Graceful degradation<br>- User feedback mechanism | If Wikipedia success rate <50% |

### Market Risks

| Risk | Severity | Validation Strategy | Learning Objective |
|------|----------|-------------------|-------------------|
| **Traditional Chinese market too small** | MEDIUM | - MVP with 50 early adopters<br>- User feedback collection<br>- Community engagement (Reddit, forums)<br>- Internationalization architecture ready | Validate actual market demand and willingness to adopt |
| **Fansub parsing need unclear** | HIGH | - Benchmark dataset (1,000 real filenames)<br>- User-contributed test cases<br>- Parse success rate tracking<br>- Competitor comparison testing | Measure actual parsing success rate vs user expectations |
| **Users unwilling to provide API keys** | MEDIUM | - MVP without AI (observe demand)<br>- Clear API key setup guide<br>- Cost transparency dashboard<br>- Alternative free parsing options | Assess user willingness to pay API costs |
| **Competition from existing tools** | LOW | - Focus on unique value (fansub parsing)<br>- Traditional Chinese positioning<br>- Community-driven development<br>- Open source advantage | Identify sustainable competitive advantages |

### Resource Risks

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
