# Story: Media Directory Setup Guide

Status: done

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

- [x] Task 1: Design and implement the enhanced empty state (AC: #1, #2, #3, #6)
  - [x] 1.1 Replaced bare text with structured info card at ScannerSettings.tsx:133
  - [x] 1.2 Includes: AlertCircle icon, bilingual title, /media explanation, docker run + compose examples
  - [x] 1.3 Styled as blue-tinted info card with border, background, code blocks

- [x] Task 2: Add environment variable documentation in the empty state (AC: #2, #3)
  - [x] 2.1 Shows VIDO_MEDIA_DIRS with bilingual explanation
  - [x] 2.2 Docker Compose example: VIDO_MEDIA_DIRS=/media/movies,/media/tv
  - [x] 2.3 Default /media path referenced in explanation

- [x] Task 3: Bilingual content (AC: #4)
  - [x] 3.1 Title: "媒體資料夾尚未設定 / Media directories not configured"
  - [x] 3.2 Inline bilingual text (no i18n framework in project)

- [x] Task 4: Optional "Learn More" link (AC: #5)
  - [x] 4.1 Docs exist (unraid-installation-guide) but no generic config doc
  - [x] 4.2 Omitted link to avoid dead links — card content is self-sufficient

- [x] Task 5: Tests (AC: all)
  - [x] 5.1 Test renders setup guide card with Docker examples
  - [x] 5.2 Test VIDO_MEDIA_DIRS text present
  - [x] 5.3 Test English text "Media directories not configured" present

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

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1-3: Replaced bare empty state with bilingual info card showing Docker volume mount and docker-compose examples
- Task 4: Omitted Learn More link (no generic config doc yet)
- Task 5: Added 1 test covering setup guide rendering with Docker examples and VIDO_MEDIA_DIRS. 126 files / 1563 tests pass.

### File List

- apps/web/src/components/settings/ScannerSettings.tsx (modified — enhanced empty state with setup guide)
- apps/web/src/components/settings/ScannerSettings.spec.tsx (modified — added setup guide test)
