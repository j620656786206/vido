# Story disc-2026-11-burned-in-subs-b1-detect：後端——偵測一季有沒有燒字幕、存起來、算進「有沒有中文」

Status: done

**Source:** 傘狀 `disc-2026-11-burned-in-subs`（⚖️ Alexyu 2026-11-01：1A 自動、2A 燒簡體也算有中文）。-a 考試過關（`disc-2026-11-burned-in-subs-a-exam.md`）。-b 拆兩張：本張 -b1（偵測＋存放＋判斷）、-b2（掃描後自動跑、現有片庫總檢查、每月上限、花費顯示）。-c 畫面另張。

## 做法（照 -a 過關的那套）

1. **抽畫面**：一集在片長 15／30／45／60／75／90% 各截一張（ffmpeg，`-nostdin`，寬 480），只留畫面下方 32%，6 張由上往下直排成一張 PNG。截不到（檔案壞、4K 杜比視界解碼錯）或截到少於 3 張 → 這季「不判斷」，什麼都不寫。
2. **問模型**：Claude（設定裡的 Claude 金鑰，模型用預設翻譯模型 `claude-sonnet-5`），**關思考**（`ai.WithThinkingOff`），一張圖＋-a 的提示詞，回 `{"burned_chinese": bool, "lines": [..]}`。解析不了 → 不判斷。
3. **判簡繁**：不交給模型。拿 `lines` 用 OpenCC 數 s2t／t2s 會改的字：簡多 → `zh-Hans`，繁多 → `zh-Hant`，一樣多（含都沒有）→ `zh`（有中文、看不出簡繁）。重用 `zhtw` 的轉換器，補一個三態的判斷函式。
4. **存放**：新 migration（049）在 `episodes` 加 `burned_in_subtitle TEXT NULL`（`zh-Hans`／`zh-Hant`／`zh`／`none`；NULL＝還沒檢查）和 `burned_in_checked_at`。**不寫進 `subtitle_tracks`**（那欄換檔會被重掃蓋掉）。
5. **一季一次**：`CheckSeason(seriesID, season)` 挑該季第一集「沒有字幕軌（`subtitle_tracks = '[]'`）」的集數檢查，結果寫到該季**所有** `subtitle_tracks = '[]'` 的集數。
6. **算進判斷**：`models.ChineseSubtitleVerdict` 多一個 `burned` 輸入：`zh-Hant`→`zh_hant`、`zh-Hans`→`zh_hans`、`zh`→`zh`（跟現有「繁勝簡勝看不出」的順序一起比）；`none` 視同一條確定不是中文的來源。SQL 函式 `vido_chinese_subtitle` 改成 4 個參數，所有呼叫點（series／movie／library／episode 的 verdict）一起改。電影沒有這欄就傳 ""。**五值契約不加新值。**
7. **花費**：每次問模型寫一筆 `ai_side_spend`，新 kind `burned_in_check`，記實際 token 費用。
8. **手動觸發（給 -b2／測試用）**：`POST /api/v1/subtitles/burned-in/check`，body `{"series_id": "...", "season": 1}`，回這季的結果 `{"result": "zh-Hans"|"zh-Hant"|"zh"|"none"|null, "episodes_updated": N, "cost_usd": x}`；null＝不判斷（附 reason）。沒設 Claude 金鑰 → 400 白話錯誤。

## Acceptance Criteria

1. [x] 抽畫面＋直排成一張圖（純 Go 影像處理，不加 cgo）；截到少於 3 張就不判斷。
2. [x] 問模型關思考、`max_tokens` 足夠（≥400）；解析不了就不判斷，不寫 `none`。
3. [x] 簡繁由 OpenCC 判斷，三態；測試含《白髮》那種繁體混一兩個簡體字（静）的句子判成繁。
4. [x] migration 049：兩個新欄位，NULL 預設；舊資料不動。
5. [x] `CheckSeason` 寫整季、只寫 `subtitle_tracks = '[]'` 的集數；回寫的值能讓集數與影集的 `chinese_subtitle` 變成 zh_hans／zh_hant／zh。
6. [x] `ChineseSubtitleVerdict` 與 SQL 函式 4 參數，所有呼叫點更新；既有 verdict 測試全過；新測試覆蓋四種 burned 值與「有字幕軌又有 burned」的組合。
7. [x] 每次呼叫記 `ai_side_spend`（kind `burned_in_check`）。
8. [x] 手動觸發 API＋handler 測試（成功、不判斷、沒金鑰、參數錯）。
9. [x] 不改任何前端；`go test ./...`、`go vet`、`staticcheck`、`gofmt` 乾淨。

## Dev Agent Record

### 改了什麼

- **抽畫面＋拼圖**（`services/burned_in_frames.go`）：`FFmpegFrameGrabber` 用 ffprobe 讀片長（集數有存 `duration_seconds` 就不跑）、ffmpeg `-nostdin` 在 15／30／45／60／75／90% 各截一張寬 480 的 JPEG（直接從 stdout 拿，不寫暫存檔）；`stackFrameStrips` 用純 Go（image/jpeg、image/png、image/draw）留每張下方 32%、由上往下拼成一張 PNG。解不開的畫面跳過，剩不到 3 張就不判斷。
- **問模型**（`ai/claude.go`、`ai/provider.go`）：新增 `ai.ImageCompleter`／`CompleteImageWithUsage`，一個 user turn＝圖片區塊＋文字；照 `CompleteTextWithUsage` 的方式吃 `ai.WithThinkingOff`。回空白／被拒時一樣帶著 usage 回 `ErrAIInvalidResponse`（跟文字版共用 `completionFromMessage`）。`ClaudeProviderHolder` 加轉接，金鑰與每次指定模型的規則和翻譯一樣。
- **判簡繁**（`zhtw/script.go`）：`ClassifyScript` 三態，數 s2tw 和 t2s 各會改幾個字，多的那邊不是答案，一樣多（含都不改）→ `zh`。用真的 OpenCC 驗過「我們師門向來清靜，此處很静」：s2tw 改 1 字、t2s 改 7 字 → 繁。
- **存放**（migration 049）：`episodes` 加 `burned_in_subtitle`、`burned_in_checked_at`（都可 NULL、無預設，舊資料不動）；`ai_side_spend` 的 kind CHECK 加 `burned_in_check`（SQLite 不能改 CHECK，所以重建表、保留所有列和 created_at 索引；Up 重跑是 no-op，Down 會刪掉 burned_in_check 的列再還原成 047 的樣子）。`Episode` model、`episodeSelectColumns`、`scanEpisode` 同步加欄位（Rule 15）；新增 `EpisodeRepository.SetSeasonBurnedInSubtitle`，只寫該季 `subtitle_tracks = '[]'` 的集數，只收四個合法值。
- **算進判斷**：`ChineseSubtitleVerdict`／`HasTraditionalChineseSubtitle` 多第四個參數 `burned`；SQL `vido_chinese_subtitle` 改 4 參數。呼叫點全部更新：影集彙總（`e.burned_in_subtitle`）、季清單的集數判斷（`computeEpisodeSubtitle`）、生成佇列的「已有繁中」排除與計數（集數讀 burned、電影傳 NULL）、電影列表／篩選／排序、首頁繁中計數、影集自身那列（都傳 NULL/""）。五值契約沒加新值。
- **一季一次**（`services/burned_in_subtitle_service.go`）：`CheckSeason` 先確認有 Claude 金鑰（沒有就什麼都不截），挑該季第一集「有檔案且 `subtitle_tracks = '[]'`」的集數，截圖→問 `claude-sonnet-5`（關思考、max_tokens 512、提示詞逐字照 `run.py`）→解析 JSON（跟 run.py 一樣取第一個 `{` 到最後一個 `}`，`burned_chinese` 缺了算解析不了）→OpenCC 判簡繁→寫整季。
- **花費**：每次呼叫在自己的不設上限 Budget 底下跑（`ai_service.go` meteredParse 的做法），有花錢就寫一筆 `ai_side_spend`（kind `burned_in_check`、provider claude、model claude-sonnet-5、實際 token 與費用）；回覆沒用也照記，因為已經付錢了。
- **手動觸發**：`POST /api/v1/subtitles/burned-in/check`，main.go 已接線（照 tracks sweep 的寫法對 `repos.Episodes` 做型別斷言）。

### 決定

- **模型釘死 `claude-sonnet-5`**，不跟著 `CLAUDE_MODEL`／翻譯預設走：考試過關的是它，Haiku 會把《白髮》的繁體抄成簡體。之後要換模型應該重考。
- **burned 為 `none` 不會自己把「沒讀過字幕軌」變成「沒有中文」**：規格說「視同一條確定不是中文的來源」，我照「多一條非中文軌」處理——字幕軌讀得到時它就是非中文的一條；完全沒有字幕軌資訊時不讓它單獨下「none」，因為畫面上沒燒字不代表檔案裡沒有軌。實務上 burned 只寫在 `subtitle_tracks = '[]'` 的集數，兩種讀法結果一樣，只有檔案被移走（tracks 變 NULL）時才有差。
- **「不判斷」的四種原因**（`reason`）：`no_episode_to_check`、`frames_unavailable`、`reply_unreadable`（含模型回空白／被拒）、`script_check_failed`（OpenCC 跑不起來）。都回 200、`result: null`、什麼都不寫。模型呼叫本身失敗（網路、5xx、金鑰被拒）→ 502；沒金鑰 → 400 `AI_NOT_CONFIGURED`。
- 回應多帶 `reason`（規格有提）和 `checked_episode_ids`（給 -b2／除錯用，看得出是哪幾集代表整季；一定有這欄，沒檢查任何一集時是空陣列）。
- **「none」要第二意見**（協調者 2026-11-02 本機真跑後要求）：真的 Claude＋ffmpeg 跑過，《白髮》→ zh-Hant、《超自然檔案》→ none、影集判斷跟著變成 zh_hant、花費約 US$0.002 有記到；但一段 4 分鐘、沒有對白的《白髮》打鬥片段回了 `none`——六張畫面可能剛好都落在沒台詞的時候，而 `none` 等於宣告整季沒有中文（-b2 還會讓它一直留在付費生成清單裡）。改成：
  - 第一集說沒燒，而且這季還有別集「有檔案、沒字幕軌」→ 再抽一集。挑的是這份清單**中間**那集（`清單[長度/2]`），不是下一集，因為相鄰集數的結構很像（片頭、前情提要）。
  - 第二集說有燒 → 用第二集抄到的字判簡繁，寫那個值；第二集也說沒燒 → 寫 `none`；第二集判斷不了（截不到、回覆讀不懂、OpenCC 失敗）→ 整季不判斷、什麼都不寫，`reason` 是第二集的原因。
  - 這季只有一集沒字幕軌 → 接受一次 `none`（維持原本行為），用 Info log 記下「只看了一集」。
  - 第一集就說有燒 → 不看第二集。
  - 兩次呼叫的費用加總進 `cost_usd`（用 decimal 加，最後才轉 float），每次呼叫照樣各寫一筆 `ai_side_spend`。
- 兩個新欄位在 Episode JSON 先標 `json:"-"`，不改現有 API 形狀；要不要在畫面上露出、季清單「中文來源」要不要多一種「燒在畫面上」，留給 -c 決定。
- `ClearMissingFile`／`ReleaseFileFromOtherEpisodes` 沒有順便清 burned 欄位（規格沒要求）；換成別的版本的檔案後，舊的 burned 值會留著，-b2 做自動重檢時要一起考慮。
- 燒字幕檢查的花費目前會算進花費摘要的 `parse_calls`／`parse_usd`（那欄加總整張 ai_side_spend），分開顯示是 -b2 的「花費顯示」。

### 測試證據

- 新測試：`models/chinese_subtitle_burned_test.go`（四種 burned 值 × 有／沒有字幕軌、有字幕軌又有 burned 的排序組合）、`zhtw/classify_script_test.go`（含繁體夾一個「静」仍判繁、沒有可分辨字→`zh`、平手→`zh`）、`services/burned_in_subtitle_service_test.go`（拼圖只留下方、少於 3 張不判斷、JSON 解析好的／包在文字裡／code fence／亂碼／缺欄位、CheckSeason 成功／四種不判斷都不寫／沒金鑰不截圖／模型失敗、「none 要第二意見」八種情況（第一集沒燒＋第二集有燒→寫第二集的值且挑的是中間那集、兩集都沒燒→none、第二集回覆讀不懂／截不到／OpenCC 失敗→不寫、只有兩集時第二眼看另一集、只有一集→接受 none、第一集有燒→不看第二集）、兩次呼叫費用加總且各寫一筆花費、沒存片長才跑 ffprobe、真 migration DB 端到端只寫該季 `[]` 的集數並寫進 ai_side_spend、季清單集數判斷讀得到 burned、本機有 ffmpeg 時用 lavfi 產生的片段跑真的抽圖）、`repository/episode_burned_in_test.go`（只寫該季 `[]` 集數、四種值讓影集 chinese_subtitle 變 zh_hans／zh_hant／zh／none、燒繁中的集數離開生成佇列）、`ai/claude_image_test.go`（請求內容＝圖片區塊＋文字、關思考、回空白仍帶 usage）、`handlers/burned_in_subtitle_handler_test.go`（成功、不判斷、沒金鑰 400、六種參數錯 400、502／500）、migration 049 測試（欄位 NULL、kind 接受 burned_in_check 且仍是封閉集合、索引保留、Down→Up 保留舊列、Up 冪等、已註冊）。
- 既有 verdict 測試全部照舊通過（只補第四個參數 `""`）。
- `cd apps/api && go build ./... && go vet ./... && go test ./...`：44 個套件全 ok；`staticcheck ./...` 無輸出、exit 0；改到的檔案 `gofmt -l` 乾淨（順手把 `episode_repository_test.go` 一行既有的對齊問題 gofmt 掉）。
- 沒有打真的 Claude API（測試全用假的）；真片庫的驗證留給 -b2／上線後用這支 API 跑一季。

### 本機實測（Opus，2026-11-02）

- 用正式機資料庫副本（migration 049 在 6 張資料表上跑過，`ai_side_spend` 原有 1 筆保留）＋真的 Claude（`claude-sonnet-5`）＋真的 ffmpeg／OpenCC，把季的檔案指到從 NAS 剪下的片段：
  - 《超自然檔案》S1 → `none`，寫 18 集。
  - 《白髮》S1 只給「打鬥、沒人說話」的片段 → 第一次單看一集判成 `none`（發現誤判風險，加上第二集複查）；改完後第一集用無對白片段、中間那集用有對白片段 → `zh-Hant`，寫 58 集，花 US$0.00426（兩次）。
  - 影集 API 的 `chinese_subtitle` 從 none／unknown 變成 `zh_hant`。

## Change Log

- 2026-11-01：建立。
- 2026-11-02：Amelia 實作完成（偵測、migration 049、verdict 4 參數、手動觸發 API），送 review。
- 2026-11-02：協調者本機真跑發現沒對白的片段會誤判 `none`；改成「none 要第二意見」（抽該季中間那集再問一次），回應欄位改為 `checked_episode_ids`。
