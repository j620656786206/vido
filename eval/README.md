# eval-1 盲測工作目錄（2026-09-03 產出）

Story：`_bmad-output/implementation-artifacts/eval-1-translation-blind-eval.md`
NAS 上的同一份在 `/mnt/user/appdata/vido/eval/`。

## 每個資料夾裡有什麼

| 檔案                           | 說明                                                                            |
| ------------------------------ | ------------------------------------------------------------------------------- |
| `source.srt`                   | 管線實際採用的英文軌（用 cue index + 時間碼比對出來的，100% 對齊）              |
| `source-N.srt`                 | 該片所有英文候選軌的原始抽出（N = ffprobe stream index），留著備查              |
| `haiku.srt`                    | Run A：`claude-haiku-4-5` 產出的 `.zh-Hant.srt`                                 |
| `sonnet.srt`                   | Run B：`claude-sonnet-5` 產出的 `.zh-Hant.srt`                                  |
| `sheet.csv`                    | **你要填的評分表**：50 句、左右隨機。只填 `left_score` / `right_score`（0/1/2） |
| `key.json`                     | 揭盲用，**評完前不要開**                                                        |
| `reference-existing-zh-TW.srt` | （只有 boys）片庫原本就有的 zh-TW 字幕，給你交叉比對用，不進評分                |

## 片單（與 story AC #1 的差異）

| 槽  | slug         | 實際樣本                      | 為什麼換                                                                             |
| --- | ------------ | ----------------------------- | ------------------------------------------------------------------------------------ |
| 1   | `boys`       | The Boys S01E01               | 照原案                                                                               |
| 2   | `peacemaker` | Peacemaker S01E01             | Ted Lasso / Slow Horses 都內嵌繁中軌 → route 是 deliver 不是 translate               |
| 3   | `landman`    | Landman **S02E01**            | House of Cards 六季全是 PGS 圖片字幕 → ASR；片庫只有 Landman 第二季                  |
| 4   | `rings`      | 魔戒：力量之戒 **S02E01**     | Dune: Prophecy 六集全 PGS；Foundation 內嵌繁中軌                                     |
| 5   | `hailmary`   | Project Hail Mary（極限返航） | Goodfellas 是 93 GB 4K remux，ffmpeg 抽字幕超過管線 10 分鐘上限而失敗；F1 內嵌中文軌 |

## 怎麼評

1. 用 Numbers / Excel 開 `eval/<slug>/sheet.csv`，逐句填 `left_score` 與 `right_score`：
   `0` 看不懂或翻錯　`1` 看得懂但生硬／不像台灣話　`2` 自然、不用改
2. 另外手記兩個數字：角色名前後不一致次數、簡體字漏網次數。
3. 存回 CSV（保持 UTF-8），然後揭盲：

```bash
python3 scripts/subtitle-blind-eval.py score eval/boys
python3 scripts/subtitle-blind-eval.py score eval/peacemaker
python3 scripts/subtitle-blind-eval.py score eval/landman
python3 scripts/subtitle-blind-eval.py score eval/rings
python3 scripts/subtitle-blind-eval.py score eval/hailmary
```

## 花費（`runs-all.txt` 是 `subtitle_runs` 的完整快照）

| slug       | cues | Haiku (A)  | Sonnet (B) | 備註                                                      |
| ---------- | ---- | ---------- | ---------- | --------------------------------------------------------- |
| boys       | 1060 | $0.223     | $0.539     |                                                           |
| peacemaker | 844  | $0.213     | $0.534     | A 第一次在寫檔階段失敗，另燒 $0.196                       |
| landman    | 914  | $0.197     | $0.494     | B 第一次在 cue 935 因 15 秒 API timeout 失敗，另燒 $0.468 |
| rings      | 567  | $0.118     | $0.329     |                                                           |
| hailmary   | 1591 | $0.328     | $0.920     |                                                           |
| goodfellas | —    | $0.000     | —          | 抽字幕逾時，未進翻譯                                      |
| **合計**   |      | **$1.079** | **$2.816** | 有效 $3.90，浪費 $0.66，總計 **$4.56**                    |

`prompt_version` 兩邊都是 `m1-v2`。`glossary_version` A 幾乎全空、B 全部非空（A 跑完 harvest 了詞彙表），屬 story AC #2 說的「已知偏誤」：B 的人名一致性要打折看。

## 閘門後仍保留英文原文的 cue 數（quality gate 兩次重試後放棄）

| slug       | Haiku | Sonnet |
| ---------- | ----- | ------ |
| boys       | 9     | 10     |
| peacemaker | 0     | 2      |
| landman    | 10    | 8      |
| rings      | 0     | 3      |
| hailmary   | 12    | 9      |

這些不算「翻錯」，是閘門擋下後的可見殘留；評分時遇到直接給 0。

## 片庫原有的 zh-TW 字幕（交叉比對用，2026-09-03 從 NAS 複製）

| 位置                                                 | 來源                                                     | 對齊狀況                              |
| ---------------------------------------------------- | -------------------------------------------------------- | ------------------------------------- |
| `eval/boys/reference-existing-zh-TW.srt`             | The Boys S01E01 旁邊的 `.zh-TW.srt`（2025-08）           | 時間碼只有 56% 對得上，句子切法不同   |
| `eval/rings/reference-existing-zh-TW.srt`            | 力量之戒 S02E01 旁邊的 `.zh-TW.srt`（2025-08）           | 77% 對得上                            |
| `eval/hailmary/reference-existing-zh-TW.srt`         | 極限返航旁邊的 `.zh-TW.srt`（2026-05）                   | 71% 對得上                            |
| `eval/goodfellas/reference-existing-zh-TW.srt`       | Goodfellas 旁邊的 `.zh-TW.srt`（陸譯風格，全形空格人名） | 本次沒有管線輸出可比                  |
| `eval/dune-prophecy/reference-existing-zh-TW.hi.srt` | 沙丘：預言 S01E01 的 `.zh-TW.hi.srt`（聽障版）           | 本次沒有管線輸出可比；等 ASR 路徑再用 |

Peacemaker、Landman 旁邊原本沒有任何字幕檔。

因為原有字幕和我們抽的英文軌切句不同，沒辦法逐列硬對。`eval/build-compare.py` 用「開始時間 ±0.5 秒」找最近的一句，產出 `eval/<slug>/compare.csv`：
`idx, start, source, haiku, sonnet, existing_zh_TW`，對不到的那格留白。用 Numbers 開來左右看即可。

## 第二輪（2026-09-03 上午，四部有官方參考字幕的片）

| slug         | 片                                 | cues | Haiku (A)  | Sonnet (B) | 參考字幕                | 備註                                                                                                                                                                                                        |
| ------------ | ---------------------------------- | ---- | ---------- | ---------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `shadowbone` | Shadow and Bone S01E01（太陽召喚） | 598  | $0.129     | $0.326     | Netflix 官方 zh-TW      | 奇幻名詞                                                                                                                                                                                                    |
| `lioness`    | Lioness S02E01（特種部隊：母獅）   | 767  | $0.176     | $0.434     | zh-TW.srt + 雙語 `.ass` | 軍事術語。**參考字幕時間軸對不上**（只對到 173/829），要用 .ass 或自己對；兩個模型都有 40 句左右「留英文」，實際上幾乎全是 `♪` 音樂符號 cue（SDH 過濾器沒把它濾掉，閘門判 echoed 後原樣保留），不是翻譯問題 |
| `zootopia2`  | Zootopia 2（動物方城市2）          | 1746 | $0.387     | $1.194     | 官方 zh-TW.hi           | 喜劇雙關。B 第一次 15 秒 timeout 失敗，另燒 $0.295                                                                                                                                                          |
| `knivesout3` | Wake Up Dead Man（鋒迴路轉3）      | 2217 | $0.458     | $1.181     | Netflix 風格 zh-TW      | 對白最密的一部                                                                                                                                                                                              |
| **合計**     |                                    |      | **$1.150** | **$3.135** |                         | 有效 $4.29，浪費 $0.30，第二輪 **$4.58**                                                                                                                                                                    |

兩部 20 GB 電影第一次因為同時抽字幕超過 10 分鐘上限而失敗（$0），改成一部一部排就過了。

**兩輪總花費 ≈ US$9.14**（第一輪 $4.56 + 第二輪 $4.58）。

每個資料夾一樣有 `sheet.csv`（要填）、`key.json`（別開）、`compare.csv`（對照用）、`reference-existing-zh-TW.srt`。

## Claude 代評結果（2026-09-03，AI 評分，不是人評；你的 `sheet.csv` 仍是空的）

Alexyu 選擇先讓 Claude 代評一份（`eval/<slug>/claude-judge/sheet.csv`），自己的評分留待之後對照。評分者是 Claude 子代理，**看不到 key.json**；但評 Claude 翻譯的仍是 Claude，報告引用時必須註明。

揭盲指令：`python3 eval/aggregate.py claude-judge`（全部九部）或加 slug 列表（story 原五部）。

| 範圍   | 模型   | n   | 0   | 1   | 2   | 0 分率 | 2 分率 | AC #4                                    |
| ------ | ------ | --- | --- | --- | --- | ------ | ------ | ---------------------------------------- |
| 原五部 | Haiku  | 250 | 19  | 98  | 133 | 7.6%   | 53.2%  | ❌（Landman 單片 0 分率 14% 也超過 10%） |
| 原五部 | Sonnet | 250 | 7   | 55  | 188 | 2.8%   | 75.2%  | ✅                                       |
| 九部   | Haiku  | 450 | 26  | 171 | 253 | 5.8%   | 56.2%  | ❌                                       |
| 九部   | Sonnet | 450 | 10  | 87  | 353 | 2.2%   | 78.4%  | ✅                                       |

單片：Haiku 只有極限返航過關；Sonnet 除了 Peacemaker（0 分率 6%）全過。

**AC #4 決策表對應列：Haiku ❌、Sonnet ✅ 且 0 分率 ≤ Haiku 一半 → 預設模型改 `claude-sonnet-5`（成本文案 ×3）。** 這是 AI 評分下的裁定；人評若翻案以人評為準。

0 分句子對照（34 句，含兩邊各自分數與理由）：`eval/zeros-claude-judge.csv`。

## Claude 代評 v2：含前後文（2026-09-03，**以此版為準**，取代上面無前後文那版）

`eval/build-ctx-sheet.py` 把每句前後各 3 句的英文與該側中文一併給評分者（`claude-judge-ctx/sheet-ctx.csv`），評分規則加了「這句顯示的文字是不是隔壁 cue 的內容」。結果在 `claude-judge-ctx/sheet.csv`，揭盲：`python3 eval/aggregate.py claude-judge-ctx`。

| 範圍   | 模型   | n   | 0   | 1   | 2   | 0 分率 | 2 分率 | AC #4                                         |
| ------ | ------ | --- | --- | --- | --- | ------ | ------ | --------------------------------------------- |
| 原五部 | Haiku  | 250 | 10  | 107 | 133 | 4.0%   | 53.2%  | ❌（2 分率不到 60%；Landman 單片 0 分率 12%） |
| 原五部 | Sonnet | 250 | 1   | 34  | 215 | 0.4%   | 86.0%  | ✅                                            |
| 九部   | Haiku  | 450 | 17  | 182 | 251 | 3.8%   | 55.8%  | ❌                                            |
| 九部   | Sonnet | 450 | 3   | 55  | 392 | 0.7%   | 87.1%  | ✅                                            |

單片：Haiku 只有極限返航、Lioness、Zootopia 2 過關；Sonnet 九部全過。

有了前後文，0 分少了一半（無前後文那版把合理的合併拆句也算錯），但**裁定不變：Haiku ❌、Sonnet ✅ 且 0 分率遠低於 Haiku 一半 → 預設改 `claude-sonnet-5`**。差距主要在「1 分（生硬）」：Haiku 四成句子生硬，Sonnet 一成二。

留下的 19 句 0 分（`eval/zeros-claude-judge-ctx.csv`）以**時間位移**為主（cue 顯示的是前一句或後一句的內容，例：knivesout3 #397/#1127、rings #236、peacemaker #114），以及少數真的翻錯（「No games.」→「沒有遊戲」；Life360 追蹤 app 被翻成「360 號公路」兩邊都錯）。
