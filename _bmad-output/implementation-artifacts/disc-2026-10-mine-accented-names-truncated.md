# Story：帶重音的人名不會再被切一半；名詞表裡「中文寫在英文欄」的反向詞條會被擋下並修正

Status: done

**Source:** `disc-2026-10-mine-accented-names-truncated`（P2・可修，filed 2026-10-07，力量之戒官方字幕學習實跑）。Alexyu 2026-10-08：「接著做下一張」（上一輪候選清單的第 2 張）。

## Story

身為 Vido 的使用者，
我希望 Vido 從官方字幕學人名時，「Khazad-dûm」學到的是完整的名字，不是「Khazad-d」；名詞表裡也不會出現「中文寫在英文欄」的怪詞條，
這樣翻譯時人名才會一致，詞彙面板也不會有我看不懂的垃圾。

## 查到的事（2026-10-08）

- **切一半**：`subtitle/mine/mine.go:307` `tokenRe = [A-Za-z][A-Za-z'’-]*` 只認 ASCII 字母，「Khazad-dûm」被切成「Khazad-d」＋「m」、「Pharazôn」→「Pharaz」、「Rhûn」→「Rh」。
- **反過來的比對也錯**：`mentionsWord` 用 `isWordByte` 判斷字的邊界，只看 ASCII；「Rh」比對「Rhûn」時，後面那個 byte 是 û 的第一個 byte，不算字母 → 被當成「Rhûn 裡提到 Rh」。
- **反向詞條**：正式機唯讀查 `show_glossary`，`term_src` 含中文的有 2 列，都是 `source=subtitle`、未確認：`馬言者 → The Neigh-sayer`、`風舞市長 → Mayor Winddancer`。來源是翻譯時模型回報的 `===TERMS===` 名單（`translation_service.go` `splitHarvestTrailer`）——模型把「原文=>譯名」寫反了，解析器照單全收。力量之戒那三列（凱勒布瑞博／林頓／蓋拉卓瑞爾）已被清掉，同一種。

## Acceptance Criteria

1. **候選字認得所有字母**：`tokenRe` 用 Unicode 字母（`\p{L}`）；`Khazad-dûm`、`Pharazôn`、`Rhûn`、`Míriel`、`Númenor` 是完整的候選。
2. **整字比對看 Unicode**：`mentionsWord` 的前後邊界用 rune 判斷（字母／數字／底線）；`Rh` 不會比對到 `Rhûn`，`Rhûn` 會比對到 `Rhûn's`。
3. **名單寫反就轉回來**：`===TERMS===` 一行若「原文」有中文、「譯名」沒有中文 → 對調；兩邊都沒有中文 → 丟掉（譯名一定要是中文）；其他照舊。
4. **舊的反向詞條修正**：migration 045，只動 `source=subtitle`、未確認、`term_src` 有中文且 `term_zh` 沒有中文的列：同一個 scope／language 已經有對調後的那個詞 → 刪掉反向那列；沒有 → 對調。log 寫幾列對調、幾列刪除。
5. **測試**：力量之戒三個名字的學習測試（官方字幕形狀）；`mentionsWord` 邊界表格；trailer 對調／丟棄；migration（對調、撞到既有詞時刪、已確認的不動、冪等）。
6. `go build／vet／test ./...`、`staticcheck`、`lint:all`（看「✖ N problems (0 errors」那行）全綠。

## Tasks / Subtasks

- [x] T1 `mine`：`tokenRe`＋`mentionsWord` 改 Unicode（AC #1、#2）
- [x] T2 `splitHarvestTrailer` 方向檢查（AC #3）
- [x] T3 migration 045（AC #4）
- [x] T4 測試＋閘門（AC #5、#6）

全後端，4 項。

## Dev Notes

- `tokenRe` 也用在「小寫出現過就不是人名」的檢查（`mine.go:87`），改了一起受益。
- 中文判斷用 `unicode.Is(unicode.Han, r)`。
- 修完要在正式機重學力量之戒確認詞條完整（部署後）。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（dev）；對抗式 CR 換模型（Sonnet 5.5）。

### Completion Notes List

- 2026-10-08 Bob（SM）：建單；正式機唯讀查到 2 列反向詞條，併入本單。
- 2026-10-08 Amelia（dev）：T1–T4 完成。
  - `tokenRe` 改成拉丁字母＋組合符號（`\p{Latin}[\p{Latin}\p{M}'’-]*`）；`mentionsWord` 的邊界改逐字（rune）判斷，只把拉丁字母／組合符號／數字／底線當成字的一部分。
  - `orientHarvestPair`：「原文」有中文、「譯名」沒有 → 對調；兩邊都沒中文 → 丟掉。
  - migration 045：用正式機資料庫副本實跑——1 列對調（Mayor Winddancer → 風舞市長）、1 列刪除（The Neigh-sayer 已經有正確的那列）。
  - 突變檢查：`tokenRe` 改回 ASCII，兩個新測試會紅。
- 2026-10-08 對抗式 CR（Sonnet 5.5）：2H／3M／2L，修 H1、H2（補測試證明不會撞唯一索引）、M3、M4、M5。
  - **H1** 原本改成「任何字母」會把緊貼的中文黏進英文名（`Gandalf的朋友`）→ 只認拉丁字母。
  - **H2** 兩列反向詞條指向同一個英文詞時會不會撞唯一索引 → 查重是逐列讀交易內的最新資料，第二列會看到第一列已對調而被刪；補測試。
  - **M3** 中文緊接在名字後面時整字比對失效 → 邊界只看拉丁字母。
  - **M4** 俄文、希臘文不再變成候選。
  - **M5** 譯名是空字串的列不對調。
  - L6／L7 記錄不修。
  - 閘門：`go vet`、`go test ./...` 全綠、`staticcheck` 乾淨、`lint:all` 0 errors。

- 2026-10-08 /ship：PR #722。

### File List

- apps/api/internal/subtitle/mine/mine.go ＋ mine_test.go
- apps/api/internal/services/translation_service.go ＋ translation_service_test.go
- apps/api/internal/database/migrations/045_fix_reversed_harvested_glossary_terms.go ＋ _test.go（新）
