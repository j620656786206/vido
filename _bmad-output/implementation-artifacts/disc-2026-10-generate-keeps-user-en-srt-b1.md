# Disc（後端）：先免費檢查「影片旁邊的英文字幕」跟影片對不對得上

Status: ready-for-dev

**Source:** 傘狀單 `disc-2026-10-generate-keeps-user-en-srt-b` 拆出的第一張。⚖️ Alexyu 2026-10-19（party mode 選 A）：片旁英文字幕可以拿來翻（推翻 sub-1-4 的範圍限制），但要**先免費檢查時間軸**：對得上才用；對不上就用白話告訴使用者，讓他自己選。**不准默默改走比較貴的聽寫。**

- 本單只做「檢查」這個後端能力，還不接到路由、估價或對話框。
- 接路由和估價是 `-b2`；對話框是 `-b3`。

**為什麼要做：** 使用者放在旁邊的英文字幕不一定是這個版本的。常見的情況有三種：

- 整條差幾秒：片頭長度不同。
- 越後面差越多：影格率不同，例如 23.976 和 25。
- 根本是別集。

翻一份對不上的字幕，中文也一樣對不上，錢還照付。所以要先檢查。檢查必須**不花錢**（在 NAS 本機算），而且**夠快**：要能在使用者打開對話框、看到估價之前做完，或至少先做完一部分。

## 查到的事（2026-10-20，Bob 以 `origin/main`＋正式機確認）

- **⚠️ 原本的計畫（重用 `detectSilences`）量過了，不能用。**
  - 正式機量測：對《Tears of Steel》（12 分鐘、官方字幕 76 句）跑 `ffmpeg silencedetect=noise=-35dB:d=0.4`（就是 `ai/whisper.go:649` 的參數），整部片只找到 **3 段**靜音，因為配樂和音效幾乎沒停過。
  - 3 個點不夠判斷 76 句字幕對不對得上。那組參數是給「切段時找一個安全的切點」用的，不是用來偵測人聲。
  - 所以改用「人聲偵測（VAD）＋找最佳平移」，見 AC #2。這只是換一個算法，原則沒變：免費、本機、只用 ffmpeg 和 Go，不加新的二進位檔。
- **不用現成工具**（沿用 party mode 結論）：alass 是 GPL（跟當初換掉 opencc 的理由一樣）；ffsubsync 是 Python，映像裡沒有。
- **ffmpeg 讀片的速度**：《Tears of Steel》（372 MB）整條音軌跑一次 silencedetect 只要 **1.07 秒**。但 4K remux 光抽音訊就要 4 分 55 秒（`project_unraid_nas_access` 2026-09-18 紀錄），所以**不能每次都讀整條音軌**，要抽樣（AC #3）。
- **現有零件**：
  - `subtitle/srt_parser.go:23` `ParseSRT`：讀字幕。
  - `subtitle/inventory.go` `ListSidecars`／`sidecarLanguage`：找片旁字幕、判斷語言。`-a` 之後，`.vido.en.srt` 也讀成 `en`。
  - `services/asr_chunk_store.go`：用 `cache_entries` 存快取的寫法，`type` 欄位分種類。
  - miner 的 `minUsableCues = 20`（`miner.go:745`）：「少於 20 句不算完整字幕」的先例。
- **`.vido.en.srt` 不是使用者的字幕**（`-a` Completion Notes 的提醒）：它是 Vido 自己的聽寫稿，本來就跟影片對得上。本單只檢查使用者的檔，要不要拿 `.vido.en.srt` 當翻譯來源由 `-b2` 決定。

## Story

身為把英文字幕放在影片旁邊的 NAS 使用者，
我想要 Vido 在用它翻譯之前，先幫我確認它跟影片對得上，
這樣我不會花錢翻出一份時間全錯的中文字幕。

## Acceptance Criteria

1. **一個入口、四種結果。** 新增 `subtitle.CheckSidecarAlignment(ctx, mediaPath string, audioStream int, sidecarPath string) (AlignmentVerdict, error)`，回傳下面其中一種：
   - `aligned`：對得上，可以直接翻。
   - `offset`：整條差固定秒數，附上 `OffsetSeconds`（例如 `+3.2`，正數表示字幕比聲音晚）。
   - `mismatch`：對不上。可能是越後面差越多、找不到一致的平移量，或根本是別部片。
   - `unusable`：不是可用的英文字幕。例如 parse 失敗、少於 20 句，或內容不是英文（AC #4）。

   另外附上給 `-b3` 用的數字：檢查了幾段（`Windows`）、每段的最佳平移量（`WindowOffsets`）、分數（`Score`）、花了多久（`Elapsed`）。

   `error` 只用在「檢查本身做不了」的時候，例如沒有 ffmpeg、讀不到檔。這時呼叫端要當成「不知道」，**不能當成 `aligned`**。
2. **算法：人聲偵測＋最佳平移。**
   - 用 ffmpeg 把抽樣到的音訊解成 16 kHz 單聲道 PCM，經過人聲頻段濾波（`highpass=f=300,lowpass=f=3400`）後，用 Go 算每 100 ms 的能量，再用該段能量分布的百分位數定門檻，得到「有人聲／沒有」的序列。
   - 字幕也換成同一個 100 ms 格子的「有字幕／沒有」序列。
   - 在 ±15 秒內，以 100 ms 為一步移動字幕序列，找重疊分數最高的平移量。
   - 判斷規則：
     - 每段的最佳平移量都接近 0，而且分數明顯高於隨便亂放 → `aligned`。
     - 每段的最佳平移量一致、但不是 0 → `offset`。
     - 各段的平移量不一致，或沒有明顯高峰 → `mismatch`。
   - 「接近 0」「明顯高於」「一致」三個門檻**都用考卷量出來**（AC #5），不要憑感覺寫死。
3. **抽樣：不讀整條音軌。** 從影片中挑 N 段固定長度的音訊，分散在片頭到片尾，避開片頭和片尾字幕區。每段用 `-ss` 放在 `-i` 前面做快速跳轉，所以只讀那幾段。片長來自 ffprobe。
   - 預設先試 6 段 × 90 秒，實際數字用考卷調。
   - 目標：《Tears of Steel》和一部 4K remux 在正式機上都 **≤ 15 秒**，Completion Notes 要附實測時間。
   - 片太短、湊不滿 N 段時，就整條讀。
4. **英文判斷（便宜版）。** 先 parse 字幕：句數少於 20 → `unusable`。再看文字：字母裡至少 90% 是拉丁字母，而且常見英文虛詞（the／you／I／to／a／is…）出現在足夠比例的句子裡，才算英文。這兩個門檻也用考卷量。不做語言模型偵測，也不呼叫任何 API。
5. **考卷（Rule 28 Real-Shape：真片、真字幕，在正式機量）。** 至少 5 部片，每部做 4 種字幕，期望結果固定：

   | 版本 | 怎麼做 | 期望 |
   |---|---|---|
   | 原版 | 《Tears of Steel》官方 `.en.srt`；其他片從片內英文字幕軌抽出（`ffmpeg -map 0:s:N`） | `aligned` |
   | 平移 | 原版每句加 3 秒，另做一份減 2 秒 | `offset`，誤差 ≤ 0.3 秒 |
   | 影格率錯 | 原版時間 × 25／23.976 | `mismatch` |
   | 別部片 | 換成另一部片的原版字幕 | `mismatch` |

   - 片子要涵蓋：配樂很滿的（《Tears of Steel》）、對白很多的影集一集、動作片、一部 4K remux、一部 30 分鐘內的短片。
   - 考卷腳本和結果表放在 `_bmad-output/audit/sidecar-alignment-exam-2026-10.md`。只寫片名，**不寫片庫路徑和檔名**（repo 是公開的）。
   - **全部 20 格都要對**才能交付。如果有一格錯，要在 Completion Notes 寫清楚錯在哪、為什麼，並請 Alexyu 裁定，不能自己放寬期望。
6. **快取。** 結果存進 `cache_entries`，`type = 'sidecar_alignment'`。key 由下面這些組成：媒體路徑＋大小＋mtime、字幕路徑＋大小＋mtime、音軌編號、算法版本號。
   - 任何一個檔案改了，就重算。
   - TTL 30 天，跟 `asr_chunk` 一樣。
   - 第二次問同一組檔案時不跑 ffmpeg，測試要證明這件事。
7. **單元測試（不需要 ffmpeg，CI 會跑）。** 下面這些都做成 PURE 函式，各自測試：
   - 能量 → 有沒有人聲的序列
   - 字幕 → 格子序列
   - 最佳平移搜尋
   - 判斷規則
   - 抽樣位置
   - 英文判斷

   測試用合成序列，涵蓋：完全對齊、平移 3 秒、等比拉長、隨機（亂放）、少於 20 句、中文內容。

   另外有一個整合測試：用 ffmpeg 合成「有聲段／無聲段」交錯的音訊，再配上對應的字幕。這個測試沒有 ffmpeg 就 skip，跟 `-a` 一樣。
8. **不接線。** 本單不改 `SelectAndRoute`、`PredictFromTracks`、估價 API 或前端。這個函式只被測試和考卷腳本呼叫。`router_test.go:546` 的鎖定測試留給 `-b2` 改。
9. `go test ./...`、`~/go/bin/staticcheck ./...`、`lint:all`（看「✖ N problems (X errors」那行）都要綠。

## 不做

- 路由和估價改用片旁字幕 → `-b2`。
- 對話框怎麼說、讓使用者怎麼選 → `-b3`（Sally 先畫稿）。
- 偵測到 `offset` 時自動幫使用者把時間軸修好（整條平移）。這是新功能，等 `-b3` 的設計討論時再問 Alexyu 要不要做。
- 片旁簡中字幕直接轉繁 → 不在本系列。

## Dev Agent Record

### Completion Notes

（dev 填）
