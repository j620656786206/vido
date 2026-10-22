# Release 0.3：tag `v0.3.0`、release notes

Status: in-progress

**Source:** 2026-10-22。Alexyu 問「要不要發 0.3.0」，選 A：先部署正式機驗證，再發 v0.3.0；release notes 用 founder-growth 的初稿（Alexyu 再選 A 照發）。

**為什麼現在發：** #887 之後 `:latest` 只跟正式版，現在停在 #886 那版的開發 build。10-20 FB 貼文之後照指南裝的人，拿不到「首次安裝修正」（#887、#888、#890、#892）和 Node 24 安全修補（#894）。10-27 是檢查日，不發版這些修正永遠到不了外部使用者手上。v0.2.0 之後 main 有 83 個改動（29 feat、20 fix），所以是 minor 版號。

## 前提（都要 ✅ 才 tag）

- main `f75855aa`（#894）CI 全綠：Docker、Tests、Visual Regression ✅
- 正式機部署 `main-f75855a` ✅（2026-10-22 10:24，備份 `backups/pre-deploy-20261022-1024.db`；migration 47；電影 57、影集 87 與部署前相同；首頁與前端 JS 200；開機紀錄 `AI parsing available provider=gemini`；`/subtitles/spend` 有 `parse_calls`／`parse_usd`）
- 只有 Gemini 的「試跑 20 句」實測 ✅：拋棄式容器、資料庫複本拿掉 Claude 金鑰，gemini-3.6-flash → 裁判 gemini-3.7-flash、20 句、0 分 10%、自然 90%、US$0.0073。容器與複本已刪。

## Acceptance Criteria

1. git tag `v0.3.0`（annotated）打在含本單的 main 上；Docker workflow 發出 image `0.3.0`、`0.3`、`0`、`latest`（#887 之後 `latest` 只在正式版 tag 產生）。
2. 發完查 ghcr.io：`:latest` 與 `:0.3.0` digest 相同；`:0.2` 不變。
3. GitHub release `v0.3.0`：標題「v0.3.0 — 內測第三版」，內文＝核可草稿，`--latest`。
4. sprint-status `release-0-3-tag-and-notes` → done，記下 release 連結與 digest。

## 不要做的事

- 不改 GitHub About、topics（v0.2 已改，這次不動）。
- 不對外發文；FB 留言由 Alexyu 自己決定（founder-growth 建議在 10-20 原文底下留一則，附 release 連結）。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-22。

### Completion Notes List

### File List

- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔
