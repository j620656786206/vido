# Golden sample（黃金樣本）

同一份 200 句英文字幕考卷，考每一個翻譯模型。`cmd/grade` 讀這裡的 `golden-v1.jsonl`（`go:embed`），sub-7-8c 的「試跑 20 句」讀前 20 句。

## v2 的考法（2026-10-14 起，`eval.SampleVersion = "golden-v2"`）

考卷內容跟 v1 一樣，200 句一句沒改，資料檔也還是 `golden-v1.jsonl`。只有考法改了：**考試時附上角色名對照表，做法跟正式版一樣。**

- 對照表從每一句的 `names` 欄位建出來，每個人名取**第一個**譯法。三個來源（`tos`、`sita`、`trap`）各自一份對照表，就像正式版「一部片一份詞彙表」。
- 一批句子只帶自己來源的人名。一批如果跨兩個來源，就把兩邊的對照表合起來送。
- 報告的 `glossary_version` 是整份對照表的雜湊值。
- **v1 的考法（不附對照表）還留著，當對照組用**：`cmd/grade --no-glossary`，報告會標成 `golden-v1`。它量的是「片子在 TMDb 找不到、沒有對照表的時候，模型自己能不能把人名翻得前後一致」。這種報告不能 `--merge`。
- **取捨**：`sita` 原本是「人名一致性」的考題，附上對照表之後，考的變成「模型有沒有照對照表翻」。`trap` 裡的人名題也一樣。要看模型自己翻得一不一致，請用 `--no-glossary` 那組分數。
- 本機模型用「模型自己的格式」考的時候：
  - Hy-MT2 改用官方的術語模板，只列出這一句和它的上下文裡真的出現的人名。
  - TranslateGemma 的官方格式放不進對照表，所以一律算成 v1 的考法。

## 來源與授權

| 來源                                                                                    | 句數                     | 影片授權                 | 字幕文字授權                                                                                                                                                                                                                                                  | 說明                                       |
| --------------------------------------------------------------------------------------- | ------------------------ | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| `tos` — [Tears of Steel](https://mango.blender.org/)（Blender Foundation, 2012）        | 76（全片對白）           | CC BY 3.0                | 英文：官方 `TOS-en.srt`；繁中：社群 `TOS-CH-traditional.srt`，皆在 [download.blender.org](https://download.blender.org/demo/movies/ToS/subtitles/)                                                                                                            | 真人科幻短片，吵架、俚語、片場指令         |
| `sita` — [Sita Sings the Blues](https://www.sitasingstheblues.com/)（Nina Paley, 2008） | 84（只選對白，不選歌詞） | CC0 1.0（2013-01-18 起） | 英文與繁中取自 Wikimedia Commons TimedText（[en](https://commons.wikimedia.org/wiki/TimedText:Sita_Sings_the_Blues.webm.en.srt)／[zh-hant](https://commons.wikimedia.org/wiki/TimedText:Sita_Sings_the_Blues.webm.zh-hant.srt)），該站文字貢獻為 CC BY-SA 4.0 | 美式閒聊＋印度神話人名（人名一致性的考題） |
| `trap` — 自寫                                                                           | 40                       | —                        | CC BY-SA 4.0                                                                                                                                                                                                                                                  | 見下表                                     |

Annette Hanshaw 的錄音仍有版權（至 2030），所以 Sita 的歌詞句**一律不選**。

**整份資料檔（`golden-v1.jsonl`、`authored-v1.json`）以 CC BY-SA 4.0 發布**，每一行都帶 `attribution` 欄。查過但不能用的來源：TED（CC BY-NC-ND，翻譯＝改作）、Khan Academy（NC）、OpenSubtitles 衍生語料（授權不明）。

## 每一行的欄位

```json
{
  "id": "tos-01",
  "source": "tos",
  "start": "00:00:23,000",
  "end": "00:00:24,500",
  "text": "You're a jerk, Thom.",
  "refs": ["你真是爛透了，湯姆", "湯姆，你這個混蛋。"],
  "names": { "Thom": ["湯姆"] },
  "traps": [],
  "attribution": "…"
}
```

- `refs`：至少兩句可接受的繁中。第一句是來源字幕的原譯，第二句是 Vido 維護者另寫的。裁判 AI 看它們理解意思與風格，**用字不同不扣分**。
- `names`：這句裡的人名與可接受譯法；譯文少了或換了譯法 → `name_mismatch`。
- `anchors`（選填）：必須落在這句的詞；譯文沒有自己的、卻有鄰句的 → `time_shift`。沒填時以 `names` 的第一個譯法當錨點。
- `forbid`（選填）：台灣觀眾會讀成陸港用語的寫法（軟件、視頻、質量…），出現即 `forbidden_term`。
- `traps`：自寫句的陷阱類別（下表）；借來的句子是空陣列。

## 40 句陷阱的分類

| 類別               | 句數 | 考什麼                                                             |
| ------------------ | ---- | ------------------------------------------------------------------ |
| `slang`            | 4    | sick＝讚、ghost、chill、sugarcoat                                  |
| `pun`              | 4    | interest／flies／outstanding in his field／put down 雙關           |
| `name_consistency` | 5    | 同一組人名（Sawyer／Alvarez）五句前後一致                          |
| `time_shift`       | 6    | 三組相鄰兩句語意可互換（拳擊手／擂台、七點／九點、馬可斯／艾蓮娜） |
| `simplified_bait`  | 5    | 軟體／影片／品質／資料／冷氣，不得 软件／视频／质量／数据／空调    |
| `tw_lexicon`       | 6    | 計程車、印表機、手機沒電、警察（非公安）、捷運、網路／檔案         |
| `number_unit`      | 3    | 英里不換算、twelve hundred＝1200、fall＝秋天                       |
| `register`         | 3    | 宮廷／街頭／商務三種語域                                           |
| `idiom`            | 4    | count your chickens、kick the bucket、break a leg、last straw      |

## 重建

```bash
python3 scripts/build-golden.py            # 下載兩對字幕、對齊、合併 authored-v1.json → golden-v1.jsonl
python3 scripts/build-golden.py --cache-dir /tmp/vido-golden-src   # 字幕檔快取位置（.srt 不進 repo）
```

手寫的部分全部在 `authored-v1.json`（第二參考譯法、人名表、40 句陷阱）。改了它就重跑腳本；改了句子的**組成**（增刪、換來源）要把 `eval.SampleVersion` 往上 bump（現在是 `golden-v2`），已發布的等級表要全部重評（sub-7-8b）。

## 評分怎麼算

規則層看模型的**原始輸出**（OpenCC 之前，跟管線的品質閘門同一個位置）：`missing`／`empty`／`echoed`／`simplified_leak`（重用 `subtitle.CheckChunk`）＋`name_mismatch`／`time_shift`／`forbidden_term`。任一條失敗該句 0 分。
其餘句子交給固定裁判（`claude-sonnet-5`，rubric `judge-v1`）看**交付後的文字**（OpenCC s2twp＋台灣詞庫），給 0／1／2。
等級沿 eval-1 AC #4：0 分率 ≤5% 且 2 分率 ≥60% → A；只過其一 → B；否則 C。預算打到上限 → `incomplete`，不給等級。

裁判是 Claude，評 Claude 模型時有自評偏誤；每份報告都帶這句 `judge_note`。
