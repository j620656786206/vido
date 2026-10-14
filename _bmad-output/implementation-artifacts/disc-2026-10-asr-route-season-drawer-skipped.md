# Disc：聽聲音那條路翻出來的人名，跟片內字幕那條路不一樣（季別詞彙沒套到）

Status: review

**Source:** 2026-10-14 v0.2 發版前 NAS 實測（丟棄容器、正式 DB 副本 `pre-verify-20261014-0645`、See S01E02 前 10 分鐘片段）。ElevenLabs 聽寫＋翻譯出來是「巴霸沃斯／巴莉絲／傑拉馬瑞／哈妮娃／可風」；同一集走片內英文字幕＋Gemini 翻出來是「巴巴禾斯／芭麗絲／謝拉馬威」。Alexyu：「人名又不一致了。」

## 查到的事（2026-10-14，main `35424210`）

- 詞彙表資料是對的（NAS DB 直查）：`tmdb:tv:80752`（全劇）Baba Voss→巴霸沃斯、Jerlamarel→傑拉馬瑞；`tmdb:tv:80752:s1` Baba Voss→巴巴禾斯、Jerlamarel→謝拉馬威、Paris→芭麗絲、Maghra→瑪嘉拉；`:s2` 是另一套。全部 `confirmed=0`。第 1 季的集應該套 `:s1`。
- 片內字幕那條路（`apps/api/internal/subtitle/glossary_store.go:99`）用**集的 id** 呼叫 `ResolveSeason` → 套到 s1 → 對。
- 聽聲音那條路：`apps/api/internal/services/transcription_service.go:1836-1838` `translationInputsFor` 先算 `glossaryKey := glossaryKeyFor(mediaID, episodeRow)`（集 → **父影集的 id**，sub-5-5 CR H1 的設計，讓 harvest 落在同一本），再 `s.loadGlossary(ctx, glossaryKey)`；`loadGlossary:448` 拿同一個參數去 `sr.ResolveSeason(ctx, mediaID)`（`:446-449`）。`ResolveSeason`（`glossary_scope_resolver.go:137-146`）用 `episodes.FindByID(影集 id)` → 找不到 → `ok=false` → 季別抽屜**永遠不套**。
- 實測 log 證實：`translating with per-show glossary … glossary_key=8181ee59-…（影集 id） term_count=79`——79 條就是全劇抽屜（80 條減一條非 zh-Hant），沒有 s1 的覆蓋。
- 既有測試 `TestLoadGlossary_SeasonDrawerOverlays` 之所以綠：它直接拿**集的 id** 呼叫 `loadGlossary`，跟正式路徑的呼叫方式不同，所以抓不到。
- 2026-10-07 `disc-2026-10-glossary-season-scope-a` 的正式機驗證（「See 三層 99／11／11」）驗的是片內字幕那條路。

## Acceptance Criteria

1. 聽聲音那條路翻譯時，集的季別抽屜會套在全劇抽屜上（跟片內字幕那條路一樣）：`loadGlossary` 改收「全劇 key」＋「集 id」兩個參數，季別只從集 id 解析；電影／整劇跑時集 id 為空、不查季。
2. 回歸測試走正式路徑 `translationInputsFor`：集 S01E02、全劇 Baba Voss→巴霸沃斯、s1→巴巴禾斯，結果要是巴巴禾斯、`glossaryKey` 仍是影集 id（harvest 不變）。
3. 既有的 `TestLoadGlossary_SeasonDrawerOverlays` 改成用正式的呼叫方式（全劇 key＋集 id），並加電影案例。
4. `go test ./internal/services/...`、vet、staticcheck 綠。
5. 正式機驗證：同一段片段再跑一次聽寫（不重聽、asr_chunk 快取保留，只重翻），人名要變成巴巴禾斯／謝拉馬威／芭麗絲。

## Tasks / Subtasks

- [x] T1 `loadGlossary(ctx, glossaryKey, episodeID)`；`translationInputsFor` 傳集 id（AC 1）
- [x] T2 測試（AC 2、3）
- [x] T3 全套檢查（AC 4）
- [ ] T4 NAS 重翻驗證（AC 5，合併後）

## Dev Notes

- 不改 `glossaryKeyFor`（harvest 要落在影集那本，sub-5-5 CR H1 的理由仍成立）。
- `nfo_localizer_service.go` 也有一個同名 `loadGlossary`，是別的 receiver，不動。
- 同一次實測另一個問題（每句顯示不到 1 秒）是 `disc-2026-10-asr-cues-too-short-to-read`，分開修。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- `transcription_service.go`：`loadGlossary` 多一個 `episodeID` 參數，季別只從它解析（空字串＝不查）；`translationInputsFor` 在有 episodeRow 時傳 `mediaID`。log 欄位改 `glossary_key`／集 id。
- 測試：`TestLoadGlossary_SeasonDrawerOverlays` 改用（全劇 key，集 id）呼叫、加電影案例；新增 `TestTranslationInputsFor_EpisodeRunReadsItsSeasonDrawer` 走正式路徑（用既有的 `metadataEpisodeReader`／`seasonScopes` fake）。修之前這條會拿到巴霸沃斯。
- 順便記錄同一天的兩個正式機驗證結果（見 sprint-status 的 `asr-scribe-b`／`gemini-translation-dispatch`）。

### Discovery Triage

N/A — no out-of-scope work discovered（字幕太短已有單）。

### File List

- `apps/api/internal/services/transcription_service.go`、`apps/api/internal/services/transcription_asr_prompt_test.go`
