# disc-2026-10-solo-gate-en-sidecar：只有翻譯金鑰、片旁有英文字幕時，按「生成字幕」不會再被擋成「語音辨識尚未設定」

Status: done

<!-- SM Bob create-story 2026-10-23；行號為 main `4b81b30a`。 -->

**Source:** Rule 24 lane ③，2026-10-23 在 NAS 做「Vido vs Lingarr」翻譯比較時發現（臨時容器只放 Gemini 金鑰＋TMDb，沒有 ASR 金鑰；片子是 Tears of Steel，片內沒有字幕、旁邊有官方 `.en.srt`）。Alexyu 選 A：10/27 檢查 FB 貼文前先修。

## 為什麼要做

FB 招募文主打「用自己的 Gemini 免費額度」。照著做的新用戶只會有一把 Gemini 金鑰，不會有語音辨識（ASR）金鑰。

-b2（#881）上線後，片內沒有字幕、旁邊有對得上的英文字幕時，Vido 會直接翻旁邊那份英文字幕，不需要聽寫。估價對話框也已經會顯示「用旁邊的英文字幕翻，$0.02」。

但真的按下「生成字幕」，畫面回「語音辨識尚未設定」（HTTP 503）。實測：

- `GET …/transcribe/estimate` → `"route":"sidecar"`、`"estimated_usd":0.02`、`sidecar_check.status = "aligned"`
- `POST /api/v1/movies/:id/transcribe?translate=true` → 503 `TRANSCRIPTION_DISABLED`
- 同一部片改打 `POST /api/v1/subtitles/pipeline/run` → 成功，log 寫 `subtitle route: translating the English subtitle beside the file`，76 句

也就是說：估價說可以，按鈕說不行。正式機沒踩到，因為正式機有 ASR 金鑰。

**原因：** 單部生成（SoloRunner）在開跑前有一道「會不會需要語音辨識」的檢查（`solo_runner.go:211`）。它只看片內字幕軌：沒有可用的片內字幕軌，就當成一定要聽寫；沒有 ASR 就擋。它沒問「旁邊有沒有對得上的英文字幕」，可是路由（`router.go:108`）在聽寫之前會先試旁邊的英文字幕。

## Acceptance Criteria

1. **對得上就放行**：片內沒有可用字幕、沒有 ASR、不能續跑，但旁邊的英文字幕檢查結果是「對得上」（`aligned`）→ 按「生成字幕」正常開跑（202），不回 503。
2. **還在檢查、或檢查跑不起來，也放行**：檢查結果是「檢查中」（`checking`）或「沒辦法檢查」（`unknown`）→ 放行。開跑後路由會等同一次檢查的結果（同一部片只檢查一次）；對不上時才退回聽寫，那時沒有 ASR 就是這一次執行失敗，不是按鈕被擋。理由：這道檢查本來的原則就是「判斷不了就不擋」（讀不到片子的 probe 也不擋）。
3. **確定用不了才擋**：旁邊沒有英文字幕（`none`），或檢查過、對不上（`offset`、`mismatch`、`unusable`）→ 照舊回 503「語音辨識尚未設定」。
4. **按鈕不等 ffmpeg**：這道檢查只讀已存的檢查結果（`Peek`），不會在 HTTP 請求裡解碼音訊。還沒檢查過時，和估價對話框一樣在背景開始檢查。
5. **其他路線不變**：片內有可用字幕（`extract`）、有 ASR、可以續跑，三種情況行為和現在一樣，不會去讀旁邊的字幕。
6. **測試**：
   - SoloRunner 檢查表加上 AC 1～3 的情況（旁邊的字幕可用／不可用 × 沒有 ASR）。
   - 「旁邊的字幕可不可以用」從檢查結果對應的規則有自己的測試，七種狀態都要列到。
   - extract、有 ASR 的情況下，不會去問旁邊的字幕。

## 不要做的事

- 不改估價、不改路由、不改檢查本身的判斷（-b1／-b2 的範圍）。
- 不改開跑事件裡的 `predicted_route`：走旁邊字幕時，它現在會寫 `asr`／`skip`。有 ASR 的正式機本來就是這樣，前端只在終點事件沒有 `route` 時才用它。另外立案（`disc-2026-10-solo-predicted-route-ignores-sidecar`），不在這張修。
- 不碰批次同意清單（-b2b 的範圍）。批次走 `ProcessItem` 路由，本來就會先試旁邊的字幕，沒有這道檢查。
- 不改 503 的錯誤文字。

## Tasks

- [x] Task 1 — SoloRunner 加上「旁邊的英文字幕可不可以用」這個選項，檢查只在「會需要聽寫、又沒有 ASR、又不能續跑」時才問（AC 1、3、5）
- [x] Task 2 — `sidecarcheck.Finder` 加一個只讀已存結果的判斷，把七種狀態對應成可用／不可用（AC 2、3、4）
- [x] Task 3 — `main.go` 接線（同一個 `sidecarFinder`，和路由、估價用的是同一份）
- [x] Task 4 — 測試、`go vet`、`staticcheck`、相關套件測試（AC 6）

## Dev Notes

- 檢查：`apps/api/internal/subtitle/solo_runner.go:205-216`；選項寫法照 `WithSoloResumeCheck`（`solo_runner.go:100`）。
- 路由的順序：`router.go:106-111`（沒有片內候選 → `sidecarVerdict` → `verdictWithoutTrack`）。`predict` 回 `asr` 和 `skip` 兩種都會走到旁邊的字幕，所以兩種都要問。
- 已存結果：`sidecarcheck/finder.go:239` 的 `Peek`；狀態常數在 `services/transcription_estimate.go:38-46`。
- 接線：`cmd/api/main.go:877`（`sidecarFinder`）、`:932`（`NewSoloRunner`）。
- 測試：`solo_runner_test.go:165` 的 `TestSoloRunner_Gate` 檢查表；`sidecarcheck/finder_test.go` 的 fixture 可以造出 none／checking／aligned／mismatch／unknown。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-23。

### Completion Notes List

- 檢查多問一句「旁邊的英文字幕可不可以用」，只在「會需要聽寫、沒有 ASR、不能續跑」三個都成立時才問；extract、有 ASR、可以續跑都不會去讀。
- 可不可以用的規則放在 `sidecarcheck.SidecarMayRoute`：`aligned`／`checking`／`unknown` 放行，`none`／`offset`／`mismatch`／`unusable` 擋。`Finder.MayRoute` 只讀已存結果（`Peek`），不在請求裡解碼。
- 反向驗證：把新加的條件換成 `true`，`TestSoloRunner_Gate` 兩個新案例會失敗。
- **NAS 實測（2026-10-23，臨時容器，只有 Gemini＋TMDb，沒有 ASR）：** 同一部 Tears of Steel，同一個資料庫，只換執行檔。
  - main：估價 `route=sidecar`、`aligned`；按生成字幕 → **503** `TRANSCRIPTION_DISABLED`。
  - 這個分支：估價一樣；按生成字幕 → **202**，log `subtitle route: translating the English subtitle beside the file … cues=76`，約 1.5 分鐘後片旁出現 `Tears of Steel (2012).zh-Hant.srt`（4,723 bytes）。官方 `.en.srt` 沒被動（4,764 bytes，時間戳記不變）。
  - 測完容器和執行檔都刪了。
- `go test ./internal/subtitle/... ./internal/handlers/... ./cmd/api/...` 全過；`go vet`、`staticcheck` 乾淨。

### File List

- `apps/api/internal/subtitle/solo_runner.go`
- `apps/api/internal/subtitle/solo_runner_test.go`
- `apps/api/internal/subtitle/sidecarcheck/finder.go`
- `apps/api/internal/subtitle/sidecarcheck/finder_test.go`
- `apps/api/cmd/api/main.go`
