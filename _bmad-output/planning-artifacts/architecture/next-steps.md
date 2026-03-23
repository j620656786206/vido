# Next Steps (v4 Priorities)

With the architecture updated for PRD v4, the recommended implementation sequence is:

## Priority Order

1. **Implement Subtitle Engine Pipeline (Phase 1 Priority)**
   - Define `SubtitleProvider` interface and pipeline orchestrator
   - Implement Assrt, Zimuku, OpenSubtitles providers
   - Build multi-factor scoring engine (language 40%, resolution 20%, source trust 20%, group 10%, downloads 10%)
   - Integrate OpenCC for 簡繁轉換 with cross-strait terminology correction
   - Implement content-based language detection (not filename-based)
   - Location: `/apps/api/internal/subtitle/`

2. **Implement Media Library Scanner with Scheduled Scanning (Phase 1)**
   - Build recursive file scanner for configured library paths
   - Implement TMDB matching orchestrator for new files
   - Add file system watcher for scheduled scans (configurable interval)
   - Support manual scan trigger via API
   - Location: `/apps/api/internal/scanner/`

3. **Build Plugin Architecture Foundation (Phase 3 prep, interface design in Phase 1)**
   - Define `Plugin`, `MediaServerPlugin`, `DownloaderPlugin`, `DVRPlugin` interfaces
   - Implement plugin manager with registration and health checks
   - Build per-plugin config storage in SQLite
   - Implement graceful degradation (circuit breaker pattern)
   - Location: `/apps/api/internal/plugins/`

4. **Implement SSE Hub (Phase 3, prototype in Phase 1 for scan progress)**
   - Build central event broadcaster goroutine
   - Implement client registration/deregistration
   - Define event types: `download_progress`, `scan_status`, `subtitle_status`, `notification`
   - Support `Last-Event-ID` for reconnection
   - Location: `/apps/api/internal/sse/`

5. **Remove JWT Authentication Infrastructure (No Longer Needed)**
   - Remove `auth_handler.go`, `auth_service.go`, `auth.go` middleware references
   - Remove `users` table migration
   - Remove frontend auth components (`LoginForm`, `PINEntry`, `authStore`)
   - Remove JWT-related environment variables (`JWT_SECRET`)
   - All endpoints are open (single-user deployment)

## Immediate Next Action

Begin with subtitle engine interface design and media scanner implementation, as these are Phase 1 priorities that deliver the core differentiating features of Vido v4.

---
