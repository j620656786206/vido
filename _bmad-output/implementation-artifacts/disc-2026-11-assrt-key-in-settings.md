# Story disc-2026-11-assrt-key-in-settings：在「設定 → API 金鑰」填射手網金鑰，填了馬上能搜中文字幕

Status: done

**Source:** ⚖️ Alexyu 2026-11-02 選「1：在設定頁加欄位」。背景：OpenSubtitles 金鑰找不到；專案 `.env` 有 `ASSRT_API_KEY`（射手網），實測可用——《天蠍》S1、《超自然檔案》S1 都搜得到含繁中的字幕。但 Vido 的射手網來源只在開機時讀一次 `assrt_api_key` 密鑰，設定頁沒有欄位、也不讀環境變數，NAS 開機 log 寫「Assrt provider disabled」。

## 問題

程式早就有射手網字幕來源，但使用者沒有任何辦法把金鑰交給它：設定頁沒欄位、環境變數不讀、就算直接寫進資料庫也要重開機才生效。

## 做法

1. 後端金鑰清單加 `assrt`：密鑰名稱沿用射手網來源原本讀的 `assrt_api_key`（不改名，避免搬資料）；環境變數 `ASSRT_API_KEY` 當退路；順序排最後。
2. `PUT /api/v1/settings/keys` 接受 `assrt` 欄位（同其他金鑰：空字串＝清除）。
3. 射手網來源改成每次搜尋／下載時向金鑰解析器要金鑰 → 儲存後立即生效，不用重開機。沒金鑰時行為不變（搜尋回空、下載回錯）。
4. 設定頁加一列「射手網（字幕搜尋，選填）」，說明填了能在搜尋字幕時找射手網的中文字幕、不花 AI 費用、每分鐘最多 5 次。
5. **實測發現射手網來源從沒對真的 API 跑通過**（只對著文件和假資料寫）。同一張一起修，不然填了金鑰也沒用：
   - 搜不到東西時 API 回 `"subs": {}`（物件不是陣列）→ 以前直接解析失敗。
   - `lang` 是物件 `{desc, langlist:{langcht:true…}}` 不是字串；列表裡還混了另一種格式（`fileid` 字串、`m_langn` 陣列）→ 以前整筆解析失敗。現在兩種都認，語言對到 zh-Hant／zh-Hans／zh／en。
   - 詳細資料的檔案大小是字串 `"138KB"` → 以前解析失敗。
   - 整季打包（《天蠍》S1 是 197 個檔的 .rar）以前會把整個壓縮檔當字幕。現在照影片檔名的 SxxEyy 挑那一集，優先純繁中 .srt；包裡沒有那一集就報錯，不會拿隔壁集。
   - 射手網的「單檔直接下載」對 .rar 包會回 HTTP 493（伺服器抽風頁）→ 改成下載整包、自己解壓（zip 用標準庫，rar 用 `github.com/nwaples/rardecode/v2`）。
6. **不做 OpenSubtitles**：它下載字幕要帳號＋密碼登入，不只一把金鑰；使用者還沒註冊，無法驗證。等有帳號再開單。

## Acceptance Criteria

1. [x] GET `/settings/keys` 多一列 `assrt`（configured／source／masked 規則同其他金鑰）。
2. [x] PUT `{assrt:"…"}` 存進 `assrt_api_key`；`{assrt:""}` 清除。
3. [x] 射手網來源在儲存金鑰後的下一次搜尋就帶新金鑰（不需重開機）；清除後回到「停用」。
4. [x] 環境變數 `ASSRT_API_KEY` 有值、資料庫沒存時也能用（source=env）。
5. [x] 設定頁出現射手網那一列，可填、可清除；三張 ApiKeysForm 視覺基準更新。
6. [x] `go build／vet／test`、`staticcheck`、`gofmt`、web lint／test／format 乾淨。

## Dev Agent Record

- 驗證（本機，正式機資料庫副本＋`.env` 的射手網金鑰當環境變數）：
  - `GET /settings/keys` → `assrt` configured、source=env。
  - 搜尋「Scorpion S01」→ 5 筆，前兩筆是「天蝎 第一季」繁中（zh-Hant，0.66 分）。
  - 下載 593605（整季 .rar）到 `Scorpion.S01E02.720p.mkv` 旁 → 單檔下載 493 → 改下載整包解壓 → 放好 `Scorpion.S01E02.720p.zh-Hant.srt`，950 句，第一句「靈感來自沃特·奧布萊恩真實故事」。
- 測試：provider（兩種列表格式、`{}`、每次讀金鑰、挑集數與排序、壓縮包退路、改名成員）、services（列表 6 項、assrt 存／清／環境變數）、handler（PUT assrt）、web（新列＋PUT assrt）。
- **後續（沒做）**：管理字幕的「線上搜尋」目前**只開給電影和整部劇**，單集看不到（後端 `oneof=movie series`）；而且查詢字用的是台灣片名（「天蠍行動」在射手網搜不到，要「Scorpion」）。所以填了金鑰後，劇集還要等下一張 `disc-2026-11-online-subs-for-episodes` 才用得到。
