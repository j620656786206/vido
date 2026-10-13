# Disc：選片清單的集數看起來在「第 1 季」外面

Status: done

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

- [x] T0 設計稿（Alexyu 跑 `disc-2026-10-consent-tree-indent-pen-prompt.md` → Sally review）
- [x] T1 `consentRows.ts`：每個候選列帶上它在第幾層
- [x] T2 `CandidateListPanel.tsx`：照層級縮排
- [x] T3 測試、visual

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- **Sally review**：
  - 兩段提示詞的文字、層級、縮排都對：F15-D 的電影列與單季集數縮 24、F15-M 縮 12、F15-SPEC-TREE（`ywpxh`）示範兩季加規則表。
  - Inline Agent 沒存檔（磁碟檔跟 HEAD 一樣），Sally 用選單存檔。
  - 「怪奇物語 S4E7」那一列在稿上本來就是收合（隱藏）的，包它的縮排框也跟著隱藏，這是對的。
  - Inline Agent 為季標題新增了一個變數 `bg-tertiary-60`（＝ `bg-tertiary` 加 60% 不透明度，跟程式碼 `bg-[var(--bg-tertiary)]/60` 同一件事）。`check-design-tokens` 通過，追認。
  - 新畫面加進 `SCREENS`（`f15-spec-tree`）；只留 `f15-d-v2.png` 和新的 `f15-spec-tree.png`。
- `consentRows.ts`：每個候選列多帶 `depth`（0／1／2）。沒分組的電影、金額排序是 0；電影分組、單季影集是 1；季底下的集數是 2。
- `CandidateListPanel.tsx`：新增 `INDENT` 表（1：`ml-3 @xl:ml-6`，2：`ml-6 @xl:ml-12`），候選列照 `depth` 縮；季標題從固定 `ml-6` 改成 `ml-3 @xl:ml-6`。
- 測試：`consentRows.spec` 加 4 條層級測試，`CandidateListPanel.spec` 加 3 條 class 測試。
- visual：`generation-consent/f15-collapsed`、`f15-search-hit`、`resumed` 三張故意變了（列縮排）。刪掉 `-darwin` 重出、`git rm` 舊的 `-linux`，交給 CI 補。
- 檢查：vitest 全部 4,499 個測試、`tsc`、`pnpm run lint:all`（0 errors）、`format:check`、本機 visual 全綠。`tests/e2e/batch-subtitle.spec.ts` 只用 testid 找列，沒有量位置。

### Senior Developer Review (AI) — 2026-10-13

結論：0 High／0 Medium／1 Low，接受。

- **L1（接受）**：手機版的 visual 樣本裡，影集群組都是收合的，所以沒有一張樣本拍到手機上縮排後的集數列。縮排在手機只有 12／24px，比桌機少一半；列的排版本來就跟著清單寬度走。

### File List

- `ux-design.pen`、`scripts/export-pen-screenshots.py`、`_bmad-output/screenshots/flow-f-subtitle-v2/f15-d-v2.png`、`f15-spec-tree.png`（新）、`_bmad-output/pen-node-ids.json`、`_bmad-output/pen-tokens.json`
- `apps/web/src/components/subtitle/consent/consentRows.ts`、`.spec.ts`
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`、`.spec.tsx`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/{f15-collapsed,f15-search-hit,resumed}/`
- `_bmad-output/implementation-artifacts/disc-2026-10-consent-tree-indent.md`、`sprint-status.yaml`
