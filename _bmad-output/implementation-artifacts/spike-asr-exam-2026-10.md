# Story spike-asr-exam-2026-10: 做一份「聽寫考卷」，考 Whisper、Voxtral、Scribe v2、VoiceStudio（NAS 自架）

Status: done

<!-- SM Bob create-story 2026-10-14。來源：2026-10-14 party mode（founder-pm／John／Winston／Murat），Alexyu 選 A 並加考 VoiceStudio。
     觸發：Alexyu 看 See S01E02 前 10 分鐘的 AI 字幕，打熊那段翻出「喬！」「你還好嗎？」——查明是聽寫（whisper-1）把熊吼聽成台詞，不是翻譯錯。
     量測（同段 2 分鐘跑 5 次）：亂補 14／25 句、漏 10 句、整段全丟、幾乎完美各一次；信心數字整段共用，過濾救不了。
     規模：NAS 候選清單 2,179 部要做字幕的片，1,518 部（70%）要走聽寫。 -->

## Story

身為片子沒有任何字幕軌、只能靠聽寫做字幕的 Vido 使用者（以及要決定「聽寫要不要換一家」的 Alexyu），
我要 Vido 用同一份考卷、同一個答案（片子裡附的官方英文字幕），考過現在用的 whisper-1 和三個候選，
這樣「換不換、換哪個」是靠我們自己量出來的分數決定，不是靠原廠榜單或一次試跑的印象。

## 這張單要回答的問題（事先登記，開跑前不得修改）

每個候選**同一份考卷考 3 次**（2026-10-14 已證明單次結果不能下結論）。四個分數：

| 分數 | 怎麼算 | 及格線 |
|---|---|---|
| **亂補率** | 輸出的句子，有一半以上時間落在「官方沒人說話」的區間（官方句子前後各放寬 0.5 秒）→ 算亂補。亂補句數 ÷ 輸出句數 | ≤ 5% |
| **漏句率** | 官方句子完全沒有任何輸出句子跟它重疊 → 算漏。漏掉句數 ÷ 官方句數 | ≤ 10% |
| **時間誤差** | 有對上的句子，開口時間差的中位數／p90 | 中位 ≤ 0.5 秒、p90 ≤ 1.0 秒 |
| **錯字率** | 有對上的句子，正規化後的 WER（只排名用，不當及格線） | — |

- **過關**：3 次都過三條及格線。有一次沒過就不算過（整段消失就是這種情況）。
- **自架的（VoiceStudio）另記速度**：NAS（i5-12400、無 GPU）10 分鐘的片要跑多久。≤ 10 分鐘（即時倍率 ≤ 1×）才算實用；超過就記「能用但慢」，交 Alexyu 決定。

| 結果 | 結論 |
|---|---|
| 有雲端候選過關 | 開單做「聽寫服務可選」：Voxtral 若回傳格式跟 OpenAI 一樣，可能只要改 `ASR_BASE_URL`／`ASR_MODEL` 加文件；Scribe 要新寫接頭。金鑰一律**選用**，不能變必填。 |
| **Scribe v2 分數最高** | Alexyu 2026-10-14：「應該考慮替換成它，因為還有送免費額度」。開單時把「預設改成 Scribe、Whisper 保留可選」列為選項，附上免費額度的實際規則（4.5 小時是不是每月、超過怎麼計費）給 Alexyu 拍板。 |
| VoiceStudio 過關且夠快 | 開單寫「NAS 自架聽寫」安裝指南（Vido 端只是指到它的網址），並接上 Edward 那條「本機模型」線。 |
| 全部沒過 | 留在 whisper-1，分數記進待辦，半年後（2027-04）換新模型再考，跟 `backlog-local-translation-retest-2027-04` 併同一天。 |

## 查到的事（2026-10-14）

- Vido 的聽寫已經可以換端點：容器環境變數有 `ASR_BASE_URL`、`ASR_MODEL`（正式機 `docker inspect Vido` 看到）。`apps/api/internal/ai/whisper.go:484-516` 送的是 OpenAI 的 multipart 格式（`model`、`response_format=verbose_json`、`timestamp_granularities[]`、`language`、`prompt`），幻覺過濾讀 `no_speech_prob`／`avg_logprob`／`compression_ratio`（`whisper_segments.go:442-446`）。
- **VoiceStudio**（github.com/debpalash/VoiceStudio，AGPL-3.0，63k star，2026-10-14 仍在推）：是一個包很多引擎的殼。`backend/api/routers/openai_compat.py` 有 `POST /v1/audio/transcriptions`，`_TRANSCRIPT_FORMATS` 含 `verbose_json`／`srt`，回的 segments 有 `avg_logprob`／`compression_ratio`／`no_speech_prob`（`:925-941`），`timestamp_granularities` 收 `word`／`segment`（`:985-992`）→ **Vido 不用改程式就能指過去**。底下 CPU 可用的引擎：faster-whisper（`Systran/faster-whisper-large-v3`，CPU 跑 int8，前面有 Silero VAD 先切掉沒聲音的段）、FunASR SenseVoice（中文強、只有句級時間）、Parakeet（25 種語言，無中文）。Docker image `ghcr.io/debpalash/voicestudio:stable` **只有 linux/amd64**（Mac arm64 拉不到）；需要 `OMNIVOICE_API_KEY`；首次啟動下載約 4GB 模型。
- **Mistral Voxtral Mini Transcribe 2**：`https://api.mistral.ai/v1/audio/transcriptions`，欄位照 OpenAI（`model=voxtral-mini-latest`、`timestamp_granularities` 收 `segment`／`word`），$0.003/分；文件說 `timestamp_granularities` **不能跟 `language` 一起用**（Vido 現在固定送 `language`）；回傳的 segments 有沒有 `no_speech_prob` 那三個欄位**沒人寫**，要打一次才知道。英文、中文都在 13 種語言裡。開放權重。
- **ElevenLabs Scribe v2**：$0.22/小時，免費額度 4.5 小時；字級時間、音效標記（含 music）；格式不是 OpenAI 的，要新寫接頭。2026-08 記憶有「長片整段漏掉」的回報，考卷要看這條。
- **OpenAI gpt-transcribe／gpt-4o-transcribe**：2026-10-14 實測，前者 3 次全回空字串、後者沒時間戳 → 不考。
- 答案來源：正式機 `See.S01E02…KONTRAST.mkv` 第 8 軌（eng，官方）＝答案；第 9 軌（SDH）有 `[roaring]`／`[music]` 標記，用來標「這段沒人說話」。
- 中文考題：天龍八部、藏海花等中文劇**都沒有字幕軌**，找不到答案。本單先只考英文；中文題另找有官方中文字幕軌的片再補（記在「沒做的事」）。

## Acceptance Criteria

1. **考卷材料**：用現成的 `See.S01E02.clip10.mkv`（NAS `/mnt/cache/appdata/vido-eval-clip/scratch1014/See/Season 1/`，0–10 分鐘：0–4 分純配樂、4:21–4:55 打熊、之後對話）。答案 = 第 8 軌 0–10 分鐘的官方英文句子；非說話區間 = 答案以外的時間。材料與答案抄一份到 repo `eval/asr-exam-2026-10/`（答案 srt 可進 repo；影片不進）。
2. **評分程式**（Python，放 `eval/asr-exam-2026-10/score.py`）：吃一個 verbose_json 或 srt 輸出，印上面四個分數；對同一候選的 3 次輸出印「3 次都過？」。評分規則就是上表，不能有第五種判法。
3. **考試腳本**：每個候選各 3 次，輸出原始 JSON 留檔（`eval/asr-exam-2026-10/runs/<候選>-<n>.json`）。
   - `whisper-1`（基準）：走 Vido 正式的參數（`language=en`、字級時間開）。
   - `voxtral-mini-latest`：先確認回傳格式；`language` 和 `timestamp_granularities` 擇一時，兩種都跑並記下差別。
   - `scribe_v2`：ElevenLabs 的格式，轉成同一種中間格式再評分。
   - VoiceStudio：faster-whisper large-v3（CPU int8）；時間夠再加 FunASR。記每次跑多久。
4. **結果報告** `_bmad-output/implementation-artifacts/eval-asr-exam-2026-10.md`：四個分數 × 3 次 × 每個候選一張表、速度、花費、照上表寫結論；標明答案是官方字幕、分數是程式算的（不是 AI 評）。
5. **金鑰**：Mistral／ElevenLabs 的金鑰由 Alexyu 放進本機 `~/.config/vido-eval/keys.env`（`MISTRAL_API_KEY=`、`ELEVENLABS_API_KEY=`，chmod 600），腳本只讀檔，不進 repo、不印出來。
6. **不做**：不改 Vido 產品程式；不接任何新 provider；不動 whisper 的幻覺過濾。那些是過關之後的單。

## Tasks / Subtasks

- [x] T1 材料與答案：抽第 8 軌 0–10 分鐘成 srt，連同 SDH 標記放進 `eval/asr-exam-2026-10/`
- [x] T2 `score.py` ＋ 用 2026-10-14 已有的 5 份 whisper 輸出先試評（`asr-stats-1014/run1-4.json`、正式跑那份）
- [x] T3 whisper-1 基準 3 次
- [x] T4 Voxtral 3 次（先打一次看格式）
- [x] T5 Scribe v2 3 次
- [x] T6 VoiceStudio：改成在 Mac 上直接跑它底下的引擎（faster-whisper large-v3 CPU int8＋VAD，同一顆引擎同一份權重）3 次；NAS 的 Docker 下載太慢（2 小時只到 2/16 層），Alexyu 2026-10-14 叫停，因為準確率跟引擎走、NAS 跑不會不同
- [x] T7 報告＋ sprint-status 結論

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 → Fable 5.1（claude-fable-5-1，Alexyu 中途切換），2026-10-14。

### Completion Notes List

- 結果見 `eval-asr-exam-2026-10.md`：**Scribe v2 3/3 過**；whisper-1、Voxtral、faster-whisper（VoiceStudio 的引擎）全 0/3。
- 評分程式修過兩次量法（Scribe 逐字拼句少空格；時間誤差要取「真的壓到這句的輸出裡最早開口的」），及格線沒動；修改前後排名相同，細節寫在報告。
- 金鑰：Alexyu 自己申請 Mistral（Experiment 免費方案）與 ElevenLabs（Speech to Text 單一權限），放 `~/.config/vido-eval/keys.env`。兩家都不用儲值。
- 花費：whisper-1 約 US$0.18 ＋ 事前量測 US$0.14；Scribe 用免費額度 30 分鐘；Voxtral 免費方案。
- ⚖️ Alexyu 2026-10-14 裁定 **A：預設換成 Scribe，Whisper 保留可選** → 開單（Bob）。

### File List

- `eval/asr-exam-2026-10/{answer.en.srt,answer.sdh.srt,score.py,runs/*}`
- `_bmad-output/implementation-artifacts/eval-asr-exam-2026-10.md`、`spike-asr-exam-2026-10.md`、`sprint-status.yaml`
