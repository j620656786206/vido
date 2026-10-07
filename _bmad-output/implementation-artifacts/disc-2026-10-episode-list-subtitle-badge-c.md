# Story（後端）：片內只寫「中文」的字幕軌，抽一小段出來看字，判出繁體還是簡體

Status: done

**Source:** 傘狀單 `disc-2026-10-episode-list-subtitle-badge`。⚖️ **Alexyu 2026-10-07 選 B**：現在就做，不等正式機數字。問題由 -a 的 CR 浮出：五個結論裡的 `zh`（有中文、分不出繁簡）J11-D 沒有畫；追問原因是 ffprobe 對片內字幕只給語言代碼（中文一律 `chi`）與標題，沒標題就分不出來。
**依賴：** -a（PR #717）的 `episodes.subtitle_tracks` 與背景補齊。-b（前端）等本單合併後再做，`zh` 會只剩圖片字幕。

> ⚖️ SM 裁定（Bob 2026-10-07）：
> 1. **看字的時機放在背景補齊**，不放在打開季清單（紅線：打開一季不跑 ffprobe／ffmpeg）。每集只做一次，結果跟軌道一起存進 `subtitle_tracks`。
> 2. **只抽一小段**：不抽整條軌。先抽第 4～6.5 分鐘（開頭常是片頭、沒對白），沒讀到中文字再抽 0～4 分鐘。4K remux 一次最多讀約 2 GB，1080p 約 100 MB；可接受，因為一集一次。
> 3. **不排在抽軌的單一閘門（`ExtractGate`）後面**：閘門是給整檔 demux 用的（兩個 40 分鐘的抽軌會互相拖死）；這裡是幾秒的小讀取，背景補齊本身又是單執行緒。排進去會讓整輪補齊卡在別人 40 分鐘的抽軌後面。
> 4. **圖片字幕（PGS／VobSub）不處理**：沒有字可看。這些維持 `zh`。
> 5. **電影不在本單**：電影的字幕軌由 enrichment 寫，走的是另一條路；先讓單集的結果出來，電影要不要一樣做另立條目。

## Story

身為 Vido 的使用者，
我希望片子裡那條只標「中文」的字幕軌，Vido 能自己看一眼內容，告訴我它是繁體還是簡體，
這樣季清單（-b）大部分的集數都能直接標「有繁中」或「只有簡中」，不會卡在「分不出來」。

## 查到的事（2026-10-07）

- 分繁簡的工具已經有：`subtitle.Detect`（看字，>70% 繁體獨有字 → zh-Hant、≤30% → zh-Hans、中間 → zh、沒有中文字 → und）。旁邊的字幕檔就是用它。
- 抽軌工具 `subtitle.Extractor.Extract` 抽整條軌、排單一閘門、逾時依檔案大小；不能只抽一段。本單另寫一個小的「抽一段」（`ffmpeg -ss <秒> -i <片> -t <秒> -map 0:<軌> -c:s srt <暫存檔>`），`-ss` 放在 `-i` 前面是用索引跳，很快。
- 標題關鍵字（`models.ChineseTitleScript`）現在認「繁／简／traditional／simplified／hant／hans／zh-tw／zh-cn…」，沒認壓片圈常見的 `CHT`／`CHS`／`Big5`／`GB`。
- Rule 19：`services` 不能 import `subtitle`；在 `services` 定介面、`subtitle` 實作、`main.go` 接線（-a 的 `SidecarTrackReader` 同形）。

## Acceptance Criteria

1. **看字結果存在軌道上**：`services.SubtitleTrack` 新增 `detected_language`（`zh-Hant`／`zh-Hans`，看不出來就不寫）。`models.ChineseSubtitleVerdict` 與 `ChineseSubtitleOfTrack` 讀它：中文軌有 `detected_language` 時以它為準（內容 > 標題 > 代碼）；粵語標題照舊算非中文。
2. **背景補齊順手看字**：-a 的背景工作在 ffprobe 之後，對「片內、文字格式、中文但分不出繁簡」的軌抽一小段看字，結果寫進 `detected_language`，跟軌道一起一次存入。沒有這種軌的集數不多做任何事。
3. **只抽一小段**：先抽 240～390 秒；結果是 `und`（沒讀到中文字）再抽 0～240 秒；還是 `und` 或 `zh`（混合）就不寫。每次 ffmpeg 有逾時（60 秒）。
4. **失敗不擋**：ffmpeg 沒裝 → 不看字，其餘照 -a；單集看字失敗 → 記 log、軌道照存（維持 `zh`）。結果 log 多兩個數字：看了幾條、失敗幾條。
5. **標題關鍵字補齊**：`CHT`／`Big5` → 繁體；`CHS`／`GBK`／`GB2312`／`GB18030` → 簡體（整個詞比對，不是子字串，避免誤認）。
6. **測試**：假的 ffmpeg（注入 runner）寫出繁／簡／空的 srt → 看字結果對；第一段 `und` 會抽第二段；圖片軌不抽；`ChineseSubtitleVerdict` 讀 `detected_language`；標題關鍵字新案例。
7. `go build／vet／test ./...`、`staticcheck`、`lint:all` 全綠。

## Tasks / Subtasks

- [x] T1 `SubtitleTrack.DetectedLanguage`＋`models` 讀它＋標題關鍵字（AC #1、#5）
- [x] T2 `subtitle.ScriptPeeker`（抽一段看字，兩段窗口，注入 runner）（AC #3）
- [x] T3 背景補齊接上看字＋`main.go` 接線（AC #2、#4）
- [x] T4 測試（AC #6、#7）

全是後端，4 項。

## Dev Notes

- 不要動 `Extractor`；不要用 `ExtractGate`（裁定 3）。
- `-ss` 超過片長時 ffmpeg 會輸出空檔 → `und` → 自然落到第二段窗口。
- 「管理字幕」對話框現在是現場 ffprobe，看不到存好的 `detected_language`，仍會顯示 `zh-unknown`——另立條目。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（dev）；對抗式 CR 換模型（Sonnet 5.5——Fable 額度用完）。

### Completion Notes List

- 2026-10-07 Bob（SM）：建單，五條裁定見上。
- 2026-10-07 Amelia（dev）：T1–T4 完成。
  - `SubtitleTrack.detected_language`：`zh-Hant`／`zh-Hans` 才算數；`zh`（混合或字太少）／`und`（沒讀到中文字）也存，代表「看過了、看不出來」，之後不再抽。
  - 判斷順序：標題寫粵語 → 非中文；否則看字結果 → 標題 → 代碼。
  - `subtitle.ScriptPeeker`：`-ss` 在 `-i` 前、只取那條字幕軌、先 240–390 秒再 0–240 秒；兩段的字累加，「只有一種寫法才有的字」滿 8 個才下結論。不走 `ExtractGate`（裁定 3）。
  - 背景補齊：-a 已經讀過的集數，若有「沒標題的 chi 文字軌、還沒看過字」會再處理一次（不用等檔案換過）；看字失敗照存軌道、這個 process 不再重試。
  - 標題關鍵字：`CHT`／`Big5` → 繁、`CHS`／`GB`／`GBK`／`GB2312`／`GB18030` → 簡，整個詞比對。這也會讓電影、「管理字幕」對話框裡這種標題的軌從「分不出繁簡」變成有結論（預期的改善；抽軌路由不讀這份清單，不受影響）。
  - 真的 ffmpeg 實測：做了一個 10 分鐘、只標 `chi` 的繁中軌 mkv → `zh-Hant`；100 秒的短片 → 第一段空、第二段判出 `zh-Hant`。
- 2026-10-07 對抗式 CR（Sonnet 5.5）：0H／2M／5L。
  - **M1／M2（已修）** 一段只有幾句「好。你好嗎」會被存成「混合」，一個「个」就會被判簡體（還會蓋過正確的標題）→ 兩段的字累加，滿 8 個獨有字才下結論；不夠就是 `zh`。補測試。
  - **L3（已補測試）** 雙標題「CHT/CHS」→ 繁體（先比繁體）。
  - **L7（不需修）** 審查說沒驗 ffmpeg 參數——`TestScriptPeeker_FirstWindowDecides` 已逐項斷言參數；另做了真 ffmpeg 實測。
  - **L4／L5／L6（不修，記錄）** 看字一直失敗的檔每次重開機會再試一次（最多 60 秒×2）；一集兩條軌一條失敗時只存一半結果；看字不排抽軌閘門，可能跟整檔抽軌同時讀同一顆硬碟（上限 60 秒）。
  - 修後閘門：`go test ./...` 全綠、`staticcheck` 乾淨、`lint:all` 0 errors、prettier 綠。

### Discovery Triage

- **③（不修）** 「管理字幕」對話框是現場 ffprobe，看不到存好的 `detected_language`，這種軌在對話框仍顯示「分不出繁簡」（Dev Notes 已記）。
- **③（不修）** 電影的片內 `chi` 軌沒有看字（裁定 5）。

### File List

- apps/api/internal/subtitle/script_peeker.go（新）＋ script_peeker_test.go（新）
- apps/api/internal/services/episode_subtitle_tracks_service.go ＋ _test.go、ffprobe_service.go、episode_chinese_subtitle.go
- apps/api/internal/models/chinese_subtitle.go ＋ _test.go
- apps/api/internal/subtitle/sidecar_track_reader_test.go
- apps/api/cmd/api/main.go
