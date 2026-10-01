# Story sub-7-5b: 掃描完自動從官方字幕學譯名、寫進詞彙表、設定頁一顆按鈕 — 後端＋前端

Status: in-progress

<!-- SM Bob create-story 2026-10-01，由 sub-7-5 拆出；依賴 sub-7-5a（`mine` 套件與 NAS 實測結果）。行號為 main `12a4f507`。 -->

## Story

身為 NAS 使用者，
我要掃描完片庫後 Vido 自己去讀同一部劇已有的官方繁中字幕、把學到的譯名放進那部劇的詞彙表，
也可以在設定頁按一顆按鈕重新學一次，之後翻沒字幕的那幾集就會用同樣的譯名。

## 背景（查到的事）

- `GlossaryScopeResolver.Resolve` 第一次解到 `tmdb:*` 時會呼叫 `GlossarySeeder.EnsureSeeded`（sub-7-3 的 lazy seeding 範例）——本單的「學習」可掛同一個 seam，或掛 `ScannerService.SetOnScanComplete`（`main.go:532` 已被 `postScanEnrichment` 佔用，callback 只有一個，要串）。
- 寫入：`GlossaryRepository.InsertIfAbsent`、`source=official_subtitle`、`confirmed=false`；既有 manual／confirmed 永不覆寫（insert-only 天然成立）。
- 排除 Vido 自產：7-5a 以檔名規則排除 `.zh-Hant.srt`；`subtitle_runs.output_path` 可再對照一次。
- SSE：`sse.EventNotification`（`hub.Broadcast`）。
- 設定頁：`apps/web/src/routes/settings/subtitle.tsx` 目前只有 `LocalizationLevelForm`；按鈕要 Sally 補 C9-D／C9-M 的一列（成本 $0 的按鈕，不走 ButtonCost）。
- ~~可選 LLM 精煉~~：7-5a 實測 Shadow and Bone 68 詞、抽查 19/20，**不做**。
- ⚠️ 7-5a 實測兩個前提：(a) `.zh-TW.hi` 檔名不保證官方——Scorpion S01 六集全是人人影視／ZiMuZu 字幕組檔，人名不一致、時間軸對不上；**學到的詞要帶來源檔名進 GlossaryPanel**，偵測到字幕組署名（檔頭「字幕組」「人人影視」「ZiMuZu」「翻譯：」）的檔跳過；(b) partial 判定用 ffprobe 實探（Supernatural 的 .mp4 沒有任何字幕串流，eval CSV 算錯）。

## Acceptance Criteria

1. **服務**：`services.OfficialSubtitleMiner`：給 series id → 列該劇所有集（`EpisodeRepository.FindBySeriesID`）→ 對每集 `mine.Classify`（ffprobe 軌＋sidecar 檔名）→ 可用的集抽軌／讀 sidecar → `mine.Align` → 合併全部段 → `mine.Mine`（`Known`＝該 scope 既有 `metadata`／`manual`／confirmed 詞）→ `InsertIfAbsent(scope, src→zh, source=official_subtitle, confirmed=false)`；回 `{episodes_used, terms_found, terms_inserted}`；單劇一次最多 N 分鐘、ffmpeg 走既有 ExtractGate。
2. **自動觸發**：掃描完成後，對「partial」影集（有 zh 的集 > 0 且沒 zh 但有英文軌的集 > 0）背景跑，純本機 $0；結果 `notification` SSE 一則（「從《Scorpion》的 13 集官方字幕學到 18 個譯名」）；失敗只 log。
3. **手動**：`POST /subtitles/glossary/mine?series_id=` 與設定頁「重新從官方字幕學習」按鈕（列出 partial 影集、逐劇進度）。
4. **測試**：partial 判定；Known 由既有詞彙表組成；insert-only 不覆寫；SSE 內容；handler 400／404／202。
5. **設計稿**：Sally 補 C9 的一列（按鈕＋結果行）→ Alexyu inline agent → 截圖。

## Task 0 提示詞（Sally 2026-10-01；貼給 Pencil Inline AI Agent 原樣執行，一段跑完 ⌘S 再跑下一段）

查證過的程式碼事實：設定頁「字幕設定」（`routes/settings/subtitle.tsx`）目前只有一張「AI 字幕的在地化程度」radio 卡（`LocalizationLevelForm`）。本單加的是**同一頁的第二個區塊**，不是新頁。按鈕 $0、不走 ButtonCost；跑完結果是一行字，不是 toast（SSE `notification` 前端沒有消費者，7-5b 改成「按下去→202→輪詢狀態」）。

> **A · C9-D 加「從官方字幕學譯名」區塊**
>
> 1. 開啟 `NR3zK`（C9-D · 字幕設定）。在「AI 字幕的在地化程度」整組（小標＋說明＋三張 radio 卡）**下方** `$Space/2xl` 處，`Copy` 那組的小標＋說明文字兩個節點當骨架，改成：小標「從官方字幕學譯名」（同字級同色），說明「同一部劇有些集已經有官方繁中字幕、有些集沒有時，Vido 會讀那幾集學會這部劇的人名怎麼翻，再用同樣的譯名翻其他集。掃描完會自動做；這裡可以手動再跑一次。」（`$text-secondary`，`textGrowth: fixed-width` 填滿欄寬）。
> 2. 說明下方 `$Space/md` 放一列 horizontal、`alignItems: center`、gap `$Space/md`：左邊一顆 `Component/Button/Secondary`（`YDPhc`）文字「重新從官方字幕學習」；右邊一段 `$text-secondary` Body 文字「上次 2026-10-01 14:30 · Shadow and Bone 學到 68 個詞（8 集）、Scorpion 跳過 6 個字幕組檔」。
> 3. 整張 frame `placeholder:false`，確認 `problems` 為空、新區塊沒有超出內容欄寬（與上方 radio 卡同寬）。
>
> **B · C9-M 同一區塊（手機）**
>
> 1. 開啟 `AWYm0`（C9-M）。在三張 radio 卡下方加同一組：小標、說明（填滿寬度自動換行）、按鈕改用 `Component/Button/Touch`（`JK9so`，全寬）、結果文字另起一行放在按鈕**下方**（`$text-secondary`，自動換行）。
> 2. 可捲動區照舊長高；`problems` 為空。
>
> **C · 狀態附註（不另開 spec 畫面，寫成 C9-D 右側的一張 note）**
>
> 在 C9-D 新區塊右側 `$Space/xl` 處放一個 note（同 K1-M 的 note 慣例），四行：「① 閒置：按鈕可按＋上次結果一行（第一次是『還沒跑過』）」「② 學習中：按鈕停用、文字改『學習中…』，每 3 秒輪詢；主要是讀檔與 ffmpeg 抽軌，$0」「③ 跑完：一行結果，每部劇一句，用頓號接；跳過的字幕組檔要說」「④ 沒有可學的劇：『片庫裡沒有「部分集有官方字幕」的影集』」。

**設計裁定理由**：這是 $0 的本機動作，所以用 Secondary 不用 ButtonCost（J9 的金額記號只給會花錢的按鈕）；結果用一行文字留在原地而不是 toast，因為使用者想知道「上次學到幾個」是回來看的，不是當下彈出的。放在字幕設定頁而不是媒體庫掃描頁，因為它改的是詞彙表、影響的是翻譯品質。

跑完之後：⌘S → `python3 scripts/export-pen-screenshots.py` → 只 stage `c9-d`、`c9-m` 兩張真變更的 PNG ＋ `_bmad-output/pen-tokens.json` → commit。Sally 以 MCP 複審後才開工 Task 3 的前端。

## Tasks / Subtasks

- [ ] Task 0 — 設計稿（AC #5）
- [x] Task 1 — `OfficialSubtitleMiner` 服務＋寫入（AC #1）
- [x] Task 2 — 掃描後自動觸發（AC #2；SSE 改成狀態端點輪詢，見 Dev Agent Record）
- [~] Task 3 — 端點（done）＋設定頁按鈕（等 Task 0 設計稿）（AC #3）
- [~] Task 4 — 測試（BE 9 條 done；FE spec 等 Task 3）（AC #4）

## Dev Notes

- 先拿 7-5a 的 NAS 實測結果（Scorpion S01 ≥15 詞／≥90%）；不到門檻再考慮原單 AC #4 的 LLM 精煉，並另立 7-5c。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- **後端（Task 1–3 的端點）已交付**：新套件 `internal/subtitle/miner`（放 `subtitle` 會與 `mine → subtitle` 成環；放 `services` 拿不到 Extractor）。`OfficialSubtitleMiner.MineSeries`（同步、回 `MineResult`）、`MinePartial`（片庫掃一輪，partial 判定只看 sidecar 檔名——不 ffprobe 一千集；只有內嵌中文軌的劇靠手動端點）、`ScanCallback`（掃描後背景跑，失敗只 log）、`Status`（跑中旗標＋上次結果）。字幕組檔（檔頭 4KB 命中 字幕組／人人影視／ZiMuZu／YYeTs／翻譯：…）跳過並計入 `fansub_skipped`。Known＝該 scope 既有詞彙（排除自己之前的未確認 official_subtitle 列）；寫入 `InsertIfAbsent`、`source=official_subtitle`、`confirmed=false`、`MediaID=series id`。
- `ScannerService.AppendOnScanComplete`：新方法，把 miner 串在既有 enrichment／auto-generation callback 之後，不用知道它們怎麼組的。
- 端點 `POST /subtitles/glossary/mine`（`series_id` 有→同步跑那部劇回 200；沒有→202 背景掃 partial；跑中 409 `GLOSSARY_MINE_RUNNING`；該劇失敗 500 `GLOSSARY_MINE_FAILED`）、`GET /subtitles/glossary/mine`（`MineStatus` [@contract-v1]）。
- 🔗 AC Drift：AC #2 的「`notification` SSE 一則」**不做**——前端沒有任何 `notification` 事件的消費者、後端也從未發過；改成狀態端點＋前端輪詢，結果留在設定頁那一行（也更符合「回來看上次學到幾個」）。AC #1 的「每劇最多 N 分鐘」未加硬上限：ffmpeg 已有 ExtractGate 與每集逾時。
- 📎 來源檔名進 GlossaryPanel（7-5a 建議）：`show_glossary` 沒有欄位放，本單只在 `MineResult.episodes[].zh_source` 回報；立案 `backlog-glossary-term-provenance-file`。
- 測試：miner 4 條（端到端含 Vido 自產排除與 known 不重寫、字幕組跳過、partial 只挑部分集有的劇、busy 與錯誤）、handler 3 條、scanner 串接 1 條；`go test ./...` 全綠、vet、staticcheck 乾淨。
- ⏳ **前端（Task 3 按鈕＋Task 4 FE spec）等 Task 0 設計稿**。

### Discovery Triage

- `backlog-glossary-term-provenance-file`：詞彙表列沒有「哪個檔教的」欄位；7-5a 實測證明檔名分不出官方與字幕組，使用者在面板裡看不到來源就無從判斷。要 migration（`show_glossary.provenance TEXT NULL`）＋面板一欄。非阻塞。

### File List

- apps/api/internal/subtitle/miner/miner.go、miner_test.go
- apps/api/internal/handlers/glossary_mine_handler.go、glossary_mine_handler_test.go
- apps/api/internal/services/scanner_service.go、scanner_callback_chain_test.go
- apps/api/cmd/api/main.go
- _bmad-output/implementation-artifacts/sub-7-5b-mine-official-subs-wiring.md、sub-7-5a-mine-official-subs-algorithm.md、sprint-status.yaml

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-5 拆出）。 |
| 2026-10-01 | dev-story Task 1–3 後端（Amelia）；Task 0 提示詞出稿（Sally），等 Alexyu 跑 inline agent。 |
