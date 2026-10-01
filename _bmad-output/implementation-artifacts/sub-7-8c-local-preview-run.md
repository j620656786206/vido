# Story sub-7-8c: 還沒評測的模型可以「試跑 20 句」、結果標「你的實測」— 後端＋前端

Status: in-progress

<!-- SM Bob create-story 2026-10-01，由 sub-7-8 拆出；依賴 sub-7-8a（考卷 embed）與 7-8b（等級表讀法）。行號為 main `4004c7af`。
     ⚠️ Task 0 設計稿：ModelPicker 列的三態（尚未評測＋按鈕／試跑中／你的實測）目前**沒有任何 .pen 稿**；要先請 Sally 補 F16／F19 的該列狀態與 J 系列成本按鈕規格，Alexyu 跑 inline agent，再開工 FE。 -->

## Story

身為選到一個「尚未評測」模型的使用者，
我要能花一點小錢用考卷前 20 句試跑它，看到 0 分率、2 分率和真的花了多少，
結果只留在我這台機器、標「你的實測」，不會被當成 Vido 的官方等級。

## 背景（查到的事）

- FE 已有占位文案：`components/subtitle/consent/ModelPicker.tsx:138-140`「· 可花約 $0.01 試跑 20 句」（純文字，不可按；註解點名 P1-8 接手）。`consentSelection.ts:639 ModelChoice` 有 `qualityGrade?／qualityNote?／isBestGrade`。
- 「約 $0.01」只對 Sonnet 等級成立（Haiku 20 句 ≈ $0.004、Opus ≈ $0.06）→ 金額要從 `input_per_1m／output_per_1m` 用考卷 20 句的 token 估算，不寫死。
- 試跑**不是**媒體 run：`subtitle_runs` 以 media 為主鍵，不塞；花費記在回應與 settings 即可（sub-7-6 月報不含試跑——誠實規則，文件寫明）。
- settings 表 `SettingsRepository.SetString`（`repository/settings_repository.go:233`）可存 `models.local_grade.<model_id>` JSON；`GET /settings/models` 讀出填 `local_grade?`（additive，`ModelInfo` 加選填欄位 = 0 bump）。
- 路由注意：`modelSettingsHandler.RegisterRoutes` 必須在 `settingsHandler` 之前（`main.go:1244` 既有註記），新 POST 同組。

## Acceptance Criteria

1. **端點。** `POST /api/v1/settings/models/:id/preview` `[@contract-v1]`：`:id` 必須 `ModelCatalogService.Supports`（否則 400 既有錯誤碼）；用考卷前 20 句（embed）跑 7-8a 的翻譯＋規則層＋裁判；`ai.Budget` 上限 = 估價 ×3（最多 $0.20）；回 `{model_id, cues: 20, zero_rate, natural_rate, cost_usd, judge_model, graded_at}`；同時寫入 settings `models.local_grade.<id>`。同一 model 60 秒內重複呼叫 429（省錢）。
2. **清單帶你的實測。** `GET /settings/models` 每列多 `local_grade?: {zero_rate, natural_rate, cost_usd, graded_at}`（additive）。官方 `quality_grade` 存在時 `local_grade` 仍回，FE 以官方為主。
3. **估價。** `GET /settings/models` 每列多 `preview_estimate_usd`（additive）：考卷 20 句的估計 token × 該模型費率，供按鈕文案。
4. **FE 三態。** ModelPicker 無官方等級的列：「尚未評測 · 試跑 20 句（約 $X）」按鈕 → 試跑中（按鈕 disabled＋spinner＋「約 1 分鐘」）→ 「你的實測：0 分 x%・2 分 y%・花 $z」（`--text-secondary`，不給 A/B/C 字母，不套官方徽章色）；失敗回「試跑失敗，沒有扣款」或帶實際金額。按鈕在 consent 對話框內，不觸發外層 submit。
5. **設計稿。** Task 0：Sally 補 F16-D／F16-M 該列三態＋規格註記（J 系列成本按鈕語彙），Alexyu 跑 inline agent，截圖入 repo，Sally review 後才開工 Task 3。
6. **測試。** handler：400／429／200 shape／settings 寫入／預算超額回 incomplete；FE：三態 spec、按鈕不冒泡、金額文案讀 `preview_estimate_usd`；gallery fixture 三張。

## Task 0 提示詞（Sally 2026-10-01；貼給 Pencil Inline AI Agent 原樣執行，一段跑完 ⌘S 再跑下一段）

查證過的程式碼事實（提示詞依此寫）：`ModelPicker.tsx` 每列第一行＝radio＋display name＋（預設）＋「約 $」；第二行＝等級徽章（`rounded-full px-1.5 py-0.5`，`$bg-tertiary` 底；有等級 `$text-secondary`、沒等級 `$text-muted`）＋「約 N 分鐘」；沒等級的列目前多一段死文字「· 可花約 $0.01 試跑 20 句」。現行 F16-D-v2／F16-M-v2 只畫了 Sonnet 5（A）和 Haiku 4.5（B）兩列，**沒有畫任何「尚未評測」的列**——但程式碼會列出 Opus 4.8、Sonnet 4.6 這些沒等級的模型，所以稿子少了真實會出現的那一列。試跑按鈕會花錢，照 J9「會花錢的按鈕」的規矩：按鈕上直接帶金額、算不出金額就停用。金額來自後端 `preview_estimate_usd`（依模型費率算，Opus 約 $0.06、Sonnet 約 $0.03、Haiku 約 $0.01）。

> **A · F16-D-v2 補第三列（尚未評測＋試跑按鈕）**
>
> 1. 開啟 `gmOt6`（F16-D-v2 · 確認產生字幕）。在「選擇翻譯模型」清單裡，`Copy` 第二列（Claude Haiku 4.5 那一列）貼在它正下方，成為第三列，整張 frame 先 `placeholder:true`。
> 2. 第三列第一行：名稱改「Claude Opus 4.8」，不要「（預設）」；金額改「約 $7.50」（Opus 費率是 Sonnet 的 1.67 倍，Sonnet 列是 $4.50）。
> 3. 第三列第二行：徽章文字改「尚未評測」，徽章字色改 `$text-muted`（底色維持 `$bg-tertiary`）；「約 N 分鐘」改「約 12 分鐘」；在分鐘之後加一顆小尺寸的 `Component/ButtonCost/Default`（`qAERt`）實例：高度 24、水平內距 `$Space/sm`、字級 12，文字「試跑 20 句 $0.06」（金額前的 `$` 是 J9 的固定記號，不另加圖示）。按鈕與分鐘之間 gap `$Space/sm`。
> 4. 對話框高度若因此撐高，讓 DialogFrame 依內容長高（不要裁切）；第三列的 radio 不勾選。`placeholder:false`，確認 `problems` 為空。
>
> **B · F16-M-v2 補第三列（手機）**
>
> 1. 開啟 `x45wBO`（F16-M-v2，bottom sheet）。同 A 的 1–3 步，加第三列「Claude Opus 4.8」。
> 2. 手機版第二行放不下按鈕：第二行只留「尚未評測」徽章＋「約 12 分鐘」；按鈕另起**第三行**，靠左、與第二行對齊（左側縮排與第二行相同，即 radio 寬度＋gap），高度 28（手機觸控）、字級 12、文字同「試跑 20 句 $0.06」。
> 3. sheet 高度依內容長高、可捲動區照舊。`placeholder:false`，`problems` 為空。
>
> **C · 新 spec 畫面 J10「模型列三態・試跑 20 句」**
>
> 1. 用 `FindEmptySpace` 在 `Ls4GO`（J9）正下方找空位，`Copy` `Ls4GO` 當骨架，命名「J10 · 模型列三態（試跑 20 句）」，`placeholder:true`。把 J9 的 A／B／C／D 四段內容清掉，只留版面（標題字級、段落字級、表格列的卡片樣式）。
> 2. **A · 裁定（⚖️ Alexyu 2026-10-01）** 三句：「沒評過的模型可以用考卷前 20 句、用你自己的 key 試跑；結果只留在這台機器，標『你的實測』，永遠不會變成 Vido 的官方等級。」「試跑是會花錢的動作，按鈕照 J9 的規矩直接帶金額；金額由後端依模型費率算，不寫死。」「試跑不是媒體 run：不進 subtitle_runs、不進『本月 AI 花費』。」
> 3. **B · 四個狀態（定稿文案）**，四張卡片列，每列左側是狀態名＋可不可點，中間放該狀態的「第二行」實物（徽章＋分鐘＋按鈕／文字），右側說明：
>    - ① 尚未評測（可點）：徽章「尚未評測」（`$text-muted`）＋「約 12 分鐘」＋ `ButtonCost/Default` 小尺寸「試跑 20 句 $0.06」。說明：「金額＝`preview_estimate_usd`；有官方等級的列**不顯示**這顆按鈕。」
>    - ② 試跑中（暫時不可點）：按鈕換成 `Component/ButtonCost/Loading`（`zhIx7`）小尺寸，文字「試跑 20 句」＋金額位置放骨架；按鈕右側 `$text-secondary` 12px 說明「約 1 分鐘，請勿關閉視窗」。說明：「整個對話框的『確認產生字幕』主按鈕在試跑期間一樣停用，報價不能在試跑時改變。」
>    - ③ 你的實測（可點）：徽章**維持**「尚未評測」；按鈕位置換成一段 `$text-secondary` 12px 文字「你的實測：0 分 5%・2 分 70%・花 $0.05」，文字右側一個 `$text-muted` 12px 的文字連結「再試一次」。說明：「不給 A／B／C 字母、不套官方徽章色——這是你這台機器的 20 句，不是 Vido 的 200 句。」
>    - ④ 試跑失敗（可點，可重試）：按鈕回到 ①，按鈕右側 `$danger-text` 12px「試跑失敗，沒有扣款」；若後端回了金額則「試跑失敗，已扣 $0.02」。
> 4. **C · 邊界情況** 四條：「60 秒內同一模型再按 → 按鈕停用（`ButtonCost/Disabled`，`dqE4G`），說明列『剛剛試跑過，稍後再試』。」「預算打到上限（incomplete）→ 仍顯示 ③，但文字改『只跑了 n 句就到上限：0 分 x%・2 分 y%・花 $z』。」「金鑰未設定 → 按鈕停用，說明『先到 設定 → API 金鑰 存一組 Claude 金鑰』（同 J9 ⑥）。」「`preview_estimate_usd` 缺或 0 → 按鈕停用，不得顯示 $0.00（同 J9 ⑤：算不出就不開放）。」
> 5. **D · 與月報的關係** 一句：「試跑的花費不計入『本月 AI 花費』（K5），也不出現在活動記錄；只在這一列的『你的實測』看得到。」
> 6. `placeholder:false`，確認 `problems` 為空、每張卡片沒有文字溢出。

**設計裁定理由**：試跑按鈕是「會花錢的按鈕」，所以不另造語彙，直接用 J9 的 ButtonCost 三個母版縮小一號（高 24／字 12）放在列內——使用者在同一個對話框裡已經學會「有金額的按鈕＝會扣款」。「你的實測」刻意不用徽章、不用字母，因為徽章＝Vido 背書的等級；這裡是使用者自己花錢在自己機器上量到的 20 句，用一句話呈現、`$text-secondary`，比官方等級低一階。失敗態用文字不用 toast：對話框裡的 toast 會被 sheet 蓋住，而且使用者要看到「有沒有扣錢」就在那一列。

跑完之後：⌘S → `python3 scripts/export-pen-screenshots.py` → `SCREENS` 補 J10 的 node ID（`("flow-j-specs", "j10-d")`）→ 只 stage `f16-d-v2`、`f16-m-v2`、`j10-d` 三張真變更的 PNG → commit。Sally 用 MCP 複審後才開工 Task 3。

## Tasks / Subtasks

- [ ] Task 0 — 設計稿（Sally → Alexyu inline agent → 截圖）（AC #5）
- [ ] Task 1 — preview 端點＋settings 寫入＋節流（AC #1）
- [ ] Task 2 — `GET /settings/models` 加 `local_grade`／`preview_estimate_usd`（AC #2, #3）
- [ ] Task 3 — FE 三態＋fixtures＋specs（AC #4, #6）

## Dev Notes

- BE 2＋FE 1＋設計 1，門檻內不再拆；FE 若因三態與手機版超過 3 task 再拆 c2。
- 試跑花的錢不進 sub-7-6 月報：文案與 docs 要講「試跑費用不計入本月字幕花費」。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-8 拆出）。 |
| 2026-10-01 | dev-story Task 1–2（Amelia，後端，commit a0d71601）；Task 0 提示詞出稿（Sally），等 Alexyu 跑 inline agent。 |
