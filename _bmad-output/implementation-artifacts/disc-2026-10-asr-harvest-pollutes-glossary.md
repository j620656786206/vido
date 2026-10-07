# Disc：聽寫猜出來的人名寫法不能擋住官方字幕的正解——官方的要蓋過猜的

Status: review

**Source:** See S01E02 第二次實測：名詞表 37 條 `source=subtitle` 裡有 7 種 Jerlamarel 的錯拼（傑拉·莫瑞爾、丘拉·莫瑞爾、傑拉瑪瑞爾…），都是自己聽寫＋翻譯後自動寫回的。2026-10-07 修 mine-shift 時發現：這些列還會讓官方字幕學習**跳過** Jerlamarel——`subtitle|Jerlamarel|傑拉瑪瑞爾` 被當成「已知、可信」，miner 只去驗證它、不再學 謝拉馬威；就算學到也因為同一個 term_src 已存在而寫不進去。

## Story

身為 Vido 的使用者，我希望旁邊有官方字幕時，官方的人名寫法會蓋過 Vido 自己猜的，而不是被猜的擋在門外。

## 這張做的範圍（第一半）

1. `knownRenderings`：未確認的 `subtitle` 來源（聽寫收成）不算證據，不交給 miner 當「已知」。
2. miner 寫入：學到的詞如果撞到未確認的 `subtitle` 列 → `Upsert` 覆寫（term_zh、source=official_subtitle、confirmed 仍 false）；撞到 TMDb／手動／已確認的列 → 照舊不動。新增 `terms_replaced` 計數＋log 欄位。

## 沒做的（第二半，另外再議）

- 其他沒被官方詞覆蓋到的錯拼列（「Chola Morel」「Jerla Morel」…）仍留在名詞表、仍會進翻譯提示；要不要在官方字幕存在時整批降級／刪除，或在收成寫回時就先查官方字幕——待討論。

## Acceptance Criteria

1. 名詞表有 `subtitle|Jerlamarel|傑拉瑪瑞爾` 時，miner 仍會學到 `Jerlamarel→謝拉馬威` 並覆寫那一列（`terms_replaced=1`）。
2. 已確認的 `subtitle` 列、TMDb、手動列不被覆寫。
3. `go test ./...`、lint:all 全綠。
4. **實測：** 正式機更新後對 See 重學：`terms_found ≥ 8`、`terms_replaced ≥ 1`；`show_glossary` 的 Jerlamarel 變成 `official_subtitle|謝拉馬威`。

## Dev Agent Record

- **Adversarial CR（2026-10-07，fresh agent）3L，全部處理：**
  - L1 用一般 `Upsert` 覆寫，若使用者在這一瞬間按了「確認」會被蓋掉 → 改成 repo 的 `ReplaceUnconfirmedGuess`：守門寫在 SQL（`WHERE confirmed = 0 AND source IN (subtitle, official_subtitle) AND term_zh != excluded`），順便讓 miner 重跑能更新自己上一次未確認的猜測（CR 提到的既有缺口）。
  - L2 猜對的收成被算成「取代」 → SQL 的 `term_zh != excluded` 讓相同寫法不算寫入。
  - L3 前端沒有 `termsReplaced` 型別 → `MineResult.termsReplaced?`＋卡片句尾「其中 N 個修正了聽寫猜的寫法」＋spec。
  - 寬介面 `GlossaryRepositoryInterface` 不加方法（會弄壞 services 裡一堆假物件）；miner 的窄介面匯出為 `miner.GlossaryRepo`，main 用型別斷言。
- Claude Fable 5.1。🔗 AC Drift: FOUND — sub-7-5a「insert-only」改成「可覆寫未確認的聽寫收成」。📎 Contract Stamps: NONE。🎭 A11y: N/A。🎨 UX: SKIPPED（`terms_replaced` 僅後端 JSON，設定頁區塊沒顯示，之後再補）。

### File List

- `apps/api/internal/subtitle/miner/miner.go`、`miner_test.go`、`apps/api/internal/repository/glossary_repository.go`、`glossary_repository_test.go`、`apps/api/cmd/api/main.go`（改）
- `apps/web/src/services/glossaryMineService.ts`、`apps/web/src/components/settings/OfficialSubtitleMiningCard.tsx`、`.spec.tsx`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | CR 3L：守門搬進 SQL、相同寫法不算取代、前端型別＋句尾。 |
| 2026-10-07 | create-story＋dev 同日（第一半）；全綠，狀態 review，待正式機重學驗證。 |
