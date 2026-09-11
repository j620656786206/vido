# Vido - NAS Media Management Platform

## Git Workflow & Working Agreements

These apply to **every** session, not just `/ship`.

- **Never commit to `main`.** Create a NEW feature branch first, based off `main` — never off another feature/skill branch.
- **Never use git worktrees.** Use a direct `git checkout -b`.
- **Commits are conventional** with a scope: `<type>(<scope>): <summary>` (e.g. `feat(retro-11): ...`, `fix(media-detail): ...`, `chore(visual): ...`). PRs squash-merge with the `(#NN)` number appended.
- **gh account is `j620656786206`.** Verify with `gh auth status` before any PR/CI op; if active account is `alexyu-tvbs`, run `gh auth switch --user j620656786206`.
- **`-linux` visual baselines cannot be generated locally** (this machine is darwin; CI is ubuntu). Never run `test:visual:update` and commit `-linux.png` to fix a visual-regression failure — the CI `Visual Regression` workflow auto-opens a `chore(visual): bootstrap N missing -linux baselines` PR for that; merge it instead.

### Commit messages and PR summaries: plain language first (ELI5)

Alexyu is the only reviewer and the only user of this product. A commit or PR
written in identifiers (`RequestTimeoutFor(model, maxTokens)` …) cannot be
reviewed by him — he cannot tell what the change does, whether it was worth
doing, or whether it is safe to merge. So **every** commit message and PR
summary leads with plain language, in zh-TW, and keeps the technical account
below it or in a PR comment. This applies to all sessions and all agents, not
just `/ship`.

**Commit message** — the conventional `<type>(<scope>):` prefix is still
required; the summary after it states the OUTCOME in plain words, not the
mechanism. Then a three-part body:

```
fix(sub-6-3): 排隊等抽軌的大檔不會再被誤判成「壞檔」而被永久跳過

原本會怎樣：
  使用者實際遇到的壞事，含真實數字（哪部片、燒掉多少錢、卡在第幾句）。
改成怎樣：
  現在會發生什麼；每一條都是可觀察的行為，不是函式名。
怎麼確定：
  跑了哪些測試／驗證了什麼。
```

**PR summary** — this order, no exceptions:

1. 這個 PR 在解決什麼問題（含真實痛點與數字）
2. 改成怎樣 / 你會看到的差別
3. 我怎麼確定它有效（測試證據、CR 結果）
4. 沒做的事（以及立案編號）
5. `---` 之後才是技術細節，或改用 PR 留言（`gh pr comment`）承載

**The test for "plain enough":** someone who has never read this codebase
finishes the summary and can say what changed and why it matters to them.
A sentence whose content is only identifiers fails. Keep identifiers for the
technical section; name files and symbols there, not in the plain-language part.

Write in zh-TW (the project's communication language); technical identifiers,
env var names and code stay in their original form.

## Confirm Before Coding (no premature assumptions)

Before implementing, **state and confirm** any of these you're relying on rather than guessing:

- Exact JSON field names and response shapes (don't invent field names).
- Whether a ticket actually exists — if no ClickUp/Jira ticket exists, say so; don't fabricate one.
- Required test guidelines and any utils extraction the change should follow.

For non-trivial features, work **architecture-first**: outline the design and validate it before reading/exploring code or writing implementation.

## UX Design Screenshots Workflow

**IMPORTANT:** After ANY modification to `ux-design.pen` (whether via Pencil MCP tools in this session, or externally), you MUST regenerate and commit screenshots before finishing.

### Steps:

1. Run `python3 scripts/export-pen-screenshots.py` (requires Pencil.app running)
   - The script spawns its own Pencil MCP server in **stdio** mode (Pencil 1.1.61 removed the old `--http`/`--http-port` transport) — safe to run even when Pencil MCP is already active
2. Screenshots are saved to `_bmad-output/screenshots/`, one folder per **user flow**. Each flow folder holds both desktop (`-d`) and mobile (`-m`) screens; filenames are the canvas frame codes (e.g. `b3p-d.png`, `b3p-m.png`):
   - `flow-a-browse-v2/` — Browse: empty / loading / grid / list / no-results / error (A′ pilot series)
   - `flow-b-detail-interaction/` — Hover / Context Menus / Detail menus / Fallbacks (mobile) / Image-load Fallback spec (B9)
   - `flow-b-detail-v2/` — Detail v2: movie / TV / skeleton / not-found / 延伸區塊 (B′ series)
   - `flow-c-search-settings/` — 媒體庫搜尋＋篩選 / 批次操作 / 檢視偏好，以及**設定的 12 個分頁**（c4–c14，另有 c15–c22 的載入／空／錯誤／確認狀態稿：外觀・連線・金鑰・服務狀態・字幕・自訂首頁・快取・日誌・備份・匯出匯入・效能監控；媒體庫掃描在 `flow-e-scanner/e1-d`）。分頁列與分組順序以 `SettingsLayout.tsx` 的 `SETTINGS_CATEGORIES` 為準
   - `flow-d-downloads-v2/` — Download centre v2: list / batch select / card actions / skeleton / empty / fail-soft / table / mobile sheets
   - `flow-e-scanner/` — Scanner settings / Scan progress / Complete toast / Filtered-unmatched
   - `flow-f-subtitle-v2/` — Manage subtitles / generation progress / glossary / batch / 生成工作區
   - `flow-h-homepage/` — Block CRUD modal (H3) / ExploreBlock spec (H5) — the two screens v3 did not replace
   - `flow-h-homepage-v3/` — Home v3 identity rework: full desktop / TMDb-degraded / mobile / 金額顯示規則 spec / loading-skeleton / empty-library-first-run / own-content-failed
   - `flow-i-advanced-search/` — Filter rail persistent (I5) / rail states spec (I7)
   - `flow-i-discover-v2/` — Discover v2: desktop / mobile / live suggestions / rail / save filter / skeleton / no-results / fail-soft
   - `flow-j-specs/` — Design-decision spec screens (PosterCard density, subtitle badges, cost-bearing buttons…)
   - `flow-k-activity-v2/` — Activity hub v2 (net-new D4-1 destination)
   - `flow-l-requests-v2/` — Request System (Epic 13): 想要 button 3-state / season-episode tree / 5-status request list
   - `flow-m-auth-gate/` — Login and password gate
   - `flow-n-setup-wizard/` — 首次啟動精靈五步（歡迎・qBittorrent・媒體庫・API 金鑰・完成）；路由 `/setup`，實作在 `apps/web/src/components/setup/`
   - `design-system/` — Design System Reference (夜行/日巡) + Component Library + Component Anatomy + 三張日巡證據畫面
   - ⚖️ **2026-09-10：39 張 v1 過時稿已從 `.pen` 移除**（Alexyu 裁定）。判準是「有明確後繼版本才刪」——Flow A/B/D/F/G/H/I 的 v1 稿有 A′／B′／`-v2`／`-v3` 接手，所以刪；Flow C、E、M 沒有後繼版本，仍是那些流程唯一的設計稿，**保留**。`flow-a-browse`、`flow-d-downloads`、`flow-f-subtitle`、`flow-g-ai-subtitle` 四個資料夾因此清空並刪除。
   - Canvas naming + block-layout convention: see `.claude/memory/project_pen_flow_layout_convention.md`
3. If new screens are added to the .pen file, update the `SCREENS` dict in `scripts/export-pen-screenshots.py` (key = node ID, value = `(flow-folder, code)`)
4. `git add` both the `.pen` file changes AND the updated screenshots, commit together
   - ⚠️ **A full regen is non-deterministic** — every PNG re-renders with byte diffs at the same dimensions. Only stage the screenshots whose **design actually changed**; `git checkout` the rest to avoid committing re-render noise.

### Commit convention:

- If only design changed: `feat: update UX design — [what changed]`
- Include both `ux-design.pen` and `_bmad-output/screenshots/` in the same commit

## Key Paths

- UX Design: `ux-design.pen` (Pencil app, read via MCP tools only)
- Design Screenshots: `_bmad-output/screenshots/`
- Screenshot Export Script: `scripts/export-pen-screenshots.py`
- Design Brief: `_bmad-output/planning-artifacts/epic5-media-library-design-brief.md`
- Planning Docs: `_bmad-output/planning-artifacts/`
- Implementation Specs: `_bmad-output/implementation-artifacts/`

<!-- nx configuration start-->
<!-- Leave the start & end comments to automatically receive updates. -->

## General Guidelines for working with Nx

- For navigating/exploring the workspace, invoke the `nx-workspace` skill first - it has patterns for querying projects, targets, and dependencies
- When running tasks (for example build, lint, test, e2e, etc.), always prefer running the task through `nx` (i.e. `nx run`, `nx run-many`, `nx affected`) instead of using the underlying tooling directly
- Prefix nx commands with the workspace's package manager (e.g., `pnpm nx build`, `npm exec nx test`) - avoids using globally installed CLI
- You have access to the Nx MCP server and its tools, use them to help the user
- For Nx plugin best practices, check `node_modules/@nx/<plugin>/PLUGIN.md`. Not all plugins have this file - proceed without it if unavailable.
- NEVER guess CLI flags - always check nx_docs or `--help` first when unsure

## Scaffolding & Generators

- For scaffolding tasks (creating apps, libs, project structure, setup), ALWAYS invoke the `nx-generate` skill FIRST before exploring or calling MCP tools

## When to use nx_docs

- USE for: advanced config options, unfamiliar flags, migration guides, plugin configuration, edge cases
- DON'T USE for: basic generator syntax (`nx g @nx/react:app`), standard commands, things you already know
- The `nx-generate` skill handles generator discovery internally - don't call nx_docs just to look up generator syntax

<!-- nx configuration end-->
