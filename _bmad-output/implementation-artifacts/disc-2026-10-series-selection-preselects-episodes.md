# Disc：在媒體庫選了一部影集，按「批次生成字幕」卻預先勾了整個片庫

Status: done

**Source:** 2026-10-14 NAS 驗證。媒體庫選取模式只選「龍族前傳」一部影集 → 批次生成字幕 → 選片清單預先勾了 658 部、估價 US$239.49（全片庫可抽取的）。Alexyu：「修『選影集時，就預先勾那部影集的集數』」。

## 查到的事（2026-10-14，main `66f9de59`）

- 媒體庫把選取的 id 原樣交給產生字幕：`apps/web/src/components/library/LibraryBrowseV2.tsx:516-520`（`setGenerationBatchSelection([...selectedIds])`）。影集那一列的 id 是**影集**的 id（`:818-819`、`:836-837` 用 `d.id`）。
- 那裡的註解（`:512-515`）寫明：影集的 id 對不到任何候選，所以「不會預選」，集數要自己勾。
- 選片清單決定預選：`apps/web/src/components/subtitle/consent/GenerationConsentView.tsx:217-222`。只拿 `c.mediaId` 比對；比對結果是空的就退回 `defaultSelection`（`consentSelection.ts:51-55`：全部「可抽取」且可寫入的）。所以選了影集 → 對不到 → 全片庫預設。
- 候選集數本來就帶影集 id：`apps/web/src/services/subtitleService.ts:337` 的 `seriesId`。
- 實際對過 NAS：影集「天龍八部」在 `GET /api/v1/series/{id}` 的 id `d4f7a76c-…`，和候選清單裡它 45 集的 `series_id` 是同一個值。
- 龍族前傳本身在候選清單裡 0 集（集數都已經有字幕或不需要產生）。
- 其他會傳預選的地方：只有媒體庫（`selectedMediaIds`，`LibraryBrowseV2.tsx:890`），以及批次視窗的「重試失敗項目／下次繼續」（`GenerationBatchDialogV2.tsx:1326` 的 `retryIds`）。從掃描完成通知進來（`?generate=1`）傳的是空陣列（`LibraryBrowseV2.tsx:525-530`）。

## 規則

1. 選了影集 → 預先勾那部影集**所有能選的**集數（可寫入、有報價；跟選電影時一樣，不分抽取或聽寫）。
2. **使用者有帶選擇進來，就不再退回全片庫預設。** 帶進來的東西一個都勾不到時（例如龍族前傳 0 集、或資料夾都不能寫），清單照常顯示，但什麼都不勾、金額 $0。全片庫預設只留給「沒帶選擇」的入口（掃描完成通知）。
   - 理由：使用者明確選了東西，看到 658 部、US$239 是嚇人的。上限雖然預設 $5 會停，但畫面上的數字就是錯的。

## Acceptance Criteria

1. 媒體庫選一部影集 → 選片清單預先勾它所有能選的集數；其他片都不勾。
2. 電影＋影集混選：電影和那部影集的集數都勾。
3. 選的東西一個都勾不到 → 什麼都不勾（不是全片庫預設）。
4. 沒帶選擇（`?generate=1`）→ 跟現在一樣用全片庫預設。
5. 重試失敗項目／下次繼續：行為不變（它們傳的是集數或電影的 id）。
6. `LibraryBrowseV2.tsx` 那段過時的註解改掉。
7. 測試：預選規則的單元測試（影集、混選、勾不到、沒帶選擇、不能選的集數不勾），加上清單元件一條。

## Tasks / Subtasks

- [x] T1 `consentSelection.ts`：新增預選函式（影集 id 對 `seriesId`；有帶選擇就不退回預設）
- [x] T2 `GenerationConsentView.tsx`：改用它
- [x] T3 `LibraryBrowseV2.tsx`：改註解
- [x] T4 測試

### Time-dependent visual coverage

N/A（選片清單不是 visual 樣本，`tests/e2e/batch-consent-mobile.spec.ts` 開頭有寫原因）。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-14。

### Completion Notes List

- `consentSelection.ts` 新增 `preselectionFor(candidates, ids)`：
  - 沒帶選擇 → `null`，呼叫端用全片庫預設。
  - 有帶選擇 → 比對 `mediaId`，或集數的 `seriesId`；只收能選的（可寫入、有報價）。勾不到就是空集合。
- `GenerationConsentView.tsx` 的 `seedList` 改用它（原本的 5 行換成 1 行）。
- `LibraryBrowseV2.tsx` 那段「影集不會預選」的註解改掉。
- 測試：
  - `consentSelection.spec.ts` 6 條：影集、電影＋影集、單純電影／集數、不能選的集數不勾、勾不到是空的、沒帶選擇是 null。
  - `GenerationConsentView.spec.tsx` 2 條：影集 id 只勾它的集數（$0.31，不是預設的 $0.05）；勾不到就 $0.00。舊程式碼下兩條都會紅（會退回預設的 $0.05）。
- 檢查：vitest（subtitle＋library 42 檔 1,178 個）、`tsc`、`lint:all` 0 errors、`format:check`。
- 本機 E2E：`batch-subtitle.spec.ts` 有 2 條紅，**在 main 上也一樣紅**——本機後端有翻譯模型清單，送出的請求多帶 `model_id`；CI 上這兩條是綠的（#779 的 CI 紀錄）。跟這次改動無關。

### Senior Developer Review (AI) — 2026-10-14（Fable 5.1 跨模型）

結論：0 High／1 Medium／4 Low。

- **M1（立案，不在這張修）**：選 0 部時「開始產生」是灰的，但提示還在講「達到上限會自動暫停」，沒說為什麼不能按。這個狀態以前就有（手動取消全部勾選就會遇到），這次讓「選了影集但勾不到」也會直接進到這個畫面。要加一句說明是文案，也可能動到 visual 樣本，交給 Sally → `disc-2026-10-consent-zero-selected-hint`。
- **L1（接受）**：重試失敗項目／下次繼續會重新分析；如果那些片重掃後變成不能選，以前會退回全片庫預設，現在是全不勾。符合規則 2。
- **L2（接受）**：影集、電影、集數的 id 都是 UUID，不會撞號；`seriesId` 只有集數有。
- **L3（確認）**：新測試在舊程式碼下會紅。
- **L4（已處理）**：story 勾選與 File List。

### File List

- `apps/web/src/components/subtitle/consent/consentSelection.ts`、`.spec.ts`
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx`、`.spec.tsx`
- `apps/web/src/components/library/LibraryBrowseV2.tsx`
- `_bmad-output/implementation-artifacts/disc-2026-10-series-selection-preselects-episodes.md`、`sprint-status.yaml`
