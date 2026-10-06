# Disc（前端）：「生成字幕」對話框跟上後端——依路線顯示說明、價錢、進度與結果

Status: backlog

**Depends on:** `disc-2026-10-single-generate-ignores-embedded-english-a`（後端 API 必須先合併：估價多了 `route`／`plan=extract`、status 多了 `job_id`、solo 終點事件帶 `job_id`）。

**Source:** Alexyu 2026-10-06 party mode 裁定選項 1（見 `-a`）。本單是前端半張。

> ⚖️ 2026-08-06 的文案裁定（「這個按鈕叫語音辨識」，`generateCostView.ts:48`、`ManageSubtitleDialogV2.tsx:874-876`、`sub-2-2c-f5-asr-copy-design.md:124`）已被 -a 推翻。**本單所有新文案與進度條變更需 Sally（ux-designer）重新核定**；下面的字串是建議稿。

## Story

身為 Vido 的使用者，
我希望按「生成字幕」之前就看得出這部片會「直接用片內字幕」「翻譯片內英文」還是「聽聲音」，各要多少錢，按下去之後的進度和結果也說的是同一件事，
這樣我不會被「語音辨識」四個字嚇到不敢按，也不會在跑完時看到一句錯的「已生成英文字幕；尚未翻譯」。

## 查到的事（2026-10-06，Bob 逐一開檔確認）

- **呼叫處**：`transcriptionService.ts`——`startTranscription`（`:99-104`，帶 `?translate=true`）、`startEpisodeTranscription`（`:112-117`）、`getTranscriptionStatus`（`:125-140`，型別 `TranscriptionStatus{inProgress}`）、`getTranscriptionEstimate`（`:150-169`）；`TranscriptionEstimate` 的 `plan: 'full' | 'translate_only'`（`:74-87`，註解寫著 confirmed against dsr-6a AC #2 v1）；`TranscribeStarted{jobId, message}`（`:31-34`）。
- **唯一的元件消費者**：`ManageSubtitleDialogV2.tsx`——估價 `:304`、狀態 `:316`、觸發 `:382-401`（成功就 `generation.startTracking(mediaId)`，**沒把 `jobId` 傳進去**）、重開時自動附掛 `:411-417`。
- **說明文字與封鎖規則**：`generateCostView.ts:48-58` 的常數；`:113-115` `!translateOnly && !data.asrAvailable` → 封鎖＋`ASR_NOT_CONFIGURED_LINE`；`DEFAULT_LINE`「語音辨識＋AI 翻譯，約需數分鐘」是沒有特殊情況時的預設。
- **進度條**：`GenerationProgressV2.tsx:32-39` 凍結六格「提取音訊・轉錄中・翻譯中・簡轉繁・AI校正・完成」（`Implements: Component/GenerationProgress-v2 (XkGvG)`），失敗文案 `:72-76`「提取音訊失敗／轉錄失敗／翻譯失敗」。
- **進度 hook**：`useGenerationProgress.ts`——`transcription_extracting/progress` 與 `translation_progress` 對 phase（`:190-194`）；D6 `probing/extracting → extracting`、`translating → translating`（`:197-201`）；D6 `complete/failed/skipped` 在看過管線 stage 後被當**終點並關掉 SSE**（`:297-328`）；`transcription_complete/failed` 也關 SSE；reducer 的 `COMPLETE` 把 `zhSrtPath` 設成 payload 的值或 null（`:146-158`）；`startTracking(mediaId)`（`:358-369`）。
- **對話框結果文字**：`zhSrtPath` 有就「字幕已生成完成」，沒有就「已生成英文字幕；尚未翻譯」（`ManageSubtitleDialogV2.tsx:751-755`）——D6 `complete` 沒有 `zh_srt_path`，所以抽字幕路線跑完會顯示錯的那句。
- **費用列**：`costTexts` 要 `spentUsd`＋`budgetUsd` 都是數字才顯示（`:1108-1110`）。
- **「語音辨識尚未設定」面板**：503 時顯示（`:785-791`）。
- **工作區單一任務**：只從帶 `title` 的 `transcription_*` 事件產生（`useGenerationJobsFeed.ts:22-35`、`:316-321`）；D6 事件只對批次成員讀。
- **既有測試**：`transcriptionService.spec.ts`（`:21-39` 驗 URL 帶 `translate=true`）、`generateCostView.spec.ts`（9 處引用 `DEFAULT_LINE`／語音辨識）、`ManageSubtitleDialogV2.spec.tsx`（13 處）、`GenerationProgressV2.spec.tsx`、`useGenerationProgress.spec.ts`、`useTranscriptionEstimate.spec.tsx`、`useTranscriptionStatus.spec.tsx`、`useGenerationJobsFeed.spec.ts`；E2E `tests/e2e/manage-subtitle-mobile.spec.ts`（`:40`、`:56`、`:183` 攔 status／estimate／transcribe）、`tests/e2e/glossary-mobile.spec.ts`（`:82`、`:98`）。

## 設計

### 1. 型別（`transcriptionService.ts`）

- `TranscriptionEstimate.plan: 'full' | 'translate_only' | 'extract'`；新增 `route?: 'extract' | 'asr' | 'skip'`。註解改成 `confirmed against [@contract-v2] (Story dsr-6a AC #2, bumped by disc-2026-10-single-generate-ignores-embedded-english-a)`。
- `TranscriptionStatus{inProgress, jobId?}`。

### 2. 說明文字與封鎖（`generateCostView.ts`）——建議稿，Sally 核定

| 情況 | 說明文字 | 按鈕 |
|---|---|---|
| `route=extract`，有翻譯金鑰 | 「片內已有字幕：中文直接使用（不花錢）；英文由 AI 翻譯，約需數分鐘」 | 可按，顯示價錢（估價是英文翻譯的價、往上報） |
| `route=extract`，沒翻譯金鑰 | 「尚未設定翻譯金鑰——片內若是中文字幕仍可直接使用；英文字幕要有金鑰才能翻譯」＋前往設定 | **可按**（降級≠封鎖，sub-2-2d 原則） |
| `route=asr` 或 `skip`，沒語音辨識 | 既有 `ASR_NOT_CONFIGURED_LINE` | 封鎖（跟今天一樣） |
| `route=asr`／`skip`，有語音辨識 | 既有 `DEFAULT_LINE` | 可按 |
| `plan=translate_only` | 既有 `TRANSLATE_ONLY_LINE` | 可按 |
| 沒有 `route`（legacy 後端） | 今天的全部規則 | 不變 |

封鎖規則改成：`!translateOnly && !asrAvailable && route !== 'extract'`。

### 3. 進度（`useGenerationProgress.ts`）

- `startTracking(mediaId, jobId?)`。state 加 `route: string | null`、`trackedJobId`。
- 終點判定：**有 `trackedJobId` 時，只有 `payload.jobId === trackedJobId` 的 `transcription_complete/failed` 才是終點**（才 `closeSSE`、才觸發 `onComplete`）。其他 job_id 的 complete（語音辨識那條路的服務事件）只更新 phase 為「翻譯完成等待收尾」、不關；D6 `complete/failed/skipped` 同樣不再是終點（只更新 message）。沒有 `trackedJobId`（重開附掛、或 legacy 後端沒給 job_id）→ 維持今天的行為。
- 重開附掛：`useTranscriptionStatus` 回 `jobId` 就傳進 `startTracking`。
- 起點事件 `transcription_extracting` 的 `predicted_route` 寫進 `route`；終點事件的 `route` 覆蓋。

### 4. 進度條（`GenerationProgressV2.tsx`）——需要設計稿

- 抽字幕路線（`route` 是 `deliver_direct`／`convert_then_deliver`／`translate`，或預測 `extract`）：格子改成「抽取字幕・翻譯中・簡轉繁・AI校正・完成」五格；`deliver_direct`／`convert_then_deliver` 跑完時「翻譯中」顯示為略過（不是完成）。失敗文案「抽取字幕失敗／翻譯失敗」。
- 語音辨識路線：今天的六格不變。
- **這是 `Component/GenerationProgress-v2 (XkGvG)` 的新變體，要先在 `ux-design.pen` 加變體（Sally），再實作；依 CLAUDE.md 流程重出截圖。**

### 5. 對話框（`ManageSubtitleDialogV2.tsx`）

- 結果文字依終點事件的 `route`／`message`：直接顯示後端 `message`（-a 已依路線寫好四句），不再只看 `zhSrtPath`；`partial` 的那句照舊。
- 「語音辨識尚未設定」面板：只在 503 時出現（後端只對 asr/skip 路線回 503），不用改；但 `:874-876` 的「Verb ruled 2026-08-06」註解改寫成新裁定。
- 觸發成功後 `startTracking(mediaId, outcome.result.jobId)`。
- 費用列：由 solo 終點事件的 `spent_usd/budget_usd` 供給（抽字幕免費路線會是 $0.00，要顯示「沒有花錢」而不是空白——Sally 核定）。

### 6. 工作區（`useGenerationJobsFeed.ts`）

- solo 的起點事件 `transcription_extracting` 帶 `title` → 已能建立單一任務（既有規則）；終點事件帶 `title`＋`job_id` → 已能結束。確認 D6 事件在中間不會把它誤判成批次成員（`pipeline && !member` 會略過）。預期**不用改碼**，但要補一個 spec 驗證「solo 管線 run 在工作區出現、結束」。

### 7. 測試更新

- `transcriptionService.spec.ts`：`plan: 'extract'`、`route`、`jobId` 解析。
- `generateCostView.spec.ts`：§2 表格每列一案；`route` 缺席時舊案全部不變。
- `useGenerationProgress.spec.ts`：job_id 終點判定（三發 complete 只有一發結束）、無 job_id 的舊行為不變。
- `GenerationProgressV2.spec.tsx`：五格變體；gallery fixture（`routes/test/-gallery.fixtures.tsx`）加一組。
- `ManageSubtitleDialogV2.spec.tsx`：結果文字四種、費用 $0.00 的顯示。
- E2E：`manage-subtitle-mobile.spec.ts`／`glossary-mobile.spec.ts` 的 estimate stub 補 `route`；跑那兩支。
- 視覺回歸：進度條新變體要新基準（`-linux` 由 CI bootstrap PR 產生，照 CLAUDE.md）。

## Acceptance Criteria

1. **型別與 ack：** `plan` union 含 `'extract'`、`route`、`jobId` 型別更新，註解寫 `confirmed against [@contract-v2] (Story dsr-6a AC #2)`。
2. **說明文字／封鎖：** §2 表格六列各一個 spec；`route` 缺席時 `generateCostView.spec.ts` 既有案例不改期望值全過。
3. **終點判定：** 有 jobId 時，依序收到「別的 job_id 的 complete」「D6 complete」「同 job_id 的 complete」→ 只有第三發結束追蹤並呼叫 `onComplete`，state 的 `zhSrtPath/route/spentUsd/budgetUsd` 來自第三發；無 jobId 時舊行為不變（既有 spec 全過）。
4. **進度條：** 抽字幕路線五格、`deliver_direct` 跑完「翻譯中」為略過；語音辨識路線六格不變；視覺基準有新變體。設計稿先更新並重出截圖（CLAUDE.md 流程）。
5. **對話框：** 四種路線的結果文字來自後端 `message`；抽字幕免費路線顯示「沒有花錢」；503 面板只在 503 出現。
6. **重開附掛：** status 回 `jobId` 時 `startTracking` 帶它；回 `in_progress:true` 無 `jobId`（批次在跑）時維持今天的附掛行為。
7. **工作區：** solo 管線 run 以單一任務出現並結束（spec）。
8. **Sally 核定紀錄：** 所有新字串與進度條變體在 story 裡留下 Sally 的核定（或修改）紀錄；未核定不得合併。
9. **檢查全綠：** `pnpm run lint:all`、`pnpm nx test web`、`pnpm run test:e2e -- --grep "manage-subtitle|glossary-mobile"`、視覺回歸。
10. **A11y pre-flight**（dev-story step 7）：進度條新變體、新說明文字的 `aria-live` 照既有做法。

## Tasks / Subtasks

- [ ] T0 Sally：核定 §2 文案、§4 進度條五格變體（含 .pen 變體＋截圖）、§5「沒有花錢」文案（AC #8、#4）
- [ ] T1 型別與 service（AC #1）
- [ ] T2 `generateCostView` 路線規則（AC #2）
- [ ] T3 `useGenerationProgress` job_id 終點判定＋`route`（AC #3、#6）
- [ ] T4 `GenerationProgressV2` 五格變體＋gallery fixture＋視覺基準（AC #4）
- [ ] T5 對話框結果文字／費用／startTracking 帶 jobId／註解更新（AC #5、#6）
- [ ] T6 工作區 spec（AC #7）
- [ ] T7 E2E stub＋全綠（AC #9、#10）

前端 7 項、後端 0 項。

## Dev Notes

### 不要做的事

- 不要改後端（都在 -a）。
- 不要把 D6 `subtitle_progress` 的 `complete` 當成 solo 的終點（它沒有 `zh_srt_path`、cost）。
- 不要把「語音辨識尚未設定」面板改成依 `route` 在前端自己判——後端 503 才是真相。
- 不要繞過 Sally 直接改凍結的進度條格子（`GenerationProgressV2.tsx:32` 註明 FROZEN，design handoff）。

### 已知陷阱

- `useGenerationProgress` 的 `d6PipelineSeenRef`（CR M7）邏輯要保留：搜尋引擎也會發 `subtitle_progress`。
- 三發 complete 的順序見 -a「已知陷阱」。
- 「$0.00」顯示：`ButtonCost`／`usd()` 可能把 0 當 falsy；寫 spec。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（進度條與對話框不讀 `Date.now()`；確認 `GenerationProgressV2.tsx`、`ManageSubtitleDialogV2.tsx` 沒有新增時間讀取）。

### References

- `-a` story；`dsr-6a-cost-on-paid-generate-buttons.md`；`sub-2-2c-f5-asr-copy-design.md`、`sub-2-2d-f5-cta-code-strings.md`（被推翻的文案裁定）
- `project-context.md` Rule 20、21、23、24

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-06 | Bob create-story（-b 前端半張，依賴 -a）。 |
