# Release 0.3.1：tag `v0.3.1`、release notes

Status: done

**Source:** 2026-10-22。修完 TestSprite 十月抓到的三個 bug 之後，Alexyu 選 A：發 v0.3.1，讓照指南裝的人拿到修正。

**為什麼現在發：** `:latest` 只跟正式版（#887）。三個修正（#900 混選送錯類型、#901 重新解析視窗卡住、#902 月份標錯）都是新用戶第一週會碰到的；發版前實測又抓到混選重新解析視窗最後寫 0／0，Alexyu 選 B 先修（#904）；10-27 要檢查安裝數。v0.3.0 之後只有修正和測試（4 個 fix、1 個測試環境 feat），所以是 patch 版號。

## 前提（都要 ✅ 才 tag）

- main CI 全綠（`8f1d9381`／#904）：Docker、Tests、Visual Regression（打 tag 前再確認一次）
- ⚠️ **正式機部署：這次做不到**（NAS 192.168.50.52 連不到）。替代驗證 ✅：拉 CI 打包的正式映像 `ghcr.io/j620656786206/vido:main`（revision `2917eaf8`），本機拋棄式容器、`TZ=Asia/Taipei`、假片庫複本：
  - `/subtitles/spend` 回 `from: 2026-10-01T00:00:00+08:00`，活動頁「10 月」
  - 進度連線故意延遲 1.5 秒，混選 2 部電影＋1 部影集按重新解析：送出兩個類型正確的請求，視窗走到「比對完成」
  - 容器與資料已刪
- 發版前實測發現一個 P3：混選重新解析時視窗最後寫「成功 0・失敗 0」→ `disc-2026-10-reparse-dialog-empty-followup-pass`，Alexyu 選 B 先修，PR #904 已合併。
- release notes 初稿：founder-growth；Alexyu 回「B，先修再發」＝修完照發，拿掉「還沒修好的」一節

## Acceptance Criteria

1. git tag `v0.3.1`（annotated）打在含本單的 main 上；Docker workflow 發出 image `0.3.1`、`0.3`、`0`、`latest`。
2. 發完查 ghcr.io：`:latest`、`:0.3.1`、`:0.3`、`:0` digest 相同；`:0.3.0` 不變。
3. GitHub release `v0.3.1`：標題「v0.3.1 — 內測第三版・小修」，內文＝核可草稿，`--latest`。
4. sprint-status `release-0-3-1-tag-and-notes` → done，記下 release 連結與 digest。

## 不要做的事

- 不改 GitHub About、topics。
- 不對外發文；FB 留言由 Alexyu 自己決定。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-22。

### Completion Notes List

- 2026-10-22：#905 合併（`0275b6fe`）→ `git tag -a v0.3.1` 推上去 → Docker workflow（run 39777602024）成功。
- 打 tag 前用 main 的正式映像（`8f1d9381`）在本機拋棄式容器（TZ=Asia/Taipei）再測一次四個修正：活動頁「10 月」；進度連線延遲 1.5 秒、混選 2 部電影＋1 部影集重新解析，送出兩個類型正確的請求，視窗停在「比對完成 成功 2・失敗 2」（log 有補跑 0 筆，沒有蓋掉）。容器與資料已刪。
- ghcr.io digest：`latest`、`0.3.1`、`0.3`、`0` 都是 `sha256:b8ced20a1630f9da13ed60432d3a49a563b5e364fada00a839bdf4ac67a795f8`；`0.3.0` 仍是 `sha256:41715c1d…`（沒動）。AC #1、#2 ✅。
- `gh release create v0.3.1 --latest`，內文＝founder-growth 初稿，依 Alexyu「B，先修再發」拿掉「還沒修好的」一節，第一點補一句「視窗也會顯示實際的處理結果」。Release：https://github.com/j620656786206/vido/releases/tag/v0.3.1（API `releases/latest` 回 v0.3.1）。AC #3 ✅。

### File List

- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔
