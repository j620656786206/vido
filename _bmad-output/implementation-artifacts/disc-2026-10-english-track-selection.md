# Disc: 片內有好幾條英文字幕時，挑「一般版」當主字幕，並把「強制字幕」（畫面上的字）合併進去

Status: review

**Source:** Alexyu 2026-10-06 party mode（See S01E02 實測後的討論，清單第 6 項）。Alexyu 問：「要拿影片裡的哪個英文字幕？一個影片可能有很多英文字幕。」實測紀錄：`eval-see-s01e02-asr-vs-official.md`「討論會」一節。

## Story

身為 Vido 的使用者，
我希望片子裡有好幾條英文字幕時，Vido 拿「完整的一般版」來翻，並且把「強制字幕」裡畫面上的字一起翻進去，
這樣翻出來的中文字幕不會缺了畫面上的字，也不會因為選到聽障版而多出一堆不必要的東西。

## 查到的事（2026-10-06，Bob 逐一開檔確認）

- **挑字幕的流程：** 批次／自動生成走 `process_item.go:176` → `Router.SelectAndRoute`（`router.go:90-150`）：先 `SelectCandidates`（`router.go:101`），再一次抽出所有候選，最後用 `pickBestCandidate`（`router.go:116`、`:196-245`）挑一條。
- **候選怎麼篩：** `SelectCandidates`（`extractor.go:141-161`）只收片內、文字格式、標成中文或 eng/en 的軌；**有中文軌就只看中文**，英文軌根本不會被抽出來（`:157-159`）。
- **怎麼挑：** `betterCandidate`（`router.go:276-284`）＝ ① 刪掉聽障標記後剩幾句，多的贏；② 一樣多就比內容（繁體 > 簡體 > 英文）；③ 再一樣就挑編號小的。**完全沒看軌道是「聽障版」還是「強制字幕」。**
  - 註解（`router.go:185-188`）的假設是「聽障版刪完標記後，句數會跟一般版差不多」。這只是假設：聽障版常把一句拆成好幾句，刪完標記後可能**比一般版多**，結果被選中。
- **強制字幕一定被丟掉：** 強制字幕只有幾句，在 ① 一定輸。這集的強制字幕（第 10 軌，整集 1 句 *We are not alone*＝「有外人在此」）正好就是畫面上的字，片內一般版英文字幕裡沒有這句（eval 文件「3. 畫面上的字沒有翻」）。
- **程式根本沒讀軌道標籤：** `SubtitleTrack`（`ffprobe_service.go:37-49`）只有 Language、Format、External、Title、StreamIndex；`ffprobeStream`（`:157-167`）也沒有 `disposition`。全 repo grep `disposition|hearing_impaired|HearingImpaired`：Go 程式只有三個 HTTP `Content-Disposition` header，**沒有任何地方讀 ffprobe 的 disposition**。Title 有存（`:218`），但沒有被拿來分類。
- **ffprobe 真的會給這些標籤（Rule 28 實測）：** ffprobe 8.1.2 對 mkv 跑 `-print_format json -show_streams`（Vido 用的就是這組參數，`ffprobe_service.go:118-120`），每條字幕的 `disposition` 是一個物件，值是 0／1 的整數，例如 `"disposition": {"default": 1, "forced": 1, "hearing_impaired": 0, …}`（共 19 個 key，包含 `forced`、`hearing_impaired`、`captions`）。Title 在 `tags.title`。
- **片長拿得到：** `MediaTechInfo.DurationSeconds`（`ffprobe_service.go:28-33`，`:228-230` 解析；0＝不知道），router 拿到的 `info` 就是它。
- **句子的身分是 Index：** 翻譯用 `Index` 對回每一句，不用位置（`pipeline.go:872-875`）；聽障過濾保留原本的 Index（`sdh_filter.go:40`，P7）。輸出時 `SerializeSRT`（`srt_parser.go:94-110`）照**陣列順序**寫，編號寫 `b.Index`。→ 合併進來的句子 **Index 不能跟主字幕撞號**，陣列要照開始時間排好。
- **其他用到 `SelectCandidates` 的地方：** `predict_route.go:83` 只看 `len(SelectCandidates(tracks)) > 0`，本單不改這個函式的結果，所以報價預測不受影響。
- **前端：** `apps/web/src/types/library.ts:55` 的 `subtitleTracks?: string` 是一段 JSON 字串。`SubtitleTrack` 加欄位（加 `omitempty`）不會影響前端。
- ⚠️ **未查證（dev 開工時先確認，NAS 連得到再查）：** See S01E02 那個 KONTRAST 檔，第 8／9／10 軌有沒有設 `disposition`、Title 寫什麼。所以分類一定要**兩條路都看**：標籤和軌道名稱。

## 設計

**1. 讀標籤（ffprobe → `SubtitleTrack`）**

- `ffprobeStream` 加上 `Disposition map[string]int \`json:"disposition,omitempty"\``。
- `SubtitleTrack` 加上 `Forced bool \`json:"forced,omitempty"\``、`HearingImpaired bool \`json:"hearing_impaired,omitempty"\``，在 `parseFfprobeJSON` 的字幕分支填入。
- 外掛字幕檔（`DetectExternalSubtitles`）不動，router 本來就不看它們。

**2. 把每條候選軌分成三類**（新函式，放在 `subtitle` 套件）

| 類別 | 判斷（任一成立） |
|---|---|
| 強制 | `Forced == true`；或 Title 含 `forced`（不分大小寫）、`強制` |
| 聽障版 | `HearingImpaired == true`；或 Title 含 `sdh`、`cc`（整個字）、`hearing impaired`、`聽障` |
| 一般版 | 以上都不是 |

軌道同時符合強制和聽障時，算強制。

**3. 挑主字幕**（改 `pickBestCandidate`／`betterCandidate`）

- **能當主字幕的條件：**
  - 不是強制軌；
  - 刪掉聽障標記後的句數，至少是所有候選中最多那條的 **50%**；
  - 片長已知時，最後一句的結束時間至少到片長的 **50%**。

  第二條是為了擋「沒標成強制、其實只有幾句」的軌。第三條是為了擋「只做了前半段」的軌。
- **在能當主字幕的軌裡，依序比：**
  1. 一般版 > 聽障版
  2. 句數多的贏（原本的規則）
  3. 繁體 > 簡體 > 英文（原本的規則）
  4. 編號小的贏（原本的規則）
- **沒有任何一條符合條件時：** 退回原本的挑法（句數 → 內容 → 編號），並記一行 Warn log，寫明為什麼沒有軌符合。行為**不能比現在差**。

**4. 合併強制字幕**（只在英文那一層、而且結果要走翻譯的時候）

- 條件：路線是 `RouteTranslate`，而且候選裡有分類成「強制」的英文軌。
- 強制軌一樣先抽出、做聽障過濾。抽取本來就是一次抽出所有候選（`router.go:106-111`），所以不會多跑 ffmpeg。
- **去重：** 強制字幕的某一句，如果跟主字幕的某一句時間有重疊，而且文字一樣（去掉空白、標點、大小寫後比對），就丟掉。時間重疊但文字不同的照樣保留，播放器會把兩句疊在一起顯示。
- **編號：** 合併進來的句子，Index 從「主字幕最大 Index ＋ 1」開始往上編，不能撞號（P7：主字幕原本的 Index 一個都不動）。
- **排序：** 合併後的陣列照開始時間排；開始時間一樣時，主字幕排前面。
- log：記錄合併了幾句、因為重複丟掉幾句。
- **中文那一層不合併**（見「不要做的事」）。

## Acceptance Criteria

1. **讀得到標籤：** 用真的 ffprobe JSON 解析（Rule 28：用本機 ffmpeg 做一個有三條字幕軌的小 mkv，三條分別是一般、`hearing_impaired`、`forced`，把 `ffprobe -print_format json -show_streams` 的輸出存成 `testdata` fixture），`SubtitleTrack` 的 `Forced`／`HearingImpaired` 要正確。沒有 `disposition` 的舊 JSON 照常解析，兩個值都是 false。
2. **分類：** 表格測試涵蓋：只有標籤、只有 Title（`English [SDH]`、`English (Forced)`、`English CC`、`繁體中文 (強制)`）、兩者都有、兩者都沒有。Title 裡剛好有 `cc` 字母但不是獨立的字（例如 `Accent`），**不能**算聽障版。
3. **一般版贏過句數比較多的聽障版：** 一般版 10 句、聽障版刪完標記後 12 句 → 選一般版。
4. **強制軌永遠不當主字幕：** 強制軌句數最多也不行。只有強制軌時，退回原本的挑法並記 Warn。
5. **50% 門檻：**
   - 一般版只有 3 句、聽障版 20 句 → 一般版沒資格，選聽障版。
   - 片長 3,000 秒，一般版最後一句停在 900 秒、聽障版涵蓋全片 → 選聽障版。
   - 片長不知道（0）時，跳過時間這一條。
6. **合併強制字幕（英文翻譯路線）：**
   - 主字幕 4 句＋強制 2 句（1 句和主字幕重複）→ 結果 5 句，照時間排序，Index 都不重複，主字幕原本的 Index 不變。
   - 路線是 `RouteDeliverDirect`／`RouteConvertThenDeliver`（中文那一層）時，**不合併**。
7. **原本的行為不變：** `router_test.go` 現有的測試全部照常通過，不能改測試的期望值，只能新增。特別是 `TestSelectAndRoute_PicksHighestPostFilterCueCount`（`:371`）、`TraditionalWinsTheTieAgainstSimplified`（`:183`）、`CueCountStillBeatsVariant`（`:209`）、`TieBreaksOnLowestStreamIndex`（`:435`）、`BlocksAreFilteredAndKeepOriginalNumbering`（`:470`）。
   - 如果真的有舊測試因為新規則而必須改期望值，要在 Completion Notes 寫清楚是哪一個、為什麼，並且 CR 要特別看。
8. **See S01E02 情境測試：** 用假的 prober／extractor 模擬三條英文軌：
   - 一般版：568 句，沒有標籤；
   - 聽障版：Title `English [SDH]`，句數比一般版多；
   - 強制：`forced`，1 句。

   結果：選一般版，合併 1 句，總共 569 句。
9. **檢查全綠：**
   - `apps/api` 跑 `go build ./... && go vet ./... && go test ./...`
   - `~/go/bin/staticcheck ./...`
   - `pnpm run lint:all`
10. **上線後在 NAS 實測（合併後補紀錄，不擋 PR）：** 對 See S01E02 跑一次批次的路線判斷（或看 log 的 `subtitle route decided`），記錄選到哪一條、合併了幾句。同時補上「查到的事」裡 ⚠️ 那一條。

## Tasks / Subtasks

- [x] T1 ffprobe 讀 `disposition` → `SubtitleTrack.Forced/HearingImpaired`，加上真實 fixture（AC #1）
- [x] T2 分類函式＋挑主字幕的新規則＋退回原挑法（AC #2–#5、#7）
- [x] T3 合併強制字幕：去重、編號、排序、log（AC #6、#8）
- [x] T4 檢查（AC #9）；sprint-status 更新

後端 4 項、前端 0 項 → 不拆單。

## Dev Notes

### 不要做的事

- **不要改 `SelectCandidates` 的「有中文就只看中文」**（`extractor.go:157-159`）。那是 2026-07-31 實測決定的（157 片裡 135 片同時有中文和英文軌）。
- **中文那一層不合併強制字幕。** 中文強制軌可能是簡體、主字幕是繁體，合併會變成混在一起，要另外處理轉換。真的遇到再開單。
- **不要動單集「生成字幕」那條。** 單集按鈕現在完全不看片內字幕，那是 `disc-2026-10-single-generate-ignores-embedded-english` 的事。那張單之後會重用這次改好的 router。
- **不要改 `RouteKind`／`RouteDecision`／`ExtractedTrack` 的欄位**（`router.go:13-16` 標了 `[@contract-v1]`：「Changing kinds/fields = Rule 20 bump + stale-mark」）。合併後的句子還是放在 `Track.Blocks` 裡，型別和意思都不變（都是「要交付或翻譯的句子」）。真的需要加欄位，就照 Rule 20 升版本。
- **不要在挑選時重跑 ffmpeg。** 強制軌本來就在候選清單裡，抽取時已經一起抽出來了。

### 已知陷阱

- **Index 撞號：** 主字幕的 Index 有缺號（聽障過濾留下的，P7）。合併進來的句子要從「最大 Index ＋ 1」開始編，**不是**「句數 ＋ 1」。
- **`cc` 誤判：** 要比對整個字（例如用 regexp `\bcc\b`，不分大小寫），不能只檢查有沒有出現 `cc` 這兩個字母。
- **片長單位：** `DurationSeconds` 是秒（float），`SubtitleBlock.End` 是 `"00:12:34,567"` 字串。`subtitle` 套件裡**沒有**現成的轉換函式（grep `internal/subtitle/*.go` 沒有）；有一個在 `mine/align.go:100` 的 `srtToMS`，但沒匯出，而且 `mine` 匯入了 `subtitle`（`mine/align.go:12`），`subtitle` 不能反過來匯入 `mine`（會循環）。做法：把它搬到 `subtitle` 套件並匯出，`mine` 改呼叫它，不要寫第二份。
- **退回原挑法時，結果要跟改動前一模一樣。** 用 AC #7 的舊測試來保護。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

### References

- `_bmad-output/implementation-artifacts/eval-see-s01e02-asr-vs-official.md`「討論會」
- `project-context.md` Rule 13（錯誤要有脈絡）、Rule 20（contract 版本）、Rule 28（真實形狀測試）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

- `go test ./...`（apps/api）全綠；`go vet ./...` 乾淨；`~/go/bin/staticcheck ./...`（2026.1，用 go1.26 重新編譯後）乾淨；`pnpm run lint:all` 0 errors（172 個既有 warning，本單沒有新增）；`pnpm nx test web` 全綠；`test:cleanup` 沒有殘留程序。
- 本機的 staticcheck 原本是用 go1.25 編的，對 go1.26 標準庫會回「file requires newer Go version」並報出兩個假的 U1000。用 `go install honnef.co/go/tools/cmd/staticcheck@2026.1` 重編後就沒有了。這是本機工具的問題，跟 repo 無關。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **T1：** `ffprobeStream` 讀 `disposition`（0／1 整數）；`SubtitleTrack` 新增 `Forced`、`HearingImpaired`（`omitempty`，沒有標籤時存進資料庫的 JSON 一個 byte 都不變）。Rule 28：`testdata/ffprobe-subtitle-dispositions.json` 是 ffprobe 8.1.2 對真的 mkv（一般／`hearing_impaired`／`forced` 三條英文軌）用 Vido 同一組參數跑出來的輸出，只經過 prettier 排版。
- **T2：** 新檔 `track_choice.go`：`classifyTrack`（標籤優先，再看 Title；`forced`／`sdh`／`cc`／`hearing impaired` 都要整個字才算，`強制`、`聽障` 用 contains）。`router.go` 把 `pickBestCandidate` 拆成 `parseCandidates`（解析＋聽障過濾＋偵測內容，順序和 log 跟原本一樣）與 `chooseMain`（強制軌不能當主字幕、句數 ≥ 最多那條的 50%、片長已知時要過半；合格的軌裡一般版 > 聽障版，再照原本的 `betterCandidate`；沒有合格的就退回原本的挑法並記 Warn）。
- **T3：** `mergeForcedCues`：只在 `RouteTranslate` 而且主字幕是「合格」選出來的時候做；只合併內容偵測為英文的強制軌；時間重疊且文字一樣（只比字母和數字、不分大小寫）就丟掉；新編號從主字幕最大 Index ＋1 開始；用穩定排序照開始時間排，同時間主字幕在前。合併或丟掉時記 Info log（`forced_merged`、`forced_duplicates_dropped`）。
- **共用：** `srtToMS` 從 `mine/align.go` 搬到 `subtitle.SRTTimestampMS`（`srt_parser.go`），`mine` 改呼叫它，沒有留兩份。
- **AC #7：** `router_test.go` 一個字都沒改，全部照常通過。
- **AC #8：** `TestSelectAndRoute_SeeS01E02Shape`。測試資料多加了一句片尾台詞，讓兩條完整軌都涵蓋全片，所以總數是 568＋1＋1 強制＝570。測試裡有寫明為什麼。
- **AC #10（NAS 實測）：** 還沒做。合併部署後，對 See S01E02 看 log 的 `subtitle route decided`（`track_kind`、`qualified_main`）和 `forced subtitle cues merged…`，再補「查到的事」裡 ⚠️ 那一條。
- 🔗 AC Drift: FOUND — sub-1-4 AC #5「Multi-candidate selection」：原本是「刪掉聽障標記後句數最多的贏，一樣多挑編號小的」，現在改成「先看類型與 50% 門檻，再用原本的順序」。沒有合格的軌時，結果跟原本完全一樣。sub-1-4 AC #8 範圍外清單寫的「❌ No forced/disposition flag parsing (… revisit if the pilot shows mis-selection)」，See S01E02 就是那個「pilot 選錯」，本單就是那次 revisit。
- 📎 Contract Stamps: FOUND — sub-1-4 AC #1 `[@contract-v1→v2]`（`router.go:13-20` 已改成 v2，`pipeline.go:187` 的引用同步更新）。型別都沒變，變的是語意：`RouteTranslate` 時 `ExtractedTrack.Blocks` 可能包含強制軌的句子，所以不再保證只來自 `Path` 那一條。下游 ack：sub-1-5a、sub-1-5b（都是 done），sub-1-6 done。沒有還沒做完的下游單，所以不用 stale-mark。下游只靠 Index 不重複和原本 Index 不變，這兩點都有測試保護。全 repo 也沒有任何地方讀 `Track.Path`。
- **對抗式 CR（另開一個全新的 agent，2026-10-06）：2M／5L，6 項修掉，1 項判定為誤報**
  - **M1** 有些片子把「完整的那條」也標成強制字幕。原本會整條合併，每句出現兩次，翻譯費也加倍 → 強制軌句數 ≥ 主字幕一半就不合併，記 Warn。新增 `FullTrackFlaggedForcedIsNotMerged`。
  - **M2** 兩條內容一樣的強制軌（subrip＋mov_text）會合併兩次 → 去重時也比對已經併進來的句子。新增 `TwoIdenticalForcedTracksMergeOnce`。
  - **L3** 中文那一層，「一般版優先」蓋過了「繁體優先」（繁體聽障版 vs 簡體一般版，會選到簡體）→ 「一般版優先」只在兩條都是英文時用，中文照原本的「繁體優先」。新增 `TraditionalSDHStillBeatsSimplifiedRegular`。
  - **L4** 軌道名稱判斷：`Non-Forced`／`Not Forced` 誤判成強制；`English_SDH`、`[HI]`、簡體的「强制」「听障」漏判 → 都修了，分類表測試加了 8 列。
  - **L5** 主字幕是「退回原挑法」選出來的時候，原本不合併強制字幕 → 改成只要主字幕不是強制軌就合併（spec 本來就沒寫要合格才合併）。新增 `FallbackMainStillMergesForcedTrack`。
  - **L6** 退回原挑法的 Warn 沒寫原因 → 現在每條軌都會寫被刷掉的原因（`stream 2: forced`、`stream 3: 3 of 20 cues`、`ends before half…`）。
  - **L7（誤報）** `process_item.go:765` 那個 `[@contract-v1]` 指的是 sub-1-3 的 `PipelineStage`，不是 `RouteKind`，不用改。
  - 為了配合 M1，兩個舊測試的主字幕加了幾句。原本主字幕只有 2～4 句，強制軌 1～2 句，會被新門檻擋下，不像真實的片子。測試的重點（Index 從最大值往上編）沒變。
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### Discovery Triage

| 發現 | 分道 | 追蹤 |
|---|---|---|
| `gofmt -l` 在 apps/api 列出 60 個沒動到的檔案（註解對齊，例如 `providers/opensub.go`）。CI 只跑 go vet＋staticcheck、沒跑 gofmt，所以一直沒被抓到。本單只格式化自己動到的區塊，`ffprobe_service_test.go` 裡被 gofmt 順手改掉的一行已經還原 | ③ | `preexisting-gofmt-drift`（sprint-status，本單發現時立案） |

### File List

- `apps/api/internal/services/ffprobe_service.go`（改）
- `apps/api/internal/services/ffprobe_service_test.go`（改）
- `apps/api/internal/services/testdata/ffprobe-subtitle-dispositions.json`（新）
- `apps/api/internal/subtitle/track_choice.go`（新）
- `apps/api/internal/subtitle/router.go`（改）
- `apps/api/internal/subtitle/router_track_choice_test.go`（新）
- `apps/api/internal/subtitle/srt_parser.go`（改）
- `apps/api/internal/subtitle/mine/align.go`（改）
- `apps/api/internal/subtitle/pipeline.go`（改，只改註解裡的 contract 版本）
- `_bmad-output/implementation-artifacts/disc-2026-10-english-track-selection.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（改）
- `_bmad-output/implementation-artifacts/sub-1-4-extract-filter-route.md`（AC drift reference — see Completion Notes；本單沒改這個檔）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-06 | Bob create-story。 |
| 2026-10-06 | T1：ffprobe 讀 `disposition`，加上真實 fixture。 |
| 2026-10-06 | T2：`classifyTrack`＋`chooseMain`（強制軌不當主字幕、50% 句數與片長門檻、一般版 > 聽障版、沒合格就退回原挑法）。 |
| 2026-10-06 | T3：`mergeForcedCues`（只在英文翻譯路線；去重、Index 接在最大值後、照時間排）。`srtToMS` 搬成 `subtitle.SRTTimestampMS`。 |
| 2026-10-06 | [@contract-v1→v2] AC #1（sub-1-4）：`RouteTranslate` 時 `ExtractedTrack.Blocks` 可能多出強制軌的句子，Index 從主字幕最大值往上編、照時間排序。下游受影響的是「假設 Blocks 只來自 `Path` 那一條、或 Index 都在原檔裡」的程式——目前沒有（全 repo 沒有地方讀 `Track.Path`），已 ack 的 sub-1-5a、sub-1-5b、sub-1-6 都是 done。 |
| 2026-10-06 | T4：全部檢查通過，狀態改成 review。 |
| 2026-10-06 | CR 修正：M1 不合併「被標成強制的完整軌」、M2 強制軌之間也去重、L3 中文層繁體優先、L4 軌道名稱判斷補齊、L5 退回原挑法時也合併、L6 Warn 寫原因。go test／vet／staticcheck／format 全綠。 |
