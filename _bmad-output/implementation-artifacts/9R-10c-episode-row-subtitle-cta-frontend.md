# Story 9R.10c: 分集列逐集字幕入口 ＋ dialog episode 模式（前端）

Status: blocked

**Epic:** epic-9R-subtitle-route-c · **Owner:** dev (Amelia) · **🟡 FRONTEND-ONLY**
**Created:** 2026-08-19（SM Bob, create-story）· **Priority:** P2 · **Effort:** M
**Blocked by:** `9R-UX-episode-row-cta-design`（硬阻斷，見下方 STOP GATE）
**Depends on:** `9R-10a-series-episode-trigger`（後端 API 必須先就緒；acks 其 `[@contract-v1]`）

---

## 🛑 STOP GATE（開工前必須確認）

**`9R-UX-episode-row-cta-design` 必須 `done` 且經 Sally MCP review 通過，dev 才可以開始本 story。**

⚖️ **這是 Alexyu 2026-08-19 的裁定**：9R-10a authoring 原本採 function-first
（先實作、之後補設計圖），**Alexyu 推翻，改走 design-first**。

開工時**必須從該設計 story 的 Dev Agent Record 繼承**，一律**不得自行發明**：

1. **`.pen` node ids**（新畫面／修改畫面）
2. **狀態×動作矩陣** —— 十個 `subtitle_status` 各自的動作存在與樣態（其 AC #2）
3. **文案字串表** —— 逐字採用（spec PNG 解析度偏低，字串以文字形式存在 Completion Notes）
4. **行動版裁定** —— 補畫面 or 註記，以及窄螢幕的動作行為
5. **影集層級 CTA 的新文案**（取代過期的「影集字幕生成即將推出」）

若上述任一項在該 story 的 Completion Notes 中缺漏，**不要猜** —— 回頭找 Sally 補齊。

---

## Story

As Alexyu（在影集詳情頁看到某一集缺字幕的人），
I want 直接在那一列按下生成，
so that 我不必為了補一集字幕去跑整輪候選分析、也不必動用整劇批次。

---

## Acceptance Criteria

> ⚠️ 以下 AC 描述的是**機制**。所有**外觀、文案、狀態矩陣、行動版行為**以
> `9R-UX-episode-row-cta-design` 的 Completion Notes 為準，牴觸時**設計優先**。

### AC #1 — 型別：`MergedEpisode` 帶 `episodeId`（ack `9R-10a` `[@contract-v1]`）

- `apps/web/src/types/library.ts` 的 `MergedEpisode` 新增 `episodeId?: string`。
- `fetchApi` 的 `snakeToCamel` 在邊界自動轉換（Rule 18）—— **不得**手動轉換。
- Dev Notes 必須帶 ack 行：`confirmed against 9R-10a [@contract-v1]`（Rule 20 強制）。
- ⚠️ **開工前 re-confirm**：`9R-10a` 若在合併過程中 bump 了版本，先對照其 Change Log。

### AC #2 — `transcriptionService` 新增單集觸發

`apps/web/src/services/transcriptionService.ts` 新增 `startEpisodeTranscription(episodeId: string)`：

- 打 `POST ${API_BASE_URL}/episodes/${episodeId}/transcribe`（**無** `?translate=true` —— 後端強制，見 `9R-10a` AC #4）。
- **完全重用**既有的 `TranscribeOutcome` 三態辨識（503+`TRANSCRIPTION_DISABLED` → `disabled`；
  409+`TRANSCRIPTION_IN_PROGRESS` → `inProgress`；其餘非 2xx → throw）。
- **必須抽出共用的回應解析函式**，兩個方法共用 —— 三態邏輯**不得**複製貼上。
  理由：reverse-proxy 的 503（後端整個掛掉、HTML body → 空 envelope）必須 fail-soft 成「重試」
  而**不是**渲染「尚未設定」的設定 CTA。這個微妙處只該存在一份。
- 檔頭那句「Movies-only today — the series route is 9R-10a」的註解一併更新。

### AC #3 — dialog 支援 `mediaType: 'episode'`，且**詞彙庫 id 與觸發 id 分離**

`ManageSubtitleDialogV2Props`：

- `mediaType` 型別擴為 `'movie' | 'series' | 'episode'`。
- **新增 `glossaryMediaId?: string`** —— 未給時退回 `mediaId`（電影／影集行為 byte-identical，零回歸）。
  episode 模式**必須**由呼叫端傳入 **series id**。
- `useGlossaryTerms(glossaryMediaId ?? mediaId, open)`。
- 生成觸發：episode 走 `startEpisodeTranscription(mediaId)`，movie 走既有 `startTranscription(mediaId)`。
- `isMovie` 這個 boolean 現在**不夠用**：CTA 的 enabled 條件變成「movie 或 episode」。
  請改以正向意圖命名（例：`canGenerate`），**不要**繼續在 `!isMovie` 上疊否定 ——
  sub-2-2b CR L1 修過同一類味道。
- **episode 模式隱藏 dormant fetch secondary**（見紅線 2）。
- 標題／副標如何指認「第幾季第幾集」：**依設計 story 的裁定**（AC #3 of `9R-UX`）。
- 進度：`useGenerationProgress` 以 SSE `transcription_*` 的 `media_id` 對位，episode run 的 `media_id`
  就是 episode id，**無須改動 hook**。

### AC #4 — `EpisodeList` 逐集入口

- 條件：`episode.hasLocalFile === true`（TMDb 有但本地沒有的集**不得**出現任何動作）。
- **形態／文案／哪些狀態出現：依設計 story 的狀態×動作矩陣**，不自行決定。
- 行為：開啟 `ManageSubtitleDialogV2`，`mediaType='episode'`、`mediaId=episode.episodeId`、
  `glossaryMediaId=seriesId`、`mediaFilePath=episode.filePath`、`subtitleStatus`/`subtitleLanguage` 照傳、
  **`subtitleTracks` 不傳**（見紅線 3）。
- **狀態提升**：`EpisodeList` 是 presentational（檔頭明載），**不得**在其中持有 dialog state。
  以 `onManageSubtitle?: (episode: MergedEpisode) => void` 回呼往上，由 `SeasonAccordionItem` 持有 dialog
  （`SeasonAccordion.tsx:34-37` 的 `SeasonAccordionItemProps` 已有 `seriesId`，glossary key 現成）。
- 觸控目標 ≥ 44px；`aria-label` **必須含 `SxxExx`**（同頁十幾列相同文案的按鈕，沒集號無法區辨）。
- 生成完成 → invalidate 該季 episodes query（badge 才會翻）。

### AC #5 — 影集層級 CTA 文案誠實化

`ManageSubtitleDialogV2` 於 `mediaType==='series'` 時，過期文案「影集字幕生成即將推出」
（`:457`，在 sub-4-2／sub-5-3 出貨後已是謊話）取代為**設計 story 裁定的新文案與 CTA 狀態**。

### AC #6 — 測試

- `transcriptionService.startEpisodeTranscription`：URL 正確、三態各一測（含 reverse-proxy 503 fail-soft）
- dialog episode 模式：**詞彙庫查詢用 series id、觸發用 episode id**
  （同一測中斷言兩者相異 —— 紅線 1 的回歸釘）
- dialog episode 模式：dormant fetch 區塊不渲染
- `EpisodeList`：`hasLocalFile=false` 的列無動作；`true` 的列有且 `aria-label` 帶 SxxExx
- 狀態×動作矩陣：設計裁定的每一條分支各一測
- 影集 CTA 新文案
- **視覺基準線**：分集清單新增動作會動到含分集清單的 gallery fixture →
  重產受影響的 `-darwin` 基準線。**`-linux` 一律由 CI `Visual Regression` 的 bootstrap PR 補**
  （CLAUDE.md 硬性慣例，本機是 darwin，**不得**本機產 `-linux`）。

---

## 🔴 三條紅線（踩到就是 CR High）

### 紅線 1 —— 詞彙庫 id ≠ 觸發 id

`/api/v1/media/:id/glossary` 是**逐字**採用 route 上的 id
（`glossary_handler.go:70`：`MediaID: c.Param("id") // route wins — never trust a body media_id`）。
它**不會**幫你把 episode id 解析成 series id。

後端 `glossaryMediaKey`（`transcription_service.go:801-819`，CR sub-5-5 H1 補的）只保護**管線內部**的
餵入與 harvest。前端的 F6 審核面板走的是 HTTP 路由，**沒有那層保護**。

⇒ 若 episode 模式把 episode id 傳給 `useGlossaryTerms`，詞彙庫會讀寫到「沒有任何其他集看得到」的
episode-id 列 —— 正是 sub-5-5 CR H1 修掉的那個 bug 的前端翻版。**AC #3 的 `glossaryMediaId` 分離是強制的。**

### 紅線 2 —— dormant fetch secondary 在 episode 模式必須隱藏

`subtitle_handler.go` 的三處 request binding（`:81`、`:113`、`:133`）都是
`binding:"required,oneof=movie series"` —— 送 `episode` 會被 gin 擋成 400。
線上搜尋對影集是 series 層級的能力，沒有單集面向。

⇒ episode 模式**不渲染**「搜尋線上字幕（成功率低）」整塊（capability honor，
與 dialog 檔頭既有的「CN 轉換動作不渲染因為沒有端點」同一個判準）。

### 紅線 3 —— episode 沒有內嵌字幕軌資料，不要去生一份

`MergedEpisode` 不帶 `subtitle_tracks`（`series_season.go:31-45` 全欄位已列，沒有；`9R-10a` 也只加 `episode_id`）。
episode 模式的 dialog 軌道列會是空的 —— **這是可接受的**：sub-2-2b 修掉的 ladder hole 已讓
authoritative `subtitle_status` 勝過空軌陣列，所以不會誤顯「缺字幕」。

**不得**為了填這格而新增 ffprobe 呼叫或新端點 —— 那是另一支 story 的體積，且逐集 probe 會讓
季展開變成磁碟風暴。若 dev 認為缺這格影響決策，走 Rule 24 lane ③ 立案，不要就地擴張。

---

## Tasks / Subtasks

- [ ] **Task 0 — STOP GATE 確認**
  - [ ] `9R-UX-episode-row-cta-design` = `done` 且 Sally review 通過
  - [ ] 從其 Completion Notes 抄回：node ids／狀態×動作矩陣／文案字串表／行動版裁定／影集 CTA 文案
  - [ ] `9R-10a` = merged，re-confirm `[@contract-v1]` 未 bump

- [ ] **Task 1 — 型別 ack（AC: #1）**
  - [ ] `types/library.ts` 加 `episodeId?: string`；Dev Notes 寫 ack 行

- [ ] **Task 2 — service 單集觸發（AC: #2）**
  - [ ] 抽出共用回應解析（三態辨識只留一份）＋ `startEpisodeTranscription`
  - [ ] 檔頭 movies-only 註解更新
  - [ ] 服務層測試（URL ＋ 三態 ＋ reverse-proxy 503）

- [ ] **Task 3 — dialog episode 模式（AC: #3, #5）**
  - [ ] props 擴充（`mediaType` 三值、`glossaryMediaId`）＋ `canGenerate` 正向命名重構
  - [ ] 觸發分流、dormant fetch 隱藏、episode 標題規則、影集 CTA 新文案
  - [ ] 測試：glossary 用 series id／觸發用 episode id（同測斷言相異）、fetch 不渲染、新文案

- [ ] **Task 4 — `EpisodeList` 逐集入口（AC: #4）**
  - [ ] `onManageSubtitle` 回呼 ＋ 列內動作（`hasLocalFile` 閘門、44px、aria-label 帶 SxxExx、狀態矩陣分支）
  - [ ] `SeasonAccordionItem` 持有 dialog state ＋ 傳 `seriesId` 當 `glossaryMediaId`
  - [ ] 完成後 invalidate 該季 episodes query
  - [ ] 測試 ＋ gallery fixture

- [ ] **Task 5 — UX 驗證與全閘門（AC: #6）**
  - [ ] **逐畫面比對設計截圖**（`flow-b-detail-v2`／`flow-f-subtitle-v2`）—— 三閘驗證的 Sally 關
  - [ ] `nx run web:test`、`lint:all` 0 error、`build`、`format:check`
  - [ ] 重產受影響的 `-darwin` 基準線（**不得**產 `-linux`）

---

## Dev Notes

### 跨端拆分檢查（Epic 8 Retro Agreement 5）

**FRONTEND-ONLY，BE task 數 = 0 ⇒ 門檻不適用。** BE 半邊 = `9R-10a`（已拆出，可獨立先行）。

### Rule 20 —— AC Contract Versioning

- **消費：** `9R-10a` AC #1（`MergedEpisode.episode_id`）與 AC #2（單集路由形狀），皆 `[@contract-v1]`。
  Dev Notes **必須**帶 `confirmed against [@contract-v1]` 行 —— 缺這行是 HIGH severity gap（retro-10-AI5 AC #3）。
- **產生：** 無（本 story 不定義 wire contract）。
- ⇒ Completion Notes：`📎 Contract Stamps: FOUND (2 upstream v1 from 9R-10a, ack lines present; this story stamps nothing)`

### Rule 23 —— Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.**
已驗證（2026-08-19 create-story）：`EpisodeList.tsx`、`SeasonAccordion.tsx`、`ManageSubtitleDialogV2.tsx`
三檔對 `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()` 的 grep 皆**零命中**。
`MergedEpisode.airDate` 是伺服器供給的字串，僅做格式化顯示，不與本機時鐘比較。

### Project Structure Notes

- 修改：`types/library.ts`、`services/transcriptionService.ts`（+ spec）、
  `components/subtitle/ManageSubtitleDialogV2.tsx`（+ spec）、`components/media/EpisodeList.tsx`（+ spec）、
  `components/media/SeasonAccordion.tsx`、`routes/test/-gallery.fixtures.tsx`。
- **零新元件檔**（除非設計裁定需要）、零後端檔案。

### References

- [Source: `_bmad-output/implementation-artifacts/9R-UX-episode-row-cta-design.md`] — **規格來源**（STOP GATE）
- [Source: `_bmad-output/implementation-artifacts/9R-10a-series-episode-trigger.md`] — 後端契約 `[@contract-v1]`
- [Source: `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:11-14,443,457`] — 影集死 CTA 與過期文案
- [Source: `apps/web/src/components/media/EpisodeList.tsx:36-52,98-110,115,163`] — J2-D icon grammar／`untranslated` 列／`hasLocalFile` 閘門
- [Source: `apps/web/src/components/media/SeasonAccordion.tsx:34-37`] — `SeasonAccordionItemProps` 已有 `seriesId`
- [Source: `apps/api/internal/handlers/glossary_handler.go:70`] — glossary route 逐字採用 route id（紅線 1）
- [Source: `apps/api/internal/handlers/subtitle_handler.go:81,113,133`] — 搜尋 binding `oneof=movie series`（紅線 2）
- [Source: `project-context.md`] — Rule 18／20／22／23／24
- [Source: `CLAUDE.md`] — `-linux` 基準線只能由 CI bootstrap PR 產生

---

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date | Change |
|---|---|
| 2026-08-19 | Story 建檔（SM Bob, create-story）。⚖️ Alexyu 裁定 9R-10a 的 FE 半邊改走 design-first，故本 story 由 9R-10a 拆出並置於 `blocked`。6 AC／6 task（含 Task 0 STOP GATE），FRONTEND-ONLY。三條紅線：glossary id≠觸發 id、fetch 在 episode 模式隱藏、episode 無內嵌軌不要造。 |
