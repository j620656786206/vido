# Success Criteria

## User Success

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

## Business Success

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

## Technical Success

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

## Measurable Outcomes

**User Experience Milestones:**
1. First-time user completes media search and sees Traditional Chinese results: <5 minutes
2. AI parsing of fansub naming files completes: <10 seconds
3. Download status real-time updates: <5 seconds latency
4. From download completion to metadata fetched: <30 seconds (after automation)

**Competitive Advantage Validation:**
- Vido correctly parses fansub naming that Radarr/Sonarr cannot handle: >90% success rate
- Traditional Chinese metadata coverage: >95% (vs Jellyfin/Plex's limited support)
- Chinese subtitle auto-download success rate: >90% (vs Bazarr's broken providers)
