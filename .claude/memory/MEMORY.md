# Vido Project Memory

## UX Design Screenshot Workflow
- `ux-design.pen` is the source of truth for UX designs (Pencil app)
- When .pen file changes, run `python3 scripts/export-pen-screenshots.py` to regenerate screenshots
- Screenshots saved to `_bmad-output/screenshots/` in 6 flow-based folders (A-F)
- Script connects via Pencil MCP **stdio** JSON-RPC (Pencil 1.1.61 removed `--http`/`--http-port`; `export-pen-screenshots.py` rewritten to stdio 2026-06-05). NOTE: full regen is non-deterministic (every PNG re-renders with byte diffs at same dimensions) — only commit genuinely-changed screens, not the whole regen.
- If new screens are added to .pen, update SCREENS dict in the export script
- Always commit screenshots alongside .pen changes
- [Pen Flow Layout Convention](project_pen_flow_layout_convention.md) — canvas IA: merge same-flow desktop+mobile into one block (A–G), semantic naming, captions above frames

## UX Redesign Initiative
- [UX Redesign Initiative](project_ux_redesign_initiative.md) — Phase 0 brief done 2026-06-12; Phase 1 = design language v2 + nav IA ADR (D1–D4); strangler migration, fresh session per phase

## Project Structure
- Monorepo managed with Nx
- `apps/api/` — Go backend (Gin framework)
- `apps/web/` — React frontend (TanStack Router)
- `_bmad-output/` — Planning artifacts, implementation specs, screenshots
- Uses BMAD multi-agent system for project management

## TestSprite Integration
- [TestSprite Integration Plan](project_testsprite_integration.md) — Installed but deferred until Epic 5+6 complete; 6 P0 journeys defined, prerequisites listed
- [TestSprite v4 Regeneration Plan](project_testsprite_v4_regen.md) — 2026-06-07 清理+擴充：58 plan / 54 active；30 v3 孤兒已刪、+8 P0 案；批次字幕前端 UI 未開發(已 triage)；執行仍卡 seed data (retro-8-TS2)

## Development Workflow Feedback
- [Design Verification Required](feedback_design_verification.md) — Dev must verify UI matches design screenshots after every story
- [UX Verification Mandatory in Workflow](feedback_ux_verification_mandatory.md) — Dev workflow step 9 now enforces mandatory UX screenshot comparison before completion (added 2026-03-15)
- [CSS Verify Before Iterating](feedback_css_verify_before_iterate.md) — Always check DevTools that CSS rules are generated before iterating on classes
- [Three-Gate Verification](feedback_verification_workflow.md) — Dev → Bob (SM) → Sally (UX) → User; never skip internal gates
- [No Background Tests](feedback_no_background_tests.md) — Never run test suites with run_in_background; causes orphaned vitest workers
- [Auto-Execute Stories](feedback_auto_execute_stories.md) — Don't pause between stories to ask; just continue the pipeline automatically
- [Retro Action Items Tracking](feedback_retro_action_items_tracking.md) — ALL retro items become sprint-status entries, no exceptions (Epic 8 retro)
- [Story Splitting Rule](feedback_story_splitting_rule.md) — Cross-stack stories with >3 tasks per side must be split (Epic 8 retro)
- [Bilingual Docs Rule](feedback_bilingual_docs.md) — All user-facing docs require EN + zh-TW versions (Epic 8 retro)
- [Pencil Label Overlap](feedback_pencil_label_overlap.md) — Screen labels/titles must not overlap with other content in Pencil designs
- [Pencil Spec Screens Stand Alone](feedback_pencil_spec_standalone_screen.md) — Design-decision/spec annotations get their own standalone .pen screen, never crammed into an existing mockup (bugfix-10-6)
- [Run Prettier Before Commit](feedback_format_before_commit.md) — Always run format:check before commit; subagent edits skip Prettier
- [Let User Demo Before Proposing](feedback_let_user_demo_before_proposing.md) — Don't anchor on bug-title literal framing; wait for user's demo + narrative before recommending
- [Respond in zh-TW](feedback_respond_in_zh_tw.md) — Reply in Traditional Chinese (config sets `communication_language: zh-tw`); don't drift to English even in long technical summaries

## Infrastructure
- [GitHub Account](project_gh_account.md) — push/PR with `j620656786206` (personal); switch before any PR/CI op (work repos use `tvbstw`)

## Technical Decisions
- [qBittorrent State Mapping](project_qbt_state_mapping.md) — qBT 4.x+5.0+ state→Vido status follows Sonarr/Radarr standard
- [gh Account for PRs](project_gh_account_for_prs.md) — `gh pr create` on vido needs active account `j620656786206` (default-active `alexyu-tvbs` isn't a collaborator); `gh auth switch` first

## Product Decisions
- [CN Subtitle Policy](project_cn_subtitle_policy.md) — 大陸影片保留簡體字幕不轉換；用 TMDb production_countries 判斷；Epic 8 範圍內處理
- [Multi-Library Management](project_multi_library.md) — Route 2 決定：多資料夾+手動類型指定，schema 預留自動偵測；PRD 和 UX spec 已完成，設計稿待補
