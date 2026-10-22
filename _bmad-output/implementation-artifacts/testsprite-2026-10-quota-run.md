# testsprite-2026-10-quota-run：十月 TestSprite 額度（150 點）用在「新用戶第一週會碰到的路」

Status: in-progress

**Source:** 2026-10-22 Party Mode（Murat／John／Bob／Sally／Amelia）＋ founder-pm 判決，Alexyu 選 A。額度月底歸零（見 9 月 `testsprite-2026-09-quota-run`）。

## 為什麼

- 10-22 剛發 v0.3.0，照安裝指南裝的人現在拿到 0.3.0；10-27 檢查安裝；**停損 2026-11-15**（沒有外部的人裝起來並跑出一支字幕 → 回頭檢視定位）。
- 9/29 之後 264 個 commit，精靈、媒體庫「修字幕清單」、活動記錄、AI 花費都沒有 TestSprite 案例。
- founder-pm：點數只顧「新用戶第一週會碰到、壞了就走」的路。真正的停損測試（有人用 Docker 裝起來）TestSprite 測不到，不能拿它代替找人裝。

## 點數預算（約 0.5 點／案例）

| 階段 | 做什麼 | 點數 |
|---|---|---|
| 1 | 9/29 之後有改到的區塊，既有案例重跑一次（約 40 條；不需新假資料，**今天就跑**） | ~20 |
| 2 | 新案例 TC139+ 約 18 條 × 3 次（等 `testenv-seed-2026-10` 合併） | ~27 |
| 3 | 修完 bug 重跑＋可疑 PASS 本機 Playwright 核對（下拉選單類） | ~20 |
| 4 | 全量回歸一輪 | ~50 |
| — | 備用 | ~30 |

## 新案例範圍（TC139+）

**P0 新用戶第一週**
1. 精靈只填 Gemini 金鑰能走完，並講清楚 AI 費用付給誰（#836；需 `--with-encryption-key`）
2. 精靈匿名回報那一步：預設關、看得到送出內容原文、選擇到設定頁還在（#667）
3. TMDb 內建金鑰：精靈不嚇人、設定頁寫「使用內建金鑰」（#626）
4. 「N 部缺中文」一點就篩，數字＝清單筆數（#826）
5. 依字幕狀態排序：缺中文最前、繁中最後（#820）
6. 生成字幕：沒金鑰 → 停用＋指到金鑰設定；只有 Gemini → 可按、看得到報價

**P1 錢與信任**
7. 活動記錄每次生成：時長、花費、中文原因（#828/#831/#851）
8. 灰色「已略過」vs 紅色「失敗」（#864）
9. 本月 AI 花費「檔名解析」格＋首頁「本月 $x」一致（#892）
10. AI 帳號沒錢：說哪一家、去哪儲值（#760）
11. 季清單每集中文字幕標記，hover／Tab 有說明（#719）

**不花點數**：icon tooltip、字型、名詞對照表分季、翻譯品質、聽寫（已由 Go／vitest／Playwright 顧）；想要清單（Epic 13，要接 Sonarr，第一週碰不到）。

Sally 的驗收原則：新案例要斷言「畫面說人話」（例如已略過不能寫成「生成失敗」），不只斷言元素存在。

## Acceptance Criteria

1. 階段 1 跑完：每條 FAIL 都先本機重現再定罪（產品 bug／案例過期／環境限制／通道抽風），產品 bug 依 Rule 24 立案。
2. 精靈 TC106–TC109 對照現在的步驟（多了匿名回報那一步）確認是否過期；過期就改寫。
3. 新案例寫進 `testsprite_tests/testsprite_frontend_test_plan.json`，description 不放 `…` 這類特殊字元（codegen 會產生壞掉的 Python，見 backend quirks）。
4. 每條新案例至少跑 3 次；下拉選單／排序類的 PASS 用本機 Playwright 核對一次。
5. 報告更新 `testsprite_tests/testsprite-mcp-test-report.md`（十月一節），sprint-status 記點數用量、抓到的 bug。
6. 10-31 前額度用到 ≤ 10 點。

## Dev Agent Record

### Completion Notes List

### File List
