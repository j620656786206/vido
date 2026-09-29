# Story: disc-2026-09-detail-failed-hides-kept-data — 有資料的片，詳情頁不再說「沒有找到資料」

Status: done

<!-- SM Bob create-story 2026-09-29。⚖️ Alexyu 裁定選 A：已有資料、只是最近一次重新比對失敗（或正在整理）的片，
     詳情頁當一般的片顯示，不出現「沒有找到資料」／「資料還在整理」區塊；不加新 UI、不需新設計稿。
     行號為 main `ecc0c1b0`，改動前；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為打開一部片詳情頁的人，
當這部片其實有資料（海報、簡介、演員），只是上次重新比對剛好失敗或還在整理時，
我要看到的是一般的詳情頁，而不是一塊寫著「沒有找到這部電影的資料」的錯誤訊息。

## 背景（查到的事）

- `apps/web/src/components/media/LocalDetailV2.tsx:176-180`：`noMetadata` 只看 `parseStatus`——`failed` → `'failed'`、`pending` → `'pending'`。
- `noMetadata` 控制四件事：
  - `:309-319` 在內容最上方放 `DetailNoMetadataV2` 區塊；文案在 `DetailNoMetadataV2.tsx:90-97`：failed「沒有找到這部{電影}的資料／自動比對沒有找到符合的作品。你可以自己選對的那一部。」、pending「這部片的資料還在整理／新加入的檔案會在下次掃描後自動比對…」。
  - `:210-218`「管理字幕」從主要（實色）按鈕降為次要。
  - `:237-244`「在地化資訊」（`NfoLocalizeAction`）隱藏。
  - `:404-416` 掛上「手動選片」對話框。
- **更正 CR 的說法**（`disc-2026-09-unmatched-filter-vs-parse-status` /ship CR MED-2 說 `:237` 會「藏掉資料區段」）：`:237` 只藏「在地化資訊」按鈕；簡介（`:321-326`）等區段不看 `noMetadata`，資料**有顯示**。真正的問題是上方那塊**錯誤的**訊息＋主要按鈕被換掉。
- 為什麼會有「有資料卻 failed／pending」：批次重新解析只寫 `parse_status='pending'`（`library_service.go:820,828`），之後 TMDb 斷線時整列連同舊資料被標 `failed`（`enrichment_service.go:715-727`）。細節見 `disc-2026-09-unmatched-filter-vs-parse-status.md` 背景段。
- 前端已拿得到來源：`types/library.ts:38-39`（電影）、`:103-104`（影集）有 `metadataSource?: string`；`LocalDetailV2.tsx:95` 已在用。
- 「有沒有資料」的判準與媒體庫「未匹配」一致（`apps/api/internal/repository/unmatched_condition.go`）：有 `metadata_source` 就是有資料。

## Acceptance Criteria

1. `noMetadata` 只在「`parseStatus` 是 failed／pending **而且** 沒有 `metadataSource`（undefined 或 `''`）」時成立。有來源的 failed／pending 片 → `noMetadata = null`：不出現區塊、「管理字幕」維持主要按鈕、「在地化資訊」照常出現、不掛手動選片對話框。
2. 片名旁的徽章不變（仍依 `parseStatus` 顯示「失敗」／「整理中」，`LocalDetailV2.tsx:302`），讓使用者看得出上次比對的狀態。
3. 沒有來源的 failed／pending 片行為**完全不變**（既有測試、E2E `media-detail.spec.ts` 的兩條、畫廊 B10p/B11p 夾具照過）。
4. 測試（`LocalDetailV2.spec.tsx` 的 `no metadata` describe）：failed＋來源 tmdb、failed＋來源 douban（無 tmdbId）、pending＋來源 tmdb → 無區塊、在地化資訊在、管理字幕是主要按鈕、失敗／整理中徽章仍在；failed＋來源 `''` → 仍有區塊。先紅後綠。
5. 視覺零變動（既有夾具都沒有「有來源的 failed」）。

## Tasks / Subtasks

- [x] Task 1 — `LocalDetailV2.tsx` `noMetadata` 加上來源判斷，更新註解（AC #1, #2）
- [x] Task 2 — 測試先紅後綠（AC #4）
- [x] Task 3 — `pnpm nx test web`、`lint:all`、typecheck 全綠（AC #3, #5）

## Dev Notes

- 只改一個判斷式；不要動 `DetailNoMetadataV2`、不要加新提示（Alexyu 選 A，不選 B「加一行提示＋重新比對」）。
- `lastRematch`（`:188-195`）依 `noMetadata === 'failed'`，有來源時自然為 null，不用另外處理。
- 不要改成看 `tmdbId`（`:176-178` 註解已說明：豆瓣／NFO／手動沒有 tmdbId 但有資料）。

### Project Structure Notes

- 純前端：`components/media/LocalDetailV2.tsx` ＋ spec。前端 3 個 task、後端 0 → 單張。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 立案：`disc-2026-09-unmatched-filter-vs-parse-status` 的 /ship CR MED-2
- 前例：dsr-2b-b（B10p／B11p 無資料區塊的由來）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Debug Log References

- RED：`no metadata` describe 新增 5 條（3 條「有來源 → 一般頁面」＋2 條「來源為 null／空字串仍有區塊」）；改碼前 3 條紅（後 2 條本來就綠，守住舊行為）。
- GREEN：`components/media/` 28 檔／451 條綠。

### Completion Notes List

- `LocalDetailV2.tsx`：`noMetadata` 先看 `Boolean(data.metadataSource)`，有來源一律 `null`；其餘照舊。後端 `NullString` 無值時序列化為 `null`（`apps/api/internal/models/types.go:20-25`），前端 `Boolean(null)` 為 false。
- 全量回歸：`pnpm nx test web` 291 檔／4,351 條綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- /ship 對抗式 CR：0 HIGH／0 MED／2 LOW＋1 NIT。確認前後端每條路徑都送得到 `metadataSource`（handler 直接回 model、`NullString` 無值送 `null`、`snakeToCamel` 轉名），失敗路徑都保留來源。已修 LOW-1（「不掛手動選片對話框」的斷言原本恆真——替身在關閉時回 null；改成替身永遠輸出 `stub-manual-match-mounted` 標記再斷言不存在）、LOW-2（補 `metadataSource: null` 這個 API 真正會送的值）。NIT（story 狀態）CR 讀到的是更新前的檔。
- 🔗 AC Drift: FOUND — `dsr-2b-b-no-metadata-states-frontend` AC #2／#3（無資料區塊只看 `parseStatus`）→ 本張 AC #1（再加「沒有 metadataSource」）。Alexyu 裁定 A 的語意修正；沒有來源的 failed／pending 行為不變。
- 📎 Contract Stamps: NONE（不定義也不引用 wire contract；`metadata_source` 欄位早已在回應裡）。
- 🎭 A11y Pre-Flight: PASS（1 component checked, 0 jsx-a11y warnings introduced；只改一個判斷式）。
- 🎨 UX Verification: PASS — 有來源的片顯示的就是既有的一般詳情頁（B3p／B4p），沒有來源的仍是 B10p／B11p；沒有新畫面。

### Discovery Triage

- ③ backlog-with-carry-forward-link — CR 補充：「手動選片」與單片重新比對唯一的入口是無資料區塊，有資料的片（含本張讓出來的「失敗但資料還在」）沒有介面能改錯配 → `disc-2026-09-no-rematch-entry-for-matched-items`（P3）。

### File List

- apps/web/src/components/media/LocalDetailV2.tsx
- apps/web/src/components/media/LocalDetailV2.spec.tsx
- _bmad-output/implementation-artifacts/disc-2026-09-detail-failed-hides-kept-data.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/dsr-2b-b-no-metadata-states-frontend.md（AC drift reference — see Completion Notes；未修改）

## Change Log

| Date       | Change                                                        |
| ---------- | ------------------------------------------------------------- |
| 2026-09-29 | create-story（SM Bob）：依 Alexyu 裁定 A 建單；更正 CR 對 `:237` 的描述。 |
| 2026-09-29 | dev-story（Amelia）：`noMetadata` 加上來源判斷；5 條新測試（3 條先紅後綠）；全量回歸綠 → review。 |
| 2026-09-29 | /ship 對抗式 CR：修 2 個測試品質問題；立案 `disc-2026-09-no-rematch-entry-for-matched-items`。 |
