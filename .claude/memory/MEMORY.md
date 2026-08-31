# Vido Project Memory

## UX Design Screenshot Workflow
- `ux-design.pen` is the source of truth for UX designs (Pencil app)
- When .pen file changes, run `python3 scripts/export-pen-screenshots.py` to regenerate screenshots
- Screenshots saved to `_bmad-output/screenshots/` in 6 flow-based folders (A-F)
- Script connects via Pencil MCP **stdio** JSON-RPC (1.1.61 removed `--http`; Pen 1.2.5 removed `get_screenshot` — script rewritten 2026-08-19 to `execute`+`Export(scale 1)`+`sips -Z 400`, keeps 400px 長邊縮圖慣例). NOTE: full regen is non-deterministic (every PNG re-renders with byte diffs at same dimensions) — only commit genuinely-changed screens, not the whole regen.
- If new screens are added to .pen, update SCREENS dict in the export script
- Always commit screenshots alongside .pen changes
- [Verify .pen Saved Before Commit](feedback_verify_pen_saved_before_commit.md) — MCP 匯出讀 app 記憶體不讀磁碟；commit 截圖前必須確認 .pen 已存檔
- [Pen Flow Layout Convention](project_pen_flow_layout_convention.md) — canvas IA: merge same-flow desktop+mobile into one block (A–G), semantic naming, captions above frames
- [Pen Inline-Agent Workflow](feedback_pen_inline_agent_workflow.md) — .pen 修改:Sally 出提示詞 → Alexyu 跑 Inline AI Agent → Sally MCP review;agent 不直接編輯
- [Pen Schema Gotchas](project_pen_schema_gotchas.md) — 對齊 enum 無 stretch/flex-*、fill_container 的 text 需 textGrowth、Color 只吃 hex、圖示名以 Pencil 版 lucide 為準

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
- [TestSprite MCP Backend Quirks](project_testsprite_mcp_backend_quirks.md) — backend plan 要手寫、tunnel 每批後會斷、description 全 ASCII；infra 失敗用 curl 直測補產品事實
- [TestSprite v4 Regeneration Plan](project_testsprite_v4_regen.md) — round1+round2 CLOSED 2026-07-23 (PR #171/173); plan 75 cases (+TC093-105 round3 candidates, PR #174); VERIFIED 92 credits; round2 found+fixed a real scan-SSE bug (adversarial catch); env = serve-test-env.sh :8090

## Development Workflow Feedback
- [Design Verification Required](feedback_design_verification.md) — Dev must verify UI matches design screenshots after every story
- [UX Verification Mandatory in Workflow](feedback_ux_verification_mandatory.md) — Dev workflow step 9 now enforces mandatory UX screenshot comparison before completion (added 2026-03-15)
- [CSS Verify Before Iterating](feedback_css_verify_before_iterate.md) — Always check DevTools that CSS rules are generated before iterating on classes
- [Verify Interactive States, Don't Read Them](feedback_verify_interactive_states_not_read_them.md) — hover/active 要用 CDP 強制 pseudo-state 或真機驗；桌機會藏住手機的空操作（Tailwind v4 rotate ≠ transform）
- [Three-Gate Verification](feedback_verification_workflow.md) — Dev → Bob (SM) → Sally (UX) → User; never skip internal gates
- [No Background Tests](feedback_no_background_tests.md) — Never run test suites with run_in_background; causes orphaned vitest workers
- [Auto-Execute Stories](feedback_auto_execute_stories.md) — Don't pause between stories to ask; just continue the pipeline automatically
- [Retro Action Items Tracking](feedback_retro_action_items_tracking.md) — ALL retro items become sprint-status entries, no exceptions (Epic 8 retro)
- [Check CI History Before Filing](feedback_check_ci_history_before_filing.md) — CI/基礎設施掛掉先查 workflow 歷史；長期綠、剛變紅＝外部故障，不立 Rule 24 條目
- [Story Splitting Rule](feedback_story_splitting_rule.md) — Cross-stack stories with >3 tasks per side must be split (Epic 8 retro)
- [Bilingual Docs Rule](feedback_bilingual_docs.md) — All user-facing docs require EN + zh-TW versions (Epic 8 retro)
- [Pencil Label Overlap](feedback_pencil_label_overlap.md) — Screen labels/titles must not overlap with other content in Pencil designs
- [Pencil Spec Screens Stand Alone](feedback_pencil_spec_standalone_screen.md) — Design-decision/spec annotations get their own standalone .pen screen, never crammed into an existing mockup (bugfix-10-6)
- [Run Prettier Before Commit](feedback_format_before_commit.md) — Always run format:check before commit; subagent edits skip Prettier
- [Identical Rendering = Sally's Decision](feedback_identical_rendering_is_sally_decision.md) — 兩個狀態畫面相同時交 Sally 裁決，不可自選方案或只丟 backlog
- [Questions Must Show Consequence](feedback_questions_must_show_consequence.md) — 問選擇題前先確認答案不在程式碼裡；真要問就寫明「選了會改變什麼」，否則寧可先出草稿讓他改
- [Let User Demo Before Proposing](feedback_let_user_demo_before_proposing.md) — Don't anchor on bug-title literal framing; wait for user's demo + narrative before recommending
- [Architecture Prefers Long Solutions](feedback_architecture_prefer_long_solutions.md) — 架構決策禁短解；補基礎設施而非用降級/截斷繞過
- [Respond in zh-TW](feedback_respond_in_zh_tw.md) — Reply in Traditional Chinese (config sets `communication_language: zh-tw`); don't drift to English even in long technical summaries
- [User May Merge PRs Manually](feedback_user_may_merge_prs_manually.md) — during /ship's CI-watch loop Alexyu sometimes merges the PR himself and interrupts to say so; stop polling, sync main, continue
- [No Monitor for PR CI](feedback_no_monitor_for_pr_ci.md) — Monitor 盯 CI 兩次靜默失效；預設讓 Alexyu 自己盯自己合，自動化只用簡單 until-loop

## Infrastructure
- [NAS Test Instance](project_nas_test_instance.md) — Unraid @192.168.50.52:8088，SSH root 可直連；appdata 掛 /mnt/cache（2026-08-04 修好）；2026-08-24 起有真的 template（`rebuild_container` 重建）、備份會跑、PUID/PGID 支援、幽靈資料已清乾淨
- [GitHub Account](project_gh_account.md) — push/PR with `j620656786206` (personal); switch before any PR/CI op (work repos use `tvbstw`)
- [Docker CI Timeout Quirk](project_docker_ci_timeout_quirk.md) — Build & Push job hits its 30-min cap on go.mod dep changes; rerun-failed-jobs with warm cache, don't debug
- [Visual Baseline: Intentional Change](project_visual_baseline_intentional_change.md) — 功能分支上刻意改基準要「刪 linux ＋ 對分支 workflow_dispatch」；自動 bootstrap 只補缺少的、且 PR 事件上不跑

## Technical Decisions
- [qBittorrent State Mapping](project_qbt_state_mapping.md) — qBT 4.x+5.0+ state→Vido status follows Sonarr/Radarr standard
- [gh Account for PRs](project_gh_account_for_prs.md) — `gh pr create` on vido needs active account `j620656786206` (default-active `alexyu-tvbs` isn't a collaborator); `gh auth switch` first

## Product Decisions
- [Local AI Subtitle Pipeline](project_local_ai_subtitle_generation.md) — 統一 spec (2026-07-23, `vido-subtitle-pipeline-spec.md`)：抽內嵌>ASR>搜尋;Claude 雲端翻譯預設;FunASR SenseVoice 讓主流 NAS 也能本地 ASR;§6.5 名詞庫 auto-harvest(順序無關);M1 進 BMAD PRD 規劃中
- [ElevenLabs ASR Evaluation](project_elevenlabs_asr_evaluation.md) — Scribe v2 中文贏 Whisper 34%+便宜 39% 值得 spike(先測長片漏段)；日文輸 FunASR 一倍；Forced Alignment/Dubbing/TTS 全砍，對時用 alass
- [CN Subtitle Policy](project_cn_subtitle_policy.md) — 大陸影片保留簡體字幕不轉換；用 TMDb production_countries 判斷；Epic 8 範圍內處理
- [Multi-Library Management](project_multi_library.md) — Route 2 決定：多資料夾+手動類型指定，schema 預留自動偵測；PRD 和 UX spec 已完成，設計稿待補
