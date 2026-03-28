# Story: Media Directory Setup Guide

Status: ready-for-dev

## Story

As a first-time Vido user running in Docker,
I want clear guidance on how to configure my media directories when the scanner shows "尚未設定媒體資料夾",
so that I can quickly set up my library without searching external documentation.

## Acceptance Criteria

1. The empty state in ScannerSettings no longer shows only "尚未設定媒體資料夾" — it includes actionable setup instructions
2. The empty state shows at minimum: (a) the `VIDO_MEDIA_DIRS` environment variable name, (b) a Docker volume mount example, (c) a docker-compose snippet example
3. The instructions mention the default path `/media` and explain how to map it to a host directory
4. The guidance is bilingual (English + Traditional Chinese) consistent with the project's bilingual docs rule
5. A "Learn More" link points to the project's installation/configuration documentation if available
6. The empty state is visually clear — uses an info/help card style, not just plain text

## Tasks / Subtasks

- [ ] Task 1: Design and implement the enhanced empty state (AC: #1, #2, #3, #6)
  - [ ] 1.1 `apps/web/src/components/settings/ScannerSettings.tsx:136`: Replace the bare "尚未設定媒體資料夾" text with a structured help card component
  - [ ] 1.2 Include in the card:
    - Icon (folder or info icon)
    - Title: "媒體資料夾尚未設定 / Media directories not configured"
    - Explanation: "Vido 預設掃描 `/media` 路徑。請透過 Docker volume mount 將您的媒體資料夾掛載到容器中。"
    - Code block: `docker run -v /path/to/your/media:/media ...`
    - Code block: docker-compose example with `VIDO_MEDIA_DIRS` env var
  - [ ] 1.3 Style as an info card with border, background, and appropriate spacing

- [ ] Task 2: Add environment variable documentation in the empty state (AC: #2, #3)
  - [ ] 2.1 Show `VIDO_MEDIA_DIRS` env var with explanation: "設定多個路徑請以逗號分隔 / Separate multiple paths with commas"
  - [ ] 2.2 Example: `VIDO_MEDIA_DIRS=/media/movies,/media/tv`
  - [ ] 2.3 Reference `apps/api/internal/config/config.go:100` which defaults to `/media`

- [ ] Task 3: Bilingual content (AC: #4)
  - [ ] 3.1 All text in the help card should be bilingual (zh-TW primary, English secondary) following project convention
  - [ ] 3.2 Use the same i18n pattern as other settings components, or inline bilingual text if no i18n framework is in use

- [ ] Task 4: Optional "Learn More" link (AC: #5)
  - [ ] 4.1 If `docs/` or a README section exists for configuration, link to it
  - [ ] 4.2 If no docs exist yet, link to the GitHub repo's wiki or omit the link (do not create dead links)

- [ ] Task 5: Tests (AC: all)
  - [ ] 5.1 `apps/web/src/components/settings/ScannerSettings.spec.tsx`: Test that empty state renders the help card with Docker examples
  - [ ] 5.2 Test that the `VIDO_MEDIA_DIRS` env var name appears in the empty state
  - [ ] 5.3 Test that both zh-TW and English text are present

## Dev Notes

### Root Cause Analysis

`ScannerSettings.tsx:136` renders a bare message "尚未設定媒體資料夾" when no media directories are configured. This provides zero guidance on how to fix the issue. The backend default in `config.go:100` sets `VIDO_MEDIA_DIRS` to `/media`, but users running Docker need to know to mount a volume at that path. First-time users see the message and have no idea what to do next.

### Key Files

| File | Change |
|------|--------|
| `apps/web/src/components/settings/ScannerSettings.tsx` | Replace empty state at line 136 with help card |
| `apps/api/internal/config/config.go:100` | Reference only — default `/media` path |

### References

- [Source: apps/web/src/components/settings/ScannerSettings.tsx:136] — bare empty state message
- [Source: apps/api/internal/config/config.go:100] — VIDO_MEDIA_DIRS defaults to /media
- [Rule: All user-facing docs require EN + zh-TW] — bilingual requirement

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
