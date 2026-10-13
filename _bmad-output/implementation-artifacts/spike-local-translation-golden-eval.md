# Story spike-local-translation-golden-eval: 用 200 句考卷量「NAS 本機翻譯模型」的繁中品質與速度

Status: done

<!-- SM Bob create-story 2026-10-13。來源：2026-10-13 party mode（founder-pm／Winston／Murat／John），Alexyu 選 A：
     「先改評分程式、用 NAS 跑一晚考兩個本機模型，過關才開單做『本機翻譯』選項」。
     觸發：FB 招募文底下 Edward Fang 留言（用 NAS 上的 Whisper＋本機小模型做完、不燒 Claude API）、
     GiftedJaguar（只有日文片）、黃柏銘（不想花錢）。研究：Claude Desktop 2026-10-13 查證（TranslateGemma、HY-MT1.5、NAS 硬體瓶頸）。
     行號為 origin/main bfed599d。 -->

## Story

身為想讓 Vido 的字幕翻譯不用付 Claude 錢的 NAS 使用者（以及要決定做不做「本機翻譯」選項的 Alexyu），
我要 Vido 用同一份 200 句考卷，考過在 NAS 上用 Ollama 跑的翻譯模型，並量出一集要跑多久，
這樣「本機翻譯要不要做」是靠我們自己的分數決定，不是靠原廠自己測的分數或一則留言。

## 這張單要回答的問題（事先登記，開跑前不得修改）

| 結果 | 條件（沿用 eval-1 AC #4 與 `internal/eval/grade.go:7-8`） | 結論 |
|---|---|---|
| **過關** | 任一本機模型：0 分率 ≤ 5% **且** 2 分率 ≥ 60% **且** 換算一集（835 句）≤ 60 分鐘 | 開一張新單做「本機翻譯」選項（含翻譯 provider 分派） |
| **品質過、速度沒過** | 品質門檻過，但一集 > 60 分鐘 | 記進待辦，註明「有 GPU 或更快的 CPU 才實用」，交 Alexyu 決定要不要以「慢但免費」上線 |
| **不過關** | 兩個模型品質都沒過 | 分數記進待辦，半年後（2027-04）換新模型再考 |

- 「一集 835 句」：eval-1 九部片 10,304 句 ÷ 740 分 = 每分 13.92 句（`bugfix-generation-estimate-under-quotes.md:28`），一小時約 835 句。換算：`200 句的分鐘數 × 835 / 200`。
- 分數是 **AI 評分**（Claude 評），報告與結論都要標明（`#775` 的教訓：`model_ratings.json` 把 AI 評分標成 human）。

## Acceptance Criteria

1. **評分器能考 OpenAI 相容端點的模型。** `apps/api/cmd/grade` 新增旗標：
   - `--openai-base-url`（例如 `http://192.168.50.52:11434/v1`）與 `--openai-model`（例如 `translategemma:12b`）。兩者一起給時，**受考模型**改走新的 OpenAI 相容 completer；沒給時行為與現在完全相同（既有 `main_test.go` 一字不改且全綠）。
   - 給了 `--openai-base-url` 時，跳過「只能考 Claude 目錄模型」的檢查（`main.go:130-140`），但**評審仍是 Claude**（`--judge-model` 預設 `claude-sonnet-5`，`internal/eval/judge.go:19`），所以仍需要 `CLAUDE_API_KEY`（除非 `--no-judge`）。
   - 報告的 `model_id` 寫成 `local:<model>`（例：`local:translategemma:12b`），不會跟目錄裡的模型 id 撞名；`--merge` 拒收 `local:` 開頭的報告（本機模型不進內建等級表）。
   - 可選的 `OPENAI_COMPAT_API_KEY` 環境變數：有值才帶 `Authorization: Bearer`；Ollama 預設不用。
2. **新的 completer 放在 `internal/ai`，實作既有的 `ai.TextCompleter`**（`internal/ai/provider.go:46-48`：`CompleteText(ctx, systemPrompt, userPrompt, maxTokens)`），打 `POST {base}/chat/completions`，messages 為 system＋user，讀 `choices[0].message.content`。
   - **逾時要可調、而且不受 180 秒上限限制**：現有的逾時表上限是 `MaxRequestTimeout = 180s`（`internal/ai/timeout.go:33`），沒 GPU 的 CPU 跑 10 句一批很可能超過。新旗標 `--call-timeout`（預設 10 分鐘）。
   - 有限次重試（例如 2 次，ctx-aware）；4xx（模型不存在、請求格式錯）不重試、直接失敗並印出端點回的錯誤內容。
   - 不記花費（本機 $0）：不呼叫 `Budget.RecordLLM`（`internal/ai/budget.go:210`）；評審的花費照舊由 Claude provider 記。
   - 這個 completer 這次**只給評分器用**，不接到 App（App 的翻譯分派是 `backlog-gemini-translation-dispatch` 的範圍）。
3. **兩種提示方式都要考**，用 `--prompt-style production|native` 切換：
   - `production`（預設）：走現在的 `services.TranslationService`（`main.go:164`、`:182`），也就是正式版的提示詞（`m1-v5`，`internal/ai/prompts/subtitle_translator.go:33`）、一批 10 句（`:41`）、前文 5 句（`:37`）、`[N]` 編號輸出格式（`:74-87`）。這代表「不改產品提示詞就接上去」會得到的品質。
   - `native`：用模型自己的官方提示格式，**一句一呼叫**（或很小的批次），把前 2～3 句英文當上下文帶進去（HY-MT 官方有「上下文翻譯」模板），輸出直接當譯文，不解析 `[N]`。這代表「為小模型另寫一條翻譯路」能得到的品質。只需支援兩個模型家族：TranslateGemma、HY-MT（家族用 `--openai-model` 字串判斷，或 `--native-template translategemma|hymt` 明確指定）。
   - 兩種都經過同一個 finalizer（OpenCC s2twp＋台灣用語詞庫，`main.go:244-255`）再給評審，規則層照舊讀**原始輸出**（`internal/eval/rules.go:27-37` → `subtitle.CheckChunk`），跟正式版的品質閘門一致。
4. **批次大小可調**：`--batch-size`（只影響 `production` 模式）。⚠️ 正式版的 `SubtitleTranslatorBatchSize = 10` 是常數（`subtitle_translator.go:41`，`translation_service.go:412` 直接讀），**不要為了這張單改成可設定的產品參數**。做法二選一，dev 開工時挑最小的：(a) 在 `TranslationService` 加一個 `TranslateOption`（例如 `WithBatchSize(n)`，預設不變）；(b) `production` 模式只跑 10，另外記錄「小模型建議 1／5」留給產品單。選 (a) 要有單元測試證明預設仍是 10。
5. **報告多記三件事**（加在 `eval.Report` 或 `cmd/grade` 的輸出外層，不能破壞既有 `--merge` 讀取的欄位，`internal/eval/run.go:55-80`）：
   - `episode_minutes_est`：`minutes × 835 / cues`（`Minutes` 已有，`run.go:212`）。
   - 大陸用語與簡體：`rule_fails_by_reason` 裡的 `simplified_leak`（`internal/subtitle/quality_gate.go:19`、`:105`）與 `forbidden_term`（`rules.go:16`）分開列出比例。
   - `prompt_style`、`batch_size`、`base_url`（不含金鑰）、`judge_note` 明寫「AI 評分（Claude 評）」。
6. **Real-shape 測試（Rule 28）**：completer 的測試要用**從 NAS 上的 Ollama 錄下來的真實回應**當 fixture（成功一筆、模型不存在的 404 錯誤一筆），放在 `apps/api/internal/ai/testdata/`。另有 httptest 測試覆蓋：逾時、重試次數、4xx 不重試、空內容回 `ErrAIInvalidResponse`（照 Claude provider 的規則）。
7. **在 NAS 上實際跑完並寫結果文件。** ⚠️ **在 NAS 上起 Ollama 容器、下載模型之前，先問 Alexyu 同意**（會佔約 15GB 磁碟、跑的時候吃 CPU 與記憶體，可能影響同一台 NAS 上的 Vido 與其他服務）。
   - 受考模型：**TranslateGemma 12B**（`translategemma:12b`）、**HY-MT 7B**（HY-MT1.5-7B；若騰訊已出更新的 Hy-MT2-7B 且有 GGUF，改考新版並註明）。基準：`claude-sonnet-5`（同一份考卷，`production` 模式）。
   - 每個本機模型跑 `production` 與 `native` 兩種；總共 2 模型 × 2 方式 ＋ 1 基準 = 5 份報告（`--trace` 開著，存每句原文／譯文／分數）。
   - 評分器在 Mac 上跑、透過區網打 NAS 的 Ollama，不需要把評分器裝到 NAS。
   - 結果寫進 `_bmad-output/implementation-artifacts/eval-local-translation-golden-2026-10.md`：5 份的分數表（0 分率、2 分率、等級、`simplified_leak`／`forbidden_term` 比例、200 句耗時、換算一集分鐘）、NAS 硬體與當時負載、模型 tag 與量化版本、每個模型挑 5 句好的與 5 句壞的對照、**套用上面事先登記的表得出的結論**。
8. **依結論立案**（Rule 24）：過關 → 在 `sprint-status.yaml` 開「本機翻譯選項」新條目（引用本單與結果文件）；不過關 → 開 backlog 條目寫「2027-04 換新模型重考」並附分數。兩種都要在結果文件裡寫出條目 id。
9. **花費上限**：評審（Claude）每份報告 ≤ US$0.50（`main.go:115` 的預設 `--budget-usd`），5 份合計 ≤ US$2.50。超過就停下來查，不要調高上限硬跑。

## Tasks / Subtasks

- [x] **Task 1 — OpenAI 相容 completer（AC: #2, #6）**
  - [x] 1.1 `internal/ai/openai_compat.go`：`NewOpenAICompatProvider(baseURL, model, opts...)`，實作 `TextCompleter`；逾時、重試、錯誤分類
  - [x] 1.2 httptest 單元測試（逾時／重試／4xx／空內容）
  - [x] 1.3 NAS 上 Ollama 起來後，錄真實回應 fixture（成功＋404），補 real-shape 測試
- [x] **Task 2 — 評分器旗標與兩種提示方式（AC: #1, #3, #4, #5）**
  - [x] 2.1 `--openai-base-url`／`--openai-model`／`--call-timeout`／`--prompt-style`／`--batch-size`／`--native-template` 解析與驗證；`model_id` = `local:<model>`；`--merge` 拒收 `local:`
  - [x] 2.2 `native` 模式的 `eval.Translator`（TranslateGemma 與 HY-MT 兩種官方模板，一句一呼叫＋前文）
  - [x] 2.3 報告加 `episode_minutes_est`、`simplified_leak`／`forbidden_term` 比例、`prompt_style`、`batch_size`、`base_url`
  - [x] 2.4 `main_test.go` 新增案例；既有案例不改、全綠
- [x] **Task 3 — NAS 實跑（AC: #7, #9）** ⚠️ 先取得 Alexyu 同意
  - [x] 3.1 Unraid 起 Ollama 容器（CPU），`ollama pull` 兩個模型；`ollama show <model> --template` 記下內建模板
  - [x] 3.2 跑 5 份報告（`--trace`），記錄當時 NAS 負載
- [x] **Task 4 — 結果文件與立案（AC: #7, #8）**
  - [x] 4.1 寫 `eval-local-translation-golden-2026-10.md`，套用事先登記的表
  - [x] 4.2 依結論在 `sprint-status.yaml` 開條目，雙向引用

（後端 2、腳本／維運 2、前端 0 —— 不觸發拆分。）

## Dev Notes

### 查到的事（2026-10-13，Bob 逐一開檔確認）

- **評分器現在只會建 Claude**：`cmd/grade/main.go:83-85` 的 `newCompleter` 固定回 `ai.NewClaudeProvider`；`:130-140` 擋掉非 Claude 目錄模型（註解指向 `backlog-gemini-translation-dispatch`）。`deps.newCompleter` 是測試用的縫（`:59-65`），新 completer 從這裡接最乾淨。
- **評分器跑的是真的翻譯服務**：`:164` `services.NewTranslationService(completer, nil)`、`:182` `TranslateWithGlossary(...)`，每 20 句一塊（`:38`、`:170-198`），預算碰頂時保留已翻的部分。
- **finalizer = OpenCC s2twp＋台灣用語詞庫**（`:244-255`，`zhtw.Finalize`）。OpenCC helper 不在本機時會跳過 OpenCC（`:247`），所以**在 Mac 上跑要確認 OpenCC helper 有裝**，不然本機模型吐簡體時分數會被低估；或加 `--no-opencc` 時在報告註明。⚠️ 未查證：Mac 上 helper 是否已安裝（dev 開工時先跑一次 `--limit 5 --no-judge` 看 `simplified_leak`）。
- **規則層讀原始輸出**：`internal/eval/rules.go:27-37` 呼叫 `subtitle.CheckChunk`，`simplified_leak` 判在 OpenCC 之前（`quality_gate.go:105`）——跟正式版閘門一樣。所以本機模型若習慣吐簡體，**即使 OpenCC 救得回來，這裡也會判 0 分**。這是刻意的（正式版閘門也會擋），但結果文件要把 `simplified_leak` 單獨列出，讓 Alexyu 判斷「是不是該先 OpenCC 再閘門」——那是產品單的事，不在本單改。
- **TextCompleter 介面**：`internal/ai/provider.go:46-48`，只有 `CompleteText(ctx, system, user, maxTokens) (string, error)`。
- **逾時是 provider 自己管**：`internal/ai/timeout.go:58-69` `RequestTimeoutFor`，上限 `:33` 180 秒；Claude（`claude.go:57` 註解）與 Gemini（`gemini.go:119`）各自套用。新 completer 用自己的 `--call-timeout`，不要硬套這張表（未知模型的基準值對 CPU 推論太短）。
- **正式版批次常數**：`prompts/subtitle_translator.go:37`（前文 5）、`:41`（一批 10），`translation_service.go:412-413` 直接讀；解析 `[N]` 在 `translation_service.go:825` `parseTranslationResponse`。
- **評審**：`internal/eval/judge.go:19` `DefaultJudgeModel = "claude-sonnet-5"`。等級門檻 `internal/eval/grade.go:7-8`（0 分率 ≤ 0.05、2 分率 ≥ 0.60）。報告欄位 `internal/eval/run.go:55-80`，`Minutes` 在 `:212`。
- **200 句考卷從沒正式跑過**：`model_ratings.json` 兩筆都來自 eval-1（`source: eval-1`），`git log` 只有 #641 改過它；`model-grade.yml` 沒有執行紀錄（見 `disc-2026-10-model-ratings-judge-mislabeled`，#775）。所以這次的 Sonnet 基準也是 golden-v1 第一次被拿來考 Sonnet。
- **NAS 硬體**：i5-12400、12 執行緒、31 GiB、沒有 GPU（`sprint-status.yaml` 的 `backlog-compute-aware-asr-default` 條目）。一般使用者多是 J4125／N5095 等級（同條目），**這次的速度是樂觀上限**，結果文件要寫明。
- **Vido 已能接自架語音辨識**：`ASR_BASE_URL`（`docs/deployment.md:163-167`）。本單只處理翻譯那段。

### 已知陷阱

- **TranslateGemma 的輸入上限約 2K tokens**（第三方教學引述的訓練設定，⚠️ 未查證官方數字）。正式版提示詞每批約 3,400 輸入 tokens（`bugfix-generation-estimate-under-quotes.md:20`：66 次呼叫、227,210 輸入 tokens），`production` 模式很可能被截斷或亂掉。**這本身就是要量的結果**，不要為了讓它過而偷偷縮短正式版提示詞。
- **兩個模型都有自己的官方提示格式**：TranslateGemma 要單一 user 訊息、指明來源與目標語言；HY-MT 的英翻中模板是「Translate the following segment into {target_language}, without additional explanation.」，另有上下文與術語模板。Ollama 可能已經把聊天模板內建進模型，`native` 模式送之前先 `ollama show <model> --template` 確認，避免包兩層模板。目標語言要寫「Traditional Chinese (Taiwan)」／「繁體中文（台灣）」，並在結果文件記錄實際用的字串。
- **HY-MT1.5-7B 沒有官方 GGUF**（官方只有 FP8／GPTQ-Int4）；社群有 `mradermacher/HY-MT1.5-7B-GGUF`。騰訊後來出了 **Hy-MT2**（1.8B／7B／30B-A3B），官方 GGUF 用 `ollama run hf.co/tencent/Hy-MT2-…-GGUF:Q4_K_M` 的形式。dev 開工時先查 Hy-MT2-7B 有沒有官方 GGUF，有就考新版。量化統一用 Q4_K_M，並記在結果文件。
- **CPU 推論很慢**：研究引述沒 GPU 的 NAS 跑 4B 模型每秒約 5.4 tokens。`native` 一句一呼叫 × 200 句，一份報告可能要一兩個小時；`--limit 20` 先試跑。
- **不要動產品程式的行為**：`TranslationService` 只允許加「預設不變」的 option（AC #4），`main.go`（App）與翻譯分派一律不碰。
- **Sakura（日文）不在本單**：只支援日譯簡中、授權禁止商用；日文需要的是「原文語言不是英文」的整條流程改動，另案。

### 不要做的事

- 不做產品 UI、不做設定頁、不接 App 的翻譯流程（→ 過關後的新單）。
- 不把本機模型寫進 `model_ratings.json`（`--merge` 要拒收 `local:`）。
- 不在 NAS 上裝任何東西，直到 Alexyu 同意。
- 不調高評審預算硬跑（AC #9）。

### Rule 27（外部整合五柱）

這次的 completer 只給評分器用、打的是使用者自己的 Ollama，不是對外的第三方資料來源；本單只要求**逾時＋有限重試＋錯誤分類**。之後若接進 App，由那張產品單補齊五柱（限流、快取、降級等），並在結果文件「下一步」寫明。

### Project Structure Notes

- 新檔：`apps/api/internal/ai/openai_compat.go`（＋`_test.go`、`testdata/`）。
- 改：`apps/api/cmd/grade/main.go`（＋`main_test.go`）；可能小改 `apps/api/internal/services/translation_service.go`（只加 option）。
- 文件：`_bmad-output/implementation-artifacts/eval-local-translation-golden-2026-10.md`。
- 報告 JSON（含 `--trace` 逐句）存在 `eval/local-translation-2026-10/`；`.gitignore:111-116` 只忽略 `eval/` 下的 `.srt`／`.ass`／`sheet*.csv`／`compare.csv`，JSON 報告不受影響，逐句譯文來自 CC 授權考卷，可以進 repo。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（不動前端）。

### References

- party mode 2026-10-13（本對話）；`project_byok_moat_direction`（模型是大宗商品、護城河是詞彙表）；founder-pm 原則「新功能不能讓必填的 API key 變多」
- `sub-7-8a-golden-sample-and-grader.md`（考卷來源與授權、評分器設計）、`sub-7-8b-model-ratings-embed-ci.md`
- `eval-1-translation-blind-eval.md` AC #4（門檻）、`:224-226`（AI 評分要標明）
- `backlog-gemini-translation-dispatch`（翻譯分派，過關後的產品單依賴它）、`backlog-compute-aware-asr-default`（NAS 硬體）、`9R-S2-nas-whisper-benchmark-spike`
- 外部（2026-10-13 查）：[Lingarr](https://github.com/lingarr-translate/lingarr)（已支援 Ollama）、[HY-MT1.5 技術報告](https://arxiv.org/html/2512.24092v1)、[HY-MT1.5-1.8B 模型卡（提示模板）](https://huggingface.co/tencent/HY-MT1.5-1.8B)、[Hy-MT2-1.8B GGUF](https://huggingface.co/tencent/Hy-MT2-1.8B-GGUF)、[TranslateGemma 指南](https://aicybr.com/blog/translategemma-guide)

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）；code review 由 Fable（跨模型）負責。

### Debug Log References

- 考試紀錄：scratchpad `run-all.log`（5 份依序跑，20:58～22:09）。
- NAS：`/mnt/cache/appdata/ollama`（模型 12G）、`/mnt/cache/appdata/llamacpp`（先前為 llama.cpp 下載的 GGUF，12G，現在沒用到）。Ollama 容器跑完後已停止（`docker stop ollama`），沒有刪除。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **結果**：照事先登記的門檻，兩個本機模型都不過關（第一組）→ Alexyu 2026-10-14 看完結果選 A：現在不做，2027-04 再考（門檻與「半年後再考」原本是 party mode 測試角色的建議，2027-04 是 Bob 寫 story 時算出來的日期，回報時被寫成像已定案的「判定」，10/14 更正）。第二組（`--also-judge simplified_leak,name_mismatch`）：Hy-MT2 7B 翻壞 6.5%、翻得自然 80%，Sonnet 是 3%／92%；一集約 43 分。全文：`eval-local-translation-golden-2026-10.md`。
- **跟計畫不同的地方**（結果文件有完整表格）：
  - 改考 Hy-MT2 7B，因為 1.5 沒有官方 GGUF。
  - 原本要用 Ollama，但 NAS 的 docker.img 40GB 放不下映像。Alexyu 把它放大到 80GB 之後才改回 Ollama。
  - 評審花費 US$3.40，超過預估的 US$2.50：Sonnet 基準第一次跑到 US$0.50 就停了，評審只評到一半。
  - 加了第二組分數，因為第一組被人名規則主導，連 Sonnet 都過不了。
- **AC #4** 選了 (a)：`services.WithBatchSize(n)`，預設不變，`TestTranslateWithGlossary_BatchSizeOption` 證明預設仍是 10。
- **AC #3 的範圍多了兩件事**：
  - `eval.Options.AlsoJudge`（第二組分數，不改第一組的等級）。
  - `eval.RejudgeSecondary`／`grade --rejudge`（評審回壞掉的 JSON 時補評）。
  - 兩件都有單元測試。
- **TranslateGemma 的目標語言**：只寫「Chinese」會吐簡體，改成「Traditional Chinese」（`native.go` 註解有寫）。
- **AC #6**：real-shape fixture 來自 NAS 上 Ollama 0.40.3 的真實回應，`internal/ai/testdata/openai_compat_ollama_{ok,404}.json`。

### Code Review（2026-10-13，Fable 5.1 跨模型）

1 HIGH、2 MEDIUM、4 LOW，全部修掉：

- **HIGH**：第二輪評分如果撞到預算，以前會把整份報告標成「未完成」，連第一組的等級也一起沒了，而且 `--rejudge` 救不回來。
  - 現在第二輪停下來時，只把剩下的句子記進 `secondary_unjudged`，原因寫在 `secondary_incomplete`，第一組照常算分。
  - 測試：`TestRun_AlsoJudge_BudgetStopKeepsHeadline`。
- **MEDIUM**：`--rejudge` 撞到預算時，存下來的報告前後數字對不上，而且結束碼是 0（看起來像成功）。
  - 現在不管從哪裡結束，都會先把計數和分數重算好再存檔；撞到預算時改成非零結束碼。
  - 測試：`TestRejudge_BudgetStopSavesAndFails`。
- **MEDIUM**：考 Claude 時也能加 `--batch-size`，這樣考出來的分數條件跟正式版不同，卻能合併進內建等級表。現在拒收。
- **LOW**：
  - `--rejudge` 現在會比對考卷版本。
  - 第二輪評分也會存評審的理由（`why`）。
  - 讀回應內容時如果被呼叫端取消，現在回 `context.Canceled`。
  - 「換算一集」原本把評審的時間也算進去了，現在改用新欄位 `translate_minutes`，只算翻譯本身。另外用 `--no-judge` 重量了 4 份本機模型的翻譯時間（`eval/local-translation-2026-10/timing/`），結果文件的「換算一集」照這 4 份更新。

### Discovery Triage

| 發現 | Lane | 條目 |
|---|---|---|
| golden-v1 人名規則偏嚴，加上不帶詞彙表時 Sonnet 會留英文人名，連 Sonnet 都只拿 B | ③ | `disc-2026-10-golden-name-rule-calibration` |
| `cmd/grade` 預算預設 0.50 不夠 Sonnet 考完（CI 的 Model Grade 會「未完成」）；評審偶爾回壞掉的 JSON | ③ | `disc-2026-10-model-grade-budget-and-judge-reliability` |
| Alexyu 選 A，2027-04 重考；如果要提早做，產品端要補「詞彙表進本機提示詞」「先轉繁體再過閘門」 | ③ | `backlog-local-translation-retest-2027-04` |
| 第二輪評審回壞掉的 JSON，會默默略過那一批 | ① | AC #5 範圍內補上 `secondary_unjudged`＋`--rejudge`（本單程式） |
| NAS docker.img 只剩 7.7GB，裝不下 Ollama | — | Alexyu 10/13 自己放大到 80GB，已解決，不用立案 |

### File List

- `apps/api/internal/ai/openai_compat.go`（新）
- `apps/api/internal/ai/openai_compat_test.go`（新）
- `apps/api/internal/ai/testdata/openai_compat_ollama_ok.json`（新）
- `apps/api/internal/ai/testdata/openai_compat_ollama_404.json`（新）
- `apps/api/cmd/grade/main.go`（改）
- `apps/api/cmd/grade/native.go`（新）
- `apps/api/cmd/grade/merge.go`（改）
- `apps/api/cmd/grade/main_test.go`（改，只新增案例）
- `apps/api/internal/eval/run.go`（改）
- `apps/api/internal/eval/run_test.go`（改，只新增案例）
- `apps/api/internal/services/translation_service.go`（改，只加 `WithBatchSize`）
- `apps/api/internal/services/translation_service_test.go`（改，只新增案例）
- `eval/local-translation-2026-10/*.json`（新，5 份逐句報告）
- `_bmad-output/implementation-artifacts/eval-local-translation-golden-2026-10.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（改）
