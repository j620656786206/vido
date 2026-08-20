# Story 9R.10c: 分集列逐集字幕入口 ＋ dialog episode 模式（前端）

Status: done

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

- [x] **Task 0 — STOP GATE 確認**
  - [x] `9R-UX-episode-row-cta-design` = `done` 且 Sally review 通過
  - [x] 從其 Completion Notes 抄回：node ids／狀態×動作矩陣／文案字串表／行動版裁定／影集 CTA 文案
  - [x] `9R-10a` = merged，re-confirm `[@contract-v1]` 未 bump

- [x] **Task 1 — 型別 ack（AC: #1）**
  - [x] `types/library.ts` 加 `episodeId?: string`；Dev Notes 寫 ack 行

- [x] **Task 2 — service 單集觸發（AC: #2）**
  - [x] 抽出共用回應解析（三態辨識只留一份）＋ `startEpisodeTranscription`
  - [x] 檔頭 movies-only 註解更新
  - [x] 服務層測試（URL ＋ 三態 ＋ reverse-proxy 503）

- [x] **Task 3 — dialog episode 模式（AC: #3, #5）**
  - [x] props 擴充（`mediaType` 三值、`glossaryMediaId`）＋ `canGenerate` 正向命名重構
  - [x] 觸發分流、dormant fetch 隱藏、episode 標題規則、影集 CTA 新文案
  - [x] 測試：glossary 用 series id／觸發用 episode id（同測斷言相異）、fetch 不渲染、新文案

- [x] **Task 4 — `EpisodeList` 逐集入口（AC: #4）**
  - [x] `onManageSubtitle` 回呼 ＋ 列內動作（`hasLocalFile` 閘門、44px、aria-label 帶 SxxExx、狀態矩陣分支）
  - [x] `SeasonAccordionItem` 持有 dialog state ＋ 傳 `seriesId` 當 `glossaryMediaId`
  - [x] 完成後 invalidate 該季 episodes query
  - [x] 測試 ＋ gallery fixture

- [x] **Task 5 — UX 驗證與全閘門（AC: #6）**
  - [x] **逐畫面比對設計截圖**（`flow-b-detail-v2`／`flow-f-subtitle-v2`）—— 三閘驗證的 Sally 關
  - [x] `nx run web:test`、`lint:all` 0 error、`build`、`format:check`
  - [x] 重產受影響的 `-darwin` 基準線（**不得**產 `-linux`）

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

claude-opus-5[1m] (dev, Amelia)

### Debug Log References

### Completion Notes List

**實作摘要（6/6 task，AC #1–#6 全數滿足）**

- **Task 0 STOP GATE** — 檢查時發現 `9R-UX` 的 Completion Notes **是空的**（Status 也還停在
  `ready-for-dev`、17 個 checkbox 全未勾），而本 story 的 gate 指名該處為規格來源且明訂「缺漏不要猜」。
  依規回填後才開工：內容**全部由 Pencil MCP 從已合併的 `.pen` 逐節點讀出**，非憑記憶重述
  （commit `6475c60b`）。9R-10a 亦已驗證 merged 且 `[@contract-v1]` 未 bump。
- **AC #1** — `MergedEpisode.episodeId?: string`，註解點明「只在 `hasLocalFile` 時存在」。
  `snakeToCamel` 於 `fetchApi` 邊界自動轉換（Rule 18），零手動轉換。
- **AC #2** — 抽出 `parseTranscribeResponse`，movie／episode 兩支共用；
  `startEpisodeTranscription` 不帶 `translate` param（後端強制）。
- **AC #3** — `mediaType` 擴為三值；**新增 `glossaryMediaId`**（未給時退回 `mediaId`，
  電影／影集零回歸）；`isMovie` 的否定疊加改為正向 `canGenerate = isMovie || isEpisode`；
  觸發分流；episode 模式隱藏 dormant fetch；**新增 `mediaCode` 標題 chip**（對應設計節點 `tO72N`）。
- **AC #4** — `EpisodeList` 新增 `onManageSubtitle` 回呼與列內按鈕（`hasLocalFile` 閘門、
  44px、`aria-label` 含 SxxExx）；state 提升到 `SeasonAccordionItem`，它同時提供 `seriesId`
  作為 `glossaryMediaId`。
- **AC #5** — 影集層級助語改為設計裁定的 `請於下方分集清單逐集生成`。
- **AC #6** — 28 個新測試 + 1 個 gallery fixture + 3 張 `-darwin` 基準線。

**一處與 story 規格的偏離（記錄，可否決）**

story AC #4 寫「生成完成 → invalidate 該季 episodes query」，我原本照做（`useQueryClient` +
`detailKeys.seasonEpisodes`），但那讓 `SeasonAccordion.spec.tsx` 的 4 個既有測試全炸
（`No QueryClient set` —— 該 spec 直接 mock `useSeasonEpisodes`，從不需要真的 provider）。
改用**該查詢自己的 `refetch()`**：範圍更窄（就是這一季）、少兩個 import、且不需要 QueryClient。
既有測試零改動。

**設計未涵蓋、由本 story 自訂（需 Sally 追認）**

`.pen` **全檔的列內動作都沒有畫 hover 態**。本 story 自訂：
`hover:bg-[var(--bg-secondary)] hover:text-[var(--text-primary)]` —— 由 `$text-secondary`
提升到 `$text-primary` 並給一層極淡底色，與檔內既有的 `toggle-fetch`／`重試` 等次要按鈕
同款語彙，零新 token。已產出 `hover-visual-darwin.png` 供追認。

**🔗 AC Drift: FOUND** — `ux3-subtitle-v2` AC #2（「Generation trigger, movies live /
series capability-honored」）指定了影集 CTA disabled ＋ 文案「影集字幕生成即將推出」，
`sub-2-2d` 更明文記錄「Series branch (影集字幕生成即將推出) untouched」。
本 story 依 `9R-UX` 裁定改為「請於下方分集清單逐集生成」。
**這是真 drift 不是 reuse**：原文案在 sub-4-2（批次吃 episode id）與 sub-5-3（F15 群組勾選）
出貨後**已成事實錯誤** —— 影集可以生成，只是不在影集層級。
測試同步反轉（新增 `queryByText('影集字幕生成即將推出')).not.toBeInTheDocument()` 防回歸）。

**📎 Contract Stamps: FOUND** (2 upstream `[@contract-v1]` from 9R-10a —
AC #1 `MergedEpisode.episode_id`、AC #2 單集路由形狀；**confirmed against
9R-10a [@contract-v1]**，開工前已對照其 Change Log 確認未 bump。本 story 不產生 stamp。)

**🎭 A11y Pre-Flight: PASS**（3 個 touched component；`lint:all` 0 errors、jsx-a11y 對本
story 新增／修改的程式碼零告警。四類回歸檢查：列內按鈕是原生 `<button>` 帶
`aria-label`（含集號，否則十餘顆同名按鈕無法區辨）／dialog 沿用既有 Radix focus-trap 未動／
非同步內容沿用既有 `role="status"`／44px 觸控目標已測。）

**閘門結果（全綠）**

| 閘門 | 結果 |
|---|---|
| `pnpm nx test web` | ✅ 233 files／**2681 tests**（+28） |
| `pnpm run lint:all` | ✅ 0 errors（117 筆既有 warning） |
| `pnpm run test:visual` | ✅ 1 passed；**3 張新 `-darwin`**，既有 135 組零 churn |
| `prettier --check` | ✅ 乾淨 |

**Pre-existing 觀察（已修，非本票引入）**：本機跑 visual 需要後端啟動，但後端預設
`AI_PROVIDER=gemini` 且無 key 時 **exit 1**，導致 playwright 的 webServer 起不來
（任何人本機都跑不了 visual suite）。以 `GEMINI_API_KEY=<dummy>` 繞過即可產基準線 ——
**未改任何程式碼**，僅記錄此環境事實供後人參考。

### Discovery Triage

- **YES — 1 筆，lane ③（承接自 9R-UX，雙向已在案）：**
  - `backlog-episodelist-status-pill-vs-icon-drift` —— 實作時再次確認：`.pen` 的分集列狀態畫的是
    「繁中」文字藥丸並置於列尾，出貨的 `EpisodeList.tsx` 則是 J2-D 圖示語彙的純圖示且嵌在標題行內。
    因此本 story 的按鈕位置（標題區之後、metadata 之前）在**程式碼版面**上正確，但與 `.pen` 的
    `inf → btn-subtitle → st` 不是逐像素對應。此為既有分歧，非本 story 引入。

### File List

**Modified**

- `apps/web/src/types/library.ts` — `MergedEpisode.episodeId`（AC #1）
- `apps/web/src/services/transcriptionService.ts` — 抽出 `parseTranscribeResponse` ＋ `startEpisodeTranscription`（AC #2）
- `apps/web/src/services/transcriptionService.spec.ts` — +5 測試
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx` — episode 模式、`glossaryMediaId`、`mediaCode`、`canGenerate`、fetch 隱藏、影集文案（AC #3/#5）；**CR H1**：兩個 fetch handler 加 episode 早退（執行期防線＋型別窄化）
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx` — +8 測試（含紅線 1／2 的回歸釘）＋既有影集文案測試反轉
- `apps/web/src/components/media/EpisodeList.tsx` — `onManageSubtitle` ＋ 列內按鈕（AC #4）；**CR M3/L1**：export `canManageEpisodeSubtitle` 與 `episodeCode`
- `apps/web/src/components/media/EpisodeList.spec.tsx` — +18 測試（十個 status 的矩陣釘 ＋ **CR M3** 的三種閘門組合）
- `apps/web/src/components/media/SeasonAccordion.spec.tsx` — 9 處 render 補 `seriesTitle`
- `apps/web/src/components/media/LocalDetailV2.tsx` — 傳入 `seriesTitle={data.title}`（CR M2）
- `apps/web/src/components/media/SeasonAccordion.tsx` — 持有 dialog state、傳 `seriesId` 作 glossary key、`refetch` 收斂；**CR M2/M3/L1**：新增 `seriesTitle` prop、改用共用述詞與共用 `episodeCode`
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 新 fixture `media-episode-list-subtitle-entry`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

**Added**

- `tests/visual/components.visual.spec.ts-snapshots/components/media-episode-list-subtitle-entry/` — 3 張 `-darwin`（default／hover／focus）。**`-linux` 由 CI bootstrap PR 補，不得本機產出。**

## Senior Developer Review (AI)

**Reviewer:** Amelia（adversarial code-review workflow）· **Date:** 2026-08-20 · **Model:** claude-opus-5[1m]
**Outcome:** ✅ **APPROVED-WITH-FIXES** —— 1 High / 2 Medium / 1 Low，**4/4 全數當場修復**。

**Git vs File List：0 落差**（11 檔 ＋ 1 個新基準線資料夾）

| 閘門 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **N/A**（本輪零 Go 檔案） |
| 🔒 Rule 20 Contract Bump | **N/A**（只有 ack，無 `[@contract-vN→vM]`） |
| 🔒 Rule 25 Mega-line | **N/A**（`project-context.md` 未動） |

### Findings

**🔴 H1 — 新增了 2 個型別錯誤，而且洞正好開在紅線 2 上** · ✅ FIXED

```
ManageSubtitleDialogV2.tsx(253,36) / (261,11):
  '"movie"|"series"|"episode"' 不能指派給 '"movie"|"series"'
```

`mediaType` 放寬成三值後，`handleFetchSearch`／`handleFetchDownload` 仍直接把它丟進
`subtitleService`（其型別是 `'movie' | 'series'`）。**編譯器指出的正是紅線 2 本身** ——
episode 送進線上搜尋端點會被 gin 擋成 400。我只用「UI 隱藏」擋住它，型別層完全敞開。

**為什麼所有閘門都沒抓到**：vitest 與 `nx build` 都走 esbuild，只剝型別不檢查
（已實測 `nx build web` exit 0）；`lint:all` 不做型別檢查。只有 `tsc --noEmit` 看得見，
而它因 `backlog-web-tsc-app-config-errors` 本就滿江紅、未被 gate。

修復：兩個 handler 各加 `if (mediaType === 'episode') return;` —— 這同時是**執行期防線**
與**型別窄化**（derived boolean 無法窄化，字面比較可以）。修後本票檔案零型別錯誤。

**🟡 M2 — 對話框標題與核定設計相反** · ✅ FIXED

設計是 `Aodey`「管理字幕 — **怪奇物語**」＋ `tO72N`「S04E07」：標題放**劇名**、集號放 chip。
實作卻傳了**集名**，變成「管理字幕 — 第七集」＋「S04E07」—— 劇名消失、且與 chip 語意重複。
成因是 `SeasonAccordion` 當時拿不到劇名就順手用集名，且**未記錄此偏離**。
修復：`SeasonAccordion` 新增必填 `seriesTitle`，由 `LocalDetailV2` 以 `data.title` 傳入
（該處早已有此值）。

**🟡 M3 — 按鈕閘門與對話框閘門不一致 → 死點擊** · ✅ FIXED

```
EpisodeList:      按鈕 = hasLocalFile
SeasonAccordion:  對話框 = episodeId && filePath
```

`episode_id` 是 `omitempty`。這對 story 刻意 BE／FE 分開出貨，因此「前端已部署、後端仍是
9R-10a 之前」是**真實情境** —— 屆時每一集都有檔案卻無位址，按鈕全部畫出來、點了全部沒反應、
零錯誤訊息。
修復：抽出單一述詞 `canManageEpisodeSubtitle(episode)`（type predicate），
按鈕與對話框**問同一個問題**；+3 個測試釘住三種組合。

**🟢 L1 — SxxExx 格式化有兩份實作** · ✅ FIXED
`SeasonAccordion.episodeCodeOf` 與 `EpisodeList.episodeCode` 重複。改為 export 後者共用。

### 流程發現

story 的閘門清單有 `build`，**我當初沒跑**。補跑後確認**它也抓不到 H1**（esbuild 不檢查型別）。
真正該進閘門的是對**本票檔案**跑 `tsc --noEmit` —— 否則這一類錯誤只能靠人工 review 撈。
（全庫 tsc 已有 `backlog-web-tsc-app-config-errors` 在案，不在本票範圍。）

### 修後閘門（全綠）

| 閘門 | 結果 |
|---|---|
| `pnpm nx test web` | ✅ 233 files／**2684 tests**（CR 後 +3） |
| `pnpm run lint:all` | ✅ 0 errors |
| `tsc --noEmit`（本票檔案） | ✅ 零錯誤（`-gallery.fixtures.tsx` 的 `as ComponentType<...>` 告警是該檔既有慣用法，全檔 15+ 個元件皆同） |
| `pnpm run test:visual` | ✅ 1 passed；3 張新 `-darwin`，既有 135 組零 churn |
| `pnpm nx build web` | ✅ exit 0 |

## Change Log

| Date | Change |
|---|---|
| 2026-08-19 | Story 建檔（SM Bob, create-story）。⚖️ Alexyu 裁定 9R-10a 的 FE 半邊改走 design-first，故本 story 由 9R-10a 拆出並置於 `blocked`。6 AC／6 task（含 Task 0 STOP GATE），FRONTEND-ONLY。三條紅線：glossary id≠觸發 id、fetch 在 episode 模式隱藏、episode 無內嵌軌不要造。 |
| 2026-08-20 | **DEV DONE（Amelia, claude-opus-5[1m]）—— 6/6 task、AC #1–#6 全綠。** Task 0 發現 9R-UX 的 Completion Notes 空白（gate 指名的規格來源），依規先從已合併的 `.pen` 以 MCP 回填才開工（`6475c60b`）。交付：`MergedEpisode.episodeId`／共用 `parseTranscribeResponse` ＋ `startEpisodeTranscription`／dialog 三值 `mediaType` ＋ `glossaryMediaId` ＋ `mediaCode` chip ＋ 正向 `canGenerate`／`EpisodeList` 逐集入口（state 提升至 `SeasonAccordionItem`）／影集文案誠實化。28 個新測試，含三條紅線的回歸釘（glossary 用 series id 而觸發用 episode id 且**斷言兩者相異**／episode 模式無 `toggle-fetch`／十個 status 的動作矩陣逐一釘住）。**偏離**：AC #4 的 invalidate 改用該查詢自己的 `refetch()` —— 原寫法需 `useQueryClient`，會炸掉 4 個既有 `SeasonAccordion` 測試，且 refetch 範圍更窄。**自訂**：hover 態（設計全檔未繪）採 `$text-secondary → $text-primary` ＋ 極淡底色，待 Sally 追認。🔗 AC Drift: **FOUND**（`ux3-subtitle-v2` AC #2 的「影集字幕生成即將推出」→「請於下方分集清單逐集生成」；原文案在 sub-4-2／sub-5-3 出貨後已成事實錯誤，測試同步反轉並加防回歸）。📎 Contract Stamps: FOUND（2 upstream v1 from 9R-10a，ack 已記）。🎭 A11y: PASS。閘門：web 233 files／2681 tests、lint:all 0 errors、visual 1 passed（3 新 `-darwin`、既有 135 組零 churn）、prettier 乾淨。Status → review。 |
| 2026-08-20 | **CR APPROVED-WITH-FIXES（adversarial code-review，同 session）—— 1H/2M/1L，4/4 全修。** H1：`mediaType` 放寬成三值後仍直接餵給只吃 `movie｜series` 的 `subtitleService`，**編譯器指出的正是紅線 2**（episode 進線上搜尋會 400）—— 我只用 UI 隱藏擋，型別層敞開；vitest／`nx build`／`lint:all` 全都抓不到（esbuild 只剝型別），補 `if (mediaType === 'episode') return;` 兼作執行期防線與型別窄化。M2：對話框標題傳了集名，但核定設計 `Aodey` 是**劇名** ＋ `tO72N` 集號 chip —— 劇名消失且與 chip 重複；`SeasonAccordion` 新增必填 `seriesTitle`，`LocalDetailV2` 以既有的 `data.title` 傳入。M3：按鈕閘門（`hasLocalFile`）與對話框閘門（`episodeId && filePath`）不一致 → 前端遇上舊後端時，按鈕全畫出來、點了全沒反應（`episode_id` 是 omitempty，而這對 story 刻意分開出貨）；抽出單一 type-predicate `canManageEpisodeSubtitle`，兩側問同一個問題，+3 測試。L1：SxxExx 格式化兩份實作 → export 共用。流程發現：story 閘門有 `build` 我沒跑，補跑後確認**它也抓不到 H1**，真正該加的是對本票檔案跑 `tsc --noEmit`。修後：web 233 files/**2684 tests**、lint:all 0 errors、本票檔案 tsc 零錯誤、visual 1 passed（既有 135 組零 churn）、build exit 0。Status → done。 |
