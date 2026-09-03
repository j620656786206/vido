# Story eval-1: 字幕翻譯品質盲測 —— Haiku 4.5 vs Sonnet 5，五部樣本，預先登記的「可看／需校稿」門檻

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m3`（M3，品質驗證，非程式碼 story）· **Risk: 🟢 LOW（operator 實測，零產品程式碼變動；唯一產物是一支 stdlib 腳本）** · **Operator-run（Alexyu 於 NAS 執行）**
**Source:** 2026-09-02 策略對話裁定（Alexyu：「押字幕，先盲測」）。前提結論：免費功能是外殼不是賣點，AI 字幕是唯一鉤子，而其品質從未被量測。
**Supersedes nothing / Blocks:** v0 對外招募內測（字幕功能開放與否）、`CLAUDE_MODEL` 預設值裁定、README 定位改寫（「Plex 旁邊的繁中字幕機器人」）。
**Cross-stack split check:** backend tasks = 0, frontend tasks = 0 → 單一 story。

---

## Story

As the product owner,
I want a blind, pre-registered comparison of the translated subtitles the pipeline actually produces on my own library,
so that「翻出來不用改就能看」is a measured fact before anyone outside is asked to spend their own API key on it.

---

## Context — 這個 story 為什麼存在

2026-09-02 盤點（程式碼實讀，非文件）：

| 事實                                     | 出處                                                                                                                                                 |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| 翻譯預設模型是最便宜的 `claude-haiku-4-5` | `apps/api/internal/ai/claude.go:31`；由 `CLAUDE_MODEL` env 覆寫，**開機時讀一次**（`main.go:679` → `subtitle.WithModelID`），換模型須重啟容器          |
| 每批 10 句、只帶前 5 句作參考             | `prompts/subtitle_translator.go` `SubtitleTranslatorBatchSize=10` / `ContextWindow=5`                                                               |
| 「品質閘門」只檢查格式                    | `subtitle/quality_gate.go`：missing / empty / echoed / simplified_leak 四類；**翻錯意思、生硬、語氣不對一律看不出來**                                    |
| 詞彙表＋metadata 有注入                   | `BuildGlossarySection` / `BuildMetadataSection`；sub-5-5 auto-harvest 已閉環                                                                          |
| 舊管線的 AI 用語校正沒接進新管線          | `prompts/terminology_corrector.go` 只在 legacy Route C；`process_item.go` 無呼叫者                                                                    |
| ASR 路徑繞過閘門                          | `backlog-asr-leg-unify-gated-pipeline`（已立案）                                                                                                      |
| 唯一「實測」是 6 月 The Boys 單集 POC     | `subtitle-v4-replan-and-feasibility-audit-2026-06.md` 附錄 A：n=1、單人看、對照組是誤標簡體的「人工」字幕。**不是品質證據**                              |
| M1 為此建了重跑機制但從未用來量品質        | `RunVersion{MetadataHash,GlossaryVersion,PromptVersion,ModelID}` + `POST /subtitles/pipeline/run {force:true}` + `subtitle_runs.spent_usd`（architecture §M1 Pilot Instrumentation） |

產品定位（2026-08-19 策略重審 + 2026-09-02 對話）明文：**不做字幕校對編輯器**，「要校稿 = 自動化失敗」。所以「可看」不是加分項，是整個對外承諾的前提。本 story 就是去量這個前提。

---

## Acceptance Criteria

### AC #1 — 樣本片單：五部、涵蓋五種對白型態、全部走 `translate` 路徑

**Given** NAS 片庫，**when** 選樣，**then** 五部樣本各屬一類，且 F15 報價清單／route prediction 顯示路徑為 **`translate`**（有內嵌英文文字軌）。**不得**選到 `no_text_source`（ASR）—— 本 story 只量翻譯層，ASR 的斷句與幻覺另案（`backlog-asr-leg-unify-gated-pipeline`、9R-5 VAD）。

| #   | 類型                   | 要考驗的能力                     | 首選（Alexyu 片庫，2026-09-02 挑）                          | 備援（首選沒有英文文字軌時）          |
| --- | ---------------------- | -------------------------------- | ----------------------------------------------------------- | ------------------------------------- |
| 1   | 動作／黑色幽默，髒話多 | 語域、俚語、台灣口語             | 黑袍糾察隊 The Boys S01E01（與 6 月 POC 同集，可直接對照） | 和平使者 Peacemaker S01E01             |
| 2   | 情境喜劇               | 雙關、笑點、快節奏短句           | 泰德拉索 Ted Lasso S01E01（英式俚語＋足球術語＋笑點）       | 外放特務組 Slow Horses S01E01（英式冷幽默） |
| 3   | 劇情／專業術語         | 政治／產業術語、正式語域         | 紙牌屋 House of Cards S01E01（美國國會政治術語）            | 石油天王 Landman S01E01（油田術語＋德州腔） |
| 4   | 科幻／奇幻自創名詞     | 自創名詞一致性（詞彙表壓力測試） | 沙丘：預言 Dune: Prophecy S01E01（Bene Gesserit 等既有繁中譯名可核對） | 基地 Foundation S01E01                 |
| 5   | 長篇一致性（電影）     | 角色名前後統一、旁白＋對白語氣穩定 | 四海好傢伙 Goodfellas（1990，2h26；旁白密、人名多、黑幫俚語） | F1電影（2025 WEB-DL，字幕軌幾乎必有）；再不行用最後生還者 S01E01（81 分鐘） |

**排除規則（片庫裡不能選的）**：所有陸劇／國漫（華語原音，route 會是 deliver／convert，不是 translate）、日本動畫（Overlord、北斗之拳、日本三國：日語原音）、韓劇（臥底廚師、今生是第一次、有本事換你來做啊）、戰酋 Chief of War（大量夏威夷語對白）、我們的星球 Our Planet（純旁白，沒有對話語域可測）。

每部取**一集**（電影取整部）。

### AC #2 — 兩個變體，同一 prompt 版本、同一詞彙表版本，只差模型

**Given** 每部樣本，**then** 產出兩份 sidecar：

- **A = `claude-haiku-4-5`**（現行預設）
- **B = `claude-sonnet-5`**（`ai/budget.go` 已有定價列，成本記帳不會落 fallback）

**且** 兩次 run 在 `subtitle_runs` 的 `prompt_version` 與 `glossary_version` **相同**、`model_id` 不同。若 glossary_version 不同（A 跑完 harvest 了新詞，B 吃到更大的詞彙表），視為**已知偏誤**記錄在報告；不重跑（成本考量），但 B 的人名一致性分數要打折看。

### AC #3 — 盲評：每部 50 句，左右隨機，評分者看不到哪邊是哪個模型

**Given** 三份 SRT（英文原文、A、B），**when** 執行 `scripts/subtitle-blind-eval.py build`，**then**：

- 從三檔共有的 cue index 中以固定 seed 抽 50 句；
- 每列左右順序隨機，對應關係只寫在 `key.json`；
- 評分者（Alexyu）在 `sheet.csv` 逐句給 `left_score` / `right_score`，**評完才**跑 `score`。

評分表（三檔，故意粗）：

| 分 | 定義                                                   |
| -- | ------------------------------------------------------ |
| 0  | 看不懂或翻錯意思 —— **不改不能看**                     |
| 1  | 看得懂但生硬、語氣不對、用語不像台灣 —— 勉強能看        |
| 2  | 自然，像人翻的 —— 不用改                                |

另外每部樣本手記兩個計數（不進 CSV）：**角色名前後不一致的次數**、**簡體字漏網次數**（閘門保證後者應為 0；不是 0 就是 bug，另立 story）。

### AC #4 — 預先登記的門檻（現在寫死，量完不得回頭改）

**Given** `score` 輸出，**then** 判定規則如下，並已同步寫進腳本常數：

| 判定                       | 條件（每個模型分別算，五部合併後的總樣本）                                       |
| -------------------------- | ------------------------------------------------------------------------------ |
| **✅ 可看（不用校稿）**    | 0 分 ≤ **5%** **且** 2 分 ≥ **60%**，且**沒有任何一部**樣本 0 分 > 10%           |
| **❌ 需校稿**              | 其餘                                                                           |

決策表：

| Haiku | Sonnet                         | 動作                                                                                                    |
| ----- | ------------------------------ | ------------------------------------------------------------------------------------------------------- |
| ✅    | 任意                           | 預設不動。改 README 定位，開始招募內測（字幕功能開放）。                                                |
| ❌    | ✅ 且 0 分率 ≤ Haiku 的一半     | `DefaultClaudeModel` → `claude-sonnet-5`（另開一支 bugfix story，附本報告），成本文案同步 ×3。再招募。 |
| ❌    | ✅ 但差距不到一半               | 換一組 seed 再抽 50 句複驗；仍不到一半 → 視為「換模型不夠」，走下一列。                                 |
| ❌    | ❌                             | **不招募**。立品質 story：(a) 上下文窗 5→整場景、(b) terminology corrector 接進 pipeline、(c) 二段式自審。 |

### AC #5 — 成本封頂與環境復原

- 每次 run 走既有 `AI_RUN_BUDGET_USD` 上限（出廠 $5），不另開預算；
- 預估總花費（見下表）**≤ US$6**，超過即停下來查（多半是 chunk 重試風暴，本身就是發現）；
- 跑完把 `CLAUDE_MODEL` 還原成裁定值並重啟；
- 媒體資料夾內每部只留**一份** `.zh-Hant.srt`（最後一次 run 的）＋ placer 自動產生的 `.zh-Hant.srt.bak`（前一次的）。NFR-I3：播放器不會撿 `.bak`，但 A 的檔案務必先複製到 eval 資料夾再跑 B。

### AC #6 — 報告落地

**Given** 五部評完，**then** 新增 `_bmad-output/implementation-artifacts/eval-1-translation-blind-eval-report.md`，內容：

1. 片單 + 每部 `subtitle_runs` 三欄（`model_id` / `cue_count` / `spent_usd`）×2；
2. `score` 原始輸出（五部各一 + 合併）；
3. 人名不一致 / 簡體漏網計數；
4. 依 AC #4 決策表的**一句話裁定**，以及它解鎖／封鎖了哪些後續 story；
5. 至少 10 句「0 分」的原文／A／B 對照（給之後修 prompt 的人看錯在哪，不是給讀者看熱鬧）。

sprint-status 本條 → `done`，並依裁定新增後續條目。

---

## 成本預估（AC #5 的依據）

假設一集 45 分鐘 ≈ 700 句 ≈ 70 批；每批輸入 ≈ 1,450 tokens（system ~700 + metadata ~300 + glossary ~100 + 5 句 context ~120 + 10 句 ~220）、輸出 ≈ 300 tokens（含 `===TERMS===` trailer）。

| 模型             | 輸入單價 / 1M | 輸出單價 / 1M | 每集（未算 prompt cache） | 電影 2h |
| ---------------- | ------------- | ------------- | ------------------------- | ------- |
| claude-haiku-4-5 | $1            | $5            | ≈ $0.21                   | ≈ $0.42 |
| claude-sonnet-5  | $3            | $15           | ≈ $0.62                   | ≈ $1.24 |

四集＋一部 2.5h 電影 × 兩模型 ≈ **US$5.5**；system block 有 `CachingCompleter` 快取，實付通常更低。

---

## 執行步驟（operator 手冊，照抄即可）

前置：NAS 已是 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`、Claude key 已設、`CLAUDE_MODEL` 未設或為 `claude-haiku-4-5`。

```bash
# 0. 準備工作目錄（不在媒體資料夾內）
mkdir -p /volume1/docker/vido/eval

# 1. 每部樣本：找英文文字軌 index，抽一份原文 SRT 給評分表用
ffprobe -v error -select_streams s -show_entries stream=index:stream_tags=language,title -of csv=p=0 "/path/to/S01E01.mkv"
ffmpeg -v error -i "/path/to/S01E01.mkv" -map 0:<index> -c:s srt /volume1/docker/vido/eval/<slug>/source.srt

# 2. Run A（Haiku）：強制重跑，繞過 pre-flight 與 segment cache
curl -s -X POST http://<nas>:8088/api/v1/subtitles/pipeline/run \
  -H 'Content-Type: application/json' \
  -d '{"media_id":"<uuid>","media_type":"episode","force":true}'
#    等 F8 進度到完成後：
cp "/path/to/S01E01.zh-Hant.srt" /volume1/docker/vido/eval/<slug>/haiku.srt
sqlite3 /volume1/docker/vido/data/vido.db \
  "select model_id,prompt_version,glossary_version,cue_count,spent_usd from subtitle_runs where media_id='<uuid>' order by started_at desc limit 1;"

# 3. 切到 Sonnet：compose 加 CLAUDE_MODEL=claude-sonnet-5，重啟容器，再 Run B（同一個 curl，force:true）
cp "/path/to/S01E01.zh-Hant.srt" /volume1/docker/vido/eval/<slug>/sonnet.srt
#    再查一次 subtitle_runs，確認 prompt_version / glossary_version 與 A 相同

# 4. 出盲評表（在任何有 python3 的機器上跑，含本機）
python3 scripts/subtitle-blind-eval.py build \
  --source eval/<slug>/source.srt --a eval/<slug>/haiku.srt --b eval/<slug>/sonnet.srt \
  --out eval/<slug> --sample 50 --seed 42

# 5. 用 Numbers / Excel 開 eval/<slug>/sheet.csv，填 left_score / right_score（0/1/2），存回 CSV

# 6. 揭盲
python3 scripts/subtitle-blind-eval.py score eval/<slug>
```

五部合併：把五個 `sheet.csv` 的資料列串成一個目錄、`key.json` 的 `rows` 合併即可（或直接把五份 `score` 輸出加總填進報告；n 夠小，手算沒問題）。

---

## Tasks

- [x] **T1** 填 AC #1 片單，逐部確認 route = `translate`（F15 清單，或 `POST /subtitles/generation-candidates/analyze` 後 `GET /subtitles/generation-candidates`）。
- [x] **T2** Run A ×5，複製 sidecar，記錄 `subtitle_runs` 三欄。
- [x] **T3** 切 `CLAUDE_MODEL=claude-sonnet-5` → Run B ×5，複製 sidecar，記錄；核對 prompt/glossary version。
- [x] **T4**（AI 代評）`build` ×5，盲評 250 句，另記人名不一致／簡體漏網。
- [x] **T5**（AI 代評）`score` ×5 + 合併，套 AC #4 決策表。
- [ ] **T6** 寫報告（AC #6），還原 `CLAUDE_MODEL`，sprint-status 更新＋依裁定立後續 story。

---

## 已知品質槓桿（本 story **不動**，量完才決定要不要動）

1. 上下文窗只有前 5 句（`SubtitleTranslatorContextWindow`）。
2. `terminology_corrector.go` 未接進 pipeline 路徑。
3. 閘門是格式閘門；沒有語意自審。
4. Prompt 規則 3「人名保留英文」與 `===TERMS===` harvest「回報你決定的中文譯名」語意互相拉扯 —— 評分時若看到同一人名一下英文一下中文，記在人名不一致計數。

## Discovery Triage（Rule 24）

- 本 story 所有發現皆為既有事實的盤點，無新 backlog；AC #4 的「❌❌」列若觸發，屆時再依報告立案。

## Dev Agent Record

（operator 執行後填：日期、片單、每步花費、偏離手冊之處。）

**2026-09-03 執行紀錄（T1–T3 由 Claude 透過 SSH 在 NAS 代跑；T4–T6 待 Alexyu 評分）**

- 工作目錄：NAS `/mnt/user/appdata/vido/eval/`，同步一份到 repo `eval/`（含 `README.md` 說明、`runs-all.txt` = `subtitle_runs` 完整快照）。手冊裡的 `/volume1/docker/...` 路徑是 Synology 風格，NAS 實際 DB 在 `/mnt/user/appdata/vido/vido.db`；主機沒有 ffprobe/python3，抽軌用 `docker exec Vido ffmpeg`。
- **片單偏離（AC #1 首選 5 部只有 1 部合格）**：內嵌任何中文文字軌就走 deliver（`SelectCandidates` 中文優先），PGS 圖片字幕走 ASR。
  - 1 The Boys S01E01 ✅ 照原案
  - 2 Ted Lasso ❌ 繁中軌 → **Peacemaker S01E01**（Slow Horses 也有繁中軌）
  - 3 House of Cards ❌ 六季全 PGS → **Landman S02E01**（片庫只有第二季）
  - 4 Dune: Prophecy ❌ 六集全 PGS；Foundation ❌ 繁中軌 → **魔戒：力量之戒 S02E01**
  - 5 Goodfellas ❌ 93 GB 4K remux，ffmpeg 抽軌超過 `defaultExtractTimeout` 10 分鐘而失敗（$0）；F1 有中文軌 → **Project Hail Mary（極限返航，1591 cues）**
- **花費**：A（Haiku）有效 $1.079；B（Sonnet）有效 $2.816；兩次失敗另燒 $0.664；總計 **$4.56**（≤ US$6 上限）。每部 A/B：boys 0.223/0.539、peacemaker 0.213/0.534、landman 0.197/0.494、rings 0.118/0.329、hailmary 0.328/0.920。Sonnet 約為 Haiku 的 2.5–2.8×。
- **版本核對**：`prompt_version` 兩邊皆 `m1-v2`。`glossary_version` A 幾乎全空（peacemaker 重跑那次除外）、B 全部非空且各不相同 → AC #2 的「已知偏誤」成立，B 的人名一致性要打折看。
- **環境變更與復原**：`/media` 由 Alexyu 在 Unraid 改成 Read/Write；`CLAUDE_MODEL` 切換是用 `docker run` 依原 inspect 重建容器（腳本 `eval/recreate.sh`），Run B 後已重建回無 `CLAUDE_MODEL` 的狀態並驗證 key test 通過。四個目標資料夾（Peacemaker/Season01、Landman/Season02、Rings of Power/Season 2、GoodFellas）加了 `o+w`，**未還原**（見發現 2）。媒體夾內每部各留 `.zh-Hant.srt`（Sonnet）+ `.zh-Hant.srt.bak`（Haiku），A 檔案已先複製。
- **管線抽樣觀察**：兩個模型都有 cue 在 quality gate 兩次重試後「保留英文原文」（echoed / simplified_leak），數量見 `eval/README.md`；rings 抽樣第 28 句 "He's something else." 兩邊都翻成「他是別的東西。」（0 分候選）。

**2026-09-03 第二輪（加測四部有官方 zh-TW 參考字幕的片，Alexyu 裁定「直接跑 A」）**

- 樣本：Shadow and Bone S01E01、Lioness S02E01、Zootopia 2、Wake Up Dead Man。挑選條件：有外掛 zh-TW 字幕（多為 Netflix／Apple 官方風格）＋內嵌英文文字軌＋無內嵌中文文字軌。全片庫掃描結果在 `eval/candidates.md`。
- 花費：A $1.150、B $3.135、浪費 $0.295（Zootopia B 又是 15 秒 timeout）；第二輪 $4.58，**兩輪合計 ≈ $9.14**。
- 新發現 7：**兩部 20 GB 檔同時排入，兩個 worker 同時 ffmpeg 抽軌互相搶 I/O，雙雙超過 10 分鐘上限失敗**；單獨跑 3.5 分鐘就好。Worker 並行度應對「抽軌」階段另設上限（或抽軌序列化、翻譯並行）。
- 新發現 8：Lioness（SDH 軌）兩個模型各有約 40 句「保留英文」，實看幾乎全是純 `♪` 的音樂 cue → `FilterSDH` 沒把純音符 cue 濾掉，每句都白白送 LLM 再被閘門判 echoed。小修：SDH 過濾器加「只含 ♪／♫ 的 cue 直接丟掉」。
- 產出：`eval/{shadowbone,lioness,zootopia2,knivesout3}/` 各有 sheet.csv / key.json / compare.csv / 參考字幕；容器已還原成預設 Haiku 並驗證。

**2026-09-03 T4/T5（AI 代評，Alexyu 裁定選項 B：Claude 先評一份、人評另存對照）**

- 評分者：9 個 Claude 子代理各評一份 50 句，看不到 key.json；結果在 `eval/<slug>/claude-judge/sheet.csv`，揭盲用 `eval/aggregate.py`。**注意：Claude 評 Claude，報告引用需標註「AI 評」，人評完成後以人評為準。**
- 原五部合併：Haiku 0 分率 7.6% / 2 分率 53.2%（Landman 單片 0 分率 14%）→ ❌；Sonnet 2.8% / 75.2% → ✅。九部合併：Haiku 5.8% / 56.2% ❌；Sonnet 2.2% / 78.4% ✅。
- 套 AC #4：**Haiku ❌、Sonnet ✅ 且 0 分率 ≤ Haiku 一半 → 預設改 `claude-sonnet-5`**（待人評確認後另開 bugfix story）。
- 0 分對照 34 句：`eval/zeros-claude-judge.csv`（供 AC #6 第 5 點）。

- 0 分句的型態：34 句裡多數不是「翻錯字」，而是**跨 cue 錯置／漏譯**——一句的內容被搬到相鄰 cue（例：knivesout3 #397 "I was a boxer" 翻成下一句的「在擂臺上殺死了一個人」；boys #514/#349 前半或主語掉到隔壁）。這是批次翻譯把 10 句當一段重排的副作用，兩個模型都有，Haiku 較多。修 prompt 時應鎖「逐 cue 對應、不得合併拆分」。

**2026-09-03 T4/T5 v2（含前後文重評，以此為準）**

- 第一版評分者只看單句，把合理的合併／拆句也判 0；重做一版給前後各 3 句上下文（`eval/build-ctx-sheet.py` → `claude-judge-ctx/`）。
- 原五部合併：Haiku 0 分率 4.0% / 2 分率 53.2%（Landman 單片 0 分率 12%）→ ❌；Sonnet 0.4% / 86.0% → ✅。九部：Haiku 3.8% / 55.8% ❌；Sonnet 0.7% / 87.1% ✅。
- **裁定不變：預設改 `claude-sonnet-5`**。Haiku 掛在「生硬」而非「翻錯」：四成句子是 1 分。
- 剩下的 0 分（19 句，`eval/zeros-claude-judge-ctx.csv`）以 cue 時間位移為主，兩個模型都有、Haiku 較多；這是發現 9 的實證。另有一例兩邊皆錯：Life360（家人定位 app）被翻成「360 號公路」——沒有世界知識注入就會錯，屬 metadata/glossary 範圍外。

**2026-09-03 T4/T5 v3（全檔逐句 AI 評，10,304 句 × 2）**

- Alexyu 要求對九部完整字幕逐句評分。`eval/build-full-sheets.py` → 38 份 × 300 句連續盲評表 → `eval/aggregate-full.py` 揭盲。彙總 `eval/full-scores.csv`，0 分 `eval/zeros-full.csv`。
- 合併：Haiku 0 分率 3.6% / 2 分率 71.8%（最差單片 Peacemaker 6.5%）→ ✅；Sonnet 1.3% / 89.6% → ✅。單片 Haiku 在 Peacemaker、Landman、Lioness 三部 0 分率 > 5% ❌。
- **與抽樣版裁定不同**：抽樣版 Haiku 2 分率 55.8% ❌，全檔版 71.8% ✅；0 分率兩版一致（3.8% / 3.6%）。差異來自 AI 評分者對「1 vs 2」的校準，不是翻譯本身。結論：**Haiku 是否過關對評分者敏感、恰在 60% 線附近；Sonnet 在任何一版都穩過且 0 分率為 Haiku 的 1/3。** 最終裁定留給人評（建議先評 Peacemaker / Landman 各 50 句校準）。
- 0 分組成：翻錯 279、漏譯／留英文 85、時間位移 120（兩模型皆有，屬管線批次重排問題）。

**每部花費／片長／處理時間（AC #6 第 1 點）**

| 片 | 片長 | 句數 | Haiku 花費 | Haiku 處理時間 | Sonnet 花費 | Sonnet 處理時間 |
| --- | --- | --- | --- | --- | --- | --- |
| The Boys S01E01 | 1h01m | 1060 | $0.223 | 10m | $0.539 | 12m |
| Peacemaker S01E01 | 47m | 844 | $0.213 | 9m | $0.534 | 11m |
| Landman S02E01 | 49m | 914 | $0.197 | 9m | $0.494 | 10m |
| 力量之戒 S02E01 | 1h16m | 567 | $0.118 | 5m | $0.329 | 6m |
| 極限返航 | 2h37m | 1591 | $0.328 | 9m | $0.920 | 18m |
| Shadow and Bone S01E01 | 52m | 598 | $0.129 | 5m | $0.326 | 7m |
| Lioness S02E01 | 44m | 767 | $0.176 | 6m | $0.434 | 8m |
| Zootopia 2 | 1h48m | 1746 | $0.387 | 13m | $1.194 | 23m |
| Wake Up Dead Man | 2h26m | 2217 | $0.458 | 19m | $1.181 | 27m |
| **合計** | **12h20m** | **10304** | **$2.229** | **1h24m** | **$5.951** | **2h03m** |

每小時片長成本：Haiku ≈ $0.18/hr、Sonnet ≈ $0.48/hr；處理速度約為片長的 11%（Haiku）／17%（Sonnet），兩 worker 並行實測。失敗重跑另燒 $0.96，兩輪總計 ≈ $9.14。

**Metadata 注入實況（2026-09-03 實查，回應 Alexyu 提問）**

- metadata **有**注入：每筆 run 的 `metadata_hash` 皆非空，`buildSystemBlocks`（`pipeline.go:937`）把它與詞彙表併成一個 system block、掛 1 小時 prompt cache。
- 但 `BuildMetadataSection` 的 7 欄只實際送出 3～4 欄：Title / Original title / Year / Overview 有值；**Genres 9 部全是 `[]`**、**Production countries movies 全空且 `seriesContext` 未填此欄**、**Cast 從未被賦值**（`TranslateContext.Cast` 只在 `pipeline.go:955` 被讀，`media_store.go` 兩條 load 路徑都沒寫 → `MetadataCastLimit=10` 為死碼）。
- 集數層級 title/overview 不送：`loadEpisode` 套用 `seriesContext(series)`，同影集各集 metadata 相同（也是 prompt cache 跨集命中的前提）。
- 反饋迴路成立：`===TERMS===` → `HarvestedTerms` → `show_glossary`（實測 261 筆）→ 下次 run 由 `BuildGlossarySection` 注入。但 261 筆 `source` **全是 `subtitle`**，schema 允許的 `metadata` 播種從未發生。

**新發現（供立案）**

10. `TranslateContext.Cast` 在管線路徑永不賦值 → prompt 的 Cast 行是死碼；演員／角色名這個最能穩住人名一致性的訊號完全沒進 prompt。
11. `genres` 掃描後為 `[]`、`production_countries` 為空，`seriesContext` 又漏填 Countries → metadata 區塊實際只剩片名＋年份＋一段簡介。
12. TMDb 比對失敗的片（Wake Up Dead Man）會把**原始檔名當 Title 送進 prompt**（`- Title: [bitsearch.to] Wake.Up...NAHOM.mkv`），是雜訊而非上下文；應在 metadata 缺失時略過該行而不是送檔名。該片仍是 Haiku 0 分率最低的一部（1.8%），可視為 metadata 邊際貢獻的粗略對照（n=1）。
13. `show_glossary` 從未由 metadata 播種（角色／演員名），這正是修人名一致性最直接的槓桿。

**執行中發現的產品問題（不在本 story 修，供立案）**

1. Unraid 模板把 `/media` 設 `Mode="ro"`，管線翻完才在 placer 寫檔失敗 → **先花錢後失敗**。pre-flight 應先檢查目標資料夾可寫。
2. 容器 `PUID=1000/PGID=1000`，片庫資料夾多為 `nobody:users`（group 100）→ 同樣是翻完才 permission denied（燒了 $0.196）。同上，pre-flight 檢查可寫；或文件註明 PGID 應設 100。
3. 大檔 remux（93 GB）抽字幕超過 10 分鐘硬上限 → 4K remux 片庫整批不能用。需可調 timeout 或改用不掃整檔的抽法。
4. Claude 呼叫硬性 **15 秒 timeout**，Sonnet 5 一批 10 句偶爾超過 → 3 次重試全逾時後整支 run 失敗（cue 935/986，燒了 $0.468）。timeout 應隨模型／輸出長度放寬。
5. 用預設模型時 `subtitle_runs.model_id` 是**空字串**（只有 env override 才寫入）→ 報告要靠 log 才知道 A 是 Haiku。
6. `POST /settings/keys/test` 在 `CLAUDE_MODEL=claude-sonnet-5` 下回 `AI_INVALID_RESPONSE: Cannot parse AI response`（key 其實有效）→ 測試 prompt 的解析太脆。

## File List

- `scripts/subtitle-blind-eval.py`（新增，stdlib）
- `scripts/README.md`（新增段落）
- `_bmad-output/implementation-artifacts/eval-1-translation-blind-eval.md`（本檔）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（新條目）
- `eval/scan-partial-zh.sh`（新增，2026-09-03 party-mode；在容器內量「同影集部分集有官方繁中」的比例）
- `eval/partial-zh.csv`（上述掃描輸出）

---

## 後續 Backlog（2026-09-03 party-mode 裁定，待 Alexyu 審閱後逐項拆 story）

> 討論脈絡：App 定位 BYOK、即將給少數朋友內測。裁定順序 = **先修「先花錢後失敗」的 bug → A 路線（讓詞彙表會累積、且第一天就有料）→ B 路線（分享）**。
> 護城河結論：模型是大宗商品、路由（有中文軌就不翻）任何腳本都寫得出來；**唯一會累積、複製不走的是 per-show 詞彙表**，但它現在只開三成（261 筆全來自 LLM 自翻、metadata 播種從未發生、綁本機 id 無法跨機器對上）。

### P0 — 朋友內測前必修（使用者的錢不能白燒；評測 $9.14 中 $0.96 = 10.5% 是翻完才失敗）

| # | 事項 | 來源 | 大小 |
| --- | --- | --- | --- |
| P0-1 | pre-flight 加「目標資料夾可寫」檢查（ro mount、PUID/PGID 不符都在這裡擋），在確認框之前就亮紅燈 | 產品問題 1、2 | 小 |
| P0-2 | Claude 呼叫 timeout 由寫死 15 秒改為隨模型／輸出長度放寬；三次逾時不得讓整支 run 死掉 | 產品問題 4 | 小 |
| P0-3 | 抽軌 timeout 可調（10 分鐘寫死 → 4K remux 整批不能用）＋ 抽軌階段序列化或另設並行上限（兩個 worker 同時抽大檔互搶 I/O 雙雙失敗） | 產品問題 3、發現 7 | 小～中 |
| P0-4 | `FilterSDH` 直接丟掉只含 ♪／♫ 的 cue（Lioness 每個模型約 40 句白送 LLM 再被閘門判 echoed） | 發現 8 | 很小 |
| P0-5 | 用預設模型時 `subtitle_runs.model_id` 也要寫入（現在是空字串，使用者不知道自己付的是哪個模型） | 產品問題 5 | 很小 |
| P0-6 | `POST /settings/keys/test` 在 `CLAUDE_MODEL=claude-sonnet-5` 下回 `AI_INVALID_RESPONSE`（key 其實有效）→ 測試 prompt 解析放寬 | 產品問題 6 | 小 |
| P0-7 | TMDb 比對失敗時略過 Title 行，不得把原始檔名（`[bitsearch.to] Wake.Up...mkv`）當片名送進 prompt | 發現 12 | 很小 |
| P0-8 | 確認框顯示模型選擇 + 本集預估價錢 + 預估時間；**預設改為 Sonnet**（評測任一版本都穩過、0 分率為 Haiku 的 1/3），Haiku 由使用者自選省錢（Alexyu 2026-09-03 裁定；確認框必須把兩者價差 2.7× 明示出來，讓省錢是使用者看得到的選擇） | party-mode → Alexyu 改裁 | 中 |
| P0-9 | TMDb attribution：logo + 「This application uses TMDB and the TMDB APIs but is not endorsed…」（條款第 3 條；JustWatch 的有做在 `StreamingAvailability.tsx:179`，TMDb 的完全沒有） | party-mode 查證 | 很小 |

### P1 — A 路線：詞彙表「會累積、且第一天就有料」

| # | 事項 | 來源 | 大小 |
| --- | --- | --- | --- |
| P1-1 | **`show_glossary` 改綁 TMDb ID**（架構評估見下一節）—— P1-2 以下全部以此為前提 | party-mode | 中 |
| P1-2 | 修 `TranslateContext.Cast` 死碼（`pipeline.go:955` 只讀不寫）、`genres` / `production_countries` 補齊、`seriesContext` 補 Countries | 發現 10、11 | 小 |
| P1-3 | TMDb 角色名／演員名在掃描當下播種 `show_glossary`（`source=metadata`，schema 已允許但從未發生）；TMDb 中文名可能是簡體或空 → OpenCC + fallback | 發現 13 | 中 |
| P1-4 | **內建在地用語詞庫**（跨片、不需累積、出貨即生效）：(a) 查表型 —— 視頻→影片、質量→品質、信息→資訊；常見品牌／App（Life360 被翻成「360 號公路」）。(b) **OTT 風格在地化**（"the grocery store" → 全聯／家樂福／喜互惠 這類 Netflix／Apple TV 譯法）—— 這是 prompt 風格規則 + 範例，**不是查表**；且是口味決定（有人討厭美劇裡出現全聯），story 要含「在地化程度」開關的 UX 裁定 | Alexyu | 中 |
| P1-5 | **加速器②：同影集內用官方繁中字幕對齊挖人名餵沒有字幕的集** —— 範圍只限「同影集／同系列」，不跨片。**量完了（見下節）：partial 影集只有 8/80 = 10%，但走翻譯的 190 集裡有 131 集（69%）落在這 8 部** → 值得做，但要以「集數覆蓋率」而非「影集比例」立論 | party-mode（Mary 修正後）+ `eval/partial-zh.csv` | 中～大 |
| P1-6 | 成本收據：單次花費 + 本月累計 + 「略過 N 集省下約 $X」（讓使用者看到路由與 cache 在替他省錢） | Sally | 中 |
| P1-7 | 內嵌預設 TMDb key（rate limit 按 key+IP，自架使用者互不排擠；條款 non-transferable 意味所有請求算 Alexyu 的使用；**商業化那天要另簽**），設定頁保留「用我自己的 key」 | Winston 查證 | 小 |
| P1-8 | 模型品質等級由 Vido 集中評測、隨 App 發：200 句黃金樣本 + `model-ratings.json` + CI 自動跑（加一行 model id 就出分）；未評測模型顯示「可花約 $0.01 試跑 20 句」而非問號 | Murat | 中～大 |
| P1-9 | prompt 鎖「逐 cue 對應、不得合併拆分」（全檔 484 個 0 分裡 120 個是時間位移，兩模型皆有） | 發現 9 | 中 |
| P1-10 | 詞彙表 UI 顯示 `source`（自己加／系統學的／官方字幕／TMDb）—— schema 有欄位，UI 沒秀（What'Sub 的「自己加 / 改字記的」標籤） | Sally | 小 |

### P2 — B 路線（A 做完再做；現在做會是一面空牆）

| # | 事項 | 大小 |
| --- | --- | --- |
| P2-1 | 詞彙表匯出／匯入（檔案；先驗證「共享」有沒有人要） | 小 |
| P2-2 | 英雄牆式公開詞彙表（參考 What'Sub，但**分享的是有標準答案的詞彙不是樣式**）：key = TMDb ID、卡片顯示「N 人使用 · M 人回報錯誤」而非愛心、同詞分歧攤開讓使用者選、官方字幕來源徽章、QR 推薦碼 | 大 |

### 片庫實測：「同影集部分集有官方繁中」的比例（P1-5 的依據，2026-09-03 `eval/scan-partial-zh.sh` 在容器內跑）

判定與 `SelectCandidates` 路由對齊：has_zh = 外掛 zh-TW／cht／tc 或內嵌 chi／zho 文字軌（deliver，$0）；translate = 無中文但有內嵌英文文字軌；asr = 其餘（PGS 或無字幕）。Vido 自產的 `.zh-Hant.srt` 不算。

| | 影集數 | 集數 | has_zh | translate | asr |
| --- | --- | --- | --- | --- | --- |
| 全片庫 TV | 80 | 2523 | 620 | **190** | 1713 |
| partial（has_zh>0 且 translate>0） | **8（10%）** | 335 | 61 | **131（= 全部 translate 的 69%）** | 143 |
| 全集都有中文 | 30 | | | | |
| 全集都要翻 | 4 | | | | |
| 全集都走 ASR | 30 | | | | |
| 電影 | 55 | | 30 | 3 | 22 |

partial 的 8 部：Scorpion（13 有 → 79 要翻）、Supernatural（7 → 25）、Shadow and Bone（8 → 8）、Lioness（5 → 3）、牧神記（2 → 13）、Clevatess（3 → 1）、Chief of War（8 → 1）、See（15 → 1）。

**解讀**

- 用「影集比例」看是 10%，會被砍；用「要翻的集數覆蓋率」看是 69% —— **真正會花錢翻譯的集，七成有同影集的官方繁中可以挖**。P1-5 值得做，story 要以後者立論。
- Scorpion 一部就佔 79 集：一個 13 集的官方詞彙表能餵 79 集，這是加速器②最好的示範案例，也適合當 P1-5 的驗收樣本。
- 另一個沒討論過的事實：**走 translate 的只有 190/2523 = 7.5%，走 ASR 的有 1713 集（68%）**。多數是中文原音的動畫／陸劇（無字幕或硬字幕）—— 這群不需要翻譯，但意味著 ASR 路徑的成本與品質才是這個片庫的大宗；要不要另開 eval，Alexyu 裁定。

### 待人評

- Peacemaker / Landman 各 50 句人評校準（Haiku 抽樣版 ❌、全檔版 ✅，差在評分者對 1/2 分的校準）。Sonnet 任何版本都過。

### 架構評估：`show_glossary` 從本機 `media_id` 改綁 TMDb ID（P1-1）

**現況（2026-09-03 實查）**

- `show_glossary.media_id TEXT NOT NULL`；unique `(media_id, term_src, language)`；index `media_id`；**沒有 FK、沒有 cascade**（migration 028）。
- key 由 `glossaryKeyFor(ref, showKey)`（`process_item.go:801`）決定：集數／影集 → `series.ID`、電影 → `movie.ID`；`ShowKey` 在 `media_store.go:100/126/156` 三條 load 路徑填入。
- **四個消費者**都吃 `mediaID string`：pipeline（`glossary_store.go:44`）、`glossary_service.go`（UI 列表／Update／ConfirmAll）、`nfo_localizer_service.go:97`、`transcription_service.go:245`。
- HTTP 是 `/media/:id/glossary`（`glossary_handler.go:30`），route id 原樣往下傳；web `glossaryService.ts` 與 `ManageSubtitleDialogV2.tsx:17` 直接傳 series id。
- `TMDbID` 在 movie／series／episode 都是 `NullInt64` → **可能為空**（Wake Up Dead Man 就比對失敗）。
- segment cache 的 `GlossaryVersion` 是 hash 詞彙**內容**（`segment_cache.go:165`），不含 key → **換 key 不會讓 cache 失效**。
- 沒有「重新比對 TMDb」流程（handlers／services 搜 rematch／override 無結果）。

**建議：不要「把 media_id 換成 tmdb_id」，加一層 scope**

1. 新欄 `scope TEXT NOT NULL`，值域：`tmdb:tv:<id>`、`tmdb:movie:<id>`、`local:<media_id>`（比對失敗的退路）。
2. 新 unique index `(scope, term_src, language)`；`media_id` 欄保留一版供稽核，下一個 migration 再移除。
3. 回填：series／movies 有 `tmdb_id` 者寫 `tmdb:*`，其餘寫 `local:<media_id>`。
4. 新增 `GlossaryScopeResolver`（services 層）：`Resolve(ctx, mediaID) → scope`，查 movie／series 的 `tmdb_id`；四個消費者改先 resolve 再查 repo。
5. **HTTP 與 web 不動**：`/media/:id/glossary` 照舊，handler 內部 resolve。
6. 電影系列（鋒迴路轉 1／2／3）：TMDb `belongs_to_collection` → 之後可加 `tmdb:collection:<id>` 當第二層查詢，本階段不做。

**風險**

- 將來若做「手動重新比對 TMDb」，scope 必須跟著搬（現在沒有這條流程，做的時候一併處理）。
- 同一 TMDb ID 有多份檔案（重複片、不同版本）會自然共用詞彙表 —— 這是想要的行為，但 UI 要標示。
- 掃描時 TMDb 比對晚於檔案入庫 → 先寫 `local:` 再升級成 `tmdb:` 的搬移要有（或 resolver 每次查、不快取）。

**A → B（共享）銜接檢查（Alexyu 2026-09-03 提問）**

scope 設計本身對 B 是中性的（匯出格式就是 `{scope, term_src, term_zh, language, source, confirmed}`，傳輸方式檔案／repo／中央服務都不受影響；`local:*` 的條目天生不能分享，符合預期）。但有**兩件事必須在 A 的同一個 migration 裡一起做**，否則 B 會繼承痛點：

1. **`source` 是 CHECK enum**（`'subtitle','metadata','manual'`，migration 028），SQLite 不能 ALTER CHECK → 要改就得重建表。B 需要 `official_subtitle`（加速器②挖出來的，信任度最高）和 `community`（匯入的）。**A 重建表時直接拿掉 CHECK，enum 改由 Go model 驗證**（`models.GlossaryTerm.Validate` 已經在做同一件事）—— 之後任何新來源都不必再重建表。
2. **`term_src` 沒有正規化**：unique index 沒有 `COLLATE NOCASE`，repo 也沒 `ToLower/TrimSpace` → `Demogorgon` 與 `demogorgon` 是兩筆。單機只是小髒，跨機器合併會變成牆上的重複條目。**A 就定好正規化規則**（至少 trim + 大小寫；unique index 加 `COLLATE NOCASE`）。

B 才需要、A 可以不做的：`origin`／`author`（誰分享的）、`remote_id`（對應牆上的條目，用於更新／撤回）、`imported_at`；「N 人使用」需要匿名安裝 ID，與 glossary schema 無關。另一個 B 才會浮現的風險：使用者的 TMDb 比對錯了，詞彙表會發布到錯的劇 → **只發布 `confirmed=1` 或 `official_subtitle` 的條目**，LLM 自挖（`subtitle`）的一律不上牆。

**彈性檢查：不走 A 也不走 B 的未來（Alexyu 2026-09-03 要求「保有彈性」）**

這個設計刻意只鎖兩件事，其餘都留開：

| 鎖住的 | 為什麼可以鎖 |
| --- | --- |
| 一列 = 一組「來源詞 → 譯詞」配對 | 這是詞彙表的定義；**風格規則**（P1-4b 的「超市→全聯」在地化程度）不是配對，**另開表**，不硬塞 |
| 一列有一個字串 `scope` | 字串 + 命名空間前綴，新的 scope 種類是**加值**不是改表 |

留開的（未來需求變了也不用重建表）：

- **換掉或並存 TMDb**（IMDb／TVDB／自家 ID）：`scope` 是 `imdb:tt123` 也行，前綴就是命名空間；換 ID 來源 = 改 resolver 一個檔案。**這就是為什麼不做 `tmdb_id INTEGER` 欄** —— 那會把 TMDb 焊死。
- **跨劇／全域詞庫**（電影系列 `tmdb:collection:*`、P1-4a 的 `global:zh-TW`、某個宇宙 `universe:mcu`）：新前綴 + resolver 回傳多個 scope 依優先序合併，表不動。
- **多目標語言**（zh-HK、ja）：`language` 欄已在 unique key 裡。
- **多使用者同一台 NAS**／**代管雲端版**：per-user 覆寫或 server-side 同步都是在 `scope` 外再包一層（`user:` 前綴或另一張 override 表），基表不動。
- **新來源**（`community`、`official_subtitle`、將來的 `llm_review`）：CHECK 拿掉後只改 Go 常數。
- **分享的傳輸方式**（檔案／repo／中央服務）：表裡不放任何傳輸細節（沒有 URL、沒有 sync 狀態），這些屬於 B 自己的表。

**真正的單行道只有一條**：詞彙表的顆粒度是「影集」不是「集」。要改成每集不同譯名，才需要重想。目前沒有這個需求。

**工程量**：1 migration（加 scope + 回填 + 重建表拿掉 CHECK + NOCASE index）+ 1 resolver + 4 個 call site + 測試 = **中**。UI 零改動。
