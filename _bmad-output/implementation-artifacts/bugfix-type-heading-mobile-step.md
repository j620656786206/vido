# Story: bugfix-type-heading-mobile-step — 手機上的標題降一階

Status: done

## Story

身為在手機上用 vido 的人，我要頁面大標在手機上小一號，
讓標題不會吃掉半個螢幕（DESIGN.md §Hierarchy「手機只縮標題，不縮內文」）。

## 背景（查到的事，附親自讀過的位置）

- DESIGN.md:351-360 字級表：Display 36→30、H1 30→24、H2 24→20、H3 20→18，H4 以下手機不變。DESIGN.md:596：`sm`（640）是手機／桌機的主分界。
- **原 disc 的兩條事實已過期**：`text-3xl` 就是 H1（30），在表上；程式碼已有 44 處 `sm:text-*`，不是零。真正不在表上的只有 `ColorPlaceholder.tsx:63` 的 `text-5xl`（48）。
- 全 app：`text-4xl` 4、`text-3xl` 3、`text-2xl` 15、`text-xl` 23 處；其中有些已有 `sm:`／`max-sm:` 搭配。
- 沒掛載的 v1 元件：`ColorPlaceholder`、`MediaDetailPanel`、`RecentMediaPanel`、`DownloadPanel`（歸 `disc-2026-09-unmounted-v1-components`）。

## Acceptance Criteria

1. 每個 `text-xl`～`text-4xl` 在 `sm` 以下降一階（`text-lg sm:text-xl` 寫法），已有 `sm:`／`max-sm:` 搭配的不動。
2. 超過 `text-4xl` 的字級不存在（48 → 36）。
3. 守門測試：兩條（無 5xl+；每個標題都有手機搭配），豁免清單寫明理由。
4. 視覺基準、DESIGN.md、sprint-status。

## Tasks / Subtasks

- [x] Task 1 — 16 處補手機降階、48→36（AC #1, #2）
- [x] Task 2 — 守門測試＋mutation check（AC #3）
- [x] Task 3 — 視覺基準、文件（AC #4）

## Dev Agent Record

### Completion Notes List

- **16 處補上**：頁面 h1（活動、探索、搜尋媒體、待解析、下載）24→手機 20；空狀態／錯誤狀態標題、區段標題、對話框標題 20→手機 18；字幕批次對話框的金額讀數 20→手機 18（與生成工作區同一個讀數已有 `max-sm:text-lg`，對齊）。
- **沒動**：`HeroBanner`（已 24/30/36 三段）、`DetailHeroV2` 標題（已 24/30）、`SettingsPageHeader`、登入／精靈／側軌的品牌字（已有 `sm:`）。⚠️ 表上說 Display 用在「詳情頁 hero 標題」，但程式碼是 H1 30——沒有查 `.pen` 確認哪邊對，**這次不改**。
- **豁免（守門測試裡逐條寫理由）**：emoji 當圖示 5 處（🔍🎬👤⚠）；placeholder 首字 4 處（跟方塊大小，不跟視窗）；`routes/test/` 開發頁（改了會讓每張手機截圖背後的 gallery 標題變動，使用者看不到）。
- **Mutation**：把 `routes/pending.tsx` 改回單獨的 `text-2xl`、`ColorPlaceholder` 改回 `text-5xl` → 兩條測試都紅；還原後綠。
- **Spec**：`GlossaryRowV2.spec.tsx` 刪除對話框標題斷言改為 `text-lg` ＋ `sm:text-xl`。
- **視覺**：4 張 darwin 更新（批次對話框手機 ×2、媒體庫手機空狀態、ColorPlaceholder），對應 `-linux` 刪除交 CI。
- 驗證：web 4313/4313、lint 0 errors、prettier 綠、typecheck 綠。

### File List

- 15 個 `apps/web/src/**` 元件／路由（字級）、`ColorPlaceholder.tsx`
- `apps/web/src/styles-type-scale.spec.ts`、`apps/web/src/components/subtitle/GlossaryRowV2.spec.tsx`
- `tests/visual/**`：4 張 darwin 更新、4 張 linux 刪除
- `DESIGN.md`、`_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-28 | 建單＋開發完成（由 `disc-2026-09-type-scale-display-and-mobile-step` 升級）→ review |
| 2026-09-28 | PR #567 合併（1269d7d1），CI 全綠 → done |
