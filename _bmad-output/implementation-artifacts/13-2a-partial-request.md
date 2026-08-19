# Story 13.2a: 部分請求（選季/選集）—— 後端：wire 形狀、驗證、Sonarr 選取感知 AddSeries、coverage 端點

Status: done

**Epic:** `epic-13`（Request System，G-2/P3-002，artery #4）· **Risk: 🟡 MED（13-4b AC #2 契約 bump＋Sonarr 三步驟新 API 面）** · **BACKEND-ONLY（cross-stack 拆分之 a 半）**
**Source:** `epics/epic-13-request-system.md` §13-2 · sprint-status `13-2-partial-request` seed
**Cross-stack split check:** backend tasks = 5, frontend tasks = 0（b 半 = `13-2b-partial-request`，依賴本 story）

---

## ⚖️ Authoring 裁定（Alexyu 2026-08-19，AskUserQuestion 確認）

**v1 不做追加請求（選項 A）**：一部劇同時僅一筆 active request（唯一索引 `idx_requests_active_unique` **不動**）。部分請求在建立當下一次選好；要加選 → 走 13-7 取消再重新請求，或等完成後再請缺的。已有 active request 時 FE 按鈕已是「已請求·處理中」pill（不開樹），故 v1 無 Sonarr already-exists 更新路徑。**追加請求（additive）與 Sonarr 既有-series 更新機制是同一件事** —— 已併入既有 backlog `disc-2026-07-arr-already-exists-loop`（條目已於本次 authoring 擴充，雙向連結）。

---

## 🔎 現實盤點（authoring 時驗證）

| 零件 | 現況 |
| --- | --- |
| `requests.seasons`/`episodes` 欄位 | ✅ 已在 migration 027（TEXT，恆 NULL —— 13-1a 預留給本 story）；`requestColumns`/scan 已含（Rule 15 同步無缺） |
| 唯一索引 | `idx_requests_active_unique ON (tmdb_id, media_type) WHERE status IN ('pending','searching','downloading')` —— ⚖️ A 裁定下**不動** |
| resource shape | `models.Request.Seasons/Episodes`（NullString）已在 13-1a AC #2/#3 `[@contract-v1]` 的 JSON 內（FE `MediaRequest.seasons/episodes` 已存在，恆 null） |
| Sonarr `AddSeries` | whole-series only（`monitored:true` 全季＋`addOptions.monitor:"all"`）—— 13-4b AC #2 `[@contract-v1]` **明文預告本 story bump** |
| `plugins.AddOptions` | `{QualityProfileID, RootFolderPath, SearchNow}` —— 13-4a AC #1 `[@contract-v1]`（加 optional 欄位 = additive，不 bump） |
| TMDB season 資料 | `GetTVShowDetails`（含 `Seasons[]` 摘要＋`episode_count`）與 `GetSeasonDetails`（整季集清單，cache 24h TTL）**client/cache 方法皆已存在**；但 `GET /tmdb/tv/:id/season/:n` **HTTP 路由未註冊**（Rule 15「client method exists ≠ route registered」的 10-2 教訓——本 story 補上） |
| owned guard | 13-1a 為 title-level（series 已入庫 → 全面 409）—— 部分請求需要 episode-level（部分擁有的劇可請缺的集） |
| FulfilmentService | `fulfil()` 共用 gate→add→transition 流程；`AddOptions` 由 settings 組成 —— selection 需從 request row 帶入 |

---

## Story

As a user requesting a TV series,
I want to request specific seasons or individual episodes instead of the whole show,
so that Vido (via Sonarr) only acquires what I actually need.

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` 部分選取 wire 形狀（create body 擴充，additive on 13-1a）

- `POST /api/v1/requests` body 增加 **optional** `seasons`（int array）＋ `episodes`（object：season number 字串 key → int array，Rule 6 snake_case）。兩者皆缺/空 = 今日 whole-title 行為 **byte-identical**（回溯相容，movie 與整劇 TV 不變）。
- 僅 `media_type=tv` 接受 selection；movie 帶任一欄 → 400 `VALIDATION_INVALID_FORMAT`。
- Canonical 化後存庫：`seasons` 排序去重（如 `[1,3]`）；`episodes` 各 list 排序去重非空（如 `{"2":[5,6]}`）；**同一季不得同時出現在 seasons 與 episodes keys**（整季即整季、選集即選集，混填 → 400）。whole-series（無 selection）→ 兩欄維持 NULL（與既有 rows 同義）。
- create/list 回傳的 `seasons`/`episodes` 由恆 null 開始承載 canonical JSON 字串 —— **additive** on 13-1a AC #2/#3 `[@contract-v1]`（欄位早已在 shape 裡），**不 bump**；ack＋Change Log。
- 本 AC 的 selection canonical 形狀 stamp `[@contract-v1]`（consumers：13-2b tree、13-2b requests-list 顯示）。

### AC #2 — 驗證 vs TMDB season 資料（Epic 2 chain）

- create 時經既有 TMDb service（Rule 27 by reuse——零新 client/limiter）驗證：season number 必須存在於 `GetTVShowDetails.Seasons`；episode number 必須落在該季合法集數（用 `GetSeasonDetails` 的集清單，**只對 episodes keys 涉及的季抓**——seasons 整季選取不需逐集驗）。季 0（特別篇）跟隨 TMDB 資料：有列出就可選，無特判。
- 非法 selection → 400，**Rule 7 新碼 `REQUEST_INVALID_SELECTION`**（既有 `REQUEST_` prefix 下的 code-list 擴充——prefix 數維持 16、`code-review/instructions.xml` 零改動，sub-1-3/13-4b「code-list update only」先例；驗證後記錄於 Completion Notes）＋ zh-TW message（Rule 3 envelope）。
- TMDb 暫時故障 → 既有 typed TMDB_* 錯誤原樣上拋（resolveTitle 同款路徑，不吞）。

### AC #3 — Guards（⚖️ A 裁定落地）

- **active-duplicate guard 不變**：同 title 已有 active request（不論 whole/partial）→ 409 `REQUEST_DUPLICATE`；唯一索引零改動。
- **owned guard 升級為 episode-level（僅 TV＋帶 selection 時）**：series 已入庫但 selection 含**未擁有**的集 → 放行；selection **全部已擁有** → 409 `REQUEST_ALREADY_IN_LIBRARY`；selection 與已擁有**部分重疊** → 400 `REQUEST_INVALID_SELECTION`（誠實拒絕、不默默 trim——FE 樹已 disable 已擁有列，正常流程不會送出重疊）。movie 與 whole-series TV 維持今日 title-level 行為。
- ownership 資料源：本地 series（by tmdb_id）＋ episodes 表（season/episode numbers）——新窄 helper（repository 層，簽章 impl 時定；與 AC #5 coverage 共用同一查詢路徑，一處實作兩處消費）。

### AC #4 — Sonarr 選取感知 AddSeries（13-4b AC #2 `[@contract-v1→v2]` bump）

- `plugins.AddOptions` 增加 **optional** `Seasons []int`＋`Episodes map[int][]int`（零值 = 今日行為——**additive** on 13-4a AC #1 `[@contract-v1]`，不 bump；ack＋所有 fakes 同步）。
- `sonarr.AddSeries` selection-aware：
  - **whole**（空 selection）→ 現行為 **byte-identical**（全季 monitored＋`monitor:"all"`＋`searchForMissingEpisodes`）——既有測試全綠不改。
  - **partial** → POST /series 時：selected 季（seasons ∪ episodes keys）`monitored:true`、其餘 `false`；`addOptions.monitor:"none"`＋`searchForMissingEpisodes:false`（搜尋改由下述 command 顯式觸發，避免整劇誤搜）。
  - **選集季的集級處理**（series 建立後）：`GET /api/v3/episode?seriesId={id}&seasonNumber={n}` → 該季**未選取**的集 `PUT /api/v3/episode/monitor {episodeIds, monitored:false}`（選取集保持 monitored）→ 搜尋觸發：整季選取 `POST /api/v3/command {name:"SeasonSearch", seriesId, seasonNumber}`、選集 `POST /api/v3/command {name:"EpisodeSearch", episodeIds}`（僅 `opts.SearchNow` 時）。⚠️ 三個端點皆 Sonarr v3+ 官方 API（episode list / episode monitor / command）——impl 時以 httptest 釘住 payload 形狀，против v4 現況 web-verify（13-4b TVDB gotcha 的查證紀律）。
  - collection 步驟任一失敗 → 整個 AddSeries 回 typed `DVR_ADD_FAILED`（series 已建立但 selection 未完成 = 誠實失敗；request 走既有 stayPending 路徑，14-7/13-3a reconcile 領域）。
- **13-4b AC #2 `[@contract-v1→v2]` bump 義務（Rule 20 producer-side，於實作同 change 執行）**：
  1. 13-4b story 檔 AC #2 stamp 改 v2＋兩段式 Change Log row（what changed：AddSeries 由 whole-series-only 變 selection-aware／what breaks downstream：對 AddOptions 零值呼叫行為不變，但語意契約含 selection 分支）。
  2. Grep 下游 ack（backtick-tolerant）：已知 **13-3a done（queue-mapping 半，FROZEN 不動）**；**13-7a/13-7b not-done（ready-for-dev，Sources 引用 13-4b AC #2）→ 兩支各補 `⚠️ STALE [@contract-v1→v2]` 於 story Dev Notes＋sprint-status 條目**（預期 re-confirm 結論：cancel 只在 pending、無 *arr un-add——selection 分支不影響，但 stale-mark 讓 13-7 dev 顯式確認）。
  3. 掃描結果記入 Change Log。
- `FulfilmentService.fulfil`：tv 路徑從 request row parse selection → 填入 `AddOptions.Seasons/Episodes`；movie 路徑零改動。

### AC #5 — 樹的資料源：coverage 端點＋season 路由補登

- **`GET /api/v1/requests/tv/:tmdb_id/coverage`** `[@contract-v1]`（consumer 13-2b）→ `{owned: {"1":[1,2,…]}, requested: {"2":[5,6]} 或 {"2":"all"}, whole_series_requested: bool, active_request: bool}`（snake_case；owned/requested 皆 season-key→episode numbers；requested 由 active request 的 canonical selection 展開，whole → `whole_series_requested:true`）。無本地 series、無 active request → 空物件＋false（200，非 404——樹對全新劇也要能開）。Handler→Service→Repository 分層（Rule 4）；與 AC #3 共用 ownership helper。
- **`GET /api/v1/tmdb/tv/:id/season/:season_number` 路由補登**（handler 薄轉接既有 `CacheService.GetSeasonDetails`；Swagger 註解；Rule 15 route↔client 同步——client/cache 方法早已存在，這正是 10-2 的「client method exists ≠ route registered」教訓的正向補課）。

### AC #6 — 範圍圍籬＋可觀測性＋全回歸

- **不改**：唯一索引（⚖️ A）、13-7a/b cancel/retry 語意（stale-mark 除外）、13-3a poller derivation（selection 不影響 status 推導——ExternalID join 不變；**ack 13-3a AC #2 `[@contract-v1]`**）、`request_progress` payload、Radarr 路徑、13-6 settings 面。
- fulfil 成功 log 加 `selection`（如 `seasons=[1] episodes={2:[5,6]}` 或 `whole`）；create log 同。
- 全回歸閘門：`go test ./...`、`pnpm nx test web`、`pnpm run lint:all`、`format:check`。

---

## Tasks / Subtasks

- [x] **Task 1 — wire 形狀＋canonical 化＋儲存（AC: #1）** 🔴 BE
  - [x] `CreateMediaRequestRequest` 加 optional `Seasons []int`＋`Episodes map[string][]int`；canonical 化（排序去重、seasons∩episodes-keys=∅、movie 拒收）；存 `requests.seasons/episodes`（canonical JSON）
  - [x] 紅線測試：無 selection 的 create 與今日 **byte-identical**（body/row/response）；canonical 化表驅動測試
- [x] **Task 2 — TMDB 驗證＋`REQUEST_INVALID_SELECTION`（AC: #2）** 🔴 BE
  - [x] season/episode 合法性驗證（只抓 episodes 涉及的季）；Rule 7 code-list 擴充＋zh-TW envelope；TMDb 故障原樣上拋測試
- [x] **Task 3 — guards：episode-level owned＋duplicate 不變（AC: #3）** 🔴 BE
  - [x] ownership 窄 helper（series by tmdb_id＋episodes numbers；real-sqlite 測試）；三分支測試（含未擁有→放行／全擁有→409／部分重疊→400）；active-duplicate 對 partial 仍 409 測試
- [x] **Task 4 — Sonarr selection-aware＋13-4b AC #2 bump＋fulfil 接線（AC: #4）** 🔴 BE
  - [x] `AddOptions` additive 加寬＋fakes 同步；`AddSeries` 三分支（whole byte-identical 紅線／partial seasons monitored／選集季 episode-monitor＋command 觸發）httptest 釘 payload
  - [x] 13-4b stamp v1→v2＋Change Log＋13-7a/b stale-mark（story Dev Notes＋sprint-status 兩處）
  - [x] `FulfilmentService` tv 路徑帶 selection；失敗 → `DVR_ADD_FAILED`→stayPending 測試
- [x] **Task 5 — coverage 端點＋season 路由＋觀測＋全回歸（AC: #5, #6）** 🔴 BE
  - [x] `GET /requests/tv/:tmdb_id/coverage`（[@contract-v1]）＋`GET /tmdb/tv/:id/season/:n` 路由補登＋Swagger；log 加 selection；全回歸閘門

（後端 task 5 個、前端 0 個 —— b 半見 `13-2b-partial-request`。）

---

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| TMDB season/episode 資料 | `tmdb.Client.GetTVShowDetails`（Seasons 摘要）＋`GetSeasonDetails`（整季集清單）＋`CacheService` 同名（24h TTL）—— 只缺 HTTP 路由 |
| 請求驗證/guard 樣式 | `RequestService.CreateRequest` 既有 owned/duplicate 兩段 guard（13-1a）—— 擴充而非重寫 |
| Sonarr 呼叫紀律 | `sonarr.Client.doRequest`（limiter 首行、typed PluginError、400→ADD_FAILED）—— 新端點照用 |
| lookup-object enrich 慣例 | `AddSeries` 現行「lookup 物件原地 enrich」idiom（手組 minimal body = Sonarr 400 的經典來源）|
| additive-no-bump 先例 | `TranslateResult.HarvestedTerms`（sub-5-5）/ `default_budget_usd`（sub-5-1）—— `AddOptions` 加欄位同款 |
| bump＋stale-mark 程序 | project-context.md Rule 20 🔁 全文＋19-6/19-7 precedent |

### 關鍵決策（authoring 已裁）

- **⚖️ A：v1 不追加**（Alexyu 2026-08-19 確認）—— 唯一索引不動、無 Sonarr already-exists 更新路徑；additive 併入 `disc-2026-07-arr-already-exists-loop`。
- **partial 時 `monitor:"none"`＋顯式 command 觸發**：`searchForMissingEpisodes` 是整劇級開關，partial 用它會誤搜未選取季；SeasonSearch/EpisodeSearch 精準觸發。
- **部分重疊已擁有 → 400 而非 silent trim**：伺服端永不默默改寫使用者的 selection（誠實 API）；FE 樹 disable 已擁有列讓正常流程不會踩到。
- **coverage 是一支端點不是三支**：owned＋requested＋whole 旗標一次回——樹開啟只打兩支（coverage＋TMDB tv details；集展開才 lazy 打 season 路由）。
- **`REQUEST_INVALID_SELECTION` 進 REQUEST_ prefix**：code-list update only（sub-1-3 先例），instructions.xml 零改動。

### 契約姿態（Rule 20）

- **產生**：AC #1 selection canonical 形狀 `[@contract-v1]`；AC #5 coverage 形狀 `[@contract-v1]`（consumers 13-2b）。
- **Bump**：13-4b AC #2 `[@contract-v1→v2]`（本 story 的核心契約動作——producer stale-mark 義務見 AC #4，13-7a/b 為已知 not-done consumers）。
- **消費（ack 於實作時 verbatim 記錄）**：confirmed against `[@contract-v1]` (13-1a AC #2/#3 — seasons/episodes 欄位啟用為 additive)；confirmed against `[@contract-v1]` (13-4a AC #1 — AddOptions additive 加寬)；confirmed against `[@contract-v1]` (13-3a AC #2 — status derivation 不受 selection 影響)。

### Time-dependent visual coverage

`N/A — 100% backend, no apps/web/ files touched.`

### References

- [Source: `_bmad-output/planning-artifacts/epics/epic-13-request-system.md` §13-2] — G-2 範圍句＋AC highlights
- [Source: `apps/api/internal/database/migrations/027_create_requests_table.go`] — seasons/episodes 欄位＋唯一索引（不動）
- [Source: `apps/api/internal/services/request_service.go:66-117`] — create 流程與兩段 guard
- [Source: `apps/api/internal/services/fulfilment_service.go:82-160`] — gate→add→transition 共用流程
- [Source: `apps/api/internal/plugins/plugin.go:29-45`] — AddOptions/DVRPlugin `[@contract-v1]`
- [Source: `apps/api/internal/plugins/sonarr/client.go:149-235`] — AddSeries 現行 whole-series 實作＋lookup enrich idiom
- [Source: `apps/api/internal/tmdb/client.go:48-55`] — GetTVShowDetails/GetSeasonDetails（HTTP 路由缺席）
- [Source: `_bmad-output/implementation-artifacts/13-4b-arr-dvr-plugin.md` AC #2] — bump 預告原文
- [Source: `project-context.md`] — Rule 3/4/6/7/15/20/27

### Previous Story Intelligence（13-4b／13-3a／sub-5-5 CR 模式）

- 手組 minimal body 是 Sonarr 400 經典來源 —— 一律 lookup-object enrich（13-4b Dev Notes idiom）。
- httptest 釘 payload 形狀（13-4a/b 全套先例）；「傳遞鏈值」要測試釘住（sub-5-5 CR：source/confirmed 值鏈——本 story 對應 monitored true/false 分佈與 command payload）。
- RED-first 紅線：whole-series 路徑 byte-identical、無 selection create byte-identical。
- 13-3a 的 poller/queue join 靠 ExternalID（series id）——partial 不改變 join key，勿動。

---

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — dev-story workflow, 2026-08-19

### Debug Log References

- RED gate（Rule 3 語言紅線）：`TestRequestHandler_CreateRequest_InvalidSelectionMapsTo400` 首跑 FAIL——handler 用 `err.Error()` 讓英文 sentinel（"invalid season/episode selection:"）漏進 zh-TW envelope → 引入 `InvalidSelectionError.Reason`（zh-TW 半）＋handler `errors.As` 取 Reason → GREEN。
- whole-series byte-identical 紅線：sonarr 既有 `TestClient_AddSeries_Success`（monitor:"all"／全季 monitored payload 斷言）**一字未改**全綠＝v1 行為 byte-identical 的實證；service 端另有 `TestRequestService_CreateWholeTitle_ColumnsStayNull` 釘 NULL/NULL。

### Completion Notes List

- **AC #1** — `CreateMediaRequestRequest` 加 optional `seasons`/`episodes`（binding 透明，handler 零改動）；`canonicalizeSelection`（排序去重、seasons∩episodes-keys 拒斥、movie 拒收、非法值逐項 zh-TW reason）；`selectionColumns`/`parseSelectionColumns` canonical JSON roundtrip（encoding/json map-key 排序 ⇒ 存庫形式決定性）；whole 路徑 NULL/NULL 紅線測試。
- **AC #2** — `validateAgainstTMDB`：season 對 `GetTVShowDetails.Seasons`、episode 只對 episodes keys 涉及的季抓 `GetSeasonDetails`；TMDb typed 錯誤原樣上拋（測試釘住）。新 Rule 7 碼 `REQUEST_INVALID_SELECTION`（**已同步 project-context.md Rule 7 code list；prefix 數維持 16，`code-review/instructions.xml` 零改動**——sub-1-3/13-4b code-list-only 先例，已驗證該檔只列 prefix 不列 code）。
- **AC #3** — duplicate guard 對 partial 照樣 409（⚖️ A，測試釘住）；episode-level owned guard 三分支（放行/全擁有 409/部分重疊 400）＋「整季選取 vs 部分擁有季＝重疊」語意，皆有測試。ownership helper 落在 **service 層**（`ownedEpisodeNumbers`：既有 `SeriesRepository.FindByTMDbID`＋新窄港口 `EpisodeOwnershipReader`（僅 `FindBySeriesID`，`*repository.EpisodeRepository` 直接滿足）——**零新 repository 方法**，比 story 預想的「repository 層新 helper」更薄；與 coverage 共用。
- **AC #4** — `AddOptions` additive 加寬（`Seasons/Episodes`＋`Partial()`）；`sonarr.AddSeries` 三分支；partial：selected 季 monitored、`monitor:"none"`＋顯式 command。**⚠️ 機制微偏離 AC 字面**：AC 寫「未選取集 PUT monitored:false」，實作為 `monitor:"none"`（add 時全集未監控）＋**選取集 PUT monitored:true**——AC 的寫法隱含「season.monitored=true 會自動監控其集」，該假設對 Sonarr 的 monitor 策略不可保證；顯式 promote 選取集在任何策略語意下都決定性，結果相同（selected monitored、unselected 不 monitored），httptest 釘 payload。集級 TMDB↔TVDB 編號差異：部分解析→skip＋Warn、整季全滅→`DVR_ADD_FAILED`（known limitation，測試釘住）。selection 步驟失敗→`DVR_ADD_FAILED`→既有 stayPending；後續 retry 的 already-exists 迴圈屬 `disc-2026-07-arr-already-exists-loop`（authoring 時已擴充該條目）。`FulfilmentService` tv 路徑 parse selection（malformed→誠實 stayPending「請求的選取資料無法解析」，不猜半份 selection）。**Rule 20 bump 已執行**：13-4b AC #2 stamp v1→v2＋兩段式 Change Log；下游掃描：13-3a done/FROZEN（只消費 queue-mapping 半）、13-7a/b not-done→story Dev Notes＋sprint-status 雙處 stale-mark。
- **AC #5** — coverage 端點＋`TVCoverage` service 方法（空殼 200 對全新劇）。**⚠️ 形狀偏離 AC 字面**：AC 草擬 `requested: {"2":[5,6]} 或 {"2":"all"}`（混型 JSON），實作為 `requested_seasons`/`requested_episodes`——與 create body 同一套 selection 詞彙（FE 兩個方向講同一種話），避免混型 map；[@contract-v1] 以實作形狀為準（consumer 13-2b 未動工，CR re-stamp 慣例）。`GET /tmdb/tv/:id/season/:season_number` 路由補登＋Swagger 註解（本 repo 無 swag 產檔流程，annotations-only 與 13-1a/13-4a 慣例一致）。
- **AC #6** — 唯一索引/13-3a poller/`request_progress`/Radarr/13-6 全零改動；create＋fulfil log 加 `selection`。全回歸：`go test ./...`（apps/api）全綠、`pnpm nx test web` 233 files/2653 tests 全綠、`pnpm run lint:all` 0 errors（117+119 既有 warnings 存量）、prettier 全綠、gofmt 觸及檔 clean（`backup_scheduler` 等存量未格式化檔屬 `backlog-go-gofmt-not-enforced`，未擴 scope）。
- 🔗 AC Drift: FOUND — 設計內（checked: 'AddSeries|AddOptions|FindActiveByTMDbID' across _bmad-output/implementation-artifacts/*.md）：本 story 改變 13-4b AC #2 出貨行為＝該 AC 明文預告的 bump，經 Rule 20 v1→v2＋stale-mark 完整走完；其餘 hits（13-1a shape、13-3a derivation）皆 REUSE（additive/不變）。
- 📎 Contract Stamps: FOUND（產生：AC #1 selection 形狀 v1、AC #5 coverage 形狀 v1（as-implemented）；bump：13-4b AC #2 v1→v2 含 producer 義務全套。消費 ack：confirmed against `[@contract-v1]` (13-1a AC #2/#3 — seasons/episodes 欄位由恆 null 啟用為 canonical JSON，additive 不 bump)；confirmed against `[@contract-v1]` (13-4a AC #1 — AddOptions additive 加寬，DVRPlugin 簽章不動)；confirmed against `[@contract-v1]` (13-3a AC #2 — status derivation 靠 ExternalID join，selection 不影響，poller 零改動)）
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story（樹 UI 屬 13-2b）

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time（2026-08-19）：
    - **③ backlog-with-carry-forward-link** — `disc-2026-07-arr-already-exists-loop`（既有條目**擴充**，雙向）：⚖️ A 裁定將「追加請求（additive partial re-request）」併入此條——兩者需要同一套 Sonarr 既有-series 偵測＋monitored 更新＋搜尋觸發機制；本 story 的 AC #4 集級處理（episode monitor＋command）落地後，該條目的實作面消失一半。

### File List

- `apps/api/internal/services/request_selection.go` — new：selection 型別/canonical 化/`InvalidSelectionError`/TMDB 驗證/ownership 三分支 guard/canonical JSON columns/`RequestCoverage` 型別
- `apps/api/internal/services/request_selection_test.go` — new：canonical 化表驅動/roundtrip/ownership 三分支/partial create 全流程/TVCoverage 測試
- `apps/api/internal/services/request_service.go` — modified：DTO 加 seasons/episodes；`EpisodeOwnershipReader` 窄港口＋建構子；`createPartialRequest`；`TVCoverage`；`pickTitle` 抽出；介面加 `TVCoverage`
- `apps/api/internal/services/request_status_poller.go` — modified（CR H1）：`SelectionOwnershipChecker` 窄港口＋`SetSelectionOwnershipChecker`＋`selectionSatisfied` 精修 rule 1
- `apps/api/internal/services/request_status_poller_test.go` — modified（CR H1）：4 支 partial-completion 測試（falsification 驗證過）
- `_bmad-output/implementation-artifacts/13-2b-partial-request.md` — new（上一輪 create-story 產物，與本 story 同批 commit）
- `apps/api/internal/services/request_service_test.go` — modified：episode-ownership stub＋fixture；`mockTMDbForRequests` 加 seasonDetails；建構子同步
- `apps/api/internal/services/fulfilment_service.go` — modified：tv 路徑 parse selection→AddOptions（malformed→stayPending）；log 加 selection
- `apps/api/internal/services/fulfilment_service_test.go` — modified：selection 直達 AddOptions＋malformed stayPending 測試
- `apps/api/internal/handlers/request_handler.go` — modified：`REQUEST_INVALID_SELECTION` 映射（`errors.As` 取 zh-TW Reason）；coverage 路由＋handler＋Swagger
- `apps/api/internal/handlers/request_handler_test.go` — modified：mock 加 `TVCoverage`；selection passthrough/400 語言紅線/coverage 三測試
- `apps/api/internal/handlers/tmdb_handler.go` — modified：介面加 `GetSeasonDetails`；season 路由＋handler＋Swagger（Rule 15 路由補登）
- `apps/api/internal/handlers/tmdb_handler_test.go` — modified：season 路由四測試
- `apps/api/internal/plugins/plugin.go` — modified：`AddOptions` additive 加寬（Seasons/Episodes/`Partial()`）
- `apps/api/internal/plugins/sonarr/client.go` — modified：`AddSeries` selection-aware（[@contract-v2]）＋`applySelection`（episode list/monitor/command）＋`postCommand`
- `apps/api/internal/plugins/sonarr/client_test.go` — modified：partial harness＋三測試（payload 斷言/無 SearchNow/編號 mismatch 失敗）
- `apps/api/cmd/api/main.go` — modified：RequestService 建構子接 `repos.Episodes`
- `project-context.md` — modified：Rule 7 code list 加 `REQUEST_INVALID_SELECTION`
- `_bmad-output/implementation-artifacts/13-4b-arr-dvr-plugin.md` — modified（Rule 20 bump）：AC #2 stamp v1→v2＋兩段式 Change Log row
- `_bmad-output/implementation-artifacts/13-7a-request-cancel-retry.md` — modified（Rule 20 stale-mark）
- `_bmad-output/implementation-artifacts/13-7b-request-cancel-retry.md` — modified（Rule 20 stale-mark）

---

## Senior Developer Review (AI)

**Date:** 2026-08-19 · **Outcome:** Approve（2 HIGH ＋ 2 MEDIUM 全數當場修復；2 LOW 一修一記錄）
**檢查:** Git↔File List 1 差異（13-2b story 檔，已補）· 🔒 Rule 7: PASS · 🔒 Rule 20 Bump: PASS（1 bump，13-3a done/FROZEN 正確、13-7a/b 雙處 stale-mark 齊全）· 🔒 Rule 25: N/A

### Action Items

- [x] **[H1] 13-3a poller 會把部分請求「第一個 tick 就標 completed」** — `reconcile` rule 1 用 title-level `ownedSet.has(tv, tmdbID)`，而部分請求的定義就是「這部劇已在庫、我要缺的集」⇒ 該規則恆真，request 立刻 completed 並誤觸 13-5 auto-subtitle seam，AC #6 的「selection 不影響 status 推導」是錯誤假設。**修**：`SelectionOwnershipChecker` 窄港口（`*RequestService.SelectionFullyOwned`，重用 create-time 的 `checkSelectionOwnership` 判定式 ⇒ 兩處「全擁有」語意不可能漂移）＋ poller `selectionSatisfied` 精修 rule 1（whole-title 與未接線 = 今日行為；查詢失敗 → **不完成**，因為 hold 可回復、錯誤 completed 是終局）＋ main.go 接線。測試 4 支，並以 **falsification** 驗證（移除守衛後 2 支立刻 FAIL）。
- [x] **[H2] 整季選取遇 Sonarr 該季無集：靜默成功** — 實跑證據：`GET /episode` 回 `[]` → `monitorCalls=0`、發一個 no-op SeasonSearch、`err=nil` ⇒ request 永遠停在 搜尋中。選集分支已有 `resolved==0` fail-loudly，整季分支沒有（不對稱）。**修**：整季分支同樣 `len(episodes)==0 → DVR_ADD_FAILED`；測試釘住「無 no-op 指令外送」。
- [x] **[M1] selection 大小無上限 → 上游呼叫放大** — 每季 = 1 次 TMDB（create）＋1 次 Sonarr（fulfil），一個請求可放大成數十次外部呼叫。**修**：`maxSelectedSeasons=100` / `maxSelectedEpisodes=2000` 天花板（遠高於真實影集，只為圍籬），超過 400；測試含「40 季長壽劇仍通過」。
- [x] **[M2] `13-2b-partial-request.md` 在分支內未列 File List** — 已補入 File List。
- [x] **[L1] `ownedEpisodeNumbers` 包 nil error** — 實跑確認訊息會變 `%!w(<nil>)`；改為 nil-row 專屬訊息。
- [ ] **[LOW] TMDB 未提供 `episode_count` 的季**，全擁有時判為 400（重疊）而非 409（已入庫）——訊息略失準，不影響安全性與資料。記錄不修。

### 已驗證非問題（避免後人重查）

- **gin 路由衝突**：實跑 probe 確認 `/requests/tv/:tmdb_id/coverage` 與 13-7a 規劃中的 `/requests/:id`、`/requests/:id/retry` **可共存**（gin 支援靜態段與 wildcard 混用），13-7a 無需改路由設計。

### CR 後全回歸

`go test ./...`（apps/api）全綠 · `pnpm nx test web` 233 files / 2653 tests 全綠 · `pnpm run lint:all` 0 errors · prettier / gofmt（觸及檔）全綠。

## Change Log

| Date | Change |
| --- | --- |
| 2026-08-19 | Code review (adversarial) — 2 HIGH + 2 MEDIUM + 1 LOW 修復，Status → done。**H1**：13-3a poller rule 1 的 title-level 擁有判定會讓部分請求第一個 tick 就 completed（並誤觸 13-5 seam）——新增 `SelectionOwnershipChecker` 精修，判定式與 create-time guard 共用；falsification 驗證。**H2**：整季選取遇 Sonarr 無集時靜默成功（實跑證據：monitorCalls=0＋no-op SeasonSearch＋err=nil）→ 補 fail-loudly，與選集分支對稱。**M1**：selection 加大小天花板（100 季/2000 集）擋上游呼叫放大。**M2/L1**：File List 補 13-2b、nil-wrap 訊息修正。已驗證非問題：gin 路由與 13-7a 規劃路由不衝突（probe 實測）。CR 後全回歸 Go＋web＋lint＋prettier 全綠。 |
| 2026-08-19 | dev-story 實作完成（Task 1–5 全數，Status → review）。Rule 20 bump 全套執行（13-4b AC #2 v1→v2＋13-7a/b 雙處 stale-mark＋Change Log 掃描紀錄）。兩處記錄性偏離：AC #4 集級監控改「monitor:none＋顯式 promote 選取集」（AC 隱含假設對 Sonarr monitor 策略不可保證，結果相同）；AC #5 coverage 形狀改 `requested_seasons`/`requested_episodes` 鏡射 create 詞彙（避免 AC 草擬的混型 map）。ownership helper 落 service 層窄港口（零新 repo 方法）。RED gate：zh-TW envelope 語言紅線（sentinel 漏出）實跑抓到後修。全回歸 Go＋web＋lint＋prettier 全綠。 |
| 2026-08-19 | create-story：Epic 13 artery #4 之 a 半（BE）。⚖️ A 裁定（Alexyu AskUserQuestion）：v1 不追加——唯一索引不動、無 Sonarr already-exists 路徑，additive 併入 disc-2026-07-arr-already-exists-loop。盤點確立：seasons/episodes 欄位＋resource shape 早在（13-1a 預留，additive 啟用不 bump）；Sonarr AddSeries 為 13-4b 明文預告的 bump 點（v1→v2＋13-7a/b stale-mark 義務）；GetSeasonDetails client/cache 在而路由缺（Rule 15 10-2 教訓補課）；owned guard 需升級 episode-level。新 Rule 7 碼 `REQUEST_INVALID_SELECTION`（code-list only）。產生 stamp×2（selection 形狀、coverage 形狀），consumer 13-2b。 |
