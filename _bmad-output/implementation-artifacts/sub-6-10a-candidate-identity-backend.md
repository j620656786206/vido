# Story 6.10a: 候選列身分 —— 封面、真實片長、未匹配標題（後端）

Status: done（Alexyu 2026-09-06 裁定直接改 done，未走換模型 CR）— 原 review 註記： — 5 task 全數交付（migration **035**，不是 story 寫的「033 已用」時的預期號碼）

## Story

As a BYOK NAS owner looking at the 產生字幕 consent list,
I want every row to carry a poster, the file's real duration and an honest title,
so that I can recognise what I am about to pay for — consent without recognition is not consent.

## Context — 這個 story 為什麼存在

`/impeccable critique` 2026-09-03（20/40，`.impeccable/critique/2026-09-03T15-07-46Z__apps-web-src-components-subtitle-consent.md`）P1「列身分崩塌」。正式環境截圖：2399 列全部灰方塊、全部「片長未知，以 45 分鐘估算」、多列標題是 `[bitsearch.to] Predator.Badlands.2025.2160p…`。

根因（已查證）：

| 症狀 | 根因 |
| --- | --- |
| 封面空白 | `GenerationCandidate`（`generation_candidates.go:93-118`）沒有 poster 欄位；FE 畫佔位 span |
| 片長全未知 | `candidateRow.runtimeMinutes()`（`:689`）只讀 TMDb `runtime`；ffprobe 在 `applyFFprobeTechInfo`（`enrichment_service.go:470`）量到的 `DurationSeconds`（`ffprobe_service.go:33`）**從未持久化、從未被消費** |
| 檔名當標題 | 未比對的 `movie.Title` 就是檔名（同 sub-6-7 的 prompt 側問題，這裡是 UI 側） |

**Consumed by:** sub-6-10b（前端）。

## Acceptance Criteria

1. **持久化容器時長。** migration（下一個空號；033 已由 sub-6-5 使用）：`movies` 與 `episodes` 加 `duration_seconds INTEGER NULL`（Rule 15：repo INSERT/UPDATE/SELECT/scan 全同步；`enriched_metadata_update.go` 的 tech-info 寫入加欄）。`applyFFprobeTechInfo` 把 `info.DurationSeconds` 寫入 movie；episodes 若無 ffprobe 路徑，則在候選分析的路線探測（本來就對每個 episode 跑 ffprobe 取軌道）同時擷取時長並回寫，快取語意沿用 `routeCachePlan`。

2. **估價改讀真實時長。** `runtimeMinutes()` 優先序：`duration_seconds/60`（容器）→ TMDb `runtime` → 45 分鐘 fallback；`RuntimeKnown=true` 於前兩者。additive 欄位 `runtime_source: "ffprobe"|"tmdb"|"fallback"`（sub-4-1 `[@contract-v1]` additive 不 bump，ack + Change Log）。

3. **封面。** `GenerationCandidate` additive `poster_path string`：電影用 `movies.poster_path`；集數用**影集**的 `series.poster_path`（`resolveSeriesTitle` 的 memo 擴成 `resolveSeriesMeta` 一次取 title + poster，仍是每影集一次查詢）。空字串＝無封面。

4. **未匹配標題誠實化。** additive `tmdb_matched bool` + `display_title`：已比對 → TMDb 標題；未比對 → 走既有檔名解析器（`internal/parser`）產出的乾淨標題（片名＋年份），失敗才退回檔名。`title` 既有欄位語意不動（舊 FE 不壞）。

5. **測試。** (a) migration up/down 與 SELECT/scan 同步（真 sqlite）；(b) `runtimeMinutes` 三級優先序表格；(c) 封面：電影自有、集數繼承影集、影集缺列時空字串且只 log 一次；(d) `display_title` 三種情況；(e) 候選分析 episode 時長回寫；(f) 全回歸。

## Tasks / Subtasks

- [x] **Task 1 — migration 035 + repo 同步（AC: #1）** — `movies`／`episodes` 各加 `duration_seconds INTEGER NULL`；Rule 15 同步了 SELECT 欄位清單、scan、INSERT×3、UPDATE×2、`enriched_metadata_update`，以及兩份手寫測試 schema。
- [x] **Task 2 — ffprobe 時長寫入（AC: #1）** — movie 走 `applyFFprobeTechInfo`；episode 走候選分析的路線探測（`PredictRouteWithDuration`，同一次探測多回傳一個數字，不新增 ffmpeg 行程）+ `EpisodeRepository.UpdateDurationSeconds` 回寫。
- [x] **Task 3 — 估價優先序 + `runtime_source`（AC: #2）** — `runtimeMinutes()` 改回傳三元組；`unknownRuntimeMinutes` 註解依 Rule 24 superseded-mechanism 改寫。
- [x] **Task 4 — `poster_path` / `tmdb_matched` / `display_title`（AC: #3, #4）+ 契約 ack** — `resolveSeriesTitle` → `resolveSeriesMeta`（一次取 title + poster，memo 仍是每影集一查）；FE `subtitleService.ts` 的型別鏡像同步加四個 optional 欄位。**Swagger：查證後不存在**——候選清單的 shape 沒有 Swagger 定義（`runtime_minutes` 在 `apps/api`／`docs` 全域 grep 零命中），契約真正的鏡像是 FE 那份 interface，已同步。
- [x] **Task 5 — 測試（AC: #5）** — 新增 21 個：migration 3、repo round-trip 6、estimator/identity 12。

（後端 5 task → 跨端拆分，FE 為 sub-6-10b。）

## Dev Notes

- Rule 15 精準先例：bugfix-20-1（欄位存在但 SELECT/scan 沒載 → 永遠零值）。本 story 加的是同一類欄位，測試要含真 DB 讀回。
- Rule 24 superseded-mechanism：加了 `duration_seconds` 後，`unknownRuntimeMinutes=45` 只剩 fallback 角色——註解要改寫，不得留兩套語意。
- 與 sub-6-7 的關係：6-7 管 prompt 不吃檔名，本 story 管 UI 不顯示檔名；共用 `LooksLikeFilename`。

### Time-dependent visual coverage

- N/A — 純後端。

### References

- critique snapshot（上）；`apps/api/internal/services/generation_candidates.go:93-118,665-765`、`enrichment_service.go:466-535`、`ffprobe_service.go:28-45`、`repository/enriched_metadata_update.go:84-88`

## Dev Agent Record

### Agent Model Used

Claude Code on the web（2026-09-05）

### Completion Notes List

**交付摘要**：使用者在同意清單上看到的每一列，現在有封面、有真實片長、有誠實的標題。
2,399 列全部「片長未知，以 45 分鐘估算」的根因是 ffprobe 早就量到 `DurationSeconds`
卻沒有欄位可放 —— migration 035 補上，估價改成三段階梯：容器實測 → TMDb 片長 → 45 分鐘。

**⚠️ 四點裁量待 Alexyu 過目**

1. **migration 號碼是 035，不是 story 寫的 033。** story 寫「下一個空號；033 已由 sub-6-5 使用」，
   但 034（`add_subtitle_run_stubborn_count`，sub-6-2）在那之後也落地了。sprint-status 的
   註記寫「migration 033」同樣過期。

2. **`RouteDurationPredictor` 做成「選配介面」而非直接加寬 `RoutePredictor`。**
   episode 的時長只能從既有的路線探測拿（不能為了量長度多跑一次 ffmpeg）。直接改
   `RoutePredictor.Probe` 的簽名會弄壞每一個既有 fake ——而它們測的都不是這個能力。
   改用型別斷言：實作了就拿得到時長，沒實作就完全退回本 story 之前的行為（有測試釘住這條退路）。

3. **movie 的時長只由 enrichment 寫，sweep 不寫。** 一個欄位兩個寫入者，就是欄位失去主人的
   開始。sweep 仍會用它當下量到的數字估價，只是不持久化（有測試釘住「episode writer 不會拿到
   movie id」）。副作用：**NFO 來源的電影拿不到 duration** —— `applyFFprobeTechInfo` 在
   `VideoCodec` 已設定時提早 return（AC #7 的既有快路徑），所以那些片繼續用 TMDb runtime 估。
   非退步，但值得知道。

4. **`display_title` 的第三種情況沿用 sub-6-7 CR M4 的推理**：未匹配但標題「不像檔名」的列
   **不重解析** —— 那個名字要嘛是解析器產出的、要嘛是使用者在 metadata editor 打的，
   拿同一個檔名再解析一次只會用比較差的答案蓋掉比較好的。判定用的就是 sub-6-7 的
   `prompts.LooksLikeFilename`，兩邊共用同一條「這是檔名不是片名」的定義。

**🔴 一個已知的交付缺口（需要裁定，見 Discovery Triage）**：路線快取熱的時候，episode 不會被
探測，也就拿不到時長。Alexyu 的 NAS 快取是熱的，所以**合併後第一次掃描，多數 episode 仍會顯示
45 分鐘**，要等快取條目因檔案變動或 30 天 TTL 失效才會逐步補上。三個選項寫在 backlog 條目裡。

**測試**：21 個新測試（migration 3、repo round-trip 6、estimator/identity 12）。
全 Go 回歸綠、`go build` 綠、本次改動檔案 gofmt-clean（既有 75 檔的 gofmt 欠債見
`backlog-go-gofmt-not-enforced`，未觸及）。

### Discovery Triage

- 🆕 **`backlog-episode-duration-warm-cache-gap`**（本 story 交付時填，雙向）：sub-5-4 的路線快取
  命中時不跑探測，於是本 story 的 episode 時長回寫也不會發生。
- ⚖️ **`routeVersion` 刻意不 bump**：它的 BUMP RULE 明文寫「路線判定的語意改變時才 bump」，
  本 story 沒有改變任何一個 route verdict 的意義。拿它當「清空快取」的槓桿會讓這個常數有兩種
  意思 —— 正是這個檔案再三警告的那件事。
- 查證後**不成立**：`backlog-episode-tech-info-parity` 所述「episodes 完全沒有 tech-info 欄位」
  仍然成立，本 story **只補 `duration_seconds` 一欄**，不處理 codec／解析度／軌道，該條目維持有效。

### File List

- `apps/api/internal/database/migrations/035_add_media_duration_seconds.go`（新）
- `apps/api/internal/database/migrations/035_add_media_duration_seconds_test.go`（新）
- `apps/api/internal/models/movie.go`、`apps/api/internal/models/episode.go`
- `apps/api/internal/repository/movie_repository.go`（SELECT／scan／INSERT×2／UPDATE）
- `apps/api/internal/repository/episode_repository.go`（同上 + `UpdateDurationSeconds`）
- `apps/api/internal/repository/enriched_metadata_update.go`
- `apps/api/internal/repository/interfaces.go`
- `apps/api/internal/repository/episode_duration_test.go`（新）
- `apps/api/internal/repository/movie_repository_test.go`、`episode_repository_test.go`（手寫 schema 同步）
- `apps/api/internal/services/enrichment_service.go`
- `apps/api/internal/services/generation_candidates.go`
- `apps/api/internal/services/candidate_identity_test.go`（新）
- `apps/api/internal/services/generation_candidates_test.go`（`stubSeriesResolver` 加 posters）
- `apps/api/internal/services/parse_queue_service_test.go`、`series_season_test.go`（介面新方法的 stub）
- `apps/api/internal/subtitle/predict_route.go`（`PredictRouteWithDuration`）
- `apps/api/cmd/api/route_predictor_adapter.go`、`apps/api/cmd/api/main.go`（接線）
- `apps/web/src/services/subtitleService.ts`（契約鏡像，四個 optional 欄位）
