# Disc（後端）：影片旁邊的英文字幕對得上，就直接拿來翻，不再花錢聽寫

Status: review（AC #0 考完，Alexyu 裁定 A；AC #8 正式機驗證要等合併、部署後做）

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

**2026-10-20 第一段（NAS 關機中，先做不需要 NAS 的部分）**

- **欄位（AC #3）：** ⚖️ Alexyu 2026-10-20 選 A，照提案的欄位做，`route` 也加 `sidecar`。
  - `plan` 多一個值 `"sidecar"`，`route` 多一個值 `"sidecar"`。
  - 新增 `sidecar_check: {status, offset_seconds, file_name}`。`status` 的值有：`none`／`checking`／`aligned`／`offset`／`mismatch`／`unusable`／`unknown`。
  - 每個值的意思寫在 `services/transcription_estimate.go` 的常數註解，以及前端 `SidecarCheck` 的型別註解。
- **路由（AC #1、#2、#6）：**
  - 新增 `subtitle/sidecar_route.go`，裡面有 `SidecarFinder` port、`Router.SetSidecarFinder`、`WithoutSidecars`。
  - `SelectAndRoute` 在找不到片內候選字幕軌時（包含只有 `und` 軌的情況），先問 finder；拿到對得上的檔，就回傳 `RouteTranslate`。這時 Track 的 `StreamIndex` 是 -1，`Path` 是片旁檔，Blocks 一樣經過 SDH 過濾。片旁檔只讀不寫。
  - `router_test.go:546` 原本鎖住「片旁字幕一律忽略」，換成 6 個案例：沒接 finder、對得上、`und` 軌也會改用片旁檔、對不上、片內英文軌優先（而且完全不會去查 finder）、檔案壞掉時退回原本的做法。
  - `sub-1-4` AC #8 補上推翻的紀錄。
- **Finder（`sidecarcheck/finder.go`）：**
  - 候選檔的順序：使用者的 `.en.srt`／`.eng.srt` → 其他英文 `.srt`（例如 `.en.sdh.srt`）→ `.vido.en.srt`。音軌用 `SelectEnglishTrack` 挑，跟聽寫用的同一條。
  - **偏離 story：** AC #1 列了 `.en.ass`，這次只收 `.srt`。路由用的 `ParseSRT` 讀不了 ASS，要支援得先在 subtitle 套件加 ASS→blocks 的轉換。留待之後再做。
  - `FindAligned` 給執行時用，會等檢查做完。`Peek` 給估價用，只讀存好的結果；沒有存好的結果時，就在背景開始檢查，馬上回 `checking`。
  - **CR 修正：**
    - 同一部片同時只有一個檢查在跑（singleflight），全伺服器同時只解碼一部片（slot）。
    - 最近一次的檢查結果留在記憶體 30 分鐘。這樣就算沒有快取或快取寫入失敗，也不會每次估價都重新解碼，前端也不會一直停在「檢查中」。
    - 沒辦法檢查的片，10 分鐘內回 `unknown`。
    - 過期的紀錄會自動清掉；背景工作出錯（panic）時會被接住，不會讓整個程式掛掉。
- **估價（AC #3）：**
  - 只在 pipeline 模式查，而且只在路線不是 `extract` 時查。只有 `aligned` 會改價錢，價錢跟片內英文軌的翻譯路線一樣；其他狀態一律報聽寫價。
  - 執行時會先試片旁字幕，再退回聽寫續跑；估價跟著這個順序，所以對得上的片旁字幕優先於「上次聽寫到一半、這次續跑」。
  - 一個 finder 同時給路由和估價用，兩邊讀的是同一份快取。
- **FreeOnly 自動生成（CR）：**
  - 掃描後的免費自動生成，不管片旁字幕對不對得上，都會把這部片停在「要花錢、等你決定」，所以檢查也不會改變結果。現在這條路會帶 `WithoutSidecars`，掃描時不跑 ffmpeg 檢查。
  - 使用者手動按生成，或批次生成時，照樣會檢查。
- **前端（AC #5）：**
  - `SIDECAR_LINE` 是「用旁邊的英文字幕翻譯，約需 1～2 分鐘」。
  - `sidecar` 方案不需要聽寫金鑰；沒有翻譯金鑰時會擋下來，因為按下去也做不出任何東西。
  - 狀態是 `checking` 時，每 3 秒重新估價一次，最多 60 次（約 3 分鐘）。
  - 沒有聽寫金鑰、又正在檢查時，顯示「正在檢查旁邊的英文字幕能不能直接翻譯…」，不會先閃一下「需要 ASR 金鑰」。
- **⚠️ AC #4（批次估價讀快取）延後：** 拆成 `disc-2026-10-generate-keeps-user-en-srt-b2b`。
  - 現在批次同意清單（`generation_candidates.go`）對這類片仍然報聽寫價，但批次實際執行時，對得上的片會改走翻譯。
  - 所以是報高、不是報低，使用者不會付得比畫面上多。CR 也把這點列為 nit。
- **測試：**
  - `router_test.go`：6 個片旁字幕案例。
  - `transcription_estimate_sidecar_test.go`：對得上就報翻譯價；6 種「還沒對上」的狀態都報聽寫價；`skip` 路線也會查；`extract` 不會查；對得上優先於續跑；沒有翻譯金鑰是 $0；legacy 模式不會查。
  - `finder_test.go`（`-race` 跑 3 次）：候選檔順序；挑英文音軌；旁邊沒有字幕時連音軌都不讀；`checking` 之後變成 `aligned`、存好之後不再解碼；對不上時回報第一個檔的結果；沒辦法檢查時回 `unknown`、不會每次都重試；沒有快取也靠記憶體回答；同時有很多人問時只跑一次檢查。
  - `route_integration_test.go`：真的 ffmpeg 合成一部片。片旁字幕對得上就走 `RouteTranslate`，Path 是片旁檔、100 句、英文檔一個 byte 都沒變；換成另一個版本的字幕，就是 `RouteNoTextSource`。
  - 前端：`generateCostView` 5 個案例、`estimateRefetchInterval` 2 個案例。
- **檢查：**
  - `go test ./...`、`go vet`、`~/go/bin/staticcheck ./...` 全部乾淨。
  - `lint:all`：「✖ 170 problems (0 errors」。
  - `web:typecheck` 通過；`nx test web` 298 個檔案全部通過。
- **跨模型 CR（Sonnet）：** 沒有 blocker。should-fix 1～3、nit 6～7 都修了。nit 4（批次估價）拆到 `-b2b`。nit 5（估價看的音軌）理論上會不一致，但實際上用的是同一條規則，不處理。
- ~~**還沒做：** AC #0（多考 10 部片）、AC #8（正式機驗證），都要等 NAS 開機。兩項都過了才開 PR、合併。~~ 見第二段。

**2026-10-21 第二段（NAS 開機，AC #0 考卷）**

- **結果 57/60，沒有全對。** 照 AC #0 停下來交給 Alexyu 裁定，門檻一個都沒動。
  - 10 部片：安靜劇情片 4K、喜劇劇情片 4K、劇情片原檔 85 GB、動畫電影、推理片、只有 SDH 字幕的動作片、日本動畫一集（英文配音）、30 分鐘喜劇影集、紀錄片原檔 7.1、對白多的影集。每部 6 種字幕，共 60 格。
  - 10 部片自己的原字幕都判「對得上」；平移 +3／−2 秒都找到，誤差 0.2 秒以內。
  - **錯的 3 格都是「fps 24」**（字幕給另一個版本做，時間軸越走越慢）：動畫一集、30 分鐘影集、紀錄片，該判「對不上」，卻判成「整條差 0.7／1.1／1.8 秒」。原因是片子比較短，或聲音太安靜，前後半段的漂移檢查沒抓到。
  - 沒有任何一個不該用的字幕被判成「對得上」。
- **⚖️ Alexyu 2026-10-21 選 A：照常接線。**
  - 理由：在這張單的做法裡，「整條差幾秒」和「對不上」都退回聽寫、報聽寫價，所以這 3 格錯誤最多多花聽寫的錢，不會拿歪掉的字幕去翻。
  - 風險移到 `-b3`：說明會寫「差約 1 秒」，其實是越走越歪。要做「自動修正時間軸」之前，一定要先修好漂移判斷（已補進 sprint-status 的 `-b3`）。
- **速度：** 大檔第一次檢查最久 196～241 秒（考試時正式機也在跑，NAS 負載約 8）。前端最多輪詢約 3 分鐘，之後先顯示聽寫價；不會多收錢，只是先看到比較貴的價錢。
- 細節在 `_bmad-output/audit/sidecar-alignment-exam-2026-10.md` 的「Second exam」；NAS 上的材料在 `/mnt/user/data/tmp-b2exam/`。
- **還沒做：** AC #8（正式機驗證），合併、部署後用《Tears of Steel》做。
