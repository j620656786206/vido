# User Journeys

## Journey 1: Alex (NAS Media Collector) - The Perfect Day

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

## Journey 2: Alex - When Things Don't Go as Expected (Edge Cases & Error Handling)

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

## Journey 3: Alex (System Administrator Role) - Initial Setup & Maintenance

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

## Journey 4: Developer David - API Integration (Future Flexibility Consideration)

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

## Journey Requirements Summary

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
