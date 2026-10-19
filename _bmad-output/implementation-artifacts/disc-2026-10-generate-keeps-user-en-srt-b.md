# Disc（後端＋前端）：影片旁邊有英文字幕檔，就直接拿來翻，不再花錢聽寫

Status: backlog（⚖️ 要 Alexyu 裁定：推翻 sub-1-4「片旁字幕檔歸搜尋引擎管」）

**Source:** `backlog-movie-generate-overwrites-existing-en-srt` 拆出的第二張（第一張 `-a` 是「絕不覆寫」安全網）。2026-10-19 正式機重現：《Tears of Steel》旁邊放著官方英文字幕，按「生成字幕」仍花錢聽寫（估價 $0.19、說明「語音辨識＋AI 翻譯」）。

**為什麼要做：** FB 新貼文要寫「片子已有英文字幕：一集約 NT$15」。片內有英文軌的片今天已經成立（`disc-2026-10-single-generate-ignores-embedded-english-a`）；但英文字幕是**旁邊的 `.srt` 檔**時，現在一律走比較貴、比較慢、時間軸比較不準的聽寫。

## 查到的事（2026-10-19，`git show origin/main:`）

- **路由只看片內軌**：`router.go:94` `SelectAndRoute`；`extractor.go:141-148` 跳過 `External`；`verdictWithoutTrack`（`router.go:190`）註解「M1 is embedded-only; sidecars stay the search engine's domain」；決策出處 `sub-1-4-extract-filter-route.md:159`。測試 `router_test.go:546` `TestSelectAndRoute_ExternalSidecarsAreIgnored` 鎖住現況。
- **估價同一套**：`GET /{movies|episodes}/:id/transcribe/estimate` → `PredictFromTracks`（`apps/api/internal/subtitle/predict_route.go:77-95`）也跳過 `External` → 有片旁英文檔仍估成 `asr`。
- **對話框文字只看估價**：`ManageSubtitleDialogV2.tsx:923-947` `<ButtonCost label="生成字幕">`；說明句來自 `generateCostView.ts`（`DEFAULT_LINE` `:49`「語音辨識＋AI 翻譯，約需數分鐘」、`EXTRACT_LINE` `:64` 只在 `plan === 'extract'`）。對話框**有**讀片旁字幕清單（`GET …/subtitles/inventory`，`ManageSubtitleDialogV2.tsx:344`、`buildInventoryRows` `:153-184`），但只拿來顯示，不影響按鈕。
- **翻譯路線本身不寫 `.en.srt`**：`deliverable` → `translateWithCache`（`process_item.go:396-401`）→ `placer.Place` 只寫 zh-Hant（`:269`、`pipeline.go:743`）。所以只要路由選了翻譯，就不會碰使用者的英文檔。

## 要 Alexyu 裁定的事

1. **推翻 sub-1-4 的那條線嗎？** 讓路由把「片旁的 `<base>.en.srt`／`.eng.srt`／`.en.ass`」當成翻譯來源，優先序排在片內英文軌之後、聽寫之前。Bob 建議：**推翻**——當初的理由是「片旁字幕歸搜尋引擎」，但使用者自己放的英文字幕不是搜尋結果，用它翻比聽寫便宜、時間軸也是原本的。
2. **片內英文軌和片旁英文檔都有時用哪個？** Bob 建議：片內優先（跟現在一樣），片旁只在片內沒有時才用。
3. **怎麼確定片旁那個檔真的是英文、可以用？** 至少：副檔名語言碼是 `en`／`eng`、能 parse、句數 > 0；要不要再做語言偵測？

## Acceptance Criteria（裁定後由 Bob 補齊）

- 路由與估價（`SelectAndRoute`、`PredictFromTracks`）用同一條規則判斷片旁英文檔，估價顯示翻譯路線的價錢；對話框說明句跟著估價切換（`generateCostView.ts` 新增翻譯路線的句子）。
- 翻譯完 `.zh-Hant.srt` 時間軸跟片旁英文檔一致；英文檔一個 byte 都不動。
- 更新 `router_test.go:546` 的鎖定測試與 sub-1-4 的裁定紀錄。
- 正式機用《Tears of Steel》驗證：估價變便宜、說明不再寫「語音辨識」、帳本 route 是 `translate`、沒有 ASR 花費。
- 依 Story Splitting Rule：前後端各超過 3 件就再拆 `-b1`（後端路由＋估價）／`-b2`（對話框文字）。

## 不做

- 絕不覆寫使用者檔案 → `-a`（先做）。
- 片旁簡中字幕直接轉繁（已有 `subtitle_handler.go:353` 的轉換入口）→ 不在本單。
