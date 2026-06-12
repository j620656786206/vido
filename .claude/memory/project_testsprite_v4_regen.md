---
name: testsprite-v4-regeneration-plan
description: "TestSprite v4 catalog state — Tier-0 cleanup done 2026-06-07, 58 plan / 54 active cases; execution still gated on seed data (retro-8-TS2)"
metadata: 
  node_type: memory
  type: project
  originSessionId: 841b7b63-125e-4a6c-ae96-dfe021e20f0a
---

TestSprite v4 字幕引擎測試重生計畫 — 現狀（更新 2026-06-07，TEA Murat coverage expansion）。

**權威來源：** `testsprite_tests/testsprite_frontend_test_plan.json`（**58 案**）+ `_bmad-output/audit/testsprite-queue.yaml`（**54 active** + 4 parked）。

**已完成：**
- **Tier-0 清理**：磁碟 30 個 v3 孤兒 `.py` 已 `git rm`（這才真正執行 retro-8-TS1 宣稱卻沒做的「28 廢測試移除」——它們一直還在磁碟上）。留 18 個 v4 正典 .py。
- **Tier-1 新增** 8 個 P0 旅程案 TC079-088（Downloads Monitor、Media Detail Panel、Connection 優雅降級）。4 個 Low 案（TC013/048/061/062）移到 queue 的 `parked:`，守住 Free-150 預算（54×5=270≈2.25 run/輪）。
- 設計文件：`_bmad-output/test-design-testsprite-coverage-2026-06.md`；新案 JSON：`_bmad-output/testsprite-new-cases-2026-06.json`。

**測試層級分工（重要，別重複測）：**
- 批次字幕、OpenCC s2twp 轉換、CN 政策、語言偵測 → **Go 層**（`batch_test.go` 28 測試、`engine_test.go`、`converter_test.go`），不是 TestSprite。
- Connection degrade→recover **轉場** → Playwright（mock health API）；TestSprite 無法中途翻轉後端狀態。

**已 triage 的缺口：** 批次字幕**前端 UI 從未開發**（後端 8-9/TD4 done 但 backend-only by AC）。見 sprint-status `disc-2026-06-batch-subtitle-frontend-ui: backlog` + [[project_testsprite_integration]] 旁的 `_bmad-output/discovery-triage-2026-06-07-batch-subtitle-ui.md`。

**執行仍卡 prerequisite：** `.py` 實際生成+跑 = `retro-8-TS2`（backlog）——需 deployed app + seed data（解析失敗下載、已擁有影片、未匹配項目、degraded 健康狀態）+ subtitle provider access/mocks。`scripts/seed-test-data.sh` 尚未建立。
