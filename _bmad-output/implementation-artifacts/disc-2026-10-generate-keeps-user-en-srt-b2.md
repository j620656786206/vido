# Disc（後端）：影片旁邊的英文字幕對得上，就直接拿來翻，不再花錢聽寫

Status: ready-for-dev（⚠️ AC #0 和 AC #8 要等 NAS 開機，其他可以先做）

**Source：** 傘狀單 `disc-2026-10-generate-keeps-user-en-srt-b` 拆出的第二張。
- `-b1`（#868）做好了檢查器 `internal/subtitle/sidecarcheck`，這張把它接進路由和估價。
- ⚖️ Alexyu 2026-10-19 的裁定：對得上才用；對不上要用白話告訴使用者，讓他自己選，**不准默默改走比較貴的路**。
- ⚖️ Alexyu 2026-10-20 選「做 -b2」。
- 說明文字和讓使用者選的畫面歸 `-b3`（要 Sally 先畫稿）。這張只做後端，加上一行前端說明。

**為什麼要做：** 目前片旁有英文字幕的片，按「生成字幕」仍然花錢聽寫。《Tears of Steel》估價 $0.19，翻譯路線大約只要三分之一。而且聽寫稿的時間軸，沒有官方字幕準。

## 查到的事（2026-10-20，`origin/main` 6f2b1be5）

- **執行時的路由**：`router.go:95` `SelectAndRoute`。流程是 probe → `SelectCandidates`（只挑片內軌）→ 抽出字幕軌 → `chooseMain`。沒有候選字幕軌時，交給 `verdictWithoutTrack`（`:190`）處理，它的註解寫「M1 is embedded-only」。`router_test.go:546` 的 `TestSelectAndRoute_ExternalSidecarsAreIgnored` 把這個行為鎖住。
- **估價**：`predict_route.go` `PredictFromTracks` 是純函式，只看字幕軌清單。它同時給單片估價（`GET …/transcribe/estimate`）和批次的候選掃描用，所以**不能**在裡面跑要花幾秒的檢查。
- **檢查器**：`sidecarcheck.NewChecker(prober, cache, logger).Check(ctx, media, audioStream, sidecar)`。
  - 結果有四種：`aligned`／`offset`／`mismatch`／`unusable`；如果完全沒辦法檢查，就回 error，代表「不知道」。
  - 結果會快取 30 天。
  - 每次檢查大約 1～11 秒；99 GB 的 4K 原檔第一次要 23～37 秒。
- **門檻只在 5 部片上量過**：`-b1` 的 CR 和考卷報告都說，門檻可能剛好只適合這 5 部片（`_bmad-output/audit/sidecar-alignment-exam-2026-10.md`）。
- **片旁字幕清單**：`inventory.go` 的 `ListSidecars` 已經會讀出每個片旁字幕的語言。`.vido.en.srt` 也會判成 `en`（`-a`）。
- **翻譯路線不會寫 `.en.srt`**（`-b` 傘狀單查過）：只會寫 zh-Hant，所以不會碰到使用者原本的英文檔。

## Story

身為片子旁邊放著英文字幕的 NAS 使用者，
我想要 Vido 確認這份字幕跟影片對得上之後，直接拿它來翻，
這樣我付的錢比較少、等的時間比較短，中文字幕的時間軸也跟我原本的英文字幕一樣準。

## Acceptance Criteria

0. **⚠️ 先擴大考卷（要等 NAS 開機）。** 用 `-b1` 的考卷程式，再考至少 10 部片：至少 2 部安靜的劇情片、2 部動畫、2 集不同影集，其餘不限。每部 6 種字幕，跟 `-b1` 一樣。
   - 全對 → 才可以接線。
   - 有錯 → 停下來，把結果交給 Alexyu 裁定，不可以自己調門檻硬讓它過。
   - 結果補進 audit 檔，片子一樣只寫「哪一種片」。
1. **怎樣算可用的片旁英文字幕。** 檔名是 `<base>.en.srt`、`.eng.srt`、`.en.ass`，也包含 `.en.sdh.srt` 這類多一段標籤的寫法；`ListSidecars` 判出來的語言要是 `en`。
   - 優先順序：片內英文軌 > 使用者自己的片旁字幕 > Vido 自己的 `.vido.en.srt` > 聽寫。片內英文軌優先，是跟現在的做法一致。
   - `.vido.en.srt` 是 Vido 自己從這部片聽寫出來的稿，一樣要先檢查，因為影片檔可能已經換過。
   - 同時有好幾個候選檔時，第一個 `aligned` 的勝出。
2. **執行時（`SelectAndRoute`）。** 沒有可用的片內軌時，就依 AC #1 的順序對每個候選檔跑 `Check`，這一步是同步的，因為已經在生成工作裡了。
   - 遇到第一個 `aligned`：回傳翻譯路線，來源是那個片旁檔。後續翻譯、放字幕的流程照舊，翻出來的 `.zh-Hant.srt` 時間軸要跟那個片旁檔一致。
   - `offset`、`mismatch`、`unusable`，或檢查失敗：退回現在的行為。這時候估價一定已經是聽寫的價錢（見 AC #3），所以使用者按下去之前就知道價錢，不算「默默改走比較貴的路」。
   - 把每個候選檔的檢查結果寫進 log。
3. **估價（單片）。** `GET /{movies|episodes}/:id/transcribe/estimate` 在片內沒有英文軌、但有片旁英文字幕時：
   - 有快取的檢查結果 → 直接用。`aligned` 就報 `plan: "sidecar"` 和翻譯路線的價錢；其他結果照舊報聽寫的價錢。
   - 沒有快取 → **在背景**開始檢查，同一部片同時只能有一個檢查在跑。這次先回聽寫的價錢，並標 `sidecar_check.status: "checking"`。前端之後再問一次，就會拿到結果。
   - 回應多一個欄位 `sidecar_check`，內容有 `status`（`none`／`checking`／`aligned`／`offset`／`mismatch`／`unusable`／`unknown`）、`offset_seconds`、`file_name`。`-b3` 要用這些資料畫出白話說明。**JSON 欄位名稱在動工前要先跟 Alexyu 確認**（CLAUDE.md「Confirm Before Coding」）。
4. **估價（批次／候選掃描）。** `PredictFromTracks` 和批次估價**不跑**檢查，只讀快取。快取是 `aligned` 的片算翻譯價；沒有快取的照舊算聽寫價（寧可高估），並另外回報「有幾部還沒檢查」的數字。
5. **前端最小改動。** `generateCostView.ts` 多一行，`plan === 'sidecar'` 時顯示「用旁邊的英文字幕翻譯，約需 1～2 分鐘」。文字最後由 Sally 在 `-b3` 定稿，這張先用這句。其他畫面不動。
6. **更新被鎖住的舊行為。**
   - `router_test.go:546` 改成新的規則：片旁字幕對得上就翻、對不上就照舊。
   - `verdictWithoutTrack` 的註解和 `sub-1-4-extract-filter-route.md` AC #8 補一行，寫明推翻的依據：「2026-10-19 Alexyu 推翻 M1 範圍限制，見 `-b` 傘狀單」。
7. **測試（Rule 28）：**
   - 路由：用假的 checker，測 aligned／offset／mismatch／unusable／error 各一個案例，加上優先順序（使用者的檔優先於 `.vido.en.srt`）。
   - 估價：快取命中、還沒檢查（背景只啟動一次）、檢查器出錯。
   - 批次：只讀快取，不會叫 ffmpeg。
   - 整條跑一次：真 ffmpeg 合成一部片，片旁放一個對得上的字幕。結果要走翻譯路線、沒有任何聽寫花費、英文檔一個 byte 都不動。沒有 ffmpeg 就 skip。
8. **正式機驗證（要等 NAS 開機）。** 合併、NAS 更新後，用《Tears of Steel》測：
   - 估價第一次是 `checking`，第二次變成 `sidecar`、價錢變便宜。
   - 按下「生成字幕」：帳本的 route 是 `translate`，沒有 ASR 花費。
   - 官方 `.en.srt` 跑前跑後的 sha256 一樣。
   - Completion Notes 附上前後的估價和帳本列。
9. `go test ./...`、`~/go/bin/staticcheck ./...`、`lint:all`（看「✖ N problems (X errors」那行）全部要綠。

## 不做

- 對不上時的白話說明、讓使用者選「還是要用旁邊的字幕翻」或「改用聽寫」 → `-b3`（Sally 先畫稿）。
- 偵測到整條差幾秒時，自動幫使用者修正時間軸 → 新功能，等 `-b3` 設計時再問 Alexyu。
- 片旁簡中字幕直接轉繁 → 不在這個系列。

## Dev Agent Record

### Completion Notes

（dev 填）
