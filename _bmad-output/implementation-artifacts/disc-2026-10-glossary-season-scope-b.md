# Disc：詞彙面板看得出「這個寫法是哪一季的」，刪之前會說清楚後果（前端）

Status: review

**Source:** 2026-10-07 party mode，Sally 的踩雷點：面板看到同一個詞兩條會想刪一條，刪了就把那一季弄壞。

## 範圍

1. 面板列出季子層的詞時加季別徽章（S1／S2）；全劇層不標。
2. 刪除季子層詞的確認文字：「這是第 1 季的寫法，刪掉後第 1 季會改用全劇多數：傑拉馬瑞」。
3. 「學習官方字幕」結果句尾：「其中 N 個詞兩季寫法不同」。
4. 列表 API 要能把子層一起回（或面板自己查 `scope:s<N>`）——與 A 單對齊 API 形狀後再開工。

## A 單 CR 轉過來的兩件事

5. 寫端清抽屜：使用者在面板 Edit／Confirm／Delete 某個全劇層詞時，同步清掉該 `term_src` 的 `scope:s%` 抽屜列（讀端已經不會覆蓋已確認的詞，但刪掉的詞會從抽屜「長回來」）。
6. 產品題：TMDb 種進來的「已知」詞（Maghra→瑪格拉）在第 1 季官方寫法不同（瑪嘉拉）時，目前不會有 s1 抽屜——「種子 vs 官方字幕誰大」要 Alexyu 裁定。

## 不做

- 選擇「以哪季為準」的 UI、forbidden 警告。

依賴：`disc-2026-10-glossary-season-scope-a`。

## 設計（2026-10-07 實作）

- **後端**：`GlossaryTerm.Season`（只在列出時由 scope 推出，不存）；`ListSeasonDrawers(scope)`；`GlossaryService.List` 把每個詞的季抽屜排在它全劇層那一列後面；Edit／Confirm／Delete 全劇層詞時 `ClearSeasonDrawersOfTerm(id)` 清掉該詞未確認的抽屜（抽屜列自己的 id 不會連坐）。
- **前端**：抽屜列多一個 `S1`／`S2` 徽章（中性灰，跟來源徽章同形）；刪抽屜列的確認文字改成「這是第 N 季官方字幕的寫法。刪掉後，補第 N 季的集數會改用全劇的寫法「○○」」；設定頁學習結果句尾「其中 N 個詞各季寫法不同」。
- 沒改 `.pen`：只是既有徽章樣式多一顆、對話框文案多一個分支。

## Acceptance Criteria

1. 面板：See 的 Jerlamarel 顯示三列——全劇「傑拉馬瑞」、`S1`「謝拉馬威」、`S2`「傑拉馬瑞」；Haniwa 只有一列。
2. 刪 `S1` 那列 → 確認文字說明會改用全劇寫法「傑拉馬瑞」；只刪那一列。
3. 編輯／確認／刪除全劇層的 Jerlamarel → 它的兩個抽屜一起清掉。
4. `go test ./...`、web spec、lint:all、typecheck 全綠。

## Dev Agent Record

- **Adversarial CR（2026-10-07，fresh agent）1M/2L＋2 notes：** M1「全部確認」不會處理抽屜 → 「N 條未確認」永遠歸不了零、確認過的全劇列底下還掛著 S1／S2 → `ConfirmAll` 成功後整批清抽屜（此時所有全劇列都是使用者的話，抽屜對查詢已無作用）；L2 `List` 吞掉抽屜查詢錯誤 → 改回傳錯誤；L3 孤兒抽屜（全劇列已刪）的刪除文案說錯 → 改成「全劇層已經沒有這個詞，刪掉後就不在名詞表裡」。Notes：Delete 先清抽屜再刪列沒有交易（not-found 安全）；使用者在面板確認一個抽屜列會讓它永久留下（使用者的話，接受，UI 沒提示）。
- Claude Fable 5.1。🔗 AC Drift: NONE。📎 Contract Stamps: `GlossaryTerm` 多 `season`（additive）。🎭 A11y: 徽章有 `title`。🎨 UX: 微調既有元件、未改 .pen（Sally 未另審）。
- 產品題仍開：TMDb 種子 vs 官方字幕誰大（第 6 點）。

### File List

- `apps/api/internal/models/glossary.go`（＋`glossary_season_test.go`）、`apps/api/internal/repository/glossary_repository.go`（＋test）、`apps/api/internal/services/glossary_service.go`（＋test）
- `apps/web/src/services/glossaryService.ts`、`glossaryMineService.ts`、`apps/web/src/components/subtitle/GlossaryRowV2.tsx`（＋spec）、`GlossaryPanelV2.tsx`、`apps/web/src/components/settings/OfficialSubtitleMiningCard.tsx`（＋spec）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | CR 1M/2L：全部確認清抽屜、List 不吞錯、孤兒文案。 |
| 2026-10-07 | dev：抽屜列出＋徽章＋刪除文案＋寫端清抽屜；全綠，狀態 review。 |

