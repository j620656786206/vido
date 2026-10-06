# Disc：資料庫裡的演員表全是空的——人名提示拿不到角色名

Status: backlog

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
