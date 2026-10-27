# disc-2026-10-glossary-seeds-actor-names-and-fan-surname-a：名詞對照表不再塞用不到的演員名，「范」「于」等姓氏不會被轉錯

Status: review

<!-- SM Bob create-story 2026-10-27；行號為 main `ec7c…` 之後。傘狀條目 disc-2026-10-glossary-seeds-actor-names-and-fan-surname 拆成 -a（本單：演員名＋姓氏）／-b（TMDb 沒有中文角色名時改從中文維基補）。拆的理由：-b 要新接一個外部來源並解析維基表格，規模跟 -a 不同級。 -->

**Source:** 2026-10-27 Alexyu 看《百花殺》的名詞對照表問「為什麼左邊英文、右邊官方譯名」。正式機（唯讀）：10 條全是演員名（`Meng Ziyi → 孟子義`…），角色名 0 條（TMDb 中文角色名只有拼音）；`Fan Shuaiqi → 範帥琦` 是錯的（TMDb 給「范帅琦」）。Alexyu 選 A。

## 查到的事

- `GlossarySeeder` 從 TMDb 的英文版與 zh-TW 版演職員表配對，**演員名和角色名都收**（`pairMovieCredits`／`pairTVCredits`）。演員名在戲劇台詞裡幾乎不會出現，對翻譯沒幫助，只是雜訊（也會被當成聽寫提示詞送出去）。真人節目（實境、談話、紀錄片）例外——台詞會叫本人（《種地吧》）。
- 簡轉繁走 OpenCC `s2twp`。本機實測人名：`范帅琦→範帥琦`、`范闲→範閒`、`于正→於正`、`党国文→黨國文`、`种丹妮→種丹妮`、`涂松岩→塗松巖`——姓氏被當成一般字轉掉；`叶→葉`、`钟→鍾`、`郑→鄭` 這些是對的。

## Acceptance Criteria

1. **演員名只在真人節目收**：TMDb 類型有 實境（10764）、談話（10767）、新聞（10763）、紀錄片（99）其中一個時，演員名照舊收；其他（戲劇、電影、動畫…）只收角色名。
2. **判斷不了就照舊**：拿不到類型（TMDb 失敗、用的客戶端沒有這個功能）→ 照舊收演員名，記 log。不會因為一次網路錯誤少收東西。
3. **姓氏不轉錯**：種子名詞從簡體轉繁體後，如果原本第一個字是 范／于／党／种／涂，而轉出來變成 範／於／黨／種／塗，就把第一個字改回原本的姓。只動第一個字；其他字照 OpenCC。
4. **已經存在的列不動**：舊的演員名列不刪（資料表分不出哪條是演員、哪條是角色），使用者自己刪或不理。使用者確認或手改過的本來就不會被種子碰到。
5. **測試**：
   - 戲劇（類型沒有真人節目）只收角色名；實境／談話／紀錄片照收演員名；拿不到類型照收演員名。電影、影集各一條。
   - 五個姓氏各一條：轉錯的會改回來；`叶→葉` 這種對的不動；名字中間的字不受影響。

## 不要做的事

- 不接維基（-b）。
- 不清舊資料。
- 不改 OpenCC 的設定或字典（字幕也用同一套，改字典會影響字幕）。

## Tasks

- [x] Task 1 — `FetchCredits`：拿 TMDb 類型，非真人節目就丟掉演員配對（窄介面斷言，不擴大 `GlossaryCreditsClient`）（AC 1、2）
- [x] Task 2 — `SeedFromCredits`：OpenCC 之後修正姓氏（AC 3）
- [x] Task 3 — 測試、vet、staticcheck（AC 5）

## Dev Notes

- 種子：`services/glossary_seeder.go`：`FetchCredits`（`:221`）、`SeedFromCredits`（`:253`，`toTaiwan` 在 Pass 2）、配對 `pairMovieCredits`／`pairTVCredits`（`CastPair.Kind` 已經分 actor／character）。
- 客戶端：`main.go:761` 傳的是 `tmdbService.CreditsClient()`，也就是 `tmdb.CacheService`；它有（有快取的）`GetTVShowDetails`／`GetMovieDetails`（`tmdb/cache.go:928、371`），`Genres []Genre`（`ID`）。
- 測試替身 `fakeCreditsClient`（`glossary_seeder_test.go:20`）沒有 details 方法 → 走 AC 2「照舊」，既有測試不用改。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- `FetchCredits` 配完對之後過 `keepActorsOnlyWhenUnscripted`：用窄介面 `glossaryGenreClient`（`GetTVShowDetails`／`GetMovieDetails`，`tmdb.CacheService` 有、走快取；編譯期斷言）讀 TMDb 類型；有 實境 10764／談話 10767／新聞 10763／紀錄片 99 才留演員配對；客戶端沒有這個功能或讀取失敗 → 全留、記 log。
- `SeedFromCredits` 在 OpenCC 之後逐條 `restoreSurname`：第一個字原本是 范／于／党／种／涂、轉出來變成 範／於／黨／種／塗 → 改回來；其他字照 OpenCC。對照表是 2026-10-27 用 OpenCC CLI 實測人名列出來的（`叶→葉`、`钟→鍾`、`郑→鄭` 對，不列）。
- 測試：`glossary_seeder_actor_surname_test.go`——戲劇（影集、電影）只剩角色名；四種真人節目類型留演員名；讀不到類型（失敗、客戶端沒有）照留；`restoreSurname` 11 種情況（五個姓、對的姓不動、只看第一個字、本來就是「範」、空字串）；`SeedFromCredits` 實際寫入「范閒」。反向驗證：拿掉兩段 → 4 個測試紅。
- **正式機資料（唯讀，2026-10-27）**：《百花殺》TMDb 286506 類型只有「劇情」18 → 之後的種子只收角色名；《種地吧》219934 類型「真人秀」10764 → 照收演員名。已經存在的列不動（AC 4）。
- `go test ./internal/services/ ./cmd/...` 全過；`go vet`、`staticcheck` 乾淨。

### File List

- `apps/api/internal/services/glossary_seeder.go`
- `apps/api/internal/services/glossary_seeder_actor_surname_test.go`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
