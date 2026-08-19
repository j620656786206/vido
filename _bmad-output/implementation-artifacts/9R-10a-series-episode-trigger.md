# Story 9R.10a: 單集手動生成入口 —— `POST /episodes/:id/transcribe`（後端）

Status: done

**Epic:** epic-9R-subtitle-route-c（Track 5 — Orchestration 收尾）· **Owner:** dev (Amelia)
**Created:** 2026-08-19（SM Bob, create-story）· **Priority:** P2 · **Effort:** S–M · **🔴 BACKEND-ONLY**
**Depends:** 9R-10 ✅(#139)、9R-16 ✅(#147)、9R-18 ✅、sub-2-2a ✅、sub-3-2 ✅、sub-4-2 ✅ —— 全數 merged，**無 blocking gate，可立即開工**。
**Pair:** FE 半邊 = `9R-10c-episode-row-subtitle-cta-frontend`（blocked，等 `9R-UX-episode-row-cta-design`）。本 story **不等它**。

---

## ⚖️ 範圍重新界定（authoring 時的第一件事，請先讀完再動手）

本票 2026-07-05 立案時寫的是「影集完全不能生成字幕，blocks Epic 6 TV generation」。
**這句話今天已經不成立。** create-story 盤點（2026-08-19，逐檔 grep 驗證）發現這六週內影集能力已由
M2/M3 陸續補齊：

| 立案時假設的缺口 | 現況（已驗證） |
|---|---|
| `TranscriptionService` 綁死 movie repo | ✅ sub-3-2 已 media-type-aware（`WithMediaType` dispatch，`transcription_service.go:375-389`；episode writer/reader 於 `main.go:563-566` 注入） |
| 批次不吃影集 | ✅ sub-4-2 D1：`POST /subtitles/generation-batch` 的 `scope=selected` 已接受混合 movie＋episode UUID（`generation_batch_handler.go:26-30` doc、`generation_batch.go:419`） |
| 整劇要逐集點 | ✅ sub-5-3：F15 候選清單已有 series／season 三態群組勾選（#227） |
| 自動管線不掃影集 | ✅ sub-3-1/3-2：`worker_pool.go:414` episode sweep + ASR fallback 腿 |

**因此真正剩下的真空只有一個：單一集的「手動即時觸發」入口** —— 電影有
`POST /api/v1/movies/:id/transcribe`（`transcription_handler.go:47`），影集沒有對應物。
使用者想針對「這一集」立刻重生字幕，今天唯一的路是跑一整輪候選分析 sweep 再從清單挑一列，繞得太遠。

> **⚖️ RULED 2026-08-19（Alexyu, create-story 裁定）：採「補單集手動入口（BE+FE）」。**
> 否決的替代方案：標 superseded、只把死掉的影集 CTA 改成導向同意流程畫面。
>
> **⚖️ RULED 2026-08-19（Alexyu，第二次裁定）：FE 半邊改走 design-first。**
> 原 authoring 的 function-first 裁定（明示可推翻）**已被推翻**。因此本 story 從 BE+FE 單支
> **拆為 BE 半邊**，FE 半邊獨立為 `9R-10c`（blocked on `9R-UX-episode-row-cta-design`）。
> BE 不等設計輪 —— 路由形狀與 `episode_id` 欄位不受逐集按鈕長相影響。

---

## 兩個 BE 缺口（grep 驗證，非推測）

1. **無單集路由。** `TranscriptionHandler.RegisterRoutes`（`apps/api/internal/handlers/transcription_handler.go:46-48`）
   只註冊 `POST /movies/:id/transcribe`。`grep -rn "episodes/:id" apps/api/internal/handlers` 唯一命中是
   `series_handler.go:321` 的 `GET /series/:id/seasons/:seasonNumber/episodes` —— 沒有任何 episode 動作路由。

2. **前端根本無法定址單集。** `services.MergedEpisode`（`apps/api/internal/services/series_season.go:31-45`）
   **不帶 episode row id**（欄位只有 `episode_number`/`name`/…/`has_local_file`/`subtitle_status`/
   `subtitle_language`/`file_path`）。沒有 id 就沒有 `POST /episodes/{id}/transcribe` 可打 ——
   這是 FE 半邊的真正前置，**由本 story 交付**，不是加值。

---

## Story

As Alexyu（NAS 影集觀眾），
I want 後端提供對**單獨一集**的字幕生成觸發端點，
so that 前端（9R-10c）能在分集清單上做出逐集入口，而我不必為了補一集字幕去跑整輪候選分析。

---

## Acceptance Criteria

### AC #1 — `MergedEpisode` 帶 episode row id（additive）`[@contract-v1]`

`services.MergedEpisode` 新增：

```go
// EpisodeID is the local episode ROW id (UUID string, 9R-18) — the address the
// per-episode transcribe route takes. Empty when the season has no local row
// for this TMDb episode (has_local_file=false); consumers gate on
// has_local_file, never on a non-empty id alone.
EpisodeID string `json:"episode_id,omitempty"`
```

- 於 `series_season.go:153` 的 `if local, ok := ...` 區塊內填 `merged.EpisodeID = local.ID`
  —— **與 `HasLocalFile=true` 同一個分支**，不得在分支外填（沒有本地檔的 TMDb 集不該帶 id）。
- Additive、`omitempty`、零 migration、既有欄位一個不動。
- **消費者：`9R-10c`**（前端 `types/library.ts` 的 `MergedEpisode` 加 `episodeId?: string`；
  `fetchApi` 的 `snakeToCamel` 自動轉換，Rule 18 —— 前端不手動轉）。本 story **不改前端型別**。

### AC #2 — `POST /api/v1/episodes/:id/transcribe` `[@contract-v1]`

鏡射電影路由的**每一道閘門**（`transcription_handler.go:52-138`），逐條對齊：

| 順序 | 行為 | 與電影版的差異 |
|---|---|---|
| 1 | `:id` 空 → 400 `VALIDATION_INVALID_FORMAT` | 訊息改「Invalid episode ID」 |
| 2 | 可用性閘門：`IsAvailable() == false` **且** episode 版 resume 不成立 → 503 `TRANSCRIPTION_DISABLED` | ⚠️ 見 AC #3 與紅線 3 |
| 3 | 查無 episode → 404 | `NotFoundError(c, "Episode")` |
| 4 | `file_path` 空 → 400 `VALIDATION_REQUIRED_FIELD` | 訊息點名「掃描媒體庫」 |
| 5 | `os.Stat(file_path)` 失敗 → 400 `VALIDATION_REQUIRED_FIELD` | 同電影 |
| 6 | `IsInProgress(id)` → 409 `TRANSCRIPTION_IN_PROGRESS` | 同電影（`inProgress` map 以 mediaID 為 key，UUID 天然不撞） |
| 7 | `StartTranscription(...)` 202 `{job_id, message}` | ⚠️ 見 AC #4 |

- 新窄介面（Rule 11，鏡射 `TranscriptionMovieGetter`）：
  ```go
  type TranscriptionEpisodeGetter interface {
      FindByID(ctx context.Context, id string) (*models.Episode, error)
  }
  ```
  `*repository.EpisodeRepository` 已滿足（`episode_repository.go:107`）；`main.go:754` 的建構子加一個參數注入 `repos.Episodes`。
- 路由註冊在同一個 `RegisterRoutes`：`rg.POST("/episodes/:id/transcribe", h.TranscribeEpisode)`。
- Swagger 註解比照 `generation_batch_handler.go` 的密度（`apps/api` 沒有 swag-gen，寫註解即可，不需跑產生器）。
- **Rule 7：零新錯誤碼。** 全數重用既有的 `TRANSCRIPTION_DISABLED` / `TRANSCRIPTION_IN_PROGRESS` /
  `VALIDATION_INVALID_FORMAT` / `VALIDATION_REQUIRED_FIELD`，`TRANSCRIPTION_` prefix 早於 9R-16 登記在案 →
  **`project-context.md` 與 `code-review/instructions.xml` 皆零編輯**（13-4b `DVR_TVDB_NOT_FOUND` 先例：
  code-list 不變、prefix 數不變 ⇒ 以驗證取代編輯，記在 Completion Notes）。
- **Rule 3：錯誤訊息 envelope 用 zh-TW。** ⚠️ 電影版的 400/404 訊息今天是英文
  （`transcription_handler.go:57,90,96`）—— 那是既有的 Rule 3 缺口（已由 sub-2-2d 認列於 503 那條）。
  **新路由不得複製這個缺口**：新寫的訊息一律 zh-TW。既有電影路由的英文訊息**不在本 story 範圍**
  （不要順手改，那會動到別人的測試）；若 dev 認為值得一併修，走 Rule 24 lane ③ 立案。

### AC #3 — episode 版 translate-only resume 閘門（Rule 11 手足，不是加寬）

`TranscriptionService` 新增：

```go
// CanResumeEpisodeTranslateOnly reports whether an EPISODE run would resume
// translate-only. Sibling of CanResumeTranslateOnly, NOT a widening (Rule 11) —
// same shape the EpisodeSubtitleStatusWriter/Reader pair took in sub-3-2.
func (s *TranscriptionService) CanResumeEpisodeTranslateOnly(ctx context.Context, episodeID string) bool {
    return s.canResumeTranslateOnly(ctx, models.SubtitleRunMediaEpisode, episodeID)
}
```

- 私有的 `canResumeTranslateOnly(ctx, mediaType, mediaID)`（`transcription_service.go:584`）已存在且已 media-type-aware，
  本 AC 只是把它的 episode 面向暴露給 handler。
- **不得**修改既有的 `CanResumeTranslateOnly`（movie）簽章 —— 它掛在
  `TranscriptionServiceInterface` 上，動它會連坐電影 handler 與其 fake。
- episode handler 的介面新增這個方法；**電影 handler 的介面不動**（可共用同一個 interface 並讓兩個 handler 都吃，
  但若共用，電影 handler 的既有測試 fake 必須補齊新方法 —— 由 dev 擇一，於 Completion Notes 記錄理由）。

### AC #4 — run 必須帶 `WithMediaType(episode)` ＋ `WithTranslation()`

`StartTranscription(ctx, episodeID, filePath, mediaDir, services.WithTranslation(), services.WithMediaType(models.SubtitleRunMediaEpisode))`

- `mediaDir = filepath.Dir(episode.FilePath.String)`（同電影）。
- ⚠️ **`WithMediaType` 漏掉 = 靜默資料損毀**：`generation_batch_runner.go:17` 的註解白紙黑字寫著
  「WithMediaType silently defaults to movie and would write an episode's status into the movies table」。
  這是本 AC 存在的唯一理由，測試必須 pin 住（見 AC #5 第 2 例）。
- `translate` 參數的處理：電影版是 `?translate=true` query 才加 `WithTranslation()`。
  **episode 版一律強制 `WithTranslation()`**（不看 query）—— 前端 `transcriptionService` 對電影也是永遠送
  `translate=true`（`transcriptionService.ts:36`），保留一個只會產出英文 SRT 的旁路沒有消費者，且與 F1 的
  「語音辨識＋AI 翻譯」承諾不符。此偏離明記於此，Alexyu 可否決。

### AC #5 — 測試（Go，RED-first，七例＋兩例 wire-shape）

1. happy path 202 + `job_id`
2. `WithMediaType(episode)` **確實**被送進 `StartTranscription`（fake 攔截 opts 並斷言 —— AC #4 的資料損毀防線）
3. `WithTranslation()` 亦被送進
4. 無 `file_path` → 400
5. 檔案不存在於磁碟 → 400
6. `IsAvailable()=false` **且** `CanResumeEpisodeTranslateOnly=false` → 503
7. `IsAvailable()=false` **但** `CanResumeEpisodeTranslateOnly=true` → **不 503**，照常 202（紅線 3 的回歸釘）
8. `MergedEpisode.episode_id` wire-shape：有本地檔 → 有值
9. `MergedEpisode.episode_id` wire-shape：無本地檔 → `omitempty` 不出現在 JSON

---

## 🔴 三條紅線（踩到就是 CR High）

### 紅線 1 —— `WithMediaType` 不可漏

見 AC #4。漏掉不會噴錯，會把一集的 `subtitle_status` 寫進 **movies** 表（該 id 在 movies 表不存在 → 0 rows），
使用者看到的是「跑完了但徽章沒變」，而且下次批次還會再算它一次、再付一次 ASR 錢。
`generation_batch_runner.go:17` 已為此留了告誡註解 —— 那是 sub-4-2 踩過的坑。

### 紅線 2 —— 不要順手改電影路由

電影路由的英文錯誤訊息、`?translate=true` query 語意、`TranscriptionMovieGetter` 介面，
**全部不在本 story 範圍**。它們各自掛著別人的測試與 Rule 20 契約。
本 story 是**新增**，不是重構。若發現值得修的，走 Rule 24 lane ③ 立案，不要就地擴張。

### 紅線 3 —— 503 閘門必須用 episode 版 resume check

若 episode handler 圖方便直接呼叫既有的 `CanResumeTranslateOnly`，那支方法會拿 episode id 去查
**movies** 表（`transcription_service.go:578-580` 硬編 `SubtitleRunMediaMovie`）→ 永遠 `false`
→ 一集已經是 `untranslated`（英文 SRT 已在磁碟）的影集，在沒有 ASR key 的部署上會被錯誤 503 擋掉，
而它**根本不需要 ASR**。

這正是 CR sub-2-2a M2 為電影修過的那個 bug。**AC #3 存在的唯一理由就是不讓它在 episode 上重演。**

---

## Tasks / Subtasks

- [x] **Task 1 — `MergedEpisode` 帶 `episode_id`（AC: #1）**
  - [x] `services/series_season.go` struct 加欄位（`omitempty`）＋ 於 `HasLocalFile` 分支內填 `local.ID`
  - [x] AC #5 第 8、9 例（wire-shape 兩例）

- [x] **Task 2 — `POST /episodes/:id/transcribe`（AC: #2, #4）**
  - [x] `TranscriptionEpisodeGetter` 窄介面 ＋ handler 建構子加參數 ＋ `main.go:754` 注入 `repos.Episodes`
  - [x] `TranscribeEpisode` 七道閘門依序實作（表格順序即實作順序），訊息 zh-TW
  - [x] `StartTranscription` 帶 `WithTranslation()` ＋ `WithMediaType(episode)`
  - [x] `RegisterRoutes` 註冊 ＋ Swagger 註解
  - [x] AC #5 第 1–5 例

- [x] **Task 3 — episode resume 閘門（AC: #3）**
  - [x] `CanResumeEpisodeTranslateOnly` ＋ handler 介面掛載（若共用 interface，補齊電影側 fake）
  - [x] AC #5 第 6、7 例

- [x] **Task 4 — 全閘門**
  - [x] `go test ./...`、`go vet`、`staticcheck`、`gofmt`
  - [x] `format:check`（.md/.yaml 也在 prettier 範圍內）

---

## Dev Notes

### 跨端拆分檢查（Epic 8 Retro Agreement 5 / Epic 9c Retro AI-1）

**BACKEND-ONLY，FE task 數 = 0 ⇒ 門檻不適用。**
原 authoring（2026-08-19 稍早）曾以 BE 3 / FE 4 的單支形式建檔；Alexyu 的 design-first 裁定
把 FE 半邊移出，成為 `9R-10c`。拆分理由是**設計相依**，不是 task 計數。

### 為什麼 BE 可以先走（不等設計）

逐集按鈕「長什麼樣」是 Sally 的題目；「打哪個端點、帶哪個 id、寫哪張表」不是。
本 story 交付的兩件事（`episode_id` 欄位、`POST /episodes/:id/transcribe`）在任何按鈕造型下都相同。
先例：9R-16 的 BE 早於 `ux3-subtitle-v2-batch` 出貨（13-7a/b 亦同款）。

**BE 先落地會有一段「路由已上線但無 UI 消費者」的期間** —— 這是 pair story 的常態，
不違反 capability honor（那條規範的是**畫出來的控制項**不得是死的，不是後端不得先於前端）。

### Rule 20 —— AC Contract Versioning

- 本 story **產生**兩個 stamp：AC #1（`MergedEpisode.episode_id` wire shape）、AC #2（單集路由的請求／回應形狀），
  皆為 `[@contract-v1]`。**下游消費者：`9R-10c-episode-row-subtitle-cta-frontend`**（該 story 的 Dev Notes 須帶
  `confirmed against [@contract-v1]` ack 行）。
- **上游 ack 盤點：**
  - `MergedEpisode` 原始定義來自 12-2（`12-2-season-episode-list.md:179` 明記
    `📎 Contract Stamps: NONE`）⇒ 前 Rule-20、隱含 v0，**依 forward-only 回溯規則不欠 ack**。
  - 9R-16 的 `[@contract-v3]`（generation-batch）**本 story 不消費** —— 單集路由與批次是兩條獨立入口。
  - 9R-18（media id 為 UUID string）為既成事實，非 stamped AC。
- ⇒ dev 於 Completion Notes 記：`📎 Contract Stamps: FOUND (2 new v1 in this story, consumer 9R-10c; upstream 12-2 pre-Rule-20 implicit v0, no ack owed)`

### Rule 24 —— Discovery Triage 預填

authoring 期間已發現、**已立案**的 lane ③ 一筆（由 design-first 裁定產生，已升格為 blocking 設計 story，
故列於 `9R-10c` 而非本 story）：`9R-UX-episode-row-cta-design`。
本 story 與其**無相依** —— 記於此僅為交叉索引。

dev 實作期間若再發現，一律照 lane ①/②/③ 分流並**在發現當下**寫進 `sprint-status.yaml`。

### Rule 23 —— Time-dependent visual coverage

**N/A — 本 story 為 BACKEND-ONLY，不觸及 `apps/web/src/components/**` 任何檔案。**

### Project Structure Notes

- 修改：`internal/services/series_season.go`、`internal/handlers/transcription_handler.go`（+ test）、
  `internal/services/transcription_service.go`、`cmd/api/main.go`。**零新檔、零 migration、零新表、零前端檔案**。
- Rule 11（窄介面）與 Rule 19（services ↛ subtitle）皆已由既有結構滿足，本 story 不引入新的跨層依賴。

### References

- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`] — `9R-10a` 條目（原始 Rule-24 ③ 立案文 + 兩次裁定）
- [Source: `_bmad-output/implementation-artifacts/9R-10-pipeline-orchestration.md#AC2`] — 母票的 trigger 定義與拆分理由
- [Source: `apps/api/internal/handlers/transcription_handler.go:46-138`] — 要鏡射的電影路由（七道閘門）
- [Source: `apps/api/internal/services/series_season.go:31-165`] — `MergedEpisode` 與合併迴圈
- [Source: `apps/api/internal/services/transcription_service.go:375-389,575-590`] — `WithMediaType`／resume 閘門
- [Source: `apps/api/internal/services/generation_batch_runner.go:17`] — 「WithMediaType 漏掉會寫錯表」的既有告誡
- [Source: `apps/api/internal/repository/episode_repository.go:107`] — `FindByID`（窄介面的實作者）
- [Source: `project-context.md`] — Rule 3／7／11／18／19／20／23／24

---

## Dev Agent Record

### Agent Model Used

claude-opus-5[1m] (dev, Amelia)

### Debug Log References

### Completion Notes List

**實作摘要（4/4 task，AC #1–#5 全數滿足）**

- **AC #1** — `services.MergedEpisode` 新增 `EpisodeID string \`json:"episode_id,omitempty"\``，
  於 `series_season.go` 合併迴圈的 `HasLocalFile = true` 分支內填 `local.ID`。零 migration、既有欄位未動。
- **AC #2** — `POST /api/v1/episodes/:id/transcribe` 上線，七道閘門逐條對齊電影版；
  新窄介面 `TranscriptionEpisodeGetter`（Rule 11 手足，`*repository.EpisodeRepository` 直接滿足）；
  Swagger 註解完整；**所有新的錯誤訊息 zh-TW**（Rule 3）。
  ⚠️ 精確化（CR L1）：202 成功回應的 `message` 欄位維持與電影路由一致的英文
  （`"Transcription started. Listen to SSE events for progress."`）——已查證**非使用者可見**
  （前端讀 SSE 的 `progress.message`，從不讀 202 body），故不構成 Rule 3 破口。
  既有電影路由的英文錯誤訊息依紅線 2 未動。
- **AC #3** — `TranscriptionService.CanResumeEpisodeTranslateOnly` 落地，
  委派既有的 media-type-aware 私有 `canResumeTranslateOnly(ctx, episode, id)`。
- **AC #4** — `StartTranscription` 帶 `WithTranslation()` ＋ `WithMediaType(models.SubtitleRunMediaEpisode)`，
  不看 query param（story 明記的偏離，Alexyu 可否決）。
- **AC #5** — 13 個新 Go 測試（story 要求 9 例；+1 見下，+3 為 CR 修正帶入），RED-first 驗證過
  （首跑 compile-fail：`MergedEpisode has no field EpisodeID` / `too many arguments in call to NewTranscriptionHandler`）。

**兩處與 story 規格的偏離（皆記錄，可否決）**

1. **`RegisterRoutes` 條件式掛載 episode 路由。** story AC #2 只說「建構子加一個參數」。
   實作照做了，但 `episodeService == nil` 時**不掛載**該路由（→ gin 回 404）而非讓 nil 進到 handler。
   理由：11 個既有 movie 測試以 `nil` 傳入第二參數（純機械改動、零 assertion 變更），
   若照樣掛載，任何誤打該路由的測試會 nil-panic。404 也比「捏一個 503 去怪 ASR 設定」誠實
   —— 未接線是接線錯誤，不是能力缺失。新增測試 `TestTranscribeEpisode_RouteAbsentWithoutGetter` 釘住。
2. **新增 `services.TranscriptionOptionsFor`（4 行，exported）。**
   `TranscriptionOption` 是 opaque func over 未匯出的 config，`handlers` 套件**無法**分辨
   `WithMediaType(episode)` 與 no-op —— 既有 movie 測試只能寫 `assert.Len(receivedOpts, 1)`，
   那種斷言擋不住 AC #4 的資料損毀。此 helper 讓斷言變成真的
   （`TestTranscribeEpisode_SendsEpisodeMediaTypeAndTranslation`）。
   **未回頭強化既有 movie 測試**（紅線 2：本票是新增不是重構）。

**多出來的第 10 個測試**：`TestTranscribeEpisode_ResumeGateIsEpisodeScoped` ——
把 mock 的 `canResume`(movie) 與 `canResumeEpisode` 設成**相反值**，
任何走錯閘門的實作都會翻轉結果。紅線 3 的直接釘子（story AC #5 第 7 例只驗 episode 旗標為 true 的情形）。

**🔗 AC Drift: NONE**（checked: `MergedEpisode|episode_id|episodes/:id|GetSeasonEpisodes` across
`_bmad-output/implementation-artifacts/*.md` — 12 hits，全部落在 12-2（定義者）與 12-3／12-6（引用），
`episode_id` 為 additive `omitempty` 欄位，未改變 12-2 任何 AC 描述的行為 ⇒ 全部 REUSE not DRIFT）

**📎 Contract Stamps: FOUND** (2 new `[@contract-v1]` in this story — AC #1 `MergedEpisode.episode_id`
wire shape、AC #2 單集路由形狀；下游消費者 = `9R-10c`。上游 12-2 自己記錄
`📎 Contract Stamps: NONE` ⇒ pre-Rule-20 隱含 v0，**不欠 ack**。9R-16 `[@contract-v3]` 本票不消費。)

**🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)**

**🎨 UX Verification: SKIPPED — no UI changes in this story**

**Rule 7 驗證（非編輯）**：新路由零新錯誤碼，全部重用 `TRANSCRIPTION_DISABLED` /
`TRANSCRIPTION_IN_PROGRESS` / `VALIDATION_INVALID_FORMAT` / `VALIDATION_REQUIRED_FIELD`。
`TRANSCRIPTION_` prefix 自 9R-16 起已在登記表內 ⇒ `project-context.md` 與
`code-review/instructions.xml` **皆零編輯**（13-4b `DVR_TVDB_NOT_FOUND` 先例）。

**閘門結果（全綠）**

| 閘門 | 結果 |
|---|---|
| `go test ./...`（apps/api 全套） | ✅ exit 0，34 個套件 ok，0 FAIL／0 panic |
| `go vet ./...` | ✅ exit 0 |
| `staticcheck ./...` | ✅ 本票零新增；2 筆既有無關告警（`config.go:282` unused、`images/processor.go:206` unused）＋ 1 筆工具鏈版本雜訊（go1.26 stdlib vs go1.25 建置），**未擴張範圍去清**（Epic 9c Retro AI-2：既有問題不假裝沒看到 —— 三者皆非測試失敗、非本票引入，故不另立條目） |
| `gofmt -l`（本票七檔） | ✅ 全部乾淨 |
| `pnpm nx test web` | ✅ 233 files／**2653 tests** 全綠（本票零前端檔案，跑滿全回歸門檻） |
| `prettier --check`（story／sprint-status） | ✅ 乾淨 |
| 測試程序清理 | ✅ `No test processes found` |

**Pre-existing 觀察（非失敗，不阻斷）**：`gofmt -l apps/api` 全庫列出 64 檔 —— 既有格式漂移
（疑似 Go 版本差異），**與本票無關**。實作中一度以 `gofmt -w internal/handlers/` 誤觸 10 個
無關檔案，已全數 `git checkout` 還原，最終 diff 僅含本票七檔。未立條目：這是格式化工具的
版本差異而非測試失敗，且清它會產生數千行無關 diff。

### Discovery Triage

- **NO — no out-of-scope work discovered.**
  authoring 時如此，實作時亦然：既有的 `gofmt` 全庫漂移與 2 筆 staticcheck unused 告警都是
  **既有格式／靜態分析雜訊，非測試失敗**，依 Epic 9c Retro AI-2 的判準不構成須立案的
  pre-existing failure（已於 Completion Notes 具名記錄）。
  （design-first 裁定產生的設計相依已升格為獨立 story `9R-UX-episode-row-cta-design`，
  掛在 `9R-10c` 的 blocked-by，不是本 story 的 discovery。）

### File List

**Modified**

- `apps/api/internal/services/series_season.go` — `MergedEpisode.EpisodeID` 欄位 ＋ 合併迴圈填值（AC #1）
- `apps/api/internal/services/series_season_test.go` — wire-shape 兩測（AC #5 第 8/9 例）
- `apps/api/internal/services/transcription_service.go` — `CanResumeEpisodeTranslateOnly`（AC #3）＋ `TranscriptionOptionsFor` 測試用 helper
- `apps/api/internal/handlers/transcription_handler.go` — `TranscriptionEpisodeGetter` 窄介面、介面新增 `CanResumeEpisodeTranslateOnly`、建構子第二參數、條件式路由掛載、`TranscribeEpisode`（AC #2/#4）＋ **CR M1/L2**：查詢錯誤分類（sentinel→404／其餘→500）與 log 級別調降
- `apps/api/internal/handlers/transcription_handler_test.go` — mock 補 `canResumeEpisode` 旗標＋方法；11 處建構子呼叫補 `nil` 第二參數（**零 assertion 變更**）
- `apps/api/internal/handlers/route_c_uuid_integration_test.go` — 建構子呼叫補 `nil`（一行）
- `apps/api/cmd/api/main.go` — 注入 `repos.Episodes`（掛載單集路由）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉

**Added**

- `apps/api/internal/handlers/transcription_episode_handler_test.go` — **13** 個單集路由測試（AC #5 ＋ CR M1/M2/L3 帶入的 `LookupFailureIs500`／`EmptyID`／`NilWithoutError`）

## Senior Developer Review (AI)

**Reviewer:** Amelia（adversarial code-review workflow）· **Date:** 2026-08-19 · **Model:** claude-opus-5[1m]
**Outcome:** ✅ **APPROVED-WITH-FIXES** —— 0 High / 2 Medium / 3 Low，**5/5 全數當場修復**。

**Git vs File List：0 落差**（8 個 `apps/` 檔案逐一對得上，無漏記、無假宣稱）

| 閘門 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **PASS**（4 個錯誤碼 `TRANSCRIPTION_DISABLED`／`TRANSCRIPTION_IN_PROGRESS`／`VALIDATION_INVALID_FORMAT`／`VALIDATION_REQUIRED_FIELD`，全為已登記前綴；本票零新增錯誤碼常數） |
| 🔒 Rule 20 Contract Bump | **N/A**（本次 diff 無 `[@contract-vN→vM]` bump row；本票只**產生** v1 stamp） |
| 🔒 Rule 25 Mega-line | **N/A**（`project-context.md` 未被改動） |
| 🔒 Rule 19 Boundaries | **PASS**（`TestForbiddenImportEdges` 綠——新增的 `handlers → repository` import 合法，5 個既有 handler 已這麼做，禁止的是 services→handlers／repository→services｜subtitle） |

### Findings

**🟡 M1 — 任何查詢錯誤都被回成 404，掩蓋基礎設施故障** · `transcription_handler.go` · ✅ FIXED

原本 `if err != nil || episode == nil { NotFoundError }` 把 `FindByID` 的**所有**失敗都當成「查無此集」。
但 AC #2 的閘門表明訂「**查無** episode → 404」。SQLite 被鎖住時（NAS 上 `/mnt/user` FUSE 問題已實際發生過）
這一集明明存在，使用者卻被告知「找不到」，於是跑去翻硬碟找一個從未消失的檔案，同時真正的故障被一個
看起來很正常的回應蓋掉。

修復：以 `errors.Is(err, repository.ErrEpisodeNotFound)`（外加介面允許但真實 repo 不會產生的 `nil, nil`）
判定 404，其餘一律 500。回歸釘：`TestTranscribeEpisode_LookupFailureIs500`（`"database is locked"` ⇒ 500
且**不得**啟動任何 run）＋ `TestTranscribeEpisode_NilWithoutError`。

**🟡 M2 — 七道閘門只有六道有測試** · `transcription_episode_handler_test.go` · ✅ FIXED

Task 2 打勾聲稱「七道閘門依序實作」，但空 `:id` → 400 那道無測試。電影路由**明確**測了同一條
router 走不到的分支（`TestTranscribeMovie_EmptyID`，手工 gin context）。一支存在目的就是鏡射電影路由的
story，鏡射漏一道是實質的完成度落差。

修復：`TestTranscribeEpisode_EmptyID`，同款手工 context，並斷言 `VALIDATION_INVALID_FORMAT`。

**🟢 L1 — Completion Notes 的宣稱不精確** · ✅ FIXED

原寫「新訊息全數 zh-TW」，但 202 成功回應的 `message` 是英文。已查證**非使用者可見**
（`ManageSubtitleDialogV2` 用 `generation.progress.message`（SSE），從不讀 202 body），
故非 Rule 3 破口——但宣稱本身不準。措辭已修正並附查證結論。
（此類「筆記裡的事實宣稱未經驗證」正是 `retro-cand-sprint-note-claim-verification` 追的問題。）

**🟢 L2 — 例行 404 卻寫 ERROR 級 log** · ✅ FIXED

書籤或重掃後的過期 episode id 是例行事件，不是事故。改為 `slog.Warn`；
搭配 M1 之後，`slog.Error` 只留給真正的基礎設施失敗。電影路由的同款 Error 級依紅線 2 未動。

**🟢 L3 — `episode == nil && err == nil` 分支無測試** · ✅ FIXED

真實 repo 不會回這個形狀，但窄介面允許，未來的 port 實作可能踩到。
`TestTranscribeEpisode_NilWithoutError` 補上。

### 看過、決定保留（免得下個審查者重吵）

`services.TranscriptionOptionsFor`（4 行 exported，唯一消費者是測試）乍看像為測試污染 production API。
**判定保留**：`TranscriptionOption` 是不透明 func 包著未匯出的 `transcriptionConfig`，跨套件測試**無法**
分辨 `WithMediaType(episode)` 與 no-op；`export_test.go` 對跨套件無效。沒有它，AC #4 那條
「漏了就靜默把單集狀態寫進 movies 表」的防線就只剩 `assert.Len(opts, 1)`——擋不住任何東西。

### 亦經查核、非 finding

- **gin 路由衝突**：`/episodes` 是唯一的靜態首段，全 `apiV1` 無同層 wildcard 兄弟節點 ⇒ 無 panic 風險。
- **typed-nil 陷阱**：`repos.Episodes` 為 `EpisodeRepositoryInterface`，`registry.go:45,78` 恆賦值
  `NewEpisodeRepository(db)`，永不為 nil ⇒ `episodeService != nil` 的掛載判斷安全。
- **安全性**：`os.Stat` 的路徑來自 DB 而非使用者輸入，無 traversal 面；新路由無新增外部呼叫。
- **併發**：`IsInProgress` 以 mediaID 為 key，UUID 天然不撞；與批次共用 9R-11 Governor 節流。

### 修後閘門（全綠）

| 閘門 | 結果 |
|---|---|
| `go test ./...` | ✅ exit 0，34 套件，0 FAIL／0 panic |
| `go vet ./...` | ✅ 0 |
| `staticcheck` | ✅ 本票零新增 |
| `gofmt -l`（本票檔案） | ✅ 乾淨 |
| `TestForbiddenImportEdges`（Rule 19） | ✅ PASS |
| 單集路由測試 | ✅ **13/13**（原 10 ＋ CR 帶入 3） |
| `pnpm nx test web` | ✅ 233 files／2653 tests（本票零前端檔案） |

## Change Log

| Date | Change |
|---|---|
| 2026-08-19 | Story 建檔（SM Bob, create-story）。⚖️ 範圍重新界定：立案時的「影集完全不能生成」已由 sub-3-2／sub-4-2／sub-5-3 推翻，真空縮為「單集手動觸發入口」；Alexyu 裁定採 BE+FE 補入口（否決 superseded 方案）。 |
| 2026-08-19 | ⚖️ Alexyu 第二次裁定：FE 半邊改走 **design-first**（推翻 authoring 的 function-first 裁定）。Story 從 BE+FE 單支 **拆為 BACKEND-ONLY**：FE 四個 task 移出成 `9R-10c-episode-row-subtitle-cta-frontend`（blocked），設計輪立為 `9R-UX-episode-row-cta-design`（Sally）。BE 不等設計輪。本檔案 AC 由 9 條縮為 5 條、task 由 7 縮為 4；原紅線 1（glossary id 分離）與紅線 3（fetch 隱藏）為純前端事項，隨 FE 半邊移至 `9R-10c`。 |
| 2026-08-19 | **DEV DONE（Amelia, claude-opus-5[1m]）—— 4/4 task、AC #1–#5 全綠。** `MergedEpisode.episode_id` [@contract-v1]（僅 `has_local_file` 分支填值，`omitempty`）＋ `POST /api/v1/episodes/:id/transcribe` [@contract-v1]（七道閘門對齊電影版、訊息 zh-TW、`TranscriptionEpisodeGetter` 窄介面、Swagger 完整）＋ `CanResumeEpisodeTranslateOnly`（Rule 11 手足）＋ `WithTranslation()`＋`WithMediaType(episode)` 強制帶入。10 個新 Go 測試，RED-first 已驗（compile-fail 起手）。**兩處記錄在案的偏離**：(1) `episodeService == nil` 時**不掛載**路由（404 而非 nil-panic／假 503），(2) 新增 4 行 exported `TranscriptionOptionsFor` —— 否則 `handlers` 套件無法分辨 `WithMediaType(episode)` 與 no-op，AC #4 的資料損毀防線就只是 `assert.Len(opts, 1)`。多釘一測 `ResumeGateIsEpisodeScoped`（movie／episode 旗標設相反值）鎖住紅線 3。閘門：go test ./... exit 0（34 套件）／go vet 0／staticcheck 零新增／gofmt 本票七檔乾淨／web 233 files 2653 tests 全綠／prettier 乾淨／測試程序已清理。🔗 AC Drift: NONE（12 hits 全 REUSE）／📎 Contract Stamps: FOUND（2 new v1，消費者 9R-10c；上游 12-2 pre-Rule-20 v0 不欠 ack）／🎭 A11y: N/A（100% backend）／🎨 UX: SKIPPED。Status → review。 |
| 2026-08-19 | **CR APPROVED-WITH-FIXES（adversarial code-review，同 session）—— 0H/2M/3L，5/5 全修。** M1（HIGHEST-VALUE）：`FindByID` 的任何錯誤原本一律回 404，DB 鎖死時使用者被告知「找不到這一集」並去翻一個從未消失的檔案 → 改為 sentinel(`repository.ErrEpisodeNotFound`)＋`nil,nil` 才 404、其餘 500，回歸釘 `LookupFailureIs500`。M2：七道閘門只有六道有測試（空 id → 400 缺，電影路由有同款手工-context 測試）→ 補 `EmptyID`。L1：Completion Notes「新訊息全數 zh-TW」不精確（202 成功 message 是英文；已查證非使用者可見，非 Rule 3 破口）→ 措辭修正並附查證。L2：例行 404 的 `slog.Error` → `slog.Warn`。L3：介面允許的 `nil,nil` 分支補測。閘門：Rule 7 PASS（4 碼全合規、零新常數）／Rule 20 N/A（無 bump）／Rule 25 N/A（未動 project-context.md）／Rule 19 `TestForbiddenImportEdges` PASS（新增的 handlers→repository import 合法）。修後：go test ./... exit 0（34 套件）、vet 0、staticcheck 零新增、gofmt 乾淨、單集路由 13/13、web 2653 全綠。Git vs File List 零落差。Status → done。 |
