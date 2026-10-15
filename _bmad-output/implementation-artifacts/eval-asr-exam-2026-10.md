# 聽寫考卷 2026-10：Whisper、Voxtral、Scribe v2、VoiceStudio

Story：`spike-asr-exam-2026-10`。分數是程式算的（`eval/asr-exam-2026-10/score.py`），答案是片子附的官方英文字幕，**不是 AI 評分**。

## 考卷

- 材料：《末日光明》S01E02 前 10 分鐘（`See.S01E02.clip10.mkv`，0–4 分純配樂、4:21–4:55 打熊、之後對話）。送出去的音檔跟 Vido 正式送的格式一樣（16kHz 單聲道 wav）。
- 答案：同一個 mkv 的第 8 軌（官方英文），前 10 分鐘 67 句（`eval/asr-exam-2026-10/answer.en.srt`）。其中兩句「Chet-chet!」是舌頭彈的聲音，沒有一家抓得到，所以漏句率的地板是 3.0%。
- 每家考 3 次，原始輸出在 `eval/asr-exam-2026-10/runs/`。

## 及格線（開跑前登記在 story）

亂補率 ≤ 5%、漏句率 ≤ 10%、開口時間誤差中位 ≤ 0.5 秒且 p90 ≤ 1.0 秒；3 次都過才算過。錯字率只排名。

## 結果（2026-10-14）

| 候選 | 參數 | 句數 | 亂補率 | 漏句率 | 時間中位 | 時間 p90 | 錯字率 | 3 次 |
|---|---|---|---|---|---|---|---|---|
| **ElevenLabs Scribe v2** | `scribe_v2`, `language_code=en`, 字級時間, 音效標記 | 81–83 | 0.0／1.2／1.2% | 3.0% ×3 | 0.11s | 0.18–0.19s | 5.7／7.2／5.9% | **✅ 3/3** |
| OpenAI whisper-1（現在用的） | Vido 正式參數（`language=en`、字級＋段級時間） | 66–79 | 1.4／0.0／8.9% | 11.9／17.9／10.4% | 0.26–0.28s | 0.63–0.92s | 13.4／17.5／11.6% | ❌ 0/3 |
| Mistral Voxtral Mini Transcribe 2 | `voxtral-mini-latest`, `language=en`＋段級時間 | 57 ×3 | 0.0% | 20.9% ×3 | 0.11s | 0.25s | 17.3% ×3 | ❌ 0/3 |
| 同上，不給 `language` | 段級時間 | 55 ×3 | 0.0% | 23.9% ×3 | 0.11s | 0.23s | 18.3% ×3 | ❌ 0/3 |
| （對照）Vido 正式跑那次的 whisper-1，經過幻覺過濾 | 2026-10-14 早上 | 81 | 21.0% | 9.0% | 0.29s | 0.95s | 12.6% | ❌ |
| **VoiceStudio 底下的引擎**：faster-whisper large-v3，CPU int8，Silero VAD（Mac M5 Pro 跑，同一顆引擎、同一份權重） | `language=en`, beam 5, `vad_filter=True`, 字級時間 | 56 ×3 | 8.9% ×3 | 14.9% ×3 | 0.43s | 0.59s | 21.9% ×3 | ❌ 0/3 |

### 細看

- **Scribe v2**：打熊那 35 秒 10 句全對（「Hey, face me!」「Come to me!」），熊吼、嬰兒哭標成 `[bear roaring]`／`[baby crying]`。漏的 3% 就是兩聲「Chet-chet!」。被算成亂補的 1 句是「Jerlamarel. … Jerlamarel!」兩句官方被併成一句 7 秒（切句不同，不是亂補）。三次結果幾乎一樣。
- **whisper-1**：三次差很多。沒有一次把打熊段聽對；有一次亂補 7 句、有一次漏 12 句。正式跑那次（經過 Vido 的幻覺過濾）亂補 21%，就是 Alexyu 看到「喬！」「你還好嗎？」那份。
- **Voxtral**：不亂補，但「遇到吵就閉嘴」：打熊 6 句全漏，後面安靜的對話也漏了 8 句（「May I have your hand?」「Where have you been?」）。文件說 `language` 和時間戳不能併用，實測可以，而且給 `language` 少漏 3 句。回傳格式跟 OpenAI 相似但 segments **沒有** `no_speech_prob` 那三個欄位。
- **faster-whisper（VoiceStudio 的引擎）**：三次一模一樣（本機跑是確定性的）。打熊 6 句全漏——VAD 把那 35 秒當成「沒人說話」直接切掉；另外 5 句被算成亂補，其實是真台詞但時間飄掉（「Charlie Monroe, is that you who took them away?」把 329–347 秒的四句併成一句、壓在錯的時間上），所以時間誤差也是四家裡最大的。錯字率 22%。**Mac M5 Pro CPU 10 分鐘跑 80 秒**（即時倍率 7.5×）；NAS i5-12400 估 2～3 倍慢，一集約 15～20 分鐘。
- **人名**：三家都錯（Jerlamarel → Chulumerel／Trillium Morrow／Julian Monroe）。這要靠翻譯側名詞表，不是聽寫能解的。

### 評分程式修過兩次（修的是量法，不是及格線）

1. Scribe 逐字輸出拼成句子時少了空格，錯字率算成 96%。
2. 「開口時間」原本拿「最早跟官方句子沾到邊的輸出」量，官方一句常是輸出的三句（「Who's there? Answer me. Who's there?」），抓到中間那句就多 1.3 秒。改成「真的壓到這句時間的輸出裡最早開口的」。改前連 Scribe 的 p90 都 1.36 秒，改後 0.18 秒；排名沒變。

## 錢

- 考試：Scribe 用免費額度 30 分鐘（每月 4.5 小時）；Voxtral 在免費方案（Experiment，資料會被拿去訓練）內；whisper-1 約 US$0.18。
- 正式：Scribe $0.22／小時、whisper-1 $0.36／小時、Voxtral $0.18／小時。
- ElevenLabs 免費方案不能商用、要標註來源 → Vido 只能說「比較便宜」，不能說「免費」。

## 結論（照 story 事先寫的規則）

- Scribe v2 過關。⚖️ **Alexyu 2026-10-14 選 A：預設換成 Scribe，Whisper 保留可選**（沒填 ElevenLabs 金鑰就自動退回 Whisper）。
- 要寫的東西：ElevenLabs 的接頭（格式不是 OpenAI 的；逐字→句子的切法可以照 `score.py` 的 `scribe_cues`）、幻覺過濾改成不依賴 `no_speech_prob`（Scribe 沒有，但它自己會把音效標出來）、費率表加第二家（`budget.go` 的 `whisperPerMinuteUSD` 是常數）、金鑰設定頁加 ElevenLabs（選用）。
- Voxtral 不過；分數記著。
- VoiceStudio：引擎本身在 Mac 上已考過，不過關；NAS 上的 Docker 只剩「速度」可量，準確率不會不同（同一顆引擎、同一份權重）。

## 沒做的事

- 中文考題：NAS 上的中文劇都沒有字幕軌，找不到答案。
- 長片整段漏掉（2026-08 對 Scribe 的疑慮）：這份考卷只有 10 分鐘，看不到。開單後第一次在正式機跑整集時要看。

## 補考：faster-whisper 照 FB 建議調參數（2026-10-15）

觸發：FB Ming Jwo 建議 faster-whisper Large-v3-Turbo 開 `vad_filter=True`、關 `condition_on_previous_text`，可避開配樂崩段（`disc-2026-10-asr-exam-faster-whisper-vad-retake`）。同一份考卷、同一段 10 分鐘、Mac M5 Pro CPU int8、各 3 次（確定性，三次一樣）。

| 設定 | 句數 | 亂補率 | 漏句率 | 時間中位 | 時間 p90 | 錯字率 | 3 次 | 每 10 分鐘耗時 |
|---|---|---|---|---|---|---|---|---|
| large-v3，VAD（原本那次） | 56 | 8.9% | 14.9% | 0.43s | 0.59s | 21.9% | ❌ 0/3 | 78 秒 |
| large-v3，VAD＋`condition_on_previous_text=False` | 36 | 11.1% | 13.4% | 0.51s | 1.13s | 21.6% | ❌ 0/3 | 56 秒 |
| **large-v3-turbo**，VAD＋`condition_on_previous_text=False` | 53 | **3.8%** | 19.4% | 0.31s | 0.51s | 18.6% | ❌ 0/3 | 27 秒 |
| （對照）ElevenLabs Scribe v2 | 81–83 | 0～1.2% | 3.0% | 0.11s | 0.18s | 5.7～7.2% | ✅ 3/3 | — |

- turbo 把亂補壓到門檻內（配樂段不再編台詞，Ming Jwo 說的對），但**漏句 19.4%**：打熊那段 VAD 還是整段切掉。
- large-v3 關掉 `condition_on_previous_text` 反而句數變少（36）、時間誤差變大。
- 結論不變：本機 faster-whisper 省錢可用，不是預設；預設仍是 Scribe v2。原始輸出 `eval/asr-exam-2026-10/runs/faster-whisper-{nocond,turbo-nocond}-{1,2,3}.json`，腳本 `fw_run2.py`（session scratchpad）。
