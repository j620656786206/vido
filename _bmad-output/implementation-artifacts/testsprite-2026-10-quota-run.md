# testsprite-2026-10-quota-run：十月 TestSprite 額度（150 點）用在「新用戶第一週會碰到的路」

Status: done

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

- 2026-10-22 階段 1 完成：7 批、55 次執行，額度 150 → 122.5。47 過、6 FAIL，6 條都不是產品 bug（本機 Playwright／API 逐條重現）：TC085 已立案的案例過期；TC096、TC122 案例過期 → 已改寫並重跑通過；TC098、TC110 是同一批互相干擾（加上 TC110 的 `select_option("")` 空洞操作），本機在乾淨資料上都正常；TC125 工具確認不了焦點，本機確認焦點在「取消」。TC068／TC136 空洞 PASS，本機確認掃描排程存得起來。AC 1、AC 2 ✅（精靈 TC106–108 沒過期）。
- 學到：會改資料的案例一批只放一條，或每條之間重新 seed。
- 階段 2 等 `testenv-seed-2026-10`。
- 2026-10-22 階段 2 完成（PR #898 之後）：TC139–TC148 十條寫進計畫，每條跑 3 次，30 次 29 過；唯一 FAIL 是 TC141 第 1 輪跟 TC140 同批（兩條都會走完精靈）。下拉／排序類（TC143）本機核對過。新案例原本列的 11 條裡，「TMDb 內建金鑰」那條測不了（原始碼自己編的版本沒有內建金鑰），「AI 帳號沒錢時說去哪儲值」只出現在跑到一半的批次視窗、假資料做不出來，改由 TC145 檢查活動記錄寫「AI 帳號沒有額度」。AC 3、AC 4 ✅。
- 2026-10-22 階段 4 完成：67 條全量回歸 60 過。TC134 因為新假資料過期（《教父》現在有片內繁中，對話框正確地說可直接套用）→ 改用《全面啟動》；TC125 改成按 Enter 證明焦點；TC093 退役；TC088 標環境限制。**抓到 2 個真 bug**：批次重新解析對話框漏掉瞬間完成的比對（TC115）、「全部」頁混選電影＋影集時整批送錯類型（查 TC115 時發現）。另 1 個 P3：側欄服務圓點唸英文。都已立案（見 sprint-status）。
- 額度 150 → 69。剩下的留給上面的 bug 修好後重跑；AC 6（月底前 ≤ 10 點）還沒到。
- 2026-10-22 晚：三個 bug 修完合併（PR #900、#901、#902），用 main 重新打包重跑。TC115 3／3 過（修之前 2 次 FAIL），新增 TC149（月份）、TC150（混選重新解析）各 3／3 過，其他跟批次操作有關的 7 條全過。額度 → 60.5。

- 2026-10-22 晚：Alexyu「逐一點擊每個頁面的詳細操作，補齊任何情境」。用 Playwright 盤點 25 個頁面的每個按鈕／連結／輸入框，對照既有 120 條找出沒點過的操作，新增 TC151–TC187（37 條），各跑 1 次（改寫的再跑 1 次）：33 過；TC163（搜尋頁 TMDb 斷線露出內部代碼、不列片庫結果）與 TC185（不存在網址只顯示英文 Not Found）是真 bug，已立案；TC186／TC187 手機版 TestSprite 不能改視窗→環境限制。TC173、TC177、TC183 第一版預期寫錯，改寫後過；TC159、TC175 本機核對過不是空洞 PASS。另立 `disc-2026-10-add-library-missing-path-and-duplicates`（要 Alexyu 決定）與 `disc-2026-10-ci-playwright-deps-hang`。額度 → 41.5。

- 2026-10-27 月底前最後一輪（main `4339c590` 之後重新打包；這幾天合併了重新選片 #915、季跟著換 #917、外網重新解析不卡 #919、Gemini 模型 #921、純季集檔名 #922）：新增 TC188–TC190（重新選片：已比對的片能從修改資訊切到手動選片且頁面不卡、沒比對到的片沒有這一列、沒存的修改不會留下），送出前先用本機 Playwright 走過，各跑 3 次 9／9 過。回歸 6 批 56 次 50 過：TC185 是已知的 `disc-2026-10-not-found-page-english`；TC085 是已知的 `plan-drift-tc085-tech-badges-fixture`；TC080／TC081／TC083 下載頁在沒有 qBittorrent 的測試環境沒有項目可測→計畫標 ENV-LIMITED；TC162 是搜尋頁兩個搜尋框清除不同步的舊問題（本機重現，記進 `disc-2026-10-search-page-tmdb-down-raw-code`）。TC143（依字幕狀態排序）本機核對過真的照順序排。會改資料的案例分批、中間重新 seed。沒有今天改動造成的問題。額度 41.5 → 10，AC 6 ✅。

### File List

- `testsprite_tests/testsprite_frontend_test_plan.json`（TC096、TC122 改寫）
- `testsprite_tests/TC096_Home_without_TMDb_shows_own_content_first_and_an_honest_explore_block_notice_below_it.py`（新）、舊 `TC096_Home_renders_Explore_blocks_below_own_content.py`（刪）
- `testsprite_tests/TC122_Service_status_page_speaks_Chinese_and_tells_you_where_to_fix_a_broken_TMDB.py`（重新生成）
- `testsprite_tests/testsprite-mcp-test-report.md`（十月一節；階段 2、4）
- `testsprite_tests/testsprite_frontend_test_plan.json`（新增 TC139–TC148；改寫 TC134、TC125、TC143 參數名；TC093 退役；TC088 環境限制）
- `testsprite_tests/TC139…TC148_*.py`（新）
- `testsprite_tests/TC188…TC190_*.py`（新）；`testsprite_frontend_test_plan.json` 新增 TC188–TC190、TC080／TC081／TC083 標 ENV-LIMITED（10-27）、`TC125`／`TC134`／`TC122` 重新生成、`TC093` 刪除
