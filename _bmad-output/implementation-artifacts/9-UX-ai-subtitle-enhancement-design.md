# Story 9.UX: AI Subtitle Enhancement Design

Status: ready-for-dev

## Story

As a UX designer working on Vido,
I want to design AI subtitle enhancement screens for the .pen design file,
so that the dev team has clear visual specs for AI terminology correction preview, Whisper transcription progress, and translation confirmation UI.

## Acceptance Criteria

1. Given the existing subtitle management UI (Flow G, screens G1-G6), when AI enhancement features are designed, then new screens extend the existing design language and component patterns without breaking visual consistency
2. Given a media item with a downloaded subtitle, when AI terminology correction is available (Claude API key configured), then the Detail Panel subtitle section shows an "AI 校正" action button with clear affordance
3. Given AI terminology correction is running, when the user views progress, then a modal or inline panel shows before/after comparison of corrections with accept/reject controls
4. Given a media item with no Chinese subtitles but an English audio track, when the user triggers transcription, then a progress UI shows the multi-stage pipeline: extracting audio → transcribing → (optional) translating → complete
5. Given Whisper transcription completes, when the user has Claude API configured, then a translation confirmation dialog asks whether to translate the English SRT to Traditional Chinese
6. Given the translation is running, when the user views progress, then batch progress is shown with percentage and estimated remaining blocks
7. Given all new screens, when reviewed against the design system, then they use the existing color tokens (--success, --warning, --error, --info), Lucide icons, and Tailwind utility classes from the Epic 5 design system
8. Given mobile viewports, when AI enhancement screens are displayed, then they follow the existing mobile patterns (bottom sheet for modals, simplified layouts, 48px touch targets)

## Tasks / Subtasks

- [ ] Task 1: AI Terminology Correction Preview — Desktop (AC: #2, #3, #7)
  - [ ] 1.1 Design "AI 校正" button placement in Detail Panel subtitle section (G1 extension) — position after existing subtitle status info, secondary/ghost button with Lucide Sparkles icon
  - [ ] 1.2 Design AI correction preview modal (desktop, 560px width centered) showing: original text (left) vs corrected text (right) diff view, highlighted changes, accept all / reject all / per-item toggle, "套用校正" primary CTA
  - [ ] 1.3 Design correction-in-progress state: Lucide Loader spinning + "正在分析用語..." status text, progress bar if multiple chunks
  - [ ] 1.4 Design correction complete state: summary count "已修正 N 處用語", list of changes with before→after pairs
  - [ ] 1.5 Add screen to ux-design.pen as new screen in Flow G extension

- [ ] Task 2: Whisper Transcription Progress UI — Desktop (AC: #4, #6, #7)
  - [ ] 2.1 Design "轉錄英文音軌" button in Detail Panel subtitle section — only visible when media has no Chinese subtitle AND has English audio track, primary button with Lucide Mic icon
  - [ ] 2.2 Design transcription progress panel (inline in Detail Panel or side panel, 500px) showing multi-stage pipeline:
    - Stage 1: "正在擷取音軌..." with Lucide AudioLines icon
    - Stage 2: "正在轉錄..." with Lucide Mic icon + progress percentage
    - Stage 3 (optional): "正在翻譯..." with Lucide Languages icon + batch progress (e.g., "12/45 字幕區塊")
    - Stage 4: "完成" with Lucide CheckCircle icon
  - [ ] 2.3 Design each stage with visual step indicator (vertical stepper pattern: completed=--success, current=--accent-primary+spinner, pending=--text-muted)
  - [ ] 2.4 Design cancel button (ghost, available at any stage) with confirmation: "確定要取消嗎？已完成的步驟不會還原。"
  - [ ] 2.5 Add screen to ux-design.pen

- [ ] Task 3: Translation Confirmation Dialog — Desktop (AC: #5, #7)
  - [ ] 3.1 Design translation prompt dialog (appears after Whisper transcription completes) — small centered dialog (400px):
    - Title: "英文字幕已產生"
    - Body: "是否要使用 AI 翻譯為繁體中文？（需要 Claude API）"
    - Subtitle: estimated time/cost hint based on block count
    - Actions: "翻譯為繁中" (primary) / "保留英文" (ghost) / "稍後決定" (text link)
  - [ ] 3.2 Design "not configured" variant: when Claude API key is missing, dialog shows "翻譯功能需要設定 Claude API 金鑰" with link to Settings page
  - [ ] 3.3 Add screen to ux-design.pen

- [ ] Task 4: Mobile Adaptations (AC: #8)
  - [ ] 4.1 Design AI correction preview as full-height bottom sheet (mobile): simplified single-column diff (original strikethrough → corrected), swipe to dismiss
  - [ ] 4.2 Design transcription progress as collapsible bottom sheet (peek height 72px showing current stage + progress, expandable to full detail) — same pattern as G6 batch processing mobile
  - [ ] 4.3 Design translation confirmation as centered action sheet (3 button options stacked vertically, 48px touch targets)
  - [ ] 4.4 Add mobile screens to ux-design.pen

- [ ] Task 5: Update Existing Subtitle Screens (AC: #1, #2)
  - [ ] 5.1 Update G1 (Detail Panel subtitle section) to include "AI 校正" button and "轉錄" button in appropriate positions — buttons only visible when feature is available (API keys configured + applicable media state)
  - [ ] 5.2 Update G2 (Manual Search panel) to show AI correction indicator: if a downloaded subtitle was AI-corrected, show "AI 校正済" badge next to language tag
  - [ ] 5.3 Add new subtitle status variant to G1: "AI 翻譯 (英→繁)" for Whisper+translation generated subtitles, with Lucide Languages icon in --info color

- [ ] Task 6: Export Screenshots & Commit (AC: all)
  - [ ] 6.1 Run `python3 scripts/export-pen-screenshots.py` to regenerate screenshots
  - [ ] 6.2 Update `SCREENS` dict in export script if new screen IDs were added
  - [ ] 6.3 Verify screenshots in `_bmad-output/screenshots/` cover all new screens
  - [ ] 6.4 Commit both `ux-design.pen` changes and updated screenshots together

## Dev Notes

### This is a UX Design Story — NOT a Code Implementation Story

This story is executed by the **UX Designer agent** using Pencil MCP tools to modify `ux-design.pen`. No Go or React code is written. The output is design screens that will inform future frontend implementation stories.

### Design System Reference

- **Color tokens:** `--success` (green), `--warning` (amber), `--error` (red), `--info` (blue), `--accent-primary` (brand), `--text-primary`, `--text-secondary`, `--text-muted`, `--bg-primary`, `--bg-secondary`, `--bg-tertiary`
- **Icon library:** Lucide icons only — relevant icons: Sparkles, Mic, AudioLines, Languages, CheckCircle, Loader, AlertTriangle, ArrowRight, Settings, X
- **Typography:** Inter font, sizes per Epic 5 design system
- **Dark theme:** All screens designed for dark theme (primary design mode)

### Existing Flow G Foundation

All new screens extend the Subtitle Engine UI (Flow G) defined in `_bmad-output/planning-artifacts/subtitle-engine-design-brief.md`:
- **G1** (Subtitle Status Indicators) — extended with AI correction and transcription buttons
- **G2** (Manual Search) — extended with AI correction badge
- **G3** (Download Progress) — pattern reused for transcription progress
- **G4** (Batch Processing) — pattern reused for batch translation progress
- **G5/G6** (Mobile variants) — patterns reused for mobile adaptations

### Backend Context (Already Implemented — Stories 9-1, 9-2a, 9-2b)

The backend APIs these screens will eventually call are already built:
- **AI Terminology Correction** (9-1): `TerminologyCorrectionService.Correct()` — sends subtitle text to Claude for word-level 簡→繁 fixes
- **Audio Extraction** (9-2a): `POST /api/v1/movies/:id/transcribe` — FFmpeg extraction + Whisper API transcription
- **Translation** (9-2b): `TranslationService.TranslateSRT()` — Claude translates English SRT to zh-Hant, `translate=true` query param
- **SSE Events:** `transcription_extracting`, `transcription_progress`, `transcription_complete`, `transcription_failed`, `translation_progress`

### Key Design Decisions to Make

1. **AI correction: modal vs inline?** — Recommendation: centered modal (560px) for desktop since diff view needs width; bottom sheet for mobile
2. **Transcription progress: inline vs panel?** — Recommendation: inline in Detail Panel (reuses existing space, no extra panel) with vertical stepper pattern
3. **Translation prompt: automatic vs confirm?** — Recommendation: always confirm (user pays for API calls), but allow "always translate" toggle in Settings for power users
4. **When to show AI buttons:** Only when relevant API key is configured AND media state is applicable (e.g., "AI 校正" only after subtitle downloaded, "轉錄" only when no Chinese subtitle exists)

### Content & Copy (Traditional Chinese)

| Element | Text |
|---------|------|
| AI correction button | AI 校正 |
| AI correction modal title | AI 用語校正預覽 |
| Correction in progress | 正在分析用語... |
| Correction complete | 已修正 {n} 處用語 |
| Accept all | 全部套用 |
| Reject all | 全部取消 |
| Apply corrections | 套用校正 |
| Transcribe button | 轉錄英文音軌 |
| Extracting audio | 正在擷取音軌... |
| Transcribing | 正在轉錄... |
| Translating | 正在翻譯... |
| Translation complete | 翻譯完成 |
| Cancel confirm | 確定要取消嗎？已完成的步驟不會還原。 |
| Translation prompt title | 英文字幕已產生 |
| Translation prompt body | 是否要使用 AI 翻譯為繁體中文？ |
| Translate button | 翻譯為繁中 |
| Keep English button | 保留英文 |
| Decide later link | 稍後決定 |
| API not configured | 翻譯功能需要設定 Claude API 金鑰 |
| AI corrected badge | AI 校正済 |
| AI translated status | AI 翻譯 (英→繁) |

### What NOT to Do

- NO new color tokens — reuse existing design system tokens
- NO new icon library — Lucide only
- NO separate "AI Subtitle" page — all features integrated into existing Detail Panel and Flow G screens
- NO light theme designs at this stage — dark theme only (primary design mode)
- NO implementation code — this is design-only, frontend stories come later
- NO polling-based progress — all progress designs assume SSE real-time updates

### References

- [Source: _bmad-output/planning-artifacts/subtitle-engine-design-brief.md] — Flow G complete spec (G1-G6)
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md] — Master UX spec and design philosophy
- [Source: _bmad-output/planning-artifacts/ux-design-gap-analysis-v4.md] — Flow G in gap analysis
- [Source: _bmad-output/planning-artifacts/epics/epic-9-ai-subtitle-enhancement.md] — Epic 9 definition
- [Source: _bmad-output/implementation-artifacts/9-1-ai-terminology-correction.md] — Backend: AI correction service
- [Source: _bmad-output/implementation-artifacts/9-2a-whisper-audio-transcription.md] — Backend: Whisper transcription
- [Source: _bmad-output/implementation-artifacts/9-2b-ai-subtitle-translation.md] — Backend: AI translation service
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.3] — P1-020 (AI 用語校正), P1-021 (MKV 英文軌翻譯)

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context) — SM agent (Bob) create-story workflow, YOLO mode

### Debug Log References

### Completion Notes List

- Story created via SM create-story workflow in YOLO mode
- This is a UX design story — no cross-stack split needed
- All 3 prior dev stories (9-1, 9-2a, 9-2b) are done — backend APIs are implemented
- Design screens extend existing Flow G (Subtitle Engine) from Epic 8 design brief
- Priority note from sprint-status: "after Epic 10+11 design" — check scheduling with PM

### File List

- `ux-design.pen` — Add ~4-6 new screens (AI correction modal, transcription progress, translation dialog, mobile variants)
- `_bmad-output/screenshots/` — Updated screenshots after design changes
- `scripts/export-pen-screenshots.py` — Update SCREENS dict if new screen IDs added
