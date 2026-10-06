# Disc：資料庫裡的演員表全是空的——人名提示拿不到角色名

Status: review

**Source:** 2026-10-07 第三次實測：See 的 `series.credits` 是 NULL，#697 的人名提示只拿到名詞表的 3 個演員名、0 個角色名，Jerlamarel 還是聽成 Trilla Morel。

## 2026-10-07 根因（Explore agent 唯讀 ＋ NAS DB 查證）

- **NAS 現況：81 部影集、55 部電影，`credits` 全部 NULL**（不是只有 See）。See 是 2026-08-24 比對成功（`parse_status=success`、`metadata_source=tmdb`、tmdb_id 80752）。
- 存演員表的程式是 **2026-09-07 sub-7-3（#397／#399）才加的**：TMDb 詳情呼叫不帶 credits（`tmdb/tv.go:52`、`movies.go:53`），另外用 `aggregate_credits`（`tmdb/credits.go:126`）抓，再由 `enrichSeries`（`enrichment_service.go:440`）→ `persistSeriesMatch`（`enrichment_manual_match.go:406`）→ `seriesRepo.UpdateCredits` 寫進去；電影同構。
- **只在「比對那一刻」寫**。影集只在 `parse_status` 是 pending／空時才會被比對（`findUnenrichedSeries`），之後不會自動重跑；沒有任何 migration 或 backfill 回頭補舊列。所以 9/7 之前比對好的全部永遠是 NULL。
- `GlossarySeeder.EnsureSeeded`（`glossary_seeder.go:158`）其實有抓到演員表，但只拿去種名詞表、把 cast 丟掉（`_, pairs, err :=`）——這就是名詞表裡那 3 個 `source=metadata` 演員名的來源。
- 單部重新比對（`POST /library/series/:id/reparse`）：非手動列會重搜 TMDb 並寫 cast；**手動列走 `refreshManualSeries`（:380），不抓 cast**。手動選片（`POST /metadata/apply`）會寫。
- TMDb 回應形狀沒問題：`AggregateCast.Roles[].Character` 由 `tvCreditsToModel` 併成 `CastMember.Character`。

## 最小修法

1. `refreshManualSeries`／`refreshManualMovie` 加「credits 為空就補抓」（用現成 `FetchCredits` + `persistCredits`；非空不覆蓋，手動改過的名單不會被洗掉）。
2. **Backfill**：啟動或掃描時對 `tmdb_id IS NOT NULL AND (credits IS NULL OR credits='')` 的影集／電影逐一補抓（TMDb 有速率限制，排隊慢慢補；NAS 136 列）。
3. 修好後用 NAS 片段重測 #697（`asr prompt built names=` 應從 3 變成 40、Jerlamarel 拼法應穩定）。

## Story

身為 Vido 的使用者，
我希望已經比對好的片子也有演員表（角色名＋演員名），
這樣聽聲音生成字幕時人名提示才有東西可用，詳情頁也看得到演員。

## 設計（2026-10-07 實作）

1. **開機後一次補抓**：`CreditsBackfillService.Run` 列出「有 tmdb_id、credits 為空、沒被移除」的電影與影集（各最多 200 筆），用 enrichment 同一個 `FetchCredits` 抓演員表、用窄寫入 `UpdateCredits` 存。TMDb 說沒有演員表的，這個 process 不再問；抓失敗的下次再試。`main.go` 在開機 45 秒後跑一次（讓掃描與比對先走），每次開機都可以跑，幂等。
2. **手動列重新整理也補**：`refreshManualMovie`／`refreshManualSeries` 加 `fillMissingCredits`——只有 credits 完全沒有時才抓、才寫；使用者在編輯器打過的名單一個字都不動。
3. 查詢：`MovieRepository.FindMissingCredits`／`SeriesRepository.FindMissingCredits`（`enriched_metadata_update.go`）。

## Acceptance Criteria

1. 開機 45 秒後 log 出現 `credits backfill pass finished movies_filled=… series_filled=…`；NAS 上 See 的 `series.credits` 不再是 NULL。
2. 有演員表的列（TMDb 的或使用者打的）不會被查到、不會被覆蓋。
3. 單部手動列重新整理時，credits 為空就補；非空不動。
4. `go test ./...`、vet、staticcheck、lint:all 全綠。
5. **實測：** 正式機更新後看 log 與 DB；再用 NAS 10 分鐘片段跑一次人名提示（`asr prompt built names=` 應 ≫3，Jerlamarel 拼法穩定）。

## Tasks / Subtasks

- [x] T1 兩個查詢＋測試（AC #2）
- [x] T2 `CreditsBackfillService`＋測試＋main 接線（AC #1）
- [x] T3 `fillMissingCredits`＋refreshManual 兩處＋測試（AC #3）
- [x] T4 檢查（AC #4）
- [ ] T5 實測（AC #5）

## Dev Notes

- 沒有動 `MovieRepositoryInterface`／`SeriesRepositoryInterface`（加方法會弄壞幾十個測試假物件）；backfill 用自己的窄介面，main 用型別斷言拿到具體 repo。
- `UpdateCredits` 對空名單寫 NULL，所以「TMDb 沒演員表」的列每次開機會再被列出一次、再問 TMDb 一次；in-process 的 `tried` 集合只擋同一個 process 內的重複。136 列的規模可接受。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- 🔗 AC Drift: NONE（sub-7-3 的「比對時存演員表」不變，只是補上「舊列也要有」）。
- 📎 Contract Stamps: NONE。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

N/A — no out-of-scope work discovered

### File List

- `apps/api/internal/repository/enriched_metadata_update.go`、`enriched_metadata_update_test.go`（改）
- `apps/api/internal/services/credits_backfill_service.go`、`credits_backfill_service_test.go`（新）
- `apps/api/internal/services/enrichment_manual_match.go`、`enrichment_glossary_seed_test.go`（改）
- `apps/api/cmd/api/main.go`（改）
- `_bmad-output/implementation-artifacts/disc-2026-10-series-credits-empty.md`、`sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | 根因查完（Explore agent＋NAS DB）；同日 dev：backfill 服務＋手動列補抓；全綠，狀態 review，待正式機實測。 |

