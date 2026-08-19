# Story: Bugfix D — UI 可見資料缺陷四連發（簡體 genre／poster_path 三格式／year 全空／title 對調）

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Source:** NAS BUG HUNT（2026-07-13, party-mode）P1 bundle，全數經 live NAS 資料實證；2026-08-19 What'Sub 對抗重審列為無悔四項之 N3（README 第一句承諾「把繁體中文字幕這件事做到底」，genre 欄卻顯示「动作/冒险」）。
**Risk: 🟡 MED** —— 四個獨立根因的資料管線 bundle；修 code path 讓**新掃描**正確。既有 5,922 列的髒資料回填屬 bugfix-c（data migration），明確不在本案。

---

## Story

As a zh-TW NAS owner browsing my library,
I want genres in Traditional Chinese, posters that load, years that display, and titles the right way round,
so that the library page—the product's face—stops contradicting its own core promise on every row.

---

## Context —— 四個缺陷（live NAS 實證，sprint-status bugfix-d 條目原文）

| # | 缺陷 | 實證 |
|---|---|---|
| D1 | genre 是**簡體**（动作/冒险），且 `/library/genres` 對 5,922 部電影只回 **2 個** genre | `TMDB_DEFAULT_LANGUAGE=zh-TW` 環境下仍發生 → 某條 TMDb 取值路徑沒帶語言參數，或 genre 建庫時序在 metadata 語言解析之前 |
| D2 | `poster_path` 一欄三格式：222 列抽樣 = 103 絕對 TMDb URL / 1 相對路徑 / 118 空 | FE 前綴 base URL 的地方遇到絕對 URL → 壞圖 |
| D3 | `year` 全部 null，`release_date` 卻有值（如 `2026-02-17`） | year 推導從未執行或寫入點缺席 |
| D4 | 部分列 title/original_title 對調（title:"Mag Mag" / original_title:"禍禍女"） | zh-TW 標題落在 original_title → 清單顯示羅馬拼音 |

## Acceptance Criteria

_（風險分層：本 bundle 為 lean AC —— 每缺陷一條可驗收行為＋一條回歸釘；根因定位屬 Task。）_

### AC #1 — D1 genre 繁中化
- 新掃描/enrichment 後，genre 以 zh-TW 寫入（动作→動作）；`/library/genres` 回傳數量與 TMDb zh-TW genre 清單一致（電影 19 類上限，實際依庫內容）。
- 根因修在**取值路徑**（帶 `language=zh-TW`），不是事後 OpenCC 補救（genre 是 TMDb 官方詞表，不該進轉換器）。
- 回歸釘：enrichment 單元測試斷言 genre 請求含 zh-TW 語言參數。

### AC #2 — D2 poster_path 單一格式
- 裁定並文件化**單一儲存格式**（建議：一律存 TMDb 相對路徑，FE 統一前綴 —— 與既有多數 FE 假設一致；若現況 FE 已假設絕對 URL 則反向裁定，擇一，寫入 story Dev Notes）。
- 寫入路徑全部收斂到該格式；FE 顯示層對兩種歷史格式**容錯讀取**（防 bugfix-c 回填前的舊列壞圖）。
- 回歸釘：寫入層測試斷言格式；FE 測試斷言兩種輸入都渲染出合法 img src。

### AC #3 — D3 year 推導
- 有 `release_date`（或 episode/series 對應日期欄）的列，掃描/enrichment 後 `year` 非 null；FE 年份重新出現。
- 回歸釘：解析測試 `2026-02-17 → 2026`；無日期 → year 維持 null（不發明資料）。

### AC #4 — D4 title 方向
- 寫入規則釘死：`title` = zh-TW 顯示標題（TMDb `title` with language=zh-TW），`original_title` = 原文。對調的根因（欄位映射或語言 fallback 順序）修正。
- 回歸釘：映射測試以「禍禍女／Mag Mag」為 fixture。

### AC #5 — 範圍紅線
- 既有髒資料列**不回填**（bugfix-c 的職權；本案修 code path，新掃描正確即過）。
- 不動 TMDb rate-limit／快取層行為（Rule 27 Five Pillars 既有機制）。

## Tasks / Subtasks

- [ ] Task 1: D1 根因定位＋修復（enrichment/TMDb 取值路徑語言參數）
- [ ] Task 2: D2 格式裁定＋寫入收斂＋FE 容錯
- [ ] Task 3: D3 year 推導寫入
- [ ] Task 4: D4 title 映射修正
- [ ] Task 5: 四缺陷回歸釘＋全綠驗證
