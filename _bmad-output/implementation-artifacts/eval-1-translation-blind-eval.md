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
| 5   | 長篇一致性             | 角色名前後統一、長時間語氣穩定   | 最後生還者 The Last of Us S01E01（81 分鐘，接近電影長度）   | 電影庫任一部英語電影（片單未附，Alexyu 自選） |

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

四集＋一部電影 × 兩模型 ≈ **US$5**；system block 有 `CachingCompleter` 快取，實付通常更低。

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

- [ ] **T1** 填 AC #1 片單，逐部確認 route = `translate`（F15 清單，或 `POST /subtitles/generation-candidates/analyze` 後 `GET /subtitles/generation-candidates`）。
- [ ] **T2** Run A ×5，複製 sidecar，記錄 `subtitle_runs` 三欄。
- [ ] **T3** 切 `CLAUDE_MODEL=claude-sonnet-5` → Run B ×5，複製 sidecar，記錄；核對 prompt/glossary version。
- [ ] **T4** `build` ×5，盲評 250 句，另記人名不一致／簡體漏網。
- [ ] **T5** `score` ×5 + 合併，套 AC #4 決策表。
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

## File List

- `scripts/subtitle-blind-eval.py`（新增，stdlib）
- `scripts/README.md`（新增段落）
- `_bmad-output/implementation-artifacts/eval-1-translation-blind-eval.md`（本檔）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（新條目）
