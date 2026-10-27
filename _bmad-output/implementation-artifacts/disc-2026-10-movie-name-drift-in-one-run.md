# disc-2026-10-movie-name-drift-in-one-run：同一次翻譯裡，前面定好的人名譯法會帶給後面每一批，不再一部片三種譯名

Status: done

<!-- SM Bob create-story 2026-10-27；行號為 main（#930 之後）。Rule 24 lane ③，2026-10-23 Vido vs Lingarr 比較時立案；Alexyu 10-27 選 A。 -->

**Source:** 2026-10-23 Vido vs Lingarr（Tears of Steel，76 句，Gemini 3.5 Flash-Lite，各跑一次）：女主角 Celia 在 Vido 裡是 #2「席莉亞」、#37／#45「希莉亞」、#71「西莉亞」；Lingarr 用同一個 Gemini 每次都是「西莉亞」。Vido 對外主打「人名整部一致」。

## 為什麼會這樣（查到的）

- 翻譯是分批送的：Gemini 3.5 Flash-Lite 一批 20 句（`prompts.SubtitleTranslatorBatchSizeFor`），76 句＝4 批。#2、#37、#45、#71 剛好分在第 1、2、3、4 批。
- 每批回覆後面有一段 `===TERMS===`，模型會列出這批用了哪些人名譯法（sub-5-5）。Vido **有收集**（`harvested`，先出現的為準），但只拿去做兩件事：跑完寫進名詞對照表、品質閘門比對。**沒有放進下一批的提示詞**——下一批只看得到開跑前就有的對照表，和前 N 句翻好的文字（`contextWindow`），人名沒出現在那幾句裡就等於重新猜一次。
- 電影常常沒有開跑前的對照表：Tears of Steel 的 TMDb 中文角色名是空的，種子收不到；第一次翻譯時對照表是空的。
- 兩條路都一樣：字幕管線 `subtitle/pipeline.go`（`translateTrack` 的 chunk 迴圈；這次比較走的是它）與 `TranslationService.TranslateWithGlossaryHarvest`（舊的聽寫後翻譯路徑）。
- 跟今天剛做完的名詞對照表（#928／#929）不是同一個問題：那兩張是**開跑前**從 TMDb／維基先放人名；這張是**翻譯途中**模型自己決定的譯法沒有往後傳。前者補得到的片（有中文角色名的）不會飄；補不到的（像 Tears of Steel）就靠這張。

## Acceptance Criteria

1. **字幕管線**：每一批送出時，提示詞多帶「這次翻譯前面幾批已經用過的人名譯法」（從 `===TERMS===` 收集、先出現的為準），放在使用者訊息的對照表區段，要求照用；開跑前的對照表裡已經有的詞不重複列。第一批沒有東西可帶，提示詞跟現在一樣。
2. **舊路徑**（`TranslateWithGlossaryHarvest`）同樣：從第二批起，對照表＝開跑前的對照表＋前面幾批收集到的人名（同名以開跑前的為準）。
3. **不動快取前綴**：字幕管線的系統訊息（有 prompt cache 的那段）不變；新增的人名只在每批的使用者訊息裡。
4. **順序固定**：帶入的人名照字母排序，同樣的輸入產生同樣的提示詞。
5. **被閘門整批退回的那次回覆，它的人名不帶**（沿用現有收集規則）。
6. **測試**：
   - 提示詞：有已用人名時多一段、沒有時跟現在逐字相同。
   - 字幕管線：第一批回覆 `Celia => 西莉亞` → 第二批收到的已用人名有它；開跑前對照表已有的詞不重複帶。
   - 舊路徑：第二批的提示詞有第一批收集到的人名。

## 不要做的事

- 不改每批大小、不改閘門、不改跑完寫進對照表的行為。
- 不在翻譯前先叫模型整部掃一次人名（多一次付費請求，另議）。

## Tasks

- [x] Task 1 — `prompts.EstablishedNames`（收集到的人名扣掉已有對照表、排序）（AC 1、2、4）
- [x] Task 2 — 字幕管線：`ChunkTranslator.TranslateChunk` 多收 `established`，`translateTrack` 每批帶入（AC 1、3、5）
- [x] Task 3 — 舊路徑：每批的對照表加上前面收集到的（AC 2）
- [ ] Task 4 — 測試、vet、staticcheck；真實 Gemini 重跑 Tears of Steel 對照（AC 6）

## Dev Notes

- 管線 chunk 迴圈：`subtitle/pipeline.go` `translateTrack`（`harvested` 收集與閘門用法在 `:1040-1065`）；`translateSplitting` 對半重送時要帶同一份。
- 埠：`ChunkTranslator`（`pipeline.go:65`）是未蓋章的埠，加參數要同步 4 個測試替身（`pipeline_test.go`、`pipeline_transient_test.go`、`process_item_test.go` ×2）。
- 實作：`TranslationService.TranslateChunk`（`translation_service.go:674`）改用 `BuildSubtitleTranslatorPromptWithGlossary`。
- 舊路徑：`TranslateWithGlossaryHarvest`（`translation_service.go:417`）。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- `prompts.EstablishedNames`；管線每批帶 `established`（使用者訊息，系統訊息不變）；舊路徑每批對照表＝開跑前＋前面收集到的。
- 測試：管線第 2、3 批收到前面的人名、對照表已有的不重複、系統訊息每批相同；舊路徑第 2 批有第 1 批的人名、對照表優先；`TranslateChunk` 人名在使用者訊息；提示詞沒有人名時逐字不變。反向驗證：拿掉帶入 → 2 個測試紅。go test／vet／staticcheck 全過。
- **還沒做**：Task 4 的真實 Gemini 重跑 Tears of Steel（用量上限，下次補）。

### File List
