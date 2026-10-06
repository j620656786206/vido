# Disc：連續同句規則不該把「已經被別條規則刪掉的句子」算進連續數

Status: backlog

**Source:** `disc-2026-10-asr-repeated-lines-dropped` 的 CR L1／L2；2026-10-07 實測看到配樂幻聽每 30 秒一句、剛好 4 句時留下來。

## 問題

- R2b 用文字算連續，已被 R0（music_only）／R1（silence）標掉的同文字段落也算在內；4 句真喊＋1 句被標 silence 的同句 → 5 連 → 3 句真喊被當 `repeat_run` 刪掉。
- `filterPromptEcho` 在 `filterHallucinations` **之前把段落移除**，兩段喊叫中間若夾一句名單回音，移掉後接成一條。
- 實測另見：配樂幻聽是「每 30 秒一句、文字一模一樣」（Whisper 一窗一句）；真的喊叫差幾秒。**時間間隔可當判準**：相鄰兩句同文字相隔 ≥ 20 秒就不是「連喊」，應視為迴圈（不管幾句）。

## 修法

1. `prompt_echo` 改成在 `filterHallucinations` 裡當 R0 標記，不先移除。
2. R2b 只數尚未被標的段落。
3. 新增：同文字、相鄰起點相隔 ≥ 20 秒的重複，兩句起就當迴圈（保留第一句）。
4. 用 10 分鐘片段 run1-prompt 的 24 句「guesthouse」當夾具。

P3；可與 chunk-at-silence 實測一起看還有沒有必要。
