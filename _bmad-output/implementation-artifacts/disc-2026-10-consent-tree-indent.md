# Disc：選片清單的集數看起來在「第 1 季」外面

Status: blocked — 等設計稿

**Source:** 2026-10-13 Alexyu 在本機看 `generation-consent/resumed` 樣本時發現：「第一季有消失的威爾 S01E01 跟瘋狂麥克絲 S01E02，但兩個卡片的長度都比第一季的長度還要長，看起來會不知道那兩集是在第一季裡面」。founder-pm、John、Sally 三方討論後，Alexyu 選 A（照層級縮排）。

## 查到的事（2026-10-13，main `979a25da`）

- 選片清單（`CandidateListPanel.tsx`，F15）在多季影集畫兩層：影集標題列（全寬）→ 季標題列（`ml-6`，24px）→ 集數列。
- 集數列（`CandidateRow`）**沒有縮排**，跟影集標題一樣寬，比季標題還往左突出——看起來像在季外面。
- 電影分「已匹配／未匹配」時也有標題列，電影列同樣沒縮排。
- 設計稿 F15-D-v2（`pwMzT`）只畫了單季（一層「怪奇物語 · 第 4 季」），列都貼齊標題；多季兩層從來沒畫過。

## 設計裁定（Sally，2026-10-13；Alexyu 選 A）

規則：**子項目要比它的標題縮得更多**。

| 在哪一層 | 桌機 | 手機（清單寬度 < 36rem） |
|---|---|---|
| 影集標題、電影分組標題 | 0 | 0 |
| 季標題 | 24（`$Space/xl`） | 12（`$Space/md`） |
| 標題底下的列（電影、單季影集的集數） | 24 | 12 |
| 季標題底下的集數 | 48（`$Space/4xl`） | 24 |
| 沒有標題的列（電影沒分組、改用金額排序） | 0 | 0 |

手機縮得比較少，是因為 390px 的列已經很擠（bugfix-f15-row-mobile-identity-collapse）。

founder-pm：值得修，因為這是選「哪幾集要付錢」的畫面，看錯分組可能選錯；但只修縮排，不改結構。John：「選整部」「選整季」的勾選框要保留；手機上片名與價錢不能被擠掉。

## Acceptance Criteria

1. 照上表縮排。季標題從 `ml-6` 改成手機 12、桌機 24。
2. 縮排用清單自己的寬度判斷（`@xl` 容器查詢，跟列的其他排版一樣），不用螢幕寬度。
3. 金額排序等「沒有群組」的清單不縮排。
4. 390px 手機樣本：片名與價錢都看得到，沒有溢出。
5. 測試：每一層的縮排 class；visual 基準更新（會變的都是故意的）。

## Tasks / Subtasks

- [ ] T0 設計稿（Alexyu 跑 `disc-2026-10-consent-tree-indent-pen-prompt.md` → Sally review）
- [ ] T1 `consentRows.ts`：每個候選列帶上它在第幾層
- [ ] T2 `CandidateListPanel.tsx`：照層級縮排
- [ ] T3 測試、visual

### Time-dependent visual coverage

N/A。

## Dev Agent Record
