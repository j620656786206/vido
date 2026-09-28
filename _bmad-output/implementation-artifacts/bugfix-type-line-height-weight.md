# Story: bugfix-type-line-height-weight — 全站行高與標籤字重照 DESIGN.md 字級表

Status: done

## Story

身為唯一的使用者，我要全站的字照設計規範的行距排，
讓中文段落不擠、標籤一眼就認得出是標籤。

## 背景（查到的事，附親自讀過的位置）

- DESIGN.md:356-360 字級表：Body `text-sm` 14/1.625、Label `text-xs` 12/1.5 字重 500，其餘六階各有行高。
- Tailwind v4 預設：`text-xs` 行高 16px（1.333）、`text-sm` 20px（1.429）——拉丁文預設，不是這張表。
- 全 app：`text-xs` 412 處、`text-sm` 693 處；明寫 `leading-*` 的分別只有 6 與 11 行；`text-xs` 411 行只有 91 行寫了字重。
- `apps/web/src/styles.css:487` 已有一個 `@theme` 區塊（動畫），Tailwind 版本 ^4.1.18，支援 `--text-*--line-height`／`--text-*--font-weight`。

## Acceptance Criteria

1. 八階（`text-xs`…`text-4xl`）的預設行高 = DESIGN.md 表上的比例；`text-xs` 預設字重 500。
2. 元素上明寫的 `leading-*`／`font-*` 仍然優先（Tailwind 的 `--tw-leading`／`--tw-font-weight` 機制）。
3. 守門測試讀**真的** DESIGN.md 表格與**真的** styles.css 比對；任一邊改了另一邊沒跟就紅。
4. 視覺基準重拍；`-linux` 交給 CI bootstrap。
5. DESIGN.md「程式碼尚未跟上」段落更新；disc 關閉。

## Tasks / Subtasks

- [x] Task 1 — `styles.css` 新 `@theme` 區塊（AC #1, #2）
- [x] Task 2 — 守門測試＋mutation check（AC #3）
- [x] Task 3 — 視覺基準、DESIGN.md、sprint-status（AC #4, #5）

## Dev Agent Record

### Completion Notes List

- **做法**：不逐處補 class，改在 `@theme` 設 `--text-{xs..4xl}--line-height` 與 `--text-xs--font-weight: 500`。確認建置後的 CSS：`.text-xs{…line-height:var(--tw-leading,var(--text-xs--line-height));font-weight:var(--tw-font-weight,var(--text-xs--font-weight))}`——明寫的 `leading-*`／`font-*` 會設 `--tw-*`，所以仍優先（AC #2）。
- **範圍比 disc 大一點**：disc 只點名 xs／sm，但其他六階一樣是 Tailwind 拉丁預設（例 `text-lg` 1.556 vs 表上 1.5），同一個地方一起設，視覺基準也只重拍一次。字重只動 Label——標題階在程式碼裡本來就逐處明寫 `font-bold`／`font-semibold`。
- **已知副作用**：`--tw-font-weight` 是 `inherits: false` 的註冊屬性，所以「粗體父元素裡的 `text-xs` 子元素」現在會是 500 而不是繼承粗體。基準比對沒看到這類畫面變怪；日後若要子標籤跟著粗，就在子元素上明寫 `font-*`。
- **守門測試**：`styles-type-scale.spec.ts` 新增 3 條。Mutation：把 `--text-sm--line-height` 改成 1.5 → 紅，訊息 `text-sm: table 1.625, css 1.5`；還原後綠。
- **視覺**：354 張 darwin 基準中 347 張有變（全站行高本來就會動到幾乎每張）。人工抽看備份管理、下載表格、媒體庫表格、手機空狀態、按鈕：只有行距變鬆、元件高 1–12px，沒有裁切或重疊。347 張 `-linux` 刪除交 CI。
- **CI 抓到一個真的跳動（E2E homepage-layout CLS）**：首頁讀數帶的「需要注意」格在手機上可能排兩行，格子的最低高度 84px 是照舊行高量的；新行高下兩行格是 90px，於是載入完成時整條帶子長高 6px。改成 `min-h-[90px] sm:min-h-[92px]`，數字在檔頭用規範行高算給你看。順便發現 640–1023 寬度**本來就**短 8px（舊測試只量 390），CLS 測試改成 390 與 768 各量一次。Mutation：拿掉 `sm:min-h-[92px]` → 768 紅（差 2px），390 綠。
- 驗證：web 4311/4311、lint 0 errors、prettier 綠。

### File List

- `apps/web/src/styles.css`
- `apps/web/src/styles-type-scale.spec.ts`
- `apps/web/src/components/homepage/HomeReadoutBand.tsx`、`tests/e2e/homepage-layout.spec.ts`
- `tests/visual/**`：347 `-darwin.png` 更新、347 `-linux.png` 刪除
- `DESIGN.md`、`_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-28 | 建單＋開發完成（由 `disc-2026-09-type-line-height-weight-drift` 升級）→ review |
| 2026-09-28 | PR #564 合併（5ea45595），CI 全綠 → done |
