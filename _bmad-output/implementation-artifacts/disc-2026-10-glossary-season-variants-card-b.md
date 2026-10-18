# Disc：各季寫法卡片 B（前端）——名詞對照表一個原文一張卡片

Status: backlog（blocked-by `disc-2026-10-glossary-season-variants-card-a`）

**Source:** 同 A 單；Bob 依 Story Splitting Rule 拆出。設計稿：F6s-D-v2 `mHJar`、F6s-M-v2 `o40M1M`、元件 `Component/GlossaryRow-v2/Seasons`（`QkRiY`）／`…/Seasons/Mobile`（`lcSkQ`）、規則 F6s Note `QMmMg`。

## 範圍

1. 同一原文（不分大小寫）的全劇列＋季別列合成一張卡片；季別子列只在季別寫法 ≠ 全劇寫法時顯示，照季數排序。
2. 子列：`↳`＋「第 N 季官方寫法：」＋譯名＋「編輯」「刪除」；沒有徽章、沒有「確認」。S1／S2 藥丸退場。
3. 卡片的「未確認」：全劇列或任何季別列未確認就顯示；卡片的「確認」打全劇列的 confirm（A 單後端會連季別一起確認）。
4. 卡片的「刪除」確認文字要說清楚會連各季寫法一起刪。子列刪除沿用 scope-b 文字。
5. 頁尾「共 N 條 · M 條未確認」數卡片。
6. 主列與子列的「編輯／刪除」用同一個固定間距，「編輯」上下對齊（稿上差 4px 是副作用）。
7. 孤兒季別列（全劇列已不在）：卡片主列只顯示原文，子列照常可編輯／刪除。
8. 畫廊夾具：桌面＋手機各一張《末日光明》卡片稿；視覺基準。

## Acceptance Criteria

（開工時補齊。）
