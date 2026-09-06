# Story 7.3: 掃描當下用 TMDb 角色／演員名播種 `show_glossary`（後端）

Status: done

## Story

As a BYOK NAS owner,
I want a show's glossary to already know its characters before the first episode is translated,
so that name consistency does not depend on the model getting lucky in episode one.

## Context

eval-1 發現 13：`show_glossary` 261 筆 `source` 全是 `subtitle`，schema 允許的 `metadata` 播種**從未發生**。這是修「同一人名一下英文一下中文」最直接的槓桿，也是「A 路線第一天就有料」的加速器①。

**前置：** sub-7-1（scope 綁 TMDb ID）、sub-7-2（cast 進 context）。

## Acceptance Criteria

1. **播種時機。** enrichment 寫入 credits 後（movie／series 各一處）呼叫 `GlossarySeeder.SeedFromCredits(ctx, scope, credits)`；insert-if-absent（既有詞不覆寫，`InsertNew` 語意）。
2. **播什麼。** 每位 cast：`term_src=Character`（英文角色名）→ `term_zh`＝TMDb zh-TW 的角色名（`GET /credits?language=zh-TW`；若 TMDb 回簡體或英文則 **OpenCC s2twp 轉繁**；仍是英文／空 → **不播**，不捏造）；演員本名同樣一筆（`source=metadata`，`confirmed=0`）。上限 `MetadataCastLimit`（10）。
3. **不放雜訊。** 角色名為「Self」「Narrator」「(voice)」等泛稱、或長度 > 40、或含檔名形狀 → 跳過（表格驅動，測試列舉）。
4. **重掃冪等。** 同 scope 重跑播種不產生重複（unique index + NOCASE）；使用者已改過的詞（`confirmed=1` 或 `source=manual`）永不被覆寫。
5. **可觀測。** 每次播種 `slog.Info` 帶 `scope`、`seeded`、`skipped`；`GlossaryPanelV2` 因 sub-7-1 已能顯示「中繼資料」徽章，無 FE 改動。
6. **測試。** (a) zh-TW 角色名直用；(b) 簡體 → 繁；(c) 英文角色名不播；(d) 泛稱過濾表；(e) 冪等與不覆寫；(f) enrichment 呼叫點各一條整合測試。

## Tasks / Subtasks

- [x] **Task 1 — TMDb zh-TW credits 取得（client 若已支援 language 參數則沿用）（AC: #2）**
  - [x] client 原本**沒有任何 credits 端點**（見 Discovery ①）：新增 `GetMovieCredits[WithLanguage]`（`/movie/{id}/credits`）與 `GetTVAggregateCredits[WithLanguage]`（`/tv/{id}/aggregate_credits`，一次拿整部影集的角色而不是某一季）；比照 `SearchPeople`，**不**掛進 `ClientInterface`，seeder 靠窄介面 `GlossaryCreditsClient` 拿
- [x] **Task 2 — `GlossarySeeder` + 過濾表（AC: #2, #3）**
  - [x] 同一部片各抓一次 en-US 與 zh-TW，電影以 credit_id、影集以 person id（演員）＋ credit_id（角色）配對；只看前 `MetadataCastLimit`（10）位（電影照 TMDb order，影集照集數多寡）
  - [x] 表格驅動泛稱表 `glossarySeedGenericRoles`（~60 條）＋ 括號修飾（半形／全形）剝除、「A / B」多角色只取第一個且兩邊數量要一致、`#2`／`Thug 3` 編號臨演、長度 > 40、檔名形狀（副檔名／路徑／SxxEyy／1080p 之類）
  - [x] 來源側必須含拉丁字母且不含漢字；zh 側必須含漢字（TMDb 沒翻譯時會原樣回英文 → 不播、不捏造）
  - [x] 簡→繁走**既有的** `subtitle.Converter`（官方 C++ OpenCC，s2twp）而不是新引一份 Go 移植（見裁量 2）；一部片一次 subprocess，行數對不上或 CLI 不在就原樣存
- [x] **Task 3 — enrichment 接線 + 冪等（AC: #1, #4）**
  - [x] `EnrichmentService.SetGlossarySeeder(seeder, scopes)`；三個比對落點（電影搜尋、電影 NFO→TMDb、影集搜尋）比對後抓 credits，row 寫入後經**新的窄寫入器** `UpdateCredits`（只寫 `credits` 欄，比照 `enriched_metadata_update.go` 的紀律）落地，再用 sub-7-1 的 resolver 解 scope 播種（resolver 這時會順便把 `local:` 抽屜搬進 `tmdb:`）
  - [x] cast 歸屬跟 title／poster 同一條規則：`models.ShouldOverwrite(row 原本的 metadata_source, 這次來源)`——`manual` 的 row 不動 cast（詞彙表照播，詞有自己的 provenance），`tmdb` 重比對會換成新片的 cast（第一次比錯片時不會留舊 cast）
  - [x] credits 抓失敗只 warn，row 照樣 enriched；沒有 TMDb id（Douban／Wikipedia 比對）完全不碰 seeder
  - [x] 冪等靠 repo 的 `InsertIfAbsent`（`ON CONFLICT … DO NOTHING`，NOCASE unique）；同一輪內另外以小寫 src 去重
- [x] **Task 4 — 測試（AC: #6）**
  - [x] (a)(b)(c)(d) `TestNormalizeSeedPair_Table` 43 例＋ `GenericTableIsExhaustive` 逐條驗證泛稱表；(b) 簡→繁在 `TestSeedFromCredits_ConvertsSimplifiedInOneOpenCCCall`（一部片一次 OpenCC）＋ 不在／失敗時原樣存
  - [x] (e) `TestSeedFromCredits_RescanIsIdempotentAndNeverOverwritesUserTerms` 跑**真 repo**（sqlite + 全部 migration）：manual／confirmed 不動、重跑 Seeded=0
  - [x] (f) `enrichment_glossary_seed_test.go` 14 條：NFO 路徑／搜尋路徑／影集路徑各驗「fetch → write → credits → seed」順序、credits **不**經 `UpdateEnrichedMetadata 偷渡、無 resolver 時 fallback scope、resolver 失敗 fallback、抓失敗仍 enriched、`manual` row 不覆寫 cast 但仍播種、`tmdb` 重比對換 cast、非 TMDb 來源（含**數字**豆瓣 id）不碰 seeder、NFO 沒 id 時不用 row 上舊的 tmdb_id、movie row 比成 TV 只存 cast 不播、resolver 失敗跳過不亂猜 scope、沒接 seeder 行為不變
  - [x] `repository/enriched_metadata_credits_test.go`：`UpdateCredits` 真 sqlite 來回（寫→讀回→`UpdateEnrichedMetadata` 不清掉→nil 清空→不存在的 id 報錯）
  - [x] `tmdb/credits_test.go`：兩個端點的路徑／language 參數／解析；`tmdb/cache_credits_test.go`：語言進 cache key、miss→hit、上游錯不進快取、client 未注入報錯

## Dev Notes

- Rule 27：走既有 `internal/tmdb` client（限流／快取／降級都在），不新開 client。
- 與 sub-7-5 共用 `GlossarySeeder` 介面（來源不同：TMDb vs 官方字幕）。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「發現 13」「加速器①」；`glossary_store.go` `InsertNew`；`enrichment_service.go`（credits 寫入處）

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-09-07

### Completion Notes List

- **前提修正：enrichment 從來沒寫過 credits。** 本 story 與 sub-7-2 的 Context 都寫「credits 已在 DB」／「enrichment 寫入 credits 後」——實際上 `movies.credits`／`series.credits` **只有 Metadata Editor 手填會寫**（repo 註解也這麼說：「manual-only via the Metadata Editor」），TMDb client 連 credits 端點都沒有。所以 AC #1 的「呼叫點」是本 story 新建的：enrichment 比對到 TMDb 後多打兩次 credits（zh-TW 存 row、en-US 配對），cast 存進同一次寫入。副作用是好事：詳情頁的 `CreditsSection`、`.nfo` 的 `<actor>`、sub-7-2 要接的 `Cast` context，從此對 enrichment 比對到的片都有料，不再只有手填的那幾部。
- **裁量 1（存哪個語言的 credits）**：row 上存 **zh-TW** 回應（演員名有翻譯就是中文、角色名 TMDb 通常不翻），跟 row 其他欄位（title／overview）一致；cast 上限 30 列（UI 顯示 5、NFO 全寫、prompt 取前 10，30 都夠）、crew 只留 director／writer／creator／producer 幾類（TMDb 全 crew 動輒上百列，NFO 只讀 director）。
- **裁量 2（OpenCC 用哪一份）**：一開始接了 `github.com/longbridgeapp/opencc`（純 Go、字典 embed），測試全綠後發現專案 Dockerfile **刻意**從原始碼建官方 C++ OpenCC 1.4.2、字幕安全網 `subtitle.Converter` 與豆瓣 `ChineseConverter` 都走它——詞彙表若用另一份字典轉，跟字幕安全網在片語選擇上可能不一致。改成 seeder 收 `services.OpenCCConverter`（main.go 傳同一個 `subtitleConverter`），依 [[architecture-prefer-long-solutions]] 不新引第二套字典；依賴已從 go.mod 拿掉。代價：本機 darwin 沒裝 `opencc` 時（`which opencc` 找不到）seeder 會 warn 並**原樣存簡體**（review 清單裡有簡體名總比掉一筆好）；容器內正常。
- **裁量 3（scope 從哪來）**：播種用 sub-7-1 的 `GlossaryScopeResolver.Resolve(localID)`，不是直接拼 `tmdb:movie:<id>`——因為 resolver 在這一刻會把比對前累積的 `local:` 抽屜搬進共享抽屜，這正是 sub-7-1 註解說的「upgraded in place once a match lands」的那個時刻。resolver 沒接或失敗才 fallback 拼字串。
- **裁量 4（電影 row 比對成 TV）**：`enrichMovie` 既有邏輯允許 parser 判成 TV 而把 TV 的 TMDb id 寫進 movie row；這種情況 seeder 會照 `mediaType=tv` 去抓 aggregate_credits（資料正確），scope 則交 resolver 決定（它照 row 型別拼 `tmdb:movie:<tvid>`）——沿用既有行為，不在本 story 修（見 Discovery ②）。
- **TMDb 用量**：每部比對成功的片多 2 個 request（有 rate limiter 40/10s），**經 cache 層**：`CacheService` 新增語言進 key 的 credits getter（`tmdb:movie/<id>/credits:<lang>`），重掃／重比對不再重抓；`TMDbService.CreditsClient()` 回的是 cache service 而不是 raw client。
- **CR（adversarial，high）結果與處置**：
  - ✅ 修：第一版把 cast 用 `SetCredits` 放在 struct 上、指望 `UpdateEnrichedMetadata` 帶出去——但那支窄寫入器**刻意不寫 credits 欄**（9R-10b CR-249 B），所以 cast 從沒進 DB，10 條 enrichment 測試全靠 fake 收 in-memory struct 才綠。改成 repo 各加 `UpdateCredits`（只寫一欄）＋ 真 sqlite 來回測試釘住。
  - ✅ 修：「有 cast 就當使用者的」是用內容猜 provenance；改用 row 的 `metadata_source` 走 `ShouldOverwrite`，跟 title／poster 同一套。
  - ✅ 修：credits 繞過 cache 層每次重抓 → 語言進 key 的 cached getter；`CreditsClient()` 為 nil 時 main.go 補 warn。
  - ✅ 修（第二輪）：(a) `applyMetadataToMovie/Series` 會把**任何**數字 provider id 存進 `tmdb_id`（豆瓣 subject id 也是數字）→ credits／播種改成只在 `searchResult.Source == tmdb` 才做；(b) NFO 路徑原本拿 row 上**既有**的 tmdb_id 抓 credits——重比對錯片時 tmdb_id 還留著，會種錯片的 cast → 改成只在**這一趟**真的比到 TMDb 才做；(c) movie row 被 parser 判成 TV：resolver 照表拼 `tmdb:movie:<tvid>`，會把影集的 cast 種進某部**電影**的共享抽屜 → cast 照存、播種跳過並 warn；(d) resolver 失敗時原本 fallback 自己拼 scope 先種——但 Resolve 同時是 local→tmdb 搬家、`MigrateScope` 不覆寫，先種會永久遮住使用者在未比對期間確認過的詞 → 改成跳過（下一次 resolve 再種）；(e) 三個 regex：`Agent 47`／`Android 18` 不再被當編號臨演（改錨定泛稱名詞）、括號改由內往外剝且殘留括號就拒絕（`Bob)` 不會變成永久垃圾列）、`Bruce Wayne/Batman` 無空白斜線也能拆。各補測試。
  - ❌ 不採（spec 如此）：「來源語言寫死 en-US、非英文片靜默沒播」——AC #2 明寫英文角色名，且整條翻譯管線本來就是 FR10 English→LLM（`router.go` 非英文軌直接 skip），非英文片沒有「第一集詞彙表」可言。
  - ⏭ 不做（記錄）：「每部片播種成本」——Resolve 多 2～3 次讀＋一個 MigrateScope 交易、一次 opencc subprocess、20～40 個 autocommit insert；enrichment 每部片本來就有 LLM 檔名解析＋TMDb 搜尋（秒級），這些是毫秒級，且只對 pending 列跑。審查建議的「scope 已有 metadata 詞就整批跳過」會讓 TMDb 後來補的角色永遠進不來，語意不對；`InsertManyIfAbsent` 批次交易等 sub-7-5 一起（它會插更多）。
  - ⏭ 立案：「只有 enrichment pending 列會播種——既有已比對片庫（sub-7-1 乾跑的 133 tv／87 movie）與下載完成走 parse queue 建的列（`createMovieFromMatch` 直接 `success`）永遠不會被播種」——是產品缺口，不是本 story 的 AC；審查建議的長解是把播種掛在 `GlossaryScopeResolver` 第一次解到 `tmdb:` scope 的那一刻（順便涵蓋既有片庫的下一次字幕跑），這是換 seam 的架構決定，交 Alexyu 裁定，見 `backlog-glossary-seed-existing-library-and-parse-queue`。
- 驗證：`go test ./...` 35 套件全綠；`nx lint api`（vet + staticcheck）乾淨；FE 零改動（sub-7-1 已能顯示「中繼資料」徽章）。

### Discovery Triage

- ① **文件失真**（本 story 內修正，不另立案）：sub-7-2 Context「cast 已經存在 DB」只對手填過的片成立；enrichment 路徑在本 story 之前從未寫過 credits。sprint-status 的 sub-7-2 條目已加註，sub-7-2 開工時 AC #1 的資料來源現在真的存在。
- ③ `backlog-glossary-seed-existing-library-and-parse-queue` — CR 抓到：播種只掛在 enrichment 的 pending 列。既有已比對的片庫、下載完成經 parse queue 直接建成 `success` 的列、`SaveMovieFromTMDb`（目前休眠）都不會被播種；使用者唯一能觸發的是 BatchReparse（會重跑 LLM 檔名解析＋TMDb 搜尋，有比錯片的風險）。長解候選：seed-on-first-Resolve（resolver 第一次解到共享 scope、且該 scope 沒有 `source=metadata` 詞時補種，一次 indexed EXISTS）——同時涵蓋既有片庫與 parse queue，enrichment 端的 hook 可退場；短解：啟動時對「有 tmdb_id 但沒 metadata 詞」的 row 做一次 backfill。要換 seam，交 Alexyu 裁定。
- ② 沿用既有 quirk（不立案，記錄）：movie row 被 parser 判成 TV 後 `tmdb_id` 存的是 TV id，resolver 依 row 型別拼出 `tmdb:movie:<tvid>` 抽屜——同一部影集若另有 series row 會落在 `tmdb:tv:<id>`，兩邊不共享。這是掃描器把單集放進 movies 表的老問題（bugfix-b 之前的路徑），sub-7-1 也沒處理；等 movies/series 判型收斂再一起看。

### File List

- apps/api/internal/tmdb/credits.go（新，+ _test）
- apps/api/internal/services/glossary_seeder.go（新，+ _test）
- apps/api/internal/services/enrichment_service.go（+ enrichment_glossary_seed_test.go 新）
- apps/api/internal/services/tmdb_service.go（`CreditsClient()`、`SetCreditsClient` 接線）
- apps/api/internal/tmdb/cache.go（+ cache_credits_test.go 新；語言進 key 的 credits getter）
- apps/api/internal/repository/enriched_metadata_update.go（`UpdateCredits` ×2，+ enriched_metadata_credits_test.go 新）、interfaces.go
- apps/api/internal/testutil/mocks.go、services/enrichment_nfo_test.go、services/parse_queue_service_test.go（mock 補 `UpdateCredits`）
- apps/api/cmd/api/main.go
- _bmad-output/implementation-artifacts/sub-7-3-glossary-seed-from-tmdb.md、sprint-status.yaml
