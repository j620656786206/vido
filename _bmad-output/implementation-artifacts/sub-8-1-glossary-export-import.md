# Story 8.1: 詞彙表匯出／匯入（檔案）—— 驗證「共享」有沒有人要（全端，小）

Status: review

**Depends on:** sub-7-1（scope 綁 TMDb ID；沒有共同 ID 匯入對不上）。B 路線第一步；party-mode 裁定「先做最笨的：匯出一個檔，貼給朋友匯入」。

## Story

As a NAS owner with a friend who watches the same show,
I want to export a show's glossary as a file and import theirs,
so that we stop translating the same characters two different ways — before anyone builds a server for it.

## Acceptance Criteria

1. **格式 `[@contract-v1]`。** JSON：`{format:"vido-glossary", version:1, exported_at, scope:"tmdb:tv:66732", title, language, terms:[{term_src, term_zh, source, confirmed}]}`。只匯出 `tmdb:*` scope；`local:*` 回 409 `GLOSSARY_NOT_SHAREABLE`（新碼，`SUBTITLE_` 前綴下或新 `GLOSSARY_` 前綴——Rule 7 裁定：**用 `GLOSSARY_`**，因為 8-2 還會加碼；同步 code-review instructions.xml 前綴清單，prefix 數 17→18）。
2. **端點。** `GET /media/:id/glossary/export`（下載）、`POST /media/:id/glossary/import`（multipart 或 JSON body；scope 不符 → 400 `GLOSSARY_SCOPE_MISMATCH`；同詞已存在且 `confirmed=1`／`manual` → 跳過不覆寫；其餘 insert-if-absent，`source=community`，`confirmed=0`）。回 `{imported, skipped, conflicts:[{term_src, mine, theirs}]}`。
3. **UI。** `GlossaryPanelV2` 工具列加「匯出」「匯入」；匯入後顯示結果摘要與衝突列表（保留我的／改用他的，逐筆或全部）。
4. **測試。** 匯出 shape；匯入三種情況（新增／跳過／衝突）；scope 不符；`local:*` 拒絕；FE spec + fixture。

## Task 0 提示詞（Sally 2026-10-02；貼給 Pencil Inline AI Agent 原樣執行，一段跑完 ⌘S 再跑下一段）

查證過的事實：F6-D-v2（`dlfMR`）的對話框 `glossary-dialog`（`GGwqu`）頂端 `header-row`（`oRUF2`）已有說明文字＋兩顆 Secondary（`btn-confirm-all` `REsxv`、`btn-add-term` `iNRPj`）；底部 `footer`（`VbxVw`）左邊是「共 6 條 · 3 條未確認」、右邊是空的。F6-M-v2（`buepS`）的 `glossary-sheet`（`Jm7RD`）在清單下方疊兩顆全寬 Secondary（`o938ZZ`、`YMF10`）。匯出／匯入是偶爾才用的動作，**不跟「全部確認」「新增詞彙」搶同一排**——放到 footer 右邊，用 Ghost 按鈕（`StCnR`）。

> **A · F6-D-v2 footer 加匯出／匯入**
>
> 1. 開啟 `dlfMR`，選 `footer`（`VbxVw`），`placeholder:true`。footer 改 `justifyContent: space_between`：左邊維持原本那串計數文字（把它們包進一個 horizontal frame「count」），右邊新增 horizontal frame「exchange」、gap `$Space/xs`，放兩顆 `Component/Button/Ghost`（`StCnR`）實例，文字「匯出檔案」「匯入檔案」，各帶一個 lucide icon（`download`、`upload`，14px，`$text-secondary`）在文字左側。
> 2. `placeholder:false`，`problems` 為空。
>
> **B · F6-M-v2 加匯出／匯入（手機）**
>
> 1. 開啟 `buepS`。在 `btn-add-term`（`YMF10`）下方加一列 horizontal、gap `$Space/sm`：兩顆 `Component/Button/Ghost` 各 `fill_container` 寬，文字「匯出」「匯入」，icon 同 A。sheet 照舊可捲動、長高。`problems` 為空。
>
> **C · 新畫面 F6c-D-v2「匯入結果」（桌機）**
>
> 1. `FindEmptySpace` 在 `dlfMR` 右側，`Copy` `dlfMR`，命名「F6c-D-v2 · 名詞對照表（匯入結果）」，`placeholder:true`。
> 2. 對話框 `body` 內、`header-row` 與 `glossary-list` 之間插入一個 vertical frame「import-result」（`$bg-tertiary` 底、`$radius-md`、內距 `$Space/md`、gap `$Space/sm`）：
>    - 第一行 `$text-primary` Body 600：「已匯入「怪奇物語」的詞彙表」，右側一個 Ghost 小按鈕「收起」。
>    - 第二行 `$text-secondary` Body：「新增 12 個詞（待你確認）· 3 個跟你的一樣，略過 · 2 個跟你的不一樣：」
>    - 兩列衝突（vertical gap `$Space/xs`），每列 horizontal、`justifyContent: space_between`：左邊「Darkling」（`$Type/Family/Mono`、`$text-primary`）＋「你的 闇之手 · 他的 黑暗之主」（`$text-secondary`，「闇之手」「黑暗之主」用 `$text-primary`）；右邊兩顆 Ghost 小按鈕「保留我的」「改用他的」。第二列：「Kirigan」「你的 凱利根 · 他的 基里根」。
>    - 最後一列靠右：兩顆 Ghost「全部保留我的」「全部改用他的」。
> 3. footer 計數改「共 20 條 · 15 條未確認」。`placeholder:false`，`problems` 為空。
>
> **D · 新畫面 F6c-M-v2（手機）**
>
> 1. `Copy` `buepS` 放在 F6c-D 右側，命名「F6c-M-v2 · 名詞對照表（匯入結果・手機）」。在 `explainer` 下方插入同一個「import-result」區塊，衝突列改 vertical：第一行「Darkling」、第二行「你的 闇之手 · 他的 黑暗之主」、第三行兩顆 Ghost「保留我的」「改用他的」各半寬。批次兩顆同樣各半寬。`problems` 為空。
>
> **E · 錯誤態附註（寫成 F6c-D 右側一張 note，同 C9-D Note 慣例）**
>
> 四行：「① 還沒對到 TMDb：匯出／匯入都回『這部片還沒對到 TMDb，詞彙表沒辦法分享。先在詳情頁把它對到正確的 TMDb 條目。』」「② 別部片的檔：『這個檔案是別部片的詞彙表。』」「③ 不是 Vido 的檔：『這不是 Vido 的詞彙表檔。請用 Vido「匯出」產生的 .json 檔。』」「④ 錯誤顯示在 import-result 的位置，`$danger-text`，不用 toast。」

**設計裁定理由**：匯出／匯入一部劇一年用不到幾次，跟每天會按的「全部確認」「新增詞彙」放同一排只會稀釋那兩顆；footer 右側本來就空，Ghost 按鈕的視覺重量剛好是「找得到但不搶眼」。匯入結果放在清單上方而不是 toast：衝突要逐列處理，toast 留不住。「改用他的」就是一次一般的編輯（裁定：編輯＝自動確認），所以按了就是已確認。

跑完之後：⌘S → `python3 scripts/export-pen-screenshots.py` → `SCREENS` 補 F6c-D／F6c-M 的 node ID（`("flow-f-subtitle-v2", "f6c-d-v2")`／`"f6c-m-v2"`）→ stage `f6-d-v2`、`f6-m-v2`、`f6c-d-v2`、`f6c-m-v2` 四張＋`_bmad-output/pen-tokens.json` → commit。

## Tasks / Subtasks

- [x] **Task 1 — 格式 + 端點 + 錯誤碼（AC: #1, #2）**
- [x] **Task 0 — 設計稿**（2026-10-02 Alexyu inline agent；Sally MCP 逐節點複審：F6-D／F6-M Ghost（`StCnR`）匯出／匯入、F6c-D `zv4hT`／F6c-M `x1uKHq` 結果區塊與兩列衝突、字型與色票、note 四行；`problems` 全空、token 檢查綠 210 張。Alexyu 這次沒 commit，由 Claude 代為 commit `e5a65c37`。）
- [x] **Task 2 — UI（AC: #3）**
- [x] **Task 3 — 測試（AC: #4）**

## Dev Notes

- 這支故意**不做**任何網路傳輸。用了才知道值不值得做 8-2。
- 記錄使用：匯出／匯入次數進 log（無遙測）。

### Time-dependent visual coverage

- N/A。

### References

- eval-1 backlog P2-1；party-mode John「先匯出一個檔」

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- **前提已解**：`disc-2026-09-glossary-edit-keeps-machine-source` 裁定（編輯＝自動確認、來源保留）在 PR #649；dsr-6c 早已合併（工具列寬 880）。
- **後端（Task 1）**：`services.GlossaryExchangeService{Export, Import}`（獨立於 `GlossaryServiceInterface`，handler 以 `WithExchange` 掛上，未掛＝404）。格式 `GlossaryExport` [@contract-v1]：`{format:"vido-glossary", version:1, exported_at(UTC), scope, title, language, terms:[{term_src, term_zh, source, confirmed}]}`，只匯出預設語言、依 term_src 不分大小寫排序。匯入：驗格式／版本／scope／上限（5,000 詞、每詞 200 字、body 2MB）→ scope 必須等於本片（不分大小寫）→ 本片沒有的詞 `InsertIfAbsent(source=community, confirmed=false)`；本片已有且譯法相同＝略過；譯法不同＝**一律回 conflict、不覆寫**（含原單說的「confirmed／manual 跳過」與「其餘」——兩者都不蓋，交使用者逐列決定）；檔內重複、空白＝略過。conflict 帶我方列 `id`，「改用他的」就是一般編輯（PUT，伺服器自動確認）。
- 端點：`GET /media/:id/glossary/export`（**檔案本身**、`Content-Disposition: attachment; filename="vido-glossary-tmdb-tv-<id>.json"`，不包 APIResponse）、`POST /media/:id/glossary/import`（JSON body 或 multipart `file`）。錯誤碼 `GLOSSARY_NOT_SHAREABLE`(409)／`GLOSSARY_SCOPE_MISMATCH`(400)／`GLOSSARY_IMPORT_INVALID`(400)。
- **Rule 7**：新增 `GLOSSARY_` 前綴（17→18），`project-context.md` 與 code-review `instructions.xml` 同步；順手補登 sub-7-5b 的 `GLOSSARY_MINE_RUNNING／_FAILED` 與 sub-7-8c 的 `AI_PREVIEW_TOO_SOON`（當時漏登）。
- 匯出／匯入次數記 log（無遙測）。
- 測試 7 條（服務 4：匯出形狀與排序、local 拒絕、匯入新增／略過／衝突、各種拒收不寫入；handler 3：匯出是檔案下載、JSON 與 multipart 匯入、錯誤碼對應含 2MB 上限）。
- 🔗 AC Drift：AC #2「confirmed=1／manual → 跳過不覆寫；其餘 insert-if-absent」→ 實作為「已有的詞一律不覆寫，譯法不同就回 conflict」——insert-if-absent 本來就不會蓋掉任何已存在的列，原文的「其餘」實際上也不會被覆寫；把兩種都列為 conflict，使用者才看得到差異。
- **前端（Task 2）**：`glossaryService.exportFile`（拿檔案本體＋伺服器檔名）／`importFile`（multipart）、`GlossaryExchangeError`（訊息已含後端的「怎麼辦」建議）；`useGlossaryMutations` 加 `exportFile`／`importFile`（匯入後 invalidate 清單）。`GlossaryPanelV2`：桌機 footer 右側 Ghost「匯出檔案／匯入檔案」、手機在「新增詞彙」下方兩顆半寬「匯出／匯入」；空表也能匯入（匯出停用）——朋友的檔是空表最快的起點。新元件 `GlossaryImportResultCard`：清單上方一塊結果（標題、摘要、衝突列「保留我的／改用他的」、兩顆以上衝突才出「全部」），錯誤放在同一個位置（note ④，不用 toast）；「改用他的」走一般編輯（伺服器自動確認），「全部改用他的」逐筆、第一筆失敗就停；重開面板清空。
- 測試：panel spec +8（footer 按鈕、下載、摘要與衝突、保留／改用、全部改用遇錯停、拒收訊息、空表可匯入、重開清空）、既有 2 條依新結構改寫（手機空表的動作區只剩匯出入列；footer gutter 改看外層）；web 全量 4,469 綠。
- 視覺：新增 3 張（結果卡桌機／手機／錯誤）；既有 3 張（seeded／seeded-mobile／empty）因 footer 變了重拍 darwin、刪掉舊 linux 讓 CI bootstrap 重出。逐張對過 F6-D／F6-M／F6c-D。

### Discovery Triage

- 無新單。

### File List

- apps/api/internal/services/glossary_exchange.go、glossary_exchange_test.go
- apps/api/internal/handlers/glossary_handler.go、glossary_handler_test.go
- apps/api/cmd/api/main.go
- apps/web/src/services/glossaryService.ts；hooks/useGlossary.ts；components/subtitle/GlossaryPanelV2.tsx、GlossaryPanelV2.spec.tsx、GlossaryImportResult.tsx；routes/test/-gallery.fixtures.tsx
- tests/visual/…/glossary-panel-v2/{seeded,seeded-mobile,empty}、glossary-import-result/{conflicts,conflicts-mobile,error}
- ux-design.pen、scripts/export-pen-screenshots.py、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/f6-d-v2.png、f6-m-v2.png、f6c-d-v2.png、f6c-m-v2.png
- project-context.md、_bmad/bmm/workflows/4-implementation/code-review/instructions.xml
- _bmad-output/implementation-artifacts/sub-8-1-glossary-export-import.md、sprint-status.yaml


## Change Log

| Date       | Change |
| ---------- | ------ |
| 2026-09-04 | create-story。 |
| 2026-10-02 | dev-story Task 1 後端（Amelia）；Task 0 提示詞（Sally）。 |
| 2026-10-02 | Task 0 設計稿完成（Alexyu）；Task 2–3 前端（Amelia）→ review。 |
