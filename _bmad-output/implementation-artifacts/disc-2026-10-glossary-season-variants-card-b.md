# Disc：各季寫法卡片 B（前端）——名詞對照表一個原文一張卡片

Status: review

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

1. 《末日光明》的清單（Baba Voss 全劇＋s1＋s2）在面板上是一張卡片；s1 寫法不同 → 一行子列；s2 跟全劇一樣 → 不列。
2. 子列只有「編輯」「刪除」；編輯只改那一季（PUT 季別列 id）；刪除文字點名全劇寫法。
3. 卡片的「未確認」＝全劇列或顯示中的子列未確認；「確認」打全劇列 id。
4. 刪卡片的確認文字列出會一起刪的各季寫法。
5. 頁尾數卡片。
6. 主列與子列的「編輯」x 座標相同（桌面、手機都量）。
7. 孤兒季別列：卡片寫「全劇沒有這個詞」，子列可編輯／刪除。
8. 畫廊 `glossary-panel-v2/season-cards`（桌面）與 `…-mobile`（手機）夾具＋視覺基準。
9. web spec、lint:all、typecheck、check-design-refs 綠。

## Dev Agent Record

- `glossaryCards.ts`：`groupGlossaryCards` 純函式（NOCASE 分組、季別照季數排、`shownSeasons` 只留寫法不同的、`unconfirmed` 不算隱藏的季別）。
- `GlossaryRowV2`：抽出 `useTermEditor`／`EditInput`／`SaveCancel`／`DeleteTermDialog` 讓主列與子列共用；新增 `seasons`、`unconfirmed` props 與 `SeasonLine`（`basis-full` 在卡片的 flex-wrap 裡自成一行；手機 `max-sm:basis-auto`）；S1／S2 藥丸與 `fallbackZh` prop 退場；新增 `GlossaryOrphanCardV2`。
- `GlossaryPanelV2`：清單改成卡片，頁尾與「全部確認」用卡片數。
- 本機 Playwright 量 F6s 夾具：桌面「編輯」八個全在 x=1012、「刪除」全在 1072；手機 251／311；清單內 0 溢出。
- 測試：`glossaryCards.spec.ts` 4 條、`GlossaryRowV2.spec` 改寫季別區塊 7 條（取代 4 條 S1 藥丸測試）、`GlossaryPanelV2.spec` 1 條。subtitle＋gallery 943 條綠。
