# Story disc-2026-11-burned-in-subs-c1-api：後端——讓畫面看得到「畫面字幕」，並能改判

Status: done

**Source:** 設計決定 `disc-2026-11-burned-in-subs-c-design.md`。前置 -b1／-b2／skip-missing（PR #962／#964／#965）。

## 做法

1. **集數 JSON 露出**：`models.Episode.BurnedInSubtitle` 改成 `json:"burned_in_subtitle,omitempty"`（值 `zh-Hans`／`zh-Hant`／`zh`／`none`；沒檢查過就不出現）。`burned_in_checked_at` 維持不露出。確認季集數 API（`GET /api/v1/series/:id/seasons/:n/episodes`）與單集讀取都帶得到。
2. **季清單的中文來源**：`services/episode_chinese_subtitle.go` 的來源種類多一種 `burned`（[@contract-v1] 的來源清單是開放的字串種類，加一種不破壞既有值）；`burned_in_subtitle` 是 zh 三種之一時加一筆 `{kind:"burned", language:"zh-Hans|zh-Hant|zh"}`，排在最後。
3. **改判 API**：`PUT /api/v1/subtitles/burned-in/season`，body `{"series_id": "...", "season": 1, "value": "none"}`：把該季 `subtitle_tracks='[]'` 的集數寫成 `none`（含 `burned_in_checked_at`），回 `{"episodes_updated": N}`。`value` 只接受 `"none"`（其他值 400）。「重新檢查這季」直接用既有的 `POST /api/v1/subtitles/burned-in/check`（-b1）。
4. 前端型別不在本張改（-c2）。

## Acceptance Criteria

1. [x] 季集數 API 的集數帶 `burned_in_subtitle`（沒檢查過不出現）；handler／repository 測試。
2. [x] 中文來源多 `burned`，只在 zh 三種時出現；測試。
3. [x] 改判 API：成功、參數錯（value 不是 none、缺 season）、影集不存在時的行為；只寫 `[]` 的集數。
4. [x] `go build／vet／test`、`staticcheck`、`gofmt` 乾淨。

## Dev Agent Record

### 改了什麼

- **集數 JSON 露出**：`models.Episode.BurnedInSubtitle` 改成 `json:"burned_in_subtitle,omitzero"`；`burned_in_checked_at` 維持不露出。用 `omitzero` 而不是規格寫的 `omitempty`：`NullString` 是 struct，`omitempty` 永遠不會省略，沒檢查過會寫出 `null`；`omitzero`（Go 1.24+）才會讓沒檢查過的集數完全不出現這個 key。
- **季集數 API**（`GET /api/v1/series/:id/seasons/:n/episodes`）回的是 `services.MergedEpisode`，不是 `models.Episode`，所以另外在 `MergedEpisode` 加 `burned_in_subtitle`（字串、omitempty，只在有本機檔案時填）。`none` 也會出現——-c2 要靠它判斷「不顯示畫面字幕那一列」；沒檢查過就沒有這個 key。專案裡沒有單集讀取的 API，單集的 JSON 就是 `models.Episode` 本身，已用測試確認。
- **中文來源多 `burned`**（`services/episode_chinese_subtitle.go`）：`burned_in_subtitle` 是 zh 三種之一時，排在最後加一筆 `{kind:"burned", language, label:""}`；`none`、沒檢查過不加。
- **改判 API**：`PUT /api/v1/subtitles/burned-in/season`，body `{"series_id","season","value"}` → `BurnedInSubtitleService.OverrideSeason` → 既有的 `SetSeasonBurnedInSubtitle`（只寫該季 `subtitle_tracks='[]'` 的集數，連 `burned_in_checked_at` 一起寫），回 `{"episodes_updated": N}`。不呼叫模型、不花錢；寫成 `none` 之後自動檢查就不會再碰這季。

### 決定

- **`zh` 在來源清單裡寫成 `zh-unknown`**（跟規格寫的 `zh` 不同）：既有的來源契約裡，語言只有 `zh-Hant`／`zh-Hans`／`zh-unknown` 三種（沒標簡繁的 `chi` 軌就是 `zh-unknown`）。畫面字幕要「跟其他來源同一串、同一種寫法」，就用同一套字；`burned_in_subtitle` 欄位本身還是原值 `zh`。
- **影集不存在時**：該季在片庫裡一集都沒有（影集不存在，或沒有這季）→ 404 `DB_NOT_FOUND`；這季有集數但沒有一集是 `[]` → 200、`episodes_updated: 0`。用既有的 `FindBySeasonNumber` 判斷，不多接影集 repository。
- `value` 只接受小寫 `"none"`，其他（含 `zh-Hans`、`NONE`、沒給）→ 400；handler 和 service 都擋。
- ⚠️ **-c1 不能比 -c2 早上線**：現在的前端 `EpisodeList.tsx` 用 `SOURCE_LABEL[s.kind](s)` 組來源文字，遇到沒看過的 `burned` 會變成呼叫 `undefined`，季清單會直接壞掉——production 已經有 45 季判出畫面字幕。要嘛跟 -c2 一起上，要嘛 -c2 先補上 `burned` 的文字（或至少遇到不認得的種類就跳過）。依規則本張沒有改前端。

### 測試證據

- `handlers/series_handler_test.go`：季集數 API 的 JSON 帶 `burned_in_subtitle` 和 `burned` 來源，沒檢查過的集數沒有這個 key。
- `services/series_season_burned_in_test.go`：五種集數（燒簡、none、沒檢查、旁邊有 zh-TW 檔又燒簡、燒中文但看不出簡繁）——欄位值、來源只在 zh 三種時出現且排最後、`zh` 對應 `zh-unknown`、序列化後沒檢查過的不出現；`models.Episode` 的 JSON 帶答案但不帶檢查時間。
- `services/burned_in_subtitle_service_test.go`：`OverrideSeason` 寫 `none` 且不呼叫模型、非 `none` 一律拒絕且不寫、整季沒集數 → not found；真的 migration DB 跑一次：只寫該季 `[]` 的集數（含檢查時間），有字幕軌的集數和別季不動，集數判斷變回缺中文。
- `handlers/burned_in_subtitle_handler_test.go`：PUT 成功、八種參數錯（value 不是 none／沒給／大寫、season 沒給／負數／字串、series 沒給、不是 JSON）都是 400 且不寫、季不存在 404、資料庫失敗 500。
- `cd apps/api && go build ./... && go vet ./... && go test ./...`：44 個套件全 ok；`~/go/bin/staticcheck ./...` 無輸出、exit 0；改到的檔案 `gofmt -l` 乾淨。

## Change Log

- 2026-11-02：建立。
- 2026-11-02：Amelia 實作完成（集數 JSON 露出、來源多 burned、改判 API），送 review；提醒不能比 -c2 早上線。
