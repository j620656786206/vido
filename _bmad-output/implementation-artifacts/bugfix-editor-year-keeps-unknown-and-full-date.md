# Story bugfix：「修改資訊」不會再亂改年份——不知道的年份留空，已知的上映日期不會被砍成 1 月 1 日

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who fixes a film's title or cast in「修改資訊」,
I want saving to leave the release date alone unless I actually change the year,
so that correcting a typo does not rewrite「2016-08-26」as「2016-01-01」, or turn an unknown year into this year.

## Context

升級自 `disc-2026-09-editor-unknown-year-saved-as-this-year`（`poster-upload-a` 的 /ship CR L1 立案）。建單查證時發現**更大的一半**：不只是「未知年份被寫成今年」，而是**每一次儲存都會把真正的上映日期砍成那一年的 1 月 1 日**。

### 🔴 查到的事（main `c6eb696b`；行號皆為現況）

1. **前端把未知年份變成今年**：`LocalDetailV2.tsx:129` 以 `parseInt(date?.slice(0, 4) || '0', 10)` 算年份（沒有日期 → `0`）；`MetadataEditorDialog.tsx` 的 `toFormValues` 用 `data.year || new Date().getFullYear()` → 年份欄預填**今年**。只改片名就按儲存，今年就被寫進去。
2. **後端每次都覆寫日期**（更大的問題）：`services/metadata_edit_service.go:109-111`（電影）`if req.Year > 0 { movie.ReleaseDate = fmt.Sprintf("%d-01-01", req.Year) }`，`:204-206`（影集 `FirstAirDate`）同樣。前端每次都送 `year` → **不管年份有沒有改，上映日期都被改成 `<年>-01-01`**。例：「你的名字」`2016-08-26` 只要按過一次儲存就變 `2016-01-01`。
3. **年份在 API 是必填**：`UpdateMetadataRequest.Validate`（`services/metadata_service.go:263-278`）`Year == 0` → `ErrUpdateMetadataYearRequired` → handler 400（`handlers/metadata_handler.go:379`）；e2e `tests/e2e/metadata-editor.api.spec.ts:125-140`「missing year → 400」鎖住這個行為。
4. **欄位型別**：`movies.release_date TEXT NOT NULL`、`series.first_air_date TEXT NOT NULL`（migration 001／002）；「未知」目前存成空字串。
5. 前端 schema：`year: z.number({ invalid_type_error: '請輸入年份' }).min(1900).max(2100)`（`MetadataEditorDialog.tsx`，poster-upload-a 的 /ship CR 加的），年份欄是 `type="number"` ＋ `valueAsNumber`。

### ⚖️ 建單裁定（2026-09-25，SM；Alexyu 可在 review 推翻）

1. **年份變成「沒改就不送、沒改就不動」**：前端只在使用者真的改了年份欄時才送 `year`；後端 `year` 不送（或 0）＝不動日期。這同時修好 🔴 #1 與 #2。
2. **有改年份時，保留原本的月日**：原本是 `2016-08-26`、改成 2017 → 存 `2017-08-26`；原本沒有日期（空字串）或格式不合 → 存 `<年>-01-01`（沿用現行做法）。理由：使用者改的是「年份」，不該連帶抹掉月日。
3. **未知年份在介面上是空的**：年份欄留空、placeholder「不知道」；留空＝不改。**這次不提供「把已知年份清成未知」**（要動 NOT NULL 欄位語意，收益小）——在 Dev Notes 記錄。
4. **API 的「年份必填」拿掉**：e2e「missing year → 400」改成「沒送年份 → 200 且日期不變」，屬刻意變更（AC drift）。

## Acceptance Criteria

1. **後端：年份可選、只在需要時動日期。**
   - `UpdateMetadataRequest.Year` 變成可選（0 或省略＝不變）；`Validate` 不再要求年份；年份若有值仍要在 1900–2100 之間，否則 400（沿用 `VALIDATION_REQUIRED_FIELD` 以外的既有錯誤碼或新增一個，dev 決定並寫進 Completion Notes）。
   - 電影 `ReleaseDate`／影集 `FirstAirDate`：`year` 為 0 → 不動；有值且與現有日期的年份相同 → 不動；有值且不同 → 原日期是 `YYYY-MM-DD` 則換年份保留月日（`02-29` 換到非閏年 → `02-28`），否則 `<年>-01-01`。
   - Go 測試：沒送年份日期不變；同一年日期不變（`2016-08-26` 仍是 `2016-08-26`）；改年保留月日；原本空日期改年 → `<年>-01-01`；閏年邊界；超出範圍 → 400。電影與影集各一組。
2. **前端：未知就留空、沒改就不送。**
   - `LocalDetailV2.buildEditorMetadata`：沒有日期 → `year: undefined`（不再是 0）。
   - `MetadataEditorDialog`：年份欄沒有值時顯示空白、placeholder「不知道」；schema 年份可選（空＝`undefined`，不跳錯）；有值時仍驗 1900–2100 與整數。**拿掉 `new Date().getFullYear()` 預設**（連同檔頭第 1 行的 Time-bomb 豁免說明一起更新）。
   - 送出時，只有年份欄被改過（react-hook-form `dirtyFields.year`）且有值才帶 `year`；否則不帶。
   - 測試（vitest）：未知年份打開是空的、只改片名送出時**沒有** `year`；改了年份才送；已知年份沒動時不送；清空年份不會送 0、也不跳錯；1800 仍跳「年份必須大於 1900」。
3. **e2e（API）**：`metadata-editor.api.spec.ts` 的「missing year → 400」改成「沒送年份 → 200，`release_date` 不變」；新增「同一年送出 → 日期不變」與「改年保留月日」。刻意變更逐條寫進 Completion Notes。
4. **不准回歸**：`MetadataEditorDialog.spec`、`LocalDetailV2.spec`、`metadata_edit_service_test.go`、`metadata_handler_test.go`、`metadata-editor.api.spec.ts` 其他案例全綠；視覺基準若因 placeholder 或預設值變動而改，照慣例 darwin 更新、`-linux` 交給 CI bootstrap。
5. **CI**：`pnpm nx test api`、`pnpm nx test web`、`pnpm run lint:all`、typecheck 綠；紅／守（Rule 16）；mutation check（至少：後端改回「有年份就寫 1 月 1 日」、前端改回每次都送年份、改回用今年當預設）。

## Tasks / Subtasks

- [x] **Task 1 — 後端：年份可選＋只在年份真的變了才改日期、保留月日（AC: #1）**＋測試
- [x] **Task 2 — 前端：未知留空、沒改不送（AC: #2）**＋測試；更新檔頭豁免說明
- [x] **Task 3 — e2e 與收尾（AC: #3, #4, #5）**

## Dev Notes

### 這張的重點

- **真正的傷害是 🔴 #2**：它每次儲存都在改資料，而且使用者看不出來（詳情頁多半只顯示年份）。先寫會紅的測試。
- **「沒改就不送」前後端都要做**：前端不送是第一道；後端「同一年不動」是第二道，保護其他呼叫者。

### 上游契約（Rule 20）

- 本張不消費 `[@contract-v*]`。`PUT /media/:id/metadata` 的 `year` 從必填變可選——**放寬**，既有呼叫者（前端、e2e helper）照舊能用；「沒送年份 → 400」的行為被刻意移除（AC drift：`3-8-metadata-editor` AC4）。

### 不要做的事

- 不要做「把年份清成未知」（裁定 #3）。
- 不要改 `release_date`／`first_air_date` 的欄位定義。
- 不要動其他欄位的「空值＝不改」行為（例如 `TitleEnglish`），本張只管年份與日期。

### 已知陷阱

- **`valueAsNumber` 空字串是 `NaN`**：schema 用 `z.preprocess` 或 `z.union([z.nan(), …])` 把 NaN 當 undefined；別讓 NaN 送到後端（JSON 會變 null）。
- **`dirtyFields.year`**：`reset` 後才準；`EditorBody` 每次開啟都重新掛載，`defaultValues` 即起點。
- **閏年**：`2020-02-29` 改成 2021 → `2021-02-28`。
- **gh**：`GH_TOKEN=$(gh auth token --user j620656786206)`；網路不穩時推送與建 PR 要重試。

### Source tree

```
apps/api/internal/services/metadata_service.go（Validate）、metadata_edit_service.go（日期邏輯，+tests）  ← Task 1
apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx（+spec）、components/media/LocalDetailV2.tsx（+spec） ← Task 2
tests/e2e/metadata-editor.api.spec.ts                                                                  ← Task 3
```

### Cross-Stack Split Check

後端 1、前端 1（＋e2e）→ 不拆。

### Time-dependent visual coverage

- 本張**移除**唯一一處讀現在時間的預設值（`new Date().getFullYear()`），檔頭豁免說明要跟著改或刪。

### References

- [Source: `apps/api/internal/services/metadata_edit_service.go:100-115, 195-210`；`services/metadata_service.go:249-278`；`handlers/metadata_handler.go:370-395`；migration `001`／`002`]
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx:120-135`；`components/metadata-editor/MetadataEditorDialog.tsx`（`toFormValues`、schema、送出）]
- [Source: `tests/e2e/metadata-editor.api.spec.ts:89-140`；`sprint-status.yaml` → `disc-2026-09-editor-unknown-year-saved-as-this-year`；`3-8-metadata-editor.md` AC4]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **Task 1（後端）**：`withYear(date, year)`（`metadata_edit_service.go`）——`year` 0 或同一年 → 原樣；不同年 → 保留月日（`02-29` 進非閏年 → `02-28`）；原本不是完整日期 → `<年>-01-01`。電影 `ReleaseDate` 與影集 `FirstAirDate` 都改用它。`UpdateMetadataRequest.Validate` 不再要求年份，有送則須 1900–2100，否則 `ErrUpdateMetadataYearOutOfRange` → handler 400 **`VALIDATION_OUT_OF_RANGE`**（新錯誤碼；原本的 `ErrUpdateMetadataYearRequired` 刪除，handler 的「必填」訊息改成只提片名）。測試：`TestWithYear` 8 案、`Validate` 可選且有界、電影只改片名／同一年日期不變、影集改年保留月日；handler「缺年份→400」測試改為「超出範圍→400 `VALIDATION_OUT_OF_RANGE`」。
- **Task 2（前端）**：`LocalDetailV2` 沒有日期 → `year: undefined`（原本是 0）；`MetadataEditorDialog` 年份可選（`z.preprocess` 把空欄位的 `NaN` 當成 undefined，有值仍驗整數與 1900–2100）、placeholder「不知道」、**拿掉 `new Date().getFullYear()` 預設**（連同檔頭 Time-bomb 豁免一起刪除）；送出時只有 `dirtyFields.year` 且有值才帶 `year`。`UpdateMetadataParams.year` 改可選。測試 +4（未知年份打開是空的且不送、已知年份沒動不送、改了才送、清空不報錯也不送）＋ `LocalDetailV2` 未知日期交給編輯器是 undefined（換回舊寫法會紅）。poster-upload-a 加的「清空年份 → 請輸入年份」測試移除（清空現在是合法的「不改」）。
- **Task 3（e2e）**：`metadata-editor.api.spec.ts`「缺年份→400」改寫成四條：沒送年份日期不變、同一年不會砍成 1 月 1 日、換年份保留月日、超出範圍→`VALIDATION_OUT_OF_RANGE`；e2e helper 的 `UpdateMetadataRequest.year` 改可選。本機 19／19；`custom-poster.spec.ts`（走「修改資訊」介面）3／3。跑 e2e 用的 API／前端已關，port 無殘留。
- 🔗 **AC Drift: FOUND** — `3-8-metadata-editor` AC4（年份必填、缺年份→400）→ 年份可選、只在改動時送；超出範圍 400 `VALIDATION_OUT_OF_RANGE`。`poster-upload-a` 的「清空年份顯示請輸入年份」→ 清空＝不改。
- 📎 **Contract Stamps: NONE**。
- 🎭 **A11y Pre-Flight: PASS**（年份欄仍有 label、錯誤仍以 `aria-describedby` 連結；只加 placeholder）。
- 🎨 **UX Verification: PASS**（B′13 的年份欄照舊；只有「未知」時多一個 placeholder，稿中所有片子都有年份，夾具不變、基準不變）。
- **測試**：`nx test api` 綠、`nx test web` **289 files／4307 tests** 綠、`lint:all`（0 error）、typecheck 綠。
- **Mutation 4／4 紅**：改回「有年份就寫 1 月 1 日」、每次都送年份、預設今年、拿掉範圍檢查。
- ⚠️ 本張**不修已經被砍成 1 月 1 日的資料**：以前按過儲存的片子，上映日期已經是 `<年>-01-01`，要重新比對（手動選片）才會拿回真的日期。

### 🔍 /ship Adversarial Review（2026-09-25）

0 HIGH／0 MEDIUM／1 LOW（記錄不修）：
- **L1** 已知年份的片子，使用者把年份欄清空再儲存 → 視為「不改」（日期保留），欄位上卻顯示 placeholder「不知道」；重新打開又會出現原年份。符合裁定 #3（這次不提供「清成未知」），但清空的當下有點誤導。

### Discovery Triage

- N/A — 已被砍的資料屬既成事實，重新比對即可修復，不另立案（見上一條）。

### File List

- `apps/api/internal/services/metadata_edit_service.go`、`metadata_service.go`、`metadata_year_test.go`（新）
- `apps/api/internal/handlers/metadata_handler.go`（+test）
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx`（+spec）
- `apps/web/src/components/media/LocalDetailV2.tsx`（+spec）
- `apps/web/src/services/metadata.ts`
- `tests/e2e/metadata-editor.api.spec.ts`、`tests/support/helpers/api-helpers.ts`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/3-8-metadata-editor.md`（AC drift reference — see Completion Notes）

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-25 | 建單（SM）：由 `disc-2026-09-editor-unknown-year-saved-as-this-year` 升級；查證發現每次儲存都把上映日期砍成 1 月 1 日；裁定年份「沒改不送、沒改不動、改年保留月日」 |
| 2026-09-25 | 實作完成：年份沒改不送、後端沒改不動、改年保留月日、未知留空；狀態 review |
| 2026-09-25 | /ship CR：0H／0M／1L（記錄不修） |
| 2026-09-27 | PR #551 合併（`ceaa107f`），狀態改 done |
