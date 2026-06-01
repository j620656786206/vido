<!-- 雙語文件：繁體中文（本檔）· English → testsprite-local-dev.md -->

# 本地執行 TestSprite 做測試開發

> **狀態（2026-06-01）：** 月度自動把關
> （`.github/workflows/testsprite-monthly.yml`）已**暫停（deferred）**——
> 其 `schedule:` cron 已被註解掉。在本地 journey 測試套件養成期間，改用**本地**
> 執行 TestSprite。Workflow 檔案保留，仍可手動觸發（`workflow_dispatch`）；復活
> 條件見檔頭的 `DEFERRED` 註記。Carry-forward 以
> `retro-19-OPS1-wire-testsprite-secrets` 追蹤於 `sprint-status.yaml`。

## 為什麼現在優先本地

2026-06-01 一次手動 `workflow_dispatch` 證明 `TESTSPRITE_API_KEY` secret 有效，
但暴露了兩個讓自動月度 cron 目前「弊大於利」的 blocker：

1. **runner-local endpoint = 環境噪音。** 沒有穩定的 `TESTSPRITE_TARGET_URL` 時，
   workflow 會在 runner 上臨時起一個 Vido（API :8080 + Vite :4200）。24 個案子中
   **15 個回 `error`（不是 `fail`）**——代表測試根本沒跑出結論，而非真實產品漂移。
   該次也跑了約 **43 分鐘**。
2. **commit 回推被擋。** workflow 要把 queue + 新生成的 `TC*.py` commit 回 `main`，
   但 branch protection 擋掉 `github-actions[bot]` → `git push failed after 3 attempts`。

本地開發可同時避開兩者：你掌控環境，且只跑你正在開發的那個測試。

## 前置需求

- 一把 TestSprite API key。CI 用的值對應
  `testsprite_tests/tmp/config.json` 的 `executionArgs.envs.API_KEY`（或從
  TestSprite dashboard 重新發行）。
- 本地把 Vido app 跑起來（受測目標）。

## 啟動受測 app

```bash
pnpm nx serve api      # API 在 :8080
pnpm nx serve web      # Vite dev server 在 :4200
```

## 路線一 — MCP server（推薦，互動式開發）

本 repo 已配置 **TestSprite MCP server**。在 agent 對話中可直接驅動它，把測試
範圍縮到單一頁面/journey：

- `testsprite_bootstrap` — 為專案初始化
- `testsprite_generate_frontend_test_plan` — 產生測試計畫
- `testsprite_generate_code_and_execute` — 生成 `.py` 案例並執行

這是針對單一 journey 做 red-green-refactor 最快的路。

## 路線二 — CLI（重現 CI 的跑法）

```bash
# 所有設定（testIds、目標 URL 等）皆讀自
# testsprite_tests/tmp/config.json
API_KEY=<你的-testsprite-key> \
  npx --yes "@testsprite/testsprite-mcp@0.0.37" generateCodeAndExecute
```

- CLI 讀 `process.env.API_KEY`（或 `TSMCP_API_KEY` 作 fallback）——**不是**
  `TESTSPRITE_API_KEY`。
- 生成的 `.py` 案例落在 `testsprite_tests/`。
- 要只跑部分案例，把 `testsprite_tests/tmp/config.json` 的
  `executionArgs.testIds` 縮小即可——不必每次都跑全部 24 個（約 43 分鐘）。

## 之後如何復活月度把關

當以下三點就緒，取消 `testsprite-monthly.yml` 內 `schedule:` 兩行的註解即可：

1. 本地測試套件已穩定；
2. 設定了穩定的 `TESTSPRITE_TARGET_URL`（取代 runner-local，降低 error 率）；
3. 解決 bot 推 `main` 的問題（改推 side branch / 開 PR，或把
   `github-actions[bot]` 加入 branch-protection bypass 名單）。

## 延伸參考

- Workflow：`.github/workflows/testsprite-monthly.yml`
- Story：`_bmad-output/implementation-artifacts/19-6-github-actions-testsprite-monthly.md`
- Queue：`_bmad-output/audit/testsprite-queue.yaml`
