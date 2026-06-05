---
name: TestSprite v4 Regeneration Plan
description: After Epic 8 completes, regenerate TestSprite test cases using updated standard_prd.json for subtitle engine coverage
type: project
---

Epic 8 完成後，用更新過的 `standard_prd.json` 重新讓 TestSprite 生成針對字幕引擎的測試案例。

**Why:** PRD v3→v4 遷移後（2026-03-23），TestSprite 的 62 個測試案例已過時。2026-03-25 已完成 P0 項目：更新 `standard_prd.json` 為 v4 並標記測試狀態（28 廢棄、22 需更新、12 有效）。字幕引擎是 v4 核心差異化功能，目前零測試覆蓋。

**How to apply:**
1. 確認 Epic 8 所有 Story 都已完成並通過 code review
2. 用 TestSprite MCP 重新 bootstrap：讀取更新過的 `testsprite_tests/standard_prd.json`
3. 重點生成 Subtitle Search (P1-010~P1-018) 和 Batch Processing (P1-019) 的測試案例
4. 刪除 28 個標記為 `DEPRECATED_PENDING_REPLACEMENT` 的舊測試案例
5. 更新 22 個標記為 `VALID_WITH_UPDATES_NEEDED` 的測試案例的 UI assertions
6. 注意 TestSprite Free plan 有 150 credits 限制，優先生成 P0 Journey「字幕自動化」的測試
