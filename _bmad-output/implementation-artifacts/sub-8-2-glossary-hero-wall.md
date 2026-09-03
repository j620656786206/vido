# Story 8.2: 公開詞彙表牆（英雄牆式）—— 概念級 story，待 8-1 驗證後細化

Status: backlog

**Blocked-by:** sub-7-1、sub-8-1（先看有沒有人匯出／匯入）。**Not a dev-ready story**：本檔記錄 party-mode 2026-09-03 的裁定與邊界，避免日後從零討論；要開工前需 `create-story` 重新細化 + Sally 設計。

## Story

As any Vido user,
I want to apply the community's confirmed glossary for a show with one click, and report a wrong term,
so that a character is translated right once, for everyone.

## 裁定（party-mode，參考 What'Sub 英雄牆但不照抄）

| 抄形狀 | 換機制（因為詞彙有標準答案、樣式沒有） |
| --- | --- |
| 全公開的牆、不做好友系統 | key = TMDb ID（sub-7-1） |
| 一鍵套用、可收藏、卡片顯示作者 | 卡片顯示「N 人使用中 · M 人回報錯誤」，**不是愛心數** |
| QR 推薦碼藏在分享裡 | 同一詞有分歧時**攤開讓使用者選**，不偷偷決定 |
| | 來源徽章：`official_subtitle` 的條目信任度高一級 |
| | **只發布 `confirmed=1` 或 `official_subtitle`**；LLM 自挖（`subtitle`）一律不上牆 |

## 邊界

- 需要中央服務（與 self-host 定位打架）—— 這是 B 路線最大的成本；先評估「GitHub repo 當牆」（PR 即審核、Actions 產 index.json、App 只讀）能不能撐第一版。
- 隱私：上牆的是詞對，不是字幕內容；但 scope 揭露「你有這部劇」——匿名安裝 ID、不綁帳號。
- 版權：詞對本身非受保護內容；仍不放任何整句對白。
- 商業化（代管 key）與 TMDb 條款（sub-7-7）同時觸發時要一起處理。

## 開工前要裁的問題

1. 牆在哪裡（GitHub repo vs 自架服務）。
2. 「N 人使用」的匿名計數怎麼做、要不要做。
3. 回報錯誤的處理流程（誰審）。

## References

- party-mode 2026-09-03 Sally「抄形狀，換機制」；`_bmad-output/planning-artifacts/strategy-review-whatsub-2026-08-19.md`；eval-1 backlog P2-2

## Dev Agent Record

### Discovery Triage

- （待細化）
