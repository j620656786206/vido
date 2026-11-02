# Story disc-2026-11-backfill-subtitle-cue-counts：替舊片補上字幕句數，批次清單的翻譯預估也照句數算

Status: done

**Source:** sprint-status 同名待辦（P2，PR #968 留下）。⚖️ Alexyu 2026-11-02「接著做補句數那張」。前置 `disc-2026-11-translate-estimate-ignores-cue-density`（PR #968）：`SubtitleTrack.CueCount`（ffprobe 標籤 `NUMBER_OF_FRAMES`／`nb_frames`）、`PredictCuesFromTracks`、句數 × 每千句的估價。

## 問題

批次清單（`GenerationCandidateService`）讀的是已經存在資料庫裡的 `subtitle_tracks`。這些是 #968 之前清點的，沒有 `cue_count`，所以批次退回「片長 × 每分鐘」：《天蠍》批次報約 $0.35，仍低於實際 $0.44～0.49。單集對話框會即時 probe，所以已經準了。

## 做法

1. **集數**（`services/episode_subtitle_tracks_service.go` 的背景清點）：仿照 `needsPeek`，新增「需要補句數」的判斷——存的字幕軌裡，有**會被翻譯的英文文字軌**（跟執行時同一套選軌規則 `subtitle.SelectCandidates`／`PredictCuesFromTracks` 會選的那幾條）而 `cue_count` 是 0 → 這一集重新 probe 一次（檔案簽章沒變也要）。
2. **只補一次**：重新 probe 後還是拿不到句數（檔案沒有統計標籤、MP4 也沒有 nb_frames）的軌，存成 `cue_count = -1`，代表「讀過了、沒有」，之後不再重讀。所有用句數的地方把 `<= 0` 都當成「不知道」。（JSON 欄位維持 `omitempty`；-1 會寫出來。）
3. **電影**（`services/movie_subtitle_tracks_backfill_service.go`）：同樣的規則補 `movies.subtitle_tracks`。
4. **片旁英文 .srt**：如果清點時本來就會把片旁字幕寫進 `subtitle_tracks`（external 軌），順便數條數寫進 `cue_count`（上限 8MB、只讀）；如果現在不會寫，就不在本張擴大範圍，寫進 Dev Agent Record。
5. 不另外加排程：用現有清點的時機（開機與掃描後）。一輪最多補多少集照現有清點的節奏（序列、可中斷）。
6. log：每輪結束 Info 一行「補了幾集句數、幾集讀不到標籤」。

## Acceptance Criteria

1. [x] 存的字幕軌缺句數、而且有會被翻譯的英文文字軌 → 簽章沒變也重新 probe；補上的句數寫回 `subtitle_tracks`。
2. [x] 讀不到句數 → 存 `-1`，下一輪不再重讀；估價把 `<= 0` 當不知道（含 #968 的所有呼叫點）。
3. [x] 電影同規則。
4. [x] 沒有英文文字軌的集數（純中文軌、只有圖片字幕、`[]`）不會被重讀。
5. [x] 測試覆蓋 1～4；`go build／vet／test`、`staticcheck`、`gofmt` 乾淨。
6. [x] 驗證：用正式機資料庫副本跑一輪清點（本機讀不到檔案的會被跳過，所以這條在 Dev Agent Record 說明怎麼驗證即可），或以單元測試模擬《天蠍》E2：存的英文軌沒有句數 → 清點後 1,609 → 批次估價約 US$0.72。

## Dev Agent Record

### 改了什麼

- **選軌規則**（`subtitle/predict_route.go`）：新增 `NeedsCueCount(tracks)`——`SelectCandidates` 選出的（會被翻譯的）軌是**英文**、而且其中有軌的 `CueCount == 0`（#968 之前存的、從沒讀過）才是 true。中文文字軌會被選中（直接交付、不翻譯）、只有圖片字幕、`und`、`[]`、只有片旁字幕都是 false。`CueCountRule{}` 把 `NeedsCueCount` 和 `PredictCuesFromTracks` 包成 `services.CueCountRule` 介面注入（services 不能 import subtitle，Rule 19；照 `TrackScriptPeeker` 的做法），並加編譯期檢查。
- **-1＝讀過、沒有**（`services/ffprobe_service.go`）：ffprobe 解析字幕軌時，標籤和 `nb_frames` 都沒有句數就存 `CueCountUnstated = -1`。`NeedsCueCount` 只看 0，所以任何**從現在起** probe 過的軌都不會再被重讀。
- **集數**（`services/episode_subtitle_tracks_service.go`）：仿 `needsPeek` 加 `needsCues`；字幕軌存著、簽章沒變、但 `needsCues` → 一樣排進這輪重新 probe（`forCues`）。重 probe 失敗沿用既有的「同一個檔案這個行程不再試」。寫回用既有的 `UpdateSubtitleTracks`（簽章沒變，所以燒字幕的兩欄會保留）。結果多 `CuesFilled`／`CuesUnstated`，每輪結束那行 Info log 多 `cue_counts_filled`、`cue_counts_unstated`。
- **電影**（`services/movie_subtitle_tracks_backfill_service.go`）：原本的「補 NULL」之後多一段補句數。repo 新增（選用介面 `MovieCueCountRepo`，舊的 fake 不受影響）：
  - `FindMoviesMissingCueCounts`：有檔案、沒被移除、字幕軌非空且整串 JSON 沒有 `"cue_count"` 的電影，依 id 分頁。現在起 probe 的每一條內嵌軌都會帶 `cue_count`（句數或 -1），所以重讀過一次就離開清單。
  - Go 端再用同一個 `NeedsCueCount` 過濾，重 probe、照原本的方式合併片旁字幕。
  - 寫回用新的 `ReplaceSubtitleTracks`（compare-and-set：資料庫裡還是當初讀到的那串才寫），掃描中途重讀過就不覆蓋。
  - log 一樣多兩個數字。
- **片旁英文 .srt**：集數的背景清點本來就會把片旁字幕寫進 `subtitle_tracks`，所以 `subtitle.SidecarTrackReader` 遇到英文 .srt 時順便數條數寫進 `cue_count`。上限 8MB、只讀；讀不到、太大、解析不了或空的都存 -1。檔案沒變（檔名＋簽章相同）就沿用存著的數字、不重開檔。中文片旁字幕不數。
- **接線**（`cmd/api/main.go`）：集數清點與電影補讀都 `SetCueCountRule(subtitle.CueCountRule{})`。沒有新排程，跟著現有的開機後與掃描後那兩輪，序列執行、可中斷。

### 檢查：所有讀 `SubtitleTrack.CueCount` 的地方都把 `<= 0` 當不知道

`grep CueCount` 之後，只有 `PredictCuesFromTracks` 會讀它（`max(0, …)`，-1 → 0）；下游的 `extractTranslationUSD`（`cues > 0`）、路線快取的編碼（`cues > 0`；解碼拒收負數）、估價的 Debug log（`cues > 0`）都只拿這個 0 或正數。其他名叫 `CueCount` 的欄位是 `SubtitleRun` 的，不相關。有測試鎖住 `PredictCuesFromTracks(-1) == 0`。

### 跟規格的差異與要知道的

- **-1 寫在 ffprobe 解析時，不只寫在補讀這條路**：規格寫「重新 probe 後還是拿不到句數的軌存成 -1」。我放在 `parseFfprobeJSON`，所以掃描時的豐富化、背景清點、補讀、估價的即時 probe 都一致，只有一個地方決定。代價：從現在起被 probe 的字幕軌，沒有句數的（包含圖片字幕、中文軌）JSON 會多一個 `"cue_count":-1`。舊資料等到下次 probe 才會變。
- **季清單打開時也會順便補片旁字幕的條數**：季清單本來會即時讀片旁字幕，跟存著的不一樣就寫回。舊資料的英文 .srt 沒有條數，第一次打開那一季時會完整讀一次那個檔、寫回，之後就沿用。
- **目前沒有地方用到片旁字幕的條數**：批次清單沒有片旁字幕這條路線；單集對話框是即時讀檔。這次照規格存起來給之後用。電影的片旁字幕（`DetectExternalSubtitles`）只看檔名、不讀內容，沒有加條數：每次掃電影都要完整讀一次英文 .srt，卻沒有人用，不划算。
- **電影 SQL 清單會一直列出某些電影**：只有中文或圖片字幕軌的舊電影，字幕軌裡永遠沒有 `"cue_count"`（因為不會重讀），每輪都會被列出來、在 Go 端跳過。成本只是解析一小段 JSON，片庫幾百部沒差，記一下。

### 驗證（AC #6）

- 用單元測試模擬《天蠍》E2，用的是**真的**選軌規則和真的路線轉接器（`cmd/api/cue_count_backfill_test.go`）：存的英文軌沒有句數 → 批次估價 $0.35（41 分鐘 × 0.0085）→ 清點同一個檔重新 probe 拿到 1,609 → 批次估價 **$0.72** → 下一輪不再 probe。
- 沒有用正式機資料庫副本跑一輪：本機讀不到媒體檔，清點會全部算「檔案不在」跳過，沒有意義。上線後這樣確認：
  1. 看開機約 1 分鐘後那輪的 log `episode subtitle tracks: pass finished`，有 `cue_counts_filled`、`cue_counts_unstated`；電影看 `movie subtitle tracks backfill: pass finished`。
  2. 唯讀查 `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db "SELECT count(*) FROM episodes WHERE subtitle_tracks LIKE '%\"cue_count\":%'"`，數字應該在第一輪後跳上來；再看一次《天蠍》S01E02 的 `subtitle_tracks` 是否有 `"cue_count":1609`。
  3. 打開批次生成的同意清單，《天蠍》每集應該是約 $0.7，不再是 $0.35。

### 測試證據

- `services/cue_count_backfill_test.go`（新）：集數補句數只補一次（拿到 1,609、讀不到存 -1、下一輪全部跳過、每個檔只 probe 一次）；七種不會重讀的情況（中文文字軌、只有圖片字幕、`[]`、`und`、只有片旁英文字幕、已經有句數、已經是 -1）；沒注入規則就不重讀；重讀失敗這個行程不再試。電影：同規則補、中文／圖片字幕不重讀、compare-and-set 不覆蓋較新的讀取、repo 沒有新方法時行為不變。
- `subtitle/predict_route_test.go`：`NeedsCueCount` 九種情況、-1 在估價裡是 0。`subtitle/sidecar_track_reader_test.go`：英文 .srt 數條數、中文不數、檔案沒變沿用、空的或不是字幕存 -1。
- `repository/movie_cue_counts_test.go`（新，真的 migration）：只列出舊的（沒有 `cue_count`、非空、有檔案、沒被移除）；compare-and-set 遇到舊值不寫；寫過之後離開清單。
- `cmd/api/cue_count_backfill_test.go`（新）：上面的《天蠍》端到端。
- `services/translate_estimate_cues_test.go`：#968 的 ffprobe 測試改成「沒有句數＝-1」。
- `cd apps/api && go build ./... && go vet ./... && go test ./...`：44 個套件全 ok；`~/go/bin/staticcheck ./...` 無輸出、exit 0；改到的檔案 `gofmt -l` 乾淨。前端沒有改。

## Change Log

- 2026-11-02：建立。
- 2026-11-02：Amelia 實作完成（集數與電影補句數、-1＝讀過沒有、片旁英文 .srt 條數），送 review。
