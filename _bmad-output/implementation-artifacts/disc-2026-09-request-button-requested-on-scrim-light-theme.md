# Bugfix: 日巡主題下，海報上的「已請求 · 處理中」看得清楚了

Status: review

**Source:** dsr-2 create-story 盤點時發現（2026-09-16，sprint-status `disc-2026-09-request-button-requested-on-scrim-light-theme`，P2）。Alexyu 2026-10-06 從候選中選定。

## Story

身為用日巡（亮色）主題的 Vido 使用者，
我希望滑過海報時，已經請求過的片寫的「已請求 · 處理中」看得清楚，
這樣我才不會以為還沒按過、又再按一次。

## 查到的事（2026-10-06，Bob 逐一開檔確認）

- **海報卡片的懸停遮罩：** `apps/web/src/components/media/PosterCard.tsx:321` 在海報下緣鋪 `from-[var(--overlay-scrim)]` 漸層，裡面放 `RequestButton`（`:323-330`，`owned={false}`、`requested={!!isRequested}`、`fullWidth`）。只在 `showRequestOverlay`（`:104`：TMDb 片、非選取模式、未入庫、且該頁有接請求狀態）時出現。
- **已請求那顆藥丸：** `apps/web/src/components/requests/RequestButton.tsx:203` 用 `bg-[var(--info-tint)] text-[var(--info-text)]`。
- **日巡的數值：** `apps/web/src/styles.css:255-256` `--info-tint: #1391b233`（20%）、`--info-text: #075469`（深靛青）；`:275` `--overlay-scrim: #0c1512b3`（70% 深色）。
- **實測對比（WCAG，自己算的）：** 日巡下深字壓在「20% 底色＋70% 深色遮罩＋海報」上：黑色海報 **1.82:1**、白色海報 **1.36:1**（要求 4.5:1）。夜行主題的字是亮色（`styles.css:94` `#5bc4dd`），沒有這個問題。
- **用到的地方：** `PosterCard` 被詳情頁「相關推薦」、首頁探索區、探索頁與搜尋頁的 `MediaGrid` 共用（sprint-status 原條目）。`RequestButton` 另外兩個呼叫點 `LocalDetailV2.tsx:351`、`TMDbDetailV2.tsx:106` 放在頁面底色上，不受影響。
- **既有前例（同一類錯誤已修過兩次）：** `HeroBanner.tsx:213-221` 與 `apps/web/src/components/library/PosterCardV2.tsx:136-142`：半透明的 `*-tint` 底色壓在遮罩／海報上時，先墊一層**不透明的 `--bg-secondary`**，讓顏色疊在對比檢查量過的那種底上。HeroBanner 的回歸測試在 `HeroBanner.spec.tsx:244`。

## 設計

照前例：**卡片上的已請求藥丸底下墊一層不透明 `--bg-secondary`**。
- `RequestButton` 新增 `onScrim` 屬性（預設 false）。只有 `PosterCard` 傳 true。
- `onScrim` 時，已請求藥丸外面包一層 `rounded-full bg-[var(--bg-secondary)]`（`fullWidth` 時這層也滿寬）。藥丸本身的顏色、字、圓點不變。
- 詳情頁兩個呼叫點不傳 → 畫面一個像素都不變。
- 「＋ 想要」按鈕是實底（`--accent-primary`），本來就清楚，不動。卡片上 `owned` 永遠是 false，所以「已入庫」藥丸不會出現在遮罩上，不動。
- 設計稿不用改：藥丸的樣子跟 L2 一樣，只是底下多一層不透明底，與 HeroBanner、PosterCardV2 同一個已裁定做法。

## Acceptance Criteria

1. `PosterCard` 上的已請求藥丸（含送出中的轉圈狀態）外層有不透明 `bg-[var(--bg-secondary)]` 的圓角底，兩個主題都有。
2. `RequestButton` 沒傳 `onScrim` 時，DOM 與 class 與現在完全一樣（詳情頁不變）。
3. `fullWidth` 時，外層與藥丸都滿寬、文字置中。
4. 測試：`RequestButton.spec.tsx` 加 `onScrim` 有／沒有兩案；`PosterCard.spec.tsx` 加「已請求的卡片，藥丸外層是不透明底」。
5. 檢查全綠：`pnpm nx test web`、`pnpm run lint:all`、`python3 scripts/check-design-tokens.py`（若存在）。

## Tasks / Subtasks

- [x] T1 `RequestButton` 加 `onScrim`＋外層底（AC #1–#3）
- [x] T2 `PosterCard` 傳 `onScrim`（AC #1）
- [x] T3 測試（AC #4）
- [x] T4 檢查（AC #5）

前端 4 項、後端 0 項 → 不拆單。

## Dev Notes

### 不要做的事

- 不要改 `--info-tint`／`--info-text` 的數值：其他地方（詳情頁、想要清單）用它們都是對的。
- 不要把藥丸改成 `--text-on-scrim` 那一組：那會讓卡片上的藥丸跟 L2 設計稿長得不一樣；墊底是已裁定的做法。
- 不要動 `contrast gate`（`disc-2026-08-contrast-gate-assumes-tints-sit-on-page-grounds` 那張單另外處理）。

### 已知陷阱

- 遮罩只在 `lg:` 寬度、懸停或鍵盤聚焦時出現；單元測試看的是 DOM／class，不是可見性。
- 藥丸有 `ref={pillRef}`（樹狀選單關閉後聚焦用），包外層時 ref 要留在藥丸本身。

### Time-dependent visual coverage

N/A — 不讀時鐘。視覺回歸：遮罩只在懸停時出現，現有 visual 基準沒有拍到這個狀態（`tests/visual` 搜不到 `request-pill`），預期不會有基準要更新。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

- 對比（WCAG，依 `styles.css` 數值計算）：日巡 修前 1.36–1.82:1 → 修後 **5.57:1**；夜行 修後 6.91:1。

### Completion Notes List

- `RequestButton` 新增 `onScrim`：已請求藥丸（含送出中）外包 `request-pill-underlay`（`rounded-full bg-[var(--bg-secondary)]`，`fullWidth` 時滿寬）；沒傳時 DOM 與 class 不變（AC #1–#3）。
- `PosterCard` 懸停遮罩傳 `onScrim`（AC #1）。
- 測試：`RequestButton.spec.tsx` +2（有／沒有 onScrim）；`PosterCard.spec.tsx` +1（遮罩上的按鈕收到 onScrim；該檔把 RequestButton 換成替身，所以底層樣式由 RequestButton.spec 守）（AC #4）。
- 檢查：`pnpm nx test web` 298 檔／4590 測試全綠、`lint:all` 0 errors、`check-design-tokens.py` 一致（AC #5）。
- ⚠️ 沒有在瀏覽器實際截圖：遮罩只在 `lg` 寬度懸停時出現，現有 gallery／visual 沒有「已請求」卡片的固定畫面；驗證靠 class 測試＋對比計算。

### File List

- `apps/web/src/components/requests/RequestButton.tsx`
- `apps/web/src/components/requests/RequestButton.spec.tsx`
- `apps/web/src/components/media/PosterCard.tsx`
- `apps/web/src/components/media/PosterCard.spec.tsx`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/disc-2026-09-request-button-requested-on-scrim-light-theme.md`

## Change Log

- 2026-10-06 Bob create-story：ready-for-dev。Ultimate context engine analysis completed - comprehensive developer guide created.
- 2026-10-06 dev-story：review。
