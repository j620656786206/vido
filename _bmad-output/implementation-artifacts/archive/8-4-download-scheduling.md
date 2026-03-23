# Story 8.4: Download Scheduling

Status: ready-for-dev

## Story

As a **NAS user**,
I want to **schedule downloads for specific times**,
So that **downloads run during off-peak hours**.

## Acceptance Criteria

1. **AC1: Weekly Schedule Grid**
   - Given the user opens Settings > Schedule
   - When configuring the schedule
   - Then a weekly time grid is available (7 days x 24 hours)
   - And users can select time blocks for each mode

2. **AC2: Schedule Modes**
   - Given schedule modes are: Full Speed / Alternative Speed / Pause All
   - When the user selects time blocks
   - Then each block is assigned a mode
   - And visual color coding shows the schedule

3. **AC3: Automatic Mode Switching**
   - Given a schedule is configured
   - When the scheduled time arrives
   - Then qBittorrent automatically switches modes
   - And Vido displays current schedule status

4. **AC4: Quick Schedule Presets**
   - Given the user wants a simple schedule
   - When using "Quick Schedule"
   - Then presets are available: "Night Only (00:00-06:00)", "Off-Peak (20:00-08:00)"
   - And one-click applies the schedule

5. **AC5: Schedule Enable/Disable**
   - Given a schedule is configured
   - When the user toggles the scheduler
   - Then the schedule can be enabled/disabled without losing configuration
   - And current status shows whether scheduler is active

## Tasks / Subtasks

- [ ] Task 1: Extend qBittorrent Client with Scheduler Methods (AC: 1, 2, 3, 5)
  - [ ] 1.1: Add `GetPreferences(ctx) (*Preferences, error)` to client.go (if not already from 8-3)
  - [ ] 1.2: Add `SetPreferences(ctx, prefs map[string]interface{}) error`
  - [ ] 1.3: Extract scheduler-specific preferences from response
  - [ ] 1.4: Write unit tests

- [ ] Task 2: Add Scheduler Types (AC: 1, 2)
  - [ ] 2.1: Add to types.go:
    - `SchedulerConfig` struct (enabled, from_hour, from_min, to_hour, to_min, days)
    - `SchedulerDays` enum (0=Every day, 1=Weekday, 2=Weekend, 3=Mon...9=Sun)
    - `SchedulePreset` struct (name, from, to, days)
  - [ ] 2.2: Write type tests

- [ ] Task 3: Create Scheduler Service (AC: 1, 2, 3, 4, 5)
  - [ ] 3.1: Add `GetSchedulerConfig(ctx) (*SchedulerConfig, error)` to download_service.go
  - [ ] 3.2: Add `SetSchedulerConfig(ctx, config *SchedulerConfig) error`
  - [ ] 3.3: Add `EnableScheduler(ctx, enabled bool) error`
  - [ ] 3.4: Add `ApplyPreset(ctx, preset string) error` with predefined presets
  - [ ] 3.5: Define presets:
    - "night-only": 00:00-06:00, every day, alternative speed
    - "off-peak": 20:00-08:00, every day, alternative speed
    - "weekday-night": 00:00-06:00, weekdays only
    - "weekend-full": full speed all weekend, alt speed weekdays
  - [ ] 3.6: Write service tests

- [ ] Task 4: Create Scheduler Handler Endpoints (AC: 1, 2, 3, 4, 5)
  - [ ] 4.1: Add `GET /api/v1/downloads/schedule` - get scheduler config
  - [ ] 4.2: Add `PUT /api/v1/downloads/schedule` - save scheduler config
  - [ ] 4.3: Add `POST /api/v1/downloads/schedule/toggle` - enable/disable scheduler
  - [ ] 4.4: Add `POST /api/v1/downloads/schedule/preset` - apply a preset
  - [ ] 4.5: Add `GET /api/v1/downloads/schedule/presets` - list available presets
  - [ ] 4.6: Add Swagger documentation
  - [ ] 4.7: Write handler tests

- [ ] Task 5: Register Routes (AC: all)
  - [ ] 5.1: Register scheduler routes in `main.go`

- [ ] Task 6: Create Schedule Grid Component (AC: 1, 2)
  - [ ] 6.1: Create `/apps/web/src/components/downloads/ScheduleGrid.tsx`
  - [ ] 6.2: 7-row (days) x 24-column (hours) interactive grid
  - [ ] 6.3: Click/drag to select time blocks
  - [ ] 6.4: Color coding: Green (Full Speed), Yellow (Alt Speed), Red (Pause All)
  - [ ] 6.5: Day labels (一 through 日) and hour labels (0-23)
  - [ ] 6.6: Write component tests

- [ ] Task 7: Create Schedule Settings Page (AC: 1, 2, 3, 4, 5)
  - [ ] 7.1: Create `/apps/web/src/components/downloads/ScheduleSettings.tsx`
  - [ ] 7.2: Enable/disable scheduler toggle at the top
  - [ ] 7.3: Include ScheduleGrid component
  - [ ] 7.4: Quick Schedule preset buttons
  - [ ] 7.5: Save button to apply schedule
  - [ ] 7.6: Current status indicator (scheduler active/inactive, current mode)
  - [ ] 7.7: Write component tests

- [ ] Task 8: Create Quick Preset Buttons (AC: 4)
  - [ ] 8.1: Create `/apps/web/src/components/downloads/SchedulePresets.tsx`
  - [ ] 8.2: Preset cards with name, description, visual preview
  - [ ] 8.3: One-click apply with confirmation
  - [ ] 8.4: Presets:
    - "僅夜間 (00:00-06:00)" - Night Only
    - "離峰時段 (20:00-08:00)" - Off-Peak
    - "平日夜間" - Weekday Nights
    - "週末全速" - Weekend Full Speed
  - [ ] 8.5: Write component tests

- [ ] Task 9: Create Frontend API Methods (AC: all)
  - [ ] 9.1: Add `getSchedulerConfig()` to downloadService.ts
  - [ ] 9.2: Add `setSchedulerConfig(config)`
  - [ ] 9.3: Add `toggleScheduler(enabled)`
  - [ ] 9.4: Add `applyPreset(presetName)`
  - [ ] 9.5: Add `getPresets()`
  - [ ] 9.6: Add TanStack Query hooks

- [ ] Task 10: Integrate Schedule Status into Dashboard (AC: 3, 5)
  - [ ] 10.1: Add scheduler status indicator to downloads page header
  - [ ] 10.2: Show current mode (full/alt/paused) based on schedule
  - [ ] 10.3: Show next mode change time
  - [ ] 10.4: Link to schedule settings

- [ ] Task 11: Write Tests (AC: all)
  - [ ] 11.1: Backend unit tests - coverage ≥80%
  - [ ] 11.2: Frontend component tests - coverage ≥70%
  - [ ] 11.3: E2E test: `/e2e/download-scheduling.spec.ts`

## Dev Notes

### Architecture Requirements

**FR37: Schedule downloads**
- Uses qBittorrent's built-in scheduler
- Visual weekly grid interface
- Quick presets for common scheduling patterns

### qBittorrent Web API Reference

```
Get Preferences (includes scheduler):
GET /api/v2/app/preferences
  Response includes:
    scheduler_enabled: boolean
    schedule_from_hour: int (0-23)
    schedule_from_min: int (0-59)
    schedule_to_hour: int (0-23)
    schedule_to_min: int (0-59)
    scheduler_days: int
      0 = Every day
      1 = Every weekday
      2 = Every weekend
      3 = Every Monday
      4 = Every Tuesday
      5 = Every Wednesday
      6 = Every Thursday
      7 = Every Friday
      8 = Every Saturday
      9 = Every Sunday
    alt_dl_limit: int (KiB/s)
    alt_up_limit: int (KiB/s)

Set Preferences:
POST /api/v2/app/setPreferences
  Body: json={"scheduler_enabled": true, "schedule_from_hour": 0, "schedule_from_min": 0, "schedule_to_hour": 6, "schedule_to_min": 0, "scheduler_days": 0}
  Response: HTTP 200
```

### CRITICAL: qBittorrent Scheduler Limitation

qBittorrent's built-in scheduler is **simple**: one time range per day, not per-hour blocks. The scheduler defines when **alternative speed** is active. Outside that range, full speed applies.

**Implementation approach:**
- The Vido weekly grid is a **visual representation** that maps to qBittorrent's simpler scheduler
- For complex schedules (per-hour blocks), Vido would need its own scheduler that calls toggle API — but for MVP, use qBittorrent's native scheduler
- Quick presets map directly to qBittorrent's scheduler_days + time range

### Vido API Endpoints

```
GET  /api/v1/downloads/schedule
  Response: { "success": true, "data": {
    "enabled": true,
    "from_hour": 0, "from_min": 0,
    "to_hour": 6, "to_min": 0,
    "days": 0,
    "days_label": "every_day",
    "alt_dl_limit": 500,
    "alt_up_limit": 100
  }}

PUT  /api/v1/downloads/schedule
  Body: {
    "enabled": true,
    "from_hour": 20, "from_min": 0,
    "to_hour": 8, "to_min": 0,
    "days": 0,
    "alt_dl_limit": 500,
    "alt_up_limit": 100
  }

POST /api/v1/downloads/schedule/toggle
  Body: { "enabled": true }

POST /api/v1/downloads/schedule/preset
  Body: { "preset": "night-only" }

GET  /api/v1/downloads/schedule/presets
  Response: { "success": true, "data": [
    { "id": "night-only", "name": "僅夜間", "description": "00:00-06:00 每天使用替代速度", "from_hour": 0, "to_hour": 6, "days": 0 },
    { "id": "off-peak", "name": "離峰時段", "description": "20:00-08:00 每天使用替代速度", "from_hour": 20, "to_hour": 8, "days": 0 },
    { "id": "weekday-night", "name": "平日夜間", "description": "00:00-06:00 僅平日", "from_hour": 0, "to_hour": 6, "days": 1 },
    { "id": "weekend-full", "name": "週末全速", "description": "週末全速，平日使用替代速度", "from_hour": 0, "to_hour": 0, "days": 2 }
  ]}
```

### Schedule Grid Design

The weekly grid should visually represent qBittorrent's scheduler in a user-friendly way:

```
         00 01 02 03 04 05 06 07 ... 20 21 22 23
Monday   [🟡 🟡 🟡 🟡 🟡 🟡 🟢 🟢 ... 🟡 🟡 🟡 🟡]
Tuesday  [🟡 🟡 🟡 🟡 🟡 🟡 🟢 🟢 ... 🟡 🟡 🟡 🟡]
...
Sunday   [🟡 🟡 🟡 🟡 🟡 🟡 🟢 🟢 ... 🟡 🟡 🟡 🟡]

🟢 = Full Speed   🟡 = Alternative Speed
```

Since qBittorrent uses a single time range + days filter, the grid is **read-only visual** showing which hours have alt speed vs full speed. Users configure via the from/to time pickers and day selector.

### Error Codes

- `QBIT_CONNECTION_FAILED` - qBittorrent not reachable
- `QBIT_OPERATION_FAILED` - Failed to set preferences
- `VALIDATION_OUT_OF_RANGE` - Invalid hour/minute values
- `VALIDATION_REQUIRED_FIELD` - Missing required scheduler fields

### Project Structure Notes

**Backend Files to Modify:**
```
/apps/api/internal/qbittorrent/client.go        → Add GetPreferences, SetPreferences
/apps/api/internal/qbittorrent/client_test.go    → Add tests
/apps/api/internal/qbittorrent/types.go          → Add SchedulerConfig, SchedulerDays, SchedulePreset
/apps/api/internal/services/download_service.go  → Add scheduler service methods
/apps/api/internal/handlers/download_handler.go  → Add scheduler endpoints
/apps/api/main.go                                → Register new routes
```

**Frontend Files to Create:**
```
/apps/web/src/components/downloads/ScheduleGrid.tsx
/apps/web/src/components/downloads/ScheduleGrid.spec.tsx
/apps/web/src/components/downloads/ScheduleSettings.tsx
/apps/web/src/components/downloads/ScheduleSettings.spec.tsx
/apps/web/src/components/downloads/SchedulePresets.tsx
/apps/web/src/components/downloads/SchedulePresets.spec.tsx
```

**Frontend Files to Modify:**
```
/apps/web/src/services/downloadService.ts  → Add scheduler methods
/apps/web/src/routes/downloads.tsx         → Add scheduler status indicator
```

### Dependencies

**Story Dependencies:**
- Story 8-3 (Bandwidth Settings Control) - Alternative speed limits must be configured first
- Epic 4 (qBittorrent connection, monitoring)

**Library Dependencies:**
- None (uses existing Go standard library + established qbittorrent package)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.4]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR37]
- [Source: _bmad-output/implementation-artifacts/4-1-qbittorrent-connection-configuration.md]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [qBittorrent Web API v4.1](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))

### Previous Story Intelligence

**From Story 8-3 (Bandwidth Settings Control):**
- GetPreferences/SetPreferences may already be implemented — REUSE, do not duplicate
- Alternative speed limits already configured — scheduler controls when they activate
- TransferInfo polling already established for real-time speed display

**From Story 8-1 and 8-2:**
- qBittorrent client extension patterns established
- Download handler endpoint patterns established
- Frontend mutation + cache invalidation patterns

**From Epic 4 Stories:**
- Download dashboard layout and navigation
- Real-time polling infrastructure

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
