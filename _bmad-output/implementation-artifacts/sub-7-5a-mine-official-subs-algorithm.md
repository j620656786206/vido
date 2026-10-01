# Story sub-7-5a: 從官方繁中字幕學譯名——對齊＋抽詞演算法＋驗收用指令 — 後端

Status: review

<!-- SM Bob create-story 2026-10-01，由 sub-7-5 拆出（原單 BE 5 task 含觸發／SSE／設定頁按鈕／可選 LLM，超過上一張同類單子的規模 → 拆成 7-5a 演算法＋CLI／7-5b 接線）。行號為 main `12a4f507`。 -->

## Story

身為片庫裡 Scorpion S01 有 13 集官方繁中、79 集沒有的人，
我要 Vido 先能從那 13 集「學會」這部劇的人名怎麼翻，
而且在接進掃描流程之前，我可以在 NAS 上用一個指令跑給自己看。

## 背景（查到的事）

- `eval/scan-partial-zh.sh` 定義了「官方繁中」：外掛 `.zh-TW／.cht／.tc／繁／.zh-Hant.hi／.zh-Hant.sdh`，或內嵌 `chi/zho` 文字軌；Vido 自產 `.zh-Hant.srt`（含 .bak）不算。本單的 `IsOfficialZhSidecar` 同一條規則。
- 內嵌軌要 ffmpeg 抽：`subtitle.Extractor.Extract(ctx, media, tmp, []int)` 回 stream→.srt 路徑；外掛 .ass 沒有 parser → 本單加最小 `ParseASS`（只讀 Dialogue 行）。
- `services.DetectExternalSubtitles` 只會把 `.zh-TW.hi` 整段當語言 tag，所以 sidecar 用檔名規則自己分類。
- 詞彙表寫入（`InsertIfAbsent`、`source=official_subtitle`）、scope resolver、TMDb 已播種名單都在 services——本單的 `internal/subtitle/mine` 不 import services（Rule 19），只吃字串，寫入放 7-5b。

## Acceptance Criteria

1. **來源判定**：`mine.Classify(media, sidecars, tracks)` → `Sources{ZhSidecars, EnSidecars, ZhStreams, EnStreams}`；`Usable()`＝兩邊都有；規則與 `scan-partial-zh.sh` 一致，Vido 自產與 .bak／.tmp 排除。
2. **對齊**：`mine.Align(en, zh)`：IoU ≥ 0.5 或「短的 80% 落在長的裡」配對，1:N／N:1 合併成段，±300ms 偏移容忍，沒對到的丟掉。SRT（`subtitle.ParseSRT`）與 ASS 都讀得進。
3. **抽詞（零成本）**：`mine.Mine(segments, Options)`：英文候選＝首字大寫 1–4 詞連續 token，**句首第一個 token 不算**（那個大寫是句子的不是名字的），停用詞／全大寫縮寫排除；中文側＝該候選所有句對裡共同出現的 2–4 字漢字子串，支持 ≥3、涵蓋 ≥60% 句對、且在這些句對外不常見；`Known`（TMDb 已播種／已確認詞彙）走整詞搜尋直接驗證，不受候選規則限制。
4. **驗收指令**：`go run ./cmd/mine --dir <季資料夾> [--known known.json] [--json]`：逐集列來源與段數、最後列詞＋支持度；外掛字幕不需要 ffmpeg，內嵌軌在容器裡跑。
5. **測試**：來源規則正反表；對齊（偏移、1:N、N:1、擦邊不配）；ASS／SRT 解析；候選規則（句首、停用詞、縮寫）；共現學習、Known 驗證、常見詞拒絕；CLI 端到端（外掛字幕的假影集）。

## Tasks / Subtasks

- [x] Task 1 — 來源判定＋對齊＋ASS parser（AC #1, #2）
- [x] Task 2 — 抽詞與 Known 驗證（AC #3）
- [x] Task 3 — `cmd/mine`（AC #4）
- [x] Task 4 — 測試（AC #5）

## Dev Notes

- 驗收樣本（原單 AC #6：Scorpion S01 ≥15 詞、抽查 20 筆 ≥90%）要在 NAS 容器裡跑：`docker exec -it Vido sh -c 'cd /app && ./mine --dir "/media/TV/Scorpion/Season 01"'`（或 `go run` 在本機對掛載路徑）。結果回填到本節後，7-5b 才決定要不要開 LLM 精煉。
- 單字名（Walter、Paige）只在句中出現時才會被學到；只在句首出現的名字靠 TMDb 播種名單（Known）補。這是刻意的：句首大寫無法區分「Ask」與「Walter」。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 新套件 `internal/subtitle/mine`（sources.go／align.go／mine.go）＋ `cmd/mine`。`subtitle` 多兩個薄 export（`IsChineseLanguageTag`／`IsEnglishLanguageTag`）給 sibling 套件用。
- 對齊用 group 合併（兩邊各一個 group id，交疊即併），所以 1:N 與 N:1 都是同一條路；輸出依英文起始時間排序。
- 中文側不分詞：2–4 字子串頻率＋「專屬度」（在候選句對內的次數 ≥ 全片的一半）擋掉「我們」這種常見詞；平手依序：先挑較常「獨立成段」的（「嘿，華特，」——人名會被呼喚、被標點夾住，片語則黏著鄰字），再挑在別處較少出現的，最後挑較長的——「華特」贏「我們需要」、「托比」贏「去問托比」、全名贏自己的前綴。
- 🔗 AC Drift（相對原單 7-5 AC #3）：候選長度 1–4 詞而非 2–4（否則 Walter／Toby 這種單名全部學不到）；補「句首第一 token 不算」規則作為代價。
- 測試 9 條全綠；`go test ./...` 全綠、vet、staticcheck 新套件乾淨。沒有在 NAS 真跑（AC #6 的實測交 Alexyu，指令在 Dev Notes）。

### Discovery Triage

- 無新單。

### File List

- apps/api/internal/subtitle/mine/sources.go、align.go、mine.go、mine_test.go
- apps/api/internal/subtitle/extractor.go（兩個 export）
- apps/api/cmd/mine/main.go、main_test.go
- _bmad-output/implementation-artifacts/sub-7-5a-mine-official-subs-algorithm.md、sub-7-5b-mine-official-subs-wiring.md、sub-7-5-mine-official-subs-in-series.md、sub-7-9-per-cue-alignment-lock.md、sprint-status.yaml

## Change Log

| Date       | Change                                             |
| ---------- | -------------------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-5 拆出）＋ dev-story（Amelia）→ review。 |
