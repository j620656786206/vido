# Story 5.4: 增量分析 —— 路線探測結果快取，重複分析只探新增與變動項

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** `epic-subtitle-pipeline-m3`（M3 第一波，A 群組）· **Risk: 🟡 MED-LOW（純快取層,零行為語意變更;失效判定是唯一紅線）** · **BACKEND-ONLY**
**Source:** sprint-status `epic-subtitle-pipeline-m3` seed（A:增量掃描 —— **現況掃描行為需先盤點,可能屬 scanner 面非字幕面**）· spec §8 M3「批次整季、失敗重試、增量掃描」
**Cross-stack split check:** backend tasks = 4, frontend tasks = 0 → 單一 story

---

## 🔎 盤點結論：seed 的疑慮成立，範圍必須重判

seed 明文要求先盤點,因為它懷疑「增量掃描」不屬字幕面。**盤點結果:一半屬 scanner 面且早已交付,另一半在字幕面且完全沒做。**

### (1) 檔案系統層的增量掃描 —— ✅ 2026-03 就交付了，不在本 story 範圍

`ScannerService` 自 **story 7-2**（Epic 7,scheduled scan）起就是增量的:

```go
// scanner_service.go:475-486
if existing != nil {
    sizeChanged := !existing.FileSize.Valid || existing.FileSize.Int64 != info.Size()
    mtimeNewer  := info.ModTime().After(existing.UpdatedAt)
    if !sizeChanged && !mtimeNewer {
        s.progress.FilesSkipped++   // ← 沒變就跳過
        return nil
    }
    ...
}
```
＋ `detectRemovedFiles`（同 story,註解就寫著「Story 7-2: incremental scan」）＋ `onScanComplete` 只在 `FilesCreated > 0 || FilesUpdated > 0` 時才觸發。

spec §8 的 M3 清單寫於 2026-07-23,而 7-2 早在 3 月就出貨 —— **這一項是清單寫成時就已經過期的條目**。本 story 不重做它。

### (2) 字幕面的「重複分析」—— ❌ 完全沒做，這才是真正的痛點

真正每次都從頭來的是 **F14 候選分析**（`GenerationCandidateService.Analyze`）:

```go
// generation_candidates.go — classify()
if tracks, ok := parsePersistedTracks(row.tracksJSON); ok {
    return s.predictor.FromTracks(tracks), true   // 電影:讀已存的軌道,免探測
}
route, err := s.predictor.Probe(ctx, row.filePath)  // ← 其餘一律 ffprobe
```

**而 episodes 永遠走探測分支**,因為 `episodes` 表**根本沒有 tech-info 欄位**:migration 021 只給 `movies` 與 `series` 加了 `subtitle_tracks`（migration 025 給 episodes 加的是 subtitle_status/path/language,不含 tracks）,`candidateRow.tracksJSON` 對 episode 是空字串（程式碼註解自己寫著「Empty for episodes, which have no such column」),而 `EnrichmentService` **零 episode 觸及**（整檔沒有一處 episode 參照）。

**後果**:一個 500 集的影集庫,每次分析 = 500 次 ffprobe,受 `NewFFprobeService(3, 10*time.Second)` 的 3 格 semaphore 節流。而且它**不是偶爾**發生:

| 觸發點 | 是否強制重新分析 |
| --- | --- |
| F17 掃描完成入口 | `forceAnalyze` → 是（合理,庫剛變） |
| 任何批次終局後 | `postTerminal` sticky → 是（**該 session 之後每次開都是**） |
| 伺服器重啟後首開 | snapshot 在記憶體,idle → 是 |
| 一般再開（ready snapshot 仍在） | 否 ✓ |

也就是說:**掃完 → 分析 500 次探測 → 生成一批 → 再開又 500 次探測**,即使這 500 個檔案一個位元都沒動。這正是 sub-5-3 服務的那個「TV-heavy library」使用者每天在等的東西。

⚖️ **裁定:本 story = 給路線探測加一層以檔案身分為鍵的快取**,讓重複分析只探測**新增或變動**的項目 —— 這才是「增量」在字幕面的意思。scanner 面不動一行。

---

## Story

As a NAS owner with a TV-heavy library,
I want a repeat analysis to only probe the files that are new or have actually changed,
so that opening 產生字幕 after a batch does not re-ffprobe five hundred untouched episodes before it can show me a list.

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` 路線快取的鍵：檔案身分＋述詞版本

新增 `apps/api/internal/services/route_cache.go`（或同等位置）,鍵格式**固定**:

```
subtitle:route:v{routeVersion}:{mediaID}:{fileSize}:{mtimeUnix}
```

- **失效靠鍵本身,不靠 TTL 語意** —— 檔案改了 ⇒ size 或 mtime 變 ⇒ 鍵變 ⇒ 必然 miss。這與 scanner 7-2 用的是**同一組信號**（size + mtime）,刻意對齊:兩處對「檔案變了嗎」不得有兩套答案。
- **`routeVersion` 是述詞版本,不是資料版本**:`RoutePredictor` 的判定邏輯（哪種軌道算「可用文字字幕」）一旦改變,舊 verdict 就是錯的 → **改 `Router.PredictRoute`/`FromTracks` 的語意時必須 bump 這個常數**,並在該常數旁留下 bump 規則註解。這是 sub-1-5b segment cache「key 版本化 key FORMAT／RunVersion 版本化 key INPUTS」先例的同構應用（Rule 27 pillar ②）。
- **儲存體重用 `cache_entries`**（`repos.Cache` / `CacheRepository`,已有 `Get`/`GetMany`/`Set` 與到期清掃排程）—— **零 migration、零新表**。`cacheType` 用可辨識的字串（如 `subtitle_route`）以便清掃與稽核。
- TTL:給一個**長但有限**的值（建議 30 天,與 AI 解析快取同級）—— 鍵已經負責正確性,TTL 只負責讓刪掉的媒體不永久佔位。
- 值:路線字串（`extract`/`asr`/`skip`）。**不存 runtime／金額** —— 那些來自 DB metadata 與費率,每次現算,否則費率調整不會反映（sub-5-1 費率同源的紅線）。

### AC #2 — 讀路徑：批次讀，命中免探測

`Analyze` 在進入 per-row 迴圈**之前**,以 `GetMany` 一次撈回整批鍵（`splitCachedCues` 的批次讀先例,sub-1-5b CR:每列一次查詢會讓 1200 項打 1200 次 round trip）。

- 命中 ⇒ 直接用該 route,**不呼叫 `predictor.Probe`**。
- 未命中 ⇒ 走現有 `classify`,成功後寫入快取（AC #3）。
- **既有的 `tracksJSON` 快路徑優先序不變**:persisted tracks（電影）→ 仍是第一順位,連快取都不用查（它本來就免探測）。快取是給**探測分支**用的。
- 進度回報語意不變:`progress(i+1, len(rows))` 照舊每列回報,**F14 的「分析字幕軌 234 / 1247」不改** —— 命中只是讓它跑得快，不是讓它數得少（分母改變會讓使用者以為漏掉東西）。

### AC #3 — 寫路徑：只快取「可信的判定」

- **只在 `classify` 回 `ok == true` 時寫入。** 探測失敗（Rule 13 case 2:記錄並把該項排除在報價外）**不得**被快取 —— 否則一次暫時性的 I/O 錯誤會被凍結 30 天,那個檔案就再也不會出現在候選清單裡。這是本 AC 的紅線。
- 寫入失敗 fail-soft:log Debug 後繼續（快取寫不進去只是慢,不該讓分析失敗 —— Rule 13 case 3,理由寫在註解）。
- `RouteSkipped` **要**快取（它是一個可信判定,而且 skip 的項目正是重複探測最沒價值的一群）。

### AC #4 — 可觀測性：命中率要看得見

- `Analyze` 結束的 `slog.Info("generation candidate analysis complete", ...)` 增加 `route_cache_hits` 與 `route_probes` 兩個欄位（既有欄位一個不動）。
- 沒有這兩個數字,「增量有沒有生效」只能靠碼表猜。這也是未來調 `routeVersion` 或 TTL 時唯一的證據來源。
- **不新增 SSE 事件、不改 `AnalysisSnapshot` 的 wire shape** —— 這是營運可觀測性,不是產品面（要曝給 UI 是另一個裁定）。

### AC #5 — 失效與正確性的測試紅線

至少：

- **檔案變動必然重探**:同一 mediaID,size 改變 ⇒ miss;mtime 改變 ⇒ miss;兩者皆不變 ⇒ hit（三個案例分開斷言,不可只測 happy path）。
- **`routeVersion` bump ⇒ 全庫失效**:同一檔案在 v1 寫入、v2 讀取 ⇒ miss。
- **探測失敗不進快取**:predictor 回錯誤的項目,查快取應為空;下一次分析仍會重探（凍結錯誤是本 story 最貴的失敗模式）。
- **批次讀**:N 項只發一次 `GetMany`（以 fake repo 計數;每列一查是被 sub-1-5b CR 點名過的反模式）。
- **快取儲存失敗不讓分析失敗**:`Set` 回錯誤時 `Analyze` 仍完成且結果正確。
- **命中路徑零探測**:全命中時 predictor 的 `Probe` 呼叫數為 0（以 spy 斷言 —— 這是整個 story 的價值主張,必須可否證）。
- 全回歸閘門:`go test ./...`、`pnpm nx test web`、`pnpm run lint:all`、`format:check`。

### AC #6 — 契約與範圍圍籬

- **`GenerationCandidate` 的 wire shape 一個欄位都不動** ⇒ sub-4-1 AC #7 `[@contract-v1]` 零影響,**不 bump、不需 ack**（本 story 不消費該契約的形狀,只是讓產生它的過程變快）。
- **`RoutePredictor` 介面不動**;`routePredictorAdapter`（`cmd/api`）不動。
- **scanner 面一行不改** —— 7-2 的 size/mtime 增量、`detectRemovedFiles`、`onScanComplete` 全部維持原狀（盤點結論 (1)）。
- **零 migration、零新 Rule 7 error code**（prefix 維持 16）、零 SSE 改動、零 endpoint 改動。
- 明確**不做**:episodes 的 tech-info 欄位補齊與 enrichment 的 episode 化 → lane ③（見 Discovery Triage）。

---

## Tasks / Subtasks

- [x] **Task 1 — 快取鍵與版本常數（AC: #1）** 🔴 BE
  - [x] `routeVersion` 常數＋bump 規則註解（述詞語意改變時必須 bump）
  - [x] 鍵組裝函式＋單元測試（三個變數各自參與鍵）

- [x] **Task 2 — 讀路徑批次化（AC: #2）** 🔴 BE
  - [x] `Analyze` 迴圈前 `GetMany` 一次撈回;命中免探測
  - [x] `tracksJSON` 快路徑優先序與 F14 進度分母皆不變（測試釘住）

- [x] **Task 3 — 寫路徑與 fail-soft（AC: #3）** 🔴 BE
  - [x] 只在 `ok == true` 時寫入;探測失敗不快取（紅線測試）
  - [x] `Set` 失敗 fail-soft＋Rule 13 註解

- [x] **Task 4 — 可觀測性與回歸（AC: #4, #5）** 🔴 BE
  - [x] 完成 log 加 `route_cache_hits`/`route_probes`
  - [x] 六條紅線測試＋全回歸閘門

（後端 task 4 個、前端 0 個 —— BACKEND-ONLY。）

---

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| 快取儲存 | `repository.CacheRepository`（`Get`/`GetMany`/`Set`;`cache_entries` 表已存在,到期清掃排程已在跑） |
| 批次讀先例 | `subtitle/segment_cache.go` 的 `splitCachedCues`（sub-1-5b CR:每列一查是被點名的反模式） |
| key 版本化先例 | `segmentKey` 的「prefix 版本化 FORMAT／RunVersion 版本化 INPUTS」拆分 |
| 檔案變動信號 | `scanner_service.go:477-478` 的 `size + mtime` —— **本 story 必須用同一組**,不得自創第三套 |
| 探測入口 | `classify`（`generation_candidates.go`）—— 快取包在它外面,內部邏輯不動 |
| fail-soft 先例 | 同檔 `resolveSeriesTitle`（sub-5-3）與 `classify` 的 Rule 13 case 2/3 註解 |

### 關鍵決策（authoring 已裁）

- **快取 route,不快取 tracks**:route 是判定結果,tracks 是原始資料。存 route 讓鍵可以只帶檔案身分;存 tracks 等於重建一份 episodes 的 tech-info 表,那是另一個 story（lane ③）。
- **失效靠鍵不靠 TTL**:TTL 只做垃圾回收。用 TTL 當正確性機制會產生「30 天內換了片源但拿到舊判定」的洞。
- **探測失敗絕不快取**:凍結一次暫時性 I/O 錯誤 30 天 = 那部片從候選清單消失且無人察覺,是本 story 最貴的失敗模式。
- **F14 進度分母不變**:命中讓它快,不讓它少數。分母縮水會讀成「怎麼漏了 500 部」。
- **不碰 scanner**:盤點結論 (1) —— 7-2 已交付,重做只會製造兩套增量語意。
- **不曝給 UI**:命中率是營運數字。要不要在 F14 顯示「已快取 500 項」是產品裁定,不在本 story。

### seam 資料層觸及（retro-m2-AI3 慣例）

- `cache_entries`:新增一個 `subtitle_route` 類別的鍵族（讀 `GetMany`、寫 `Set`）;既有清掃排程自動涵蓋。
- `GenerationCandidateService`:既有觸及不變（movies/episodes/series 讀）;本 story 只在 `classify` 外圍加快取。
- **零 migration、零新表、零 schema 變更。**

### 已知限制（記錄,不在本 story 解）

- episodes 仍無 tech-info 欄位 ⇒ 詳情頁的 episode 技術徽章依然空白;本 story 只讓**分析**免重探,不補資料面（lane ③ `backlog-episode-tech-info-parity`）。
- 首次分析（冷快取）成本不變 —— 本 story 優化的是**重複**分析。
- `AnalysisSnapshot` 仍在記憶體 ⇒ 重啟後仍會跑一次完整分析（但此後每項都是快取命中,所以是「快的一次」而非「慢的一次」）。
- `postTerminal` 讓每次批次終局後都強制重新分析 —— 本 story 讓那次重新分析變便宜,但**沒有**改變它會發生的事實（那是 sub-4-3 CR H2 的正確性設計,不動）。

### 契約姿態（Rule 20）

- **消費**:無。本 story 不讀任何 stamped AC 的形狀。
- **產生**:`[@contract-v1]` 標在 AC #1 的快取鍵格式（跨 story 面:未來任何寫入／讀取這個鍵族的程式必須遵守同一格式與版本規則）。
- sub-4-1 AC #7 候選信封:**wire shape 零改動** ⇒ 不 bump、不需 ack。
- `RoutePredictor` / `routePredictorAdapter` / D2 / D6 / SSE:全部不動 ⇒ 0 bump ⇒ 無 stale-mark 義務。

### Time-dependent visual coverage

`N/A — 100% backend, no apps/web/ files touched.`（mtime 是檔案屬性,非 wall clock 讀取;Rule 23 不適用。）

### References

- [Source: sprint-status `epic-subtitle-pipeline-m3` seed] — A 群組:增量掃描,**現況需先盤點,可能屬 scanner 面**
- [Source: `apps/api/internal/services/scanner_service.go:469-510,281`] — 7-2 已交付的 size/mtime 增量與 detectRemovedFiles
- [Source: `_bmad-output/implementation-artifacts/7-2-scheduled-scan-service.md:14,30-31`] — 「incremental scan」AC 與 task 原文
- [Source: `apps/api/internal/services/generation_candidates.go` classify/Analyze/enumerate] — 探測分支與每列成本
- [Source: `apps/api/internal/database/migrations/021_media_tech_info.go:21-48`] — tech-info 欄位只給 movies/series,**episodes 沒有**
- [Source: `apps/api/internal/database/migrations/025_add_episode_subtitle_fields.go:26-28`] — episodes 只有 subtitle_status/path/language
- [Source: `apps/api/internal/services/enrichment_service.go`] — 零 episode 觸及（movies-only）
- [Source: `apps/api/cmd/api/main.go:409`] — `NewFFprobeService(3, 10*time.Second)` 節流參數
- [Source: `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx`] — forceAnalyze／postTerminal 的觸發頻率
- [Source: `apps/api/internal/repository/cache_repository.go:26,66,115`] — Get/GetMany/Set
- [Source: `apps/api/internal/subtitle/segment_cache.go:51-60,143-186`] — key 版本化與批次讀先例
- [Source: `project-context.md`] — Rule 13/14/19/20/24/27

---

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`

### Debug Log References

- RED gate: `go test ./internal/services/ -run TestRouteCache` 於實作前跑，build failed（`routeCacheKey` / `routeVersion` / `SetRouteCache` / `fileIdentity` 全未定義）—— 測試先於實作存在，非事後補寫。
- GREEN：同一指令 16 個測試全綠（`-v` 逐條列出）。
- `go test -race ./internal/services/ -run 'TestRouteCache|TestAnalyze|TestStartAnalysis|TestCancel'` 綠 —— 新欄位（`routeCache`/`fileIdentity`）與既有 `sseHub` 同屬 wiring-time 寫入、run-time 唯讀，不入 `mu`。

### Completion Notes List

**交付內容（AC 對照）**

- **AC #1** — `apps/api/internal/services/route_cache.go`：`RouteCache` 窄埠（只 `GetMany`/`Set`，理由同 `subtitle.SegmentCache`）、`routeCacheKey(version, mediaID, fileSize, mtimeUnix)` 產生 `subtitle:route:v{n}:{id}:{size}:{mtime}`、`routeCacheType = "subtitle_route"`、`routeCacheTTL = 30 天`、`routeVersion = 1`（bump 規則註解明列觸發條件：`PredictFromTracks` / `PredictRoute` / `SelectCandidates` 的「可用文字軌」定義 / `RoutePrediction` 常數本身有任何語意變動即必須 bump）。`NewRouteCacheRepository` 適配 `repository.CacheRepositoryInterface` → 零 migration、零新表。size+mtime 取自 `os.Stat`，與 `scanner_service.go:477-478`（story 7-2）**同一組信號**。
- **AC #2** — `planRouteCache` 在 `Analyze` 迴圈**之前**逐列決策，`readRouteCache` 一次 `GetMany` 撈回整批。`tracksJSON` 快路徑優先序不變（有 persisted tracks 的列 `key == ""`，連查都不查）。`progress(i+1, len(rows))` 一行未動 —— 命中仍逐列回報，F14 分母不變。
- **AC #3** — 只在 `classify` 回 `ok == true` 時 `storeRoute`；`RouteSkipped` 有寫入。探測失敗（`ok == false`）**永不寫入**（紅線測試 `TestRouteCache_ProbeFailureIsNeverCached` 雙段斷言：當次不寫、下次仍重探）。`Set` 失敗 log Debug 後續跑（Rule 13 case 3 註解在函式 doc）。額外防線：`isKnownRoute` 擋掉讀回來的無法辨識值（否則會以空 `RoutePrediction` 流進報價，計 $0 且不進任何 summary 計數）。
- **AC #4** — `slog.Info("generation candidate analysis complete", …)` 增加 `route_cache_hits` / `route_probes`；既有 6 個欄位一字未改（`TestRouteCache_LogsHitAndProbeCounts` 同時斷言 `candidates=3` 仍在）。無新 SSE 事件、`AnalysisSnapshot` wire shape 零改動。
- **AC #5** — 16 個測試（`route_cache_test.go`）。六條紅線全覆蓋：size 變／mtime 變／兩者不變三案例分開斷言、`routeVersion` bump ⇒ 失效、探測失敗不進快取、N=50 只發 1 次 `GetMany`、`Set` 失敗不讓分析失敗、全命中 `Probe` 呼叫數為 0。另補 5 條：persisted-tracks 完全繞過快取、進度分母不變、讀取失敗 fail-soft、stat 失敗跳過快取但仍探測、未接快取時行為與今天完全相同。
- **AC #6** — `GenerationCandidate` / `GenerationCandidateSummary` / `AnalysisSnapshot` 欄位零改動；`RoutePredictor` 介面與 `routePredictorAdapter` 未動；scanner 面零改動；零 migration、零新 error code（prefix 維持 16）、零 SSE／endpoint 改動。

**設計決策（實作時裁定，authoring 未指定）**

- **`SetRouteCache` 用 setter 而非建構子參數** —— 與既有 `SetSSEHub` 同款。建構子加參數會改寫 `generation_candidates_test.go` 的每一個 callsite，而快取本質是選配基礎設施：不接就是今天的行為（`TestRouteCache_NilCacheKeepsProbingEveryRow` 釘住）。
- **`fileIdentity` 為可注入函式欄位** —— 與既有 `now func() time.Time` 同款。測試因此能直接操縱 size/mtime 證明失效，不必在磁碟上造真檔（真檔測試無法穩定控制 mtime 秒級邊界）。
- **`routeCacheKey` 的 version 是參數不是直接讀常數** —— 否則「bump ⇒ 全庫失效」這條 AC 無法被否證（常數無法在測試中改）。
- **`planRouteCache` 重新 parse 一次 `tracksJSON`** —— 為了不動 `classify` 的簽章。成本是每列一次小 JSON unmarshal（微秒級），相對於它省下的 ffprobe（百毫秒級）可忽略；註解已寫明取捨。

**強制檢查**

- **🔗 AC Drift: NONE** — 觸發條件 (c) 成立（`generation_candidates.go` 出現在 sub-4-1／sub-5-1／sub-5-3／sub-4-3 的 File List）。掃描：`grep -rn "分析字幕軌" *.md`（4 hits）、`grep -rn "progress(i+1\|len(rows)" *.md`（1 hit，本 story 自己）、`grep -rn "候選信封" *.md`（7 hits）、`grep -rn "cache_entries\|CacheRepository" *.md`。逐條判定全為 **REUSE 非 DRIFT**：F14 進度契約（sub-4-1 AC #8）的分母語意被本 story 明文保留且以測試釘住；候選信封（sub-4-1 AC #7）欄位零改動；`cache_entries` 新增一個 type 家族與既有 `subtitle_segment`／`tmdb` 同構，`ClearByType` 無中央註冊表需同步，共用到期清掃排程（`infra-cache-entries-expiry-sweep`）以 table 為單位自動涵蓋。
- **📎 Contract Stamps: FOUND（1 stamp produced；1 upstream ack；0 bumps）** — 產生：AC #1 的快取鍵格式 `[@contract-v1]`（跨 story 面：日後任何讀寫 `subtitle:route:*` 家族的程式必須遵守同一格式與 bump 規則；`route_cache.go` 的 `routeCacheKey` doc 留有 inline stamp 註解）。消費：**confirmed against `[@contract-v1]` (Story sub-4-1 AC #7)** —— 候選信封 wire shape **零改動**（不 additive、不 bump、不需 stale-mark；本 story 只讓產生它的過程變快）。同時 **confirmed against `[@contract-v1]` (Story sub-4-1 AC #8)** —— F14 分析進度契約的 `analyzed/total` 語意維持逐列回報，分母不隨命中縮水。Rule 20 🔁 bump-side 義務：**N/A（0 bumps）**。
- **🎭 A11y Pre-Flight: N/A（100% backend — no apps/web/ files touched）**
- **🎨 UX Verification: SKIPPED — no UI changes in this story**
- **🔒 Rule 7 Wire Format: PASS** — 本次變更 0 個新 error-code 常數，prefix 維持 16。
- **🔒 Rule 15 Wiring: PASS** — `main.go:815` 已接 `generationCandidateService.SetRouteCache(services.NewRouteCacheRepository(repos.Cache))`；無新 endpoint／無 swagger 變更／無 DB 欄位。

**全回歸閘門**

- `go build ./...` ✅ · `go vet` ✅ · `staticcheck`（`nx run api:lint --skip-nx-cache`）✅
- `go test ./...`（apps/api 全套）✅ 0 失敗
- `pnpm nx test web` ✅ 233 files / 2653 tests 全綠
- `pnpm run lint:all` ✅ 0 errors（119 warnings，全為既有前端 `no-explicit-any`／`exhaustive-deps`，本 story 未觸及 `apps/web/`）· `format:check` ✅
- `pnpm run test:cleanup` ✅ No test processes found
- **Pre-existing failures: NONE** —— 全套零失敗，Epic 9c AI-2 的 FIX-or-FILE 規則本次無觸發對象。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time（2026-08-18）：
    - **③ backlog-with-carry-forward-link** — `backlog-episode-tech-info-parity`：`episodes` 表完全沒有 tech-info 欄位（migration 021 只給 movies/series）,且 `EnrichmentService` 零 episode 觸及。後果不只本 story 的重複探測:episode 詳情頁的技術徽章沒有資料來源,且任何需要「這集有哪些軌道」的功能都得現場 ffprobe。本 story 以快取繞過,**不補資料面**。非阻塞。
  - **範圍重判記錄（非 discovery,是 seed 明文要求的盤點結果）**:seed 的「增量掃描」有一半（檔案系統層）早在 story 7-2 就交付,屬 scanner 面 ⇒ 本 story 不重做,改為交付字幕面真正缺的那半（路線探測快取）。此裁定寫在本檔開頭的盤點結論。
  - **YES — filed at implementation time（2026-08-18）**：
    - **③ backlog-with-carry-forward-link** — `backlog-go-gofmt-not-enforced`：`gofmt -l apps/api/` 列出 **75 個未格式化的檔案**（其中 20 個在 `internal/services/`），而 CI 的 `api:lint` 只跑 `go vet` + `staticcheck` —— 兩者都不檢查格式。前端有 prettier `format:check` 把關，後端沒有對等閘門，Go 格式漂移因此永遠不會被擋下。本 story 觸及的 4 個 Go 檔本身 gofmt-clean（已驗證），故非阻塞、不在本 story 清理存量（閘門與一次性 `go fmt ./...` 必須同批進 PR，否則閘門立刻紅）。

### File List

- `apps/api/internal/services/route_cache.go` — **NEW**：`RouteCache` 窄埠、`routeCacheKey` `[@contract-v1]`、`routeVersion` + bump 規則、`routeCacheType`/`routeCacheTTL`、`isKnownRoute`、`fileIdentityFunc`/`osFileIdentity`、`NewRouteCacheRepository` 適配器
- `apps/api/internal/services/route_cache_test.go` — **NEW**：20 個測試（六條紅線 + 補強 + 鍵組裝單元測試 + 未接快取的回溯相容；CR 後 +4：TTL 傳遞、adapter type/映射、cached-skip 讀側、plan ctx 早退）
- `apps/api/internal/services/generation_candidates.go` — **MODIFIED**：`routeCache`/`fileIdentity` 欄位、`SetRouteCache`、`Analyze` 迴圈接快取讀寫與 hits/probes 計數、log 加兩欄、新增 `routeCachePlan`/`planRouteCache`/`readRouteCache`/`storeRoute`
- `apps/api/cmd/api/main.go` — **MODIFIED**：`SetRouteCache(services.NewRouteCacheRepository(repos.Cache))` 接線（Rule 15）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — **MODIFIED**：本 story ready-for-dev → in-progress → review；新增 lane ③ 條目 `backlog-go-gofmt-not-enforced`
- `_bmad-output/implementation-artifacts/sub-4-1-cost-preview-backend.md` — 參照（AC #7/#8 `[@contract-v1]` ack 來源，見 Completion Notes；未修改）

---

## Senior Developer Review (AI)

**Date:** 2026-08-18 · **Reviewer:** Fable 5（換模型慣例 —— impl by Opus 5）· **Outcome:** Approve（0H/2M/3L，全數當場修復）

**Mandatory checks:** 🔒 Rule 7 Wire Format: PASS（in-review Go 檔 0 個新 error-code 常數，prefix 維持 16）· 🔒 Rule 20 Contract Bump: N/A（0 bumps；grep 到的 `→v` token 全在未觸及的既有檔案註解）· 🔒 Rule 25 Mega-line: N/A（project-context.md 不在 diff）· Git vs File List: 0 discrepancies · Task audit: 全部 [x] 有實據。

**攻擊過但未破的面**（記錄以免下次重查）：快取鍵不含 media_type —— movies/episodes 都是 `uuid.New().String()`（`scanner_service.go:503` / `media_ingest_service.go:144`），相撞不可能；批次後重分析的正確性 —— 已生成項目由 enumeration predicate（`FindMissingZhHantSubtitle`）直接排除，字幕是 sidecar 不改媒體檔，快取鍵依然有效；全套 `-race` 重跑綠。

### Action Items

- [x] **[M1] TTL/type 傳遞鏈零測試 —— `routeCacheTTL` 壞值會靜默停用整個快取。** 真 repo `Set` 對 `ttl<=0` 回 error，經 `storeRoute` fail-soft 只剩 Debug log ⇒ 永遠 miss 且無人察覺（本 story 自己命名的「靜默失效」家族）。且 `routeCacheRepository` adapter 零覆蓋。→ 修復：`fakeRouteCache` 記錄 `setTTLs`，新增 `TestRouteCache_WritesCarryThePositiveTTL`（斷言每筆寫入帶 `routeCacheTTL` 且為正）；新增 `fakeCacheRepo`（實作完整 `CacheRepositoryInterface`）＋ `TestRouteCacheRepository_TagsTheFamilyAndMapsValues`（釘 `subtitle_route` type 標記與 value 映射／miss-by-absence）。[route_cache_test.go]
- [x] **[M2] 命中路徑對 `RouteSkipped` 的讀側行為未測。** 唯一會改變 summary 數字的 hit 分支，靠「hit 與 fresh 落同一 switch」隱含保證。→ 修復：新增 `TestRouteCache_CachedSkipHitStillCountsAsSkipped`（skip 命中 ⇒ `SkippedCount==1`、不進 Candidates、零探測、零寫入）。[route_cache_test.go]
- [x] **[L1] `planRouteCache` 的 stat 掃描不檢查 ctx。** `os.Stat` 不吃 ctx，取消後會把整庫 stat 跑完才被迴圈接住。→ 修復：`planRouteCache` 改收 `ctx`，迴圈每列 `ctx.Err()` 早退（未掃到的列 zero-valued，Analyze 迴圈第一列即返回，無害）；新增 `TestRouteCache_CancelledContextStopsTheStatSweep`（cancel 後 stat 呼叫數 == 0）。[generation_candidates.go]
- [x] **[L2] mtime 秒級截斷（`.Unix()`）未寫入 `[@contract-v1]` doc。** 同秒同 size 重寫 ⇒ stale hit（實務機率趨近零，scanner 精度同級）。→ 修復：`routeCacheKey` doc 增加「KNOWN PRECISION LIMIT」段，明文這是 accepted 而非 overlooked。[route_cache.go]
- [x] **[L3] `planRouteCache` 與 `classify` 的 probe 判定是隱性 parity 依賴。** classify 未來新增免探來源（episode tech-info parity 落地時必然發生）會讓 `route_probes` 靜默失真。→ 修復：兩處互指的「PARITY CONTRACT」註解，明文點名 `backlog-episode-tech-info-parity` 是已知觸發點。[generation_candidates.go]

**修復後全回歸：** `go build` ✅ · `go test ./...` 全套 0 失敗 ✅ · `-race`（services 全包）✅ · staticcheck（`api:lint`）✅ · `pnpm nx test web` 233 files / 2653 tests ✅ · `format:check` ✅ · `test:cleanup` 無殘留 ✅。route cache 測試 16 → **20 個**。

## Change Log

| Date | Change |
| --- | --- |
| 2026-08-18 | Adversarial CR（Fable 5 審 Opus 5）：0H/2M/3L 全數當場修復 —— M1 TTL/adapter 零測試（補 `WritesCarryThePositiveTTL` + `TagsTheFamilyAndMapsValues`）、M2 cached-skip 讀側未測（補 `CachedSkipHitStillCountsAsSkipped`）、L1 plan 迴圈 ctx 早退＋測試、L2 mtime 秒級精度限制寫入 key doc、L3 plan/classify PARITY CONTRACT 互指註解。測試 16→20；全回歸 api ✅ web 233/2653 ✅ lint/format ✅。Status → done。 |
| 2026-08-18 | Task 1–4（AC #1–#6）dev-story 實作完成。新增 `route_cache.go`：`RouteCache` 窄埠（batch-only 讀，沿用 sub-1-5b 先例）、`[@contract-v1]` 鍵 `subtitle:route:v{n}:{id}:{size}:{mtime}`（size+mtime 取自 `os.Stat`，與 scanner 7-2 同一組信號）、`routeVersion=1` 附 bump 規則、TTL 30 天、type `subtitle_route`（重用 `cache_entries`，零 migration）。`Analyze` 迴圈前一次 `GetMany`；命中免 `Probe`；只在 `classify` 回 `ok==true` 時寫入（探測失敗永不快取＝本 story 紅線）；讀寫失敗皆 fail-soft（Rule 13 case 3）。完成 log 加 `route_cache_hits`/`route_probes`。wire shape／進度分母／scanner 面／`RoutePredictor` 介面全部零改動。16 個新測試；全回歸 api ✅ web 233/2653 ✅ lint 0 errors ✅ format ✅。lane ③×1（`backlog-go-gofmt-not-enforced`）。 |
| 2026-08-18 | create-story：M3 A 群組。**依 seed 要求先盤點,結果推翻了字面範圍** —— 檔案系統層增量掃描早於 2026-03 由 story 7-2 交付（size/mtime skip + detectRemovedFiles）,屬 scanner 面;真正每次從頭來的是 F14 候選分析,因為 episodes 無 tech-info 欄位而永遠走 ffprobe 探測分支。改為交付以「檔案身分＋述詞版本」為鍵的路線快取（重用 cache_entries,零 migration）。lane ③×1（episode tech-info parity）。 |
