# Disc：報價跟著新批次重算——Sonnet 片內字幕每小時 US$0.48 → 0.28、聽寫 1.30 → 0.68

Status: review

**Source:** `disc-2026-10-translation-batch-size-default` 與 `disc-2026-10-translation-batch-truncation` 之後，Sonnet 一次 50 句、不想，實際一集便宜約一半，但選片畫面和 README 還在用舊費率（報約兩倍）。Alexyu 2026-10-15 選 A：在正式機跑幾集、用實際帳單重算。

## 查到的事（2026-10-15，main `011e122d` 之後）

- 費率在 `apps/api/internal/services/generation_candidates.go`：片內字幕 `translationUSDPerMinuteByModel`（Sonnet 0.00804＝eval-1 的 $5.951／740 分）、聽寫 `asrLegTranslationUSDPerMinuteByModel`（Sonnet 0.0216＝S01E02 的 $1.022／659 句 × 13.92 句／分）、處理時間 `translationTimeShareByModel`（Sonnet 0.17）。
- **沒量過的模型是照 Sonnet 的費率按價格比例換算**（`ratePerMinuteFrom` 用 `rates[translationCalibrationModel]` 當基準）。如果直接把 Sonnet 砍半，Opus／Sonnet 4.6／Gemini 的報價也跟著砍半——但它們還是 10 句一批，**會報太低**。Haiku 的聽寫費率原本也是換算來的。
- 正式機實測（Sonnet 50 句、不想）：

| 集 | 句數 | 花費 | 處理時間 | 片長 |
|---|---|---|---|---|
| S01E04 | 474 | US$0.159795 | 186 秒 | 3,163 秒 |
| S01E05 | 414（另 50 句碰上 NAS 斷網留英文） | US$0.132408 | 205 秒（含三次重試） | 3,110 秒 |
| S01E06 | 573 | US$0.193857 | 216 秒 | 3,264 秒 |

  → 每句 US$0.000332 × 13.92 句／分 = **0.00462／分（$0.28／時）**；處理時間 0.059～0.066 → **0.07**。
- 聽寫路（丟棄容器、S01E02 前 10 分鐘、ElevenLabs）：第一批 50 句碰上斷網；第二批 28 句 5,283 input＋451 output tokens＝US$0.022614 → 每句 US$0.000808 × 13.92 = **0.0113／分（$0.68／時）**。尾批句數少、每批重送的說明攤得少，所以偏高——報價允許偏高。第一次量時因為 segment 快取還在（DB 備份裡有 S01E02 的翻譯）只送了一批，作廢、清掉快取重量。

## Acceptance Criteria

1. Sonnet：片內字幕 0.00462、聽寫 0.0113、處理時間 0.07。
2. 沒量過的模型改用「Sonnet 10 句時的費率」當換算基準（片內 0.00804、聽寫 0.0216、時間 0.17）——它們的報價不變，不會因為 Sonnet 變便宜而報低。
3. Haiku 的聽寫費率固定成原本換算出的 0.0072（它還是 10 句一批）。
4. README 價格表、`docs/deployment.md` 的費用／時間敘述更新，標明重量日期。
5. 測試照新數字改，且新增「沒量過的模型 ≥ Sonnet 現在的費率」的斷言；`go test ./...`、vet、staticcheck 綠。
6. 正式機：升級後重新分析，一集約 53 分鐘、片內有英文字幕的候選，Sonnet 報價約 US$0.25（原本約 US$0.43）。

## Tasks / Subtasks

- [x] T1 費率與基準（AC 1～3）
- [x] T2 文件（AC 4）
- [x] T3 測試（AC 5）
- [ ] T4 正式機（AC 6，合併後）

### Time-dependent visual coverage

N/A — 純後端與文件。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-15。

### Completion Notes List

- `ratePerMinuteFrom(rates, anchorRate, model)` 多一個基準參數；新增 `translationAnchorUSDPerMinute`、`asrLegTranslationAnchorUSDPerMinute`、`translationTimeShareAnchor`。
- 18 條估價測試照新費率改（60 分聽寫：$1.66 → $1.04；只翻譯：$1.30 → $0.68 …），`TestEstimateUSD_ASRRouteCoversTheMeasuredSeeS01E02Bill` 改成用新量測重算的帳單。
- 量測花費：兩集片內字幕 US$0.33、聽寫片段兩次 US$0.13。

### Discovery Triage

- ③ 50 句一批時，一次網路失敗會留 50 句英文（10 句時只有 10 句）；既有機制會讓下一次執行只補那幾句。先觀察，不開單。

### File List

- `apps/api/internal/services/generation_candidates.go`
- `apps/api/internal/services/{generation_candidates_test.go,translation_dispatcher_test.go,transcription_estimate_test.go,transcription_estimate_route_test.go,transcription_resume_credit_test.go}`
- `README.md`、`docs/deployment.md`
