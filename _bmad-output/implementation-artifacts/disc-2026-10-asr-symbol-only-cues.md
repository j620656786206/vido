# Disc：聽寫把喊叫寫成「$%&!」，翻譯照抄進字幕

Status: done

**Source:** 2026-10-11 正式機驗證 `disc-2026-10-asr-music-only-dash`（PR #740），See S01E02 的 10 分鐘片段。

## 查到的事（2026-10-12，Bob 逐一開檔／讀 NAS 材料確認）

### 實際看到的

- NAS `/mnt/cache/appdata/vido-eval-clip/verify-20261011/clip.en.srt`（聽寫出來的英文，已經過幻聽過濾）共 70 句。用 Python 掃「沒有任何字母、也沒有任何數字」的句子，剛好 3 句：
  - `00:04:23,886 --> 00:04:24,166`：`-$%&!`
  - `00:04:24,546 --> 00:04:24,686`：`$%&!`
  - `00:04:44,566 --> 00:04:45,966`：`$%&!`
- 同目錄 `clip.zh-Hant.srt` 同樣時間是 `-$%&！`、`$%&！`、`$%&！`：翻譯照抄，只把半形驚嘆號換成全形，就進了字幕。
- 這三句是打鬥時的喊叫或髒話。Whisper 用 SDH（聽障字幕）的「消音」寫法表示。觀眾在畫面上看到一串符號，看不懂它的意思。

### 為什麼現在的過濾沒擋到

- 聽寫的每一段都會先過 `filterHallucinationsWith`（`apps/api/internal/ai/whisper.go:366` 呼叫），R0 那一輪（`apps/api/internal/ai/whisper_segments.go:420-433`）只認兩種「不是對白」：
  - 提詞回音（`prompt_echo`）
  - 只有音符（`IsMusicOnlyText`，`whisper_segments.go:426`）
- `IsMusicOnlyText`（`apps/api/internal/ai/music_only.go:22-41`）只放過空白、零寬字元、說話人破折號（`-`、`–`、`—`），以及音符 `♪♫♬#`（`music_only.go:10`）。遇到其他任何字元，例如 `$`、`%`、`&`、`!`，就回 false。所以 `$%&!` 不算「只有音符」，被留了下來。
- 丟掉的原因是固定字串，會出現在 log 和測試裡（`whisper_segments.go:255-276`）。例如 2026-10-11 的 `clip.log` 第 110 行：`dropped_by_reason=map[music_only:8]`。

### 同一個判斷，另一條路

- 片內英文字幕軌也會用 `ai.IsMusicOnlyText` 丟掉只有音符的句子（`apps/api/internal/subtitle/sdh_filter.go:103`、`:115`）。這張**不改**那一條：
  - 官方字幕裡的 `#$%&!` 是人工打的消音髒話，有沒有意義要另外判斷。
  - 而且到目前為止沒看過有人抱怨。

### 花費參考

- 2026-10-11 那次 10 分鐘片段整次花 US$0.1888（`clip.log` 第 168 行 `spent_usd=0.1888269`）。其中聽寫每 2 分鐘一段，每段約 US$0.012（第 112–130 行）。

## Story

身為看 AI 字幕的人，
我希望聽寫出來「只有符號、沒有任何字」的句子（例如 `$%&!`）不要出現在字幕上，
這樣畫面上不會冒出一串看不懂的符號，也不用花錢翻譯它。

## Acceptance Criteria

1. **新判斷：** `ai` 套件新增 `IsSymbolOnlyText(text string) bool`，規則如下：
   - 去掉空白和零寬字元（跟 `IsMusicOnlyText` 同一套）之後，還要剩下**至少一個字元**。
   - 剩下的字元裡，**沒有任何字母**（`unicode.IsLetter`，中文字也算字母）和**數字**（`unicode.IsNumber`）。
   - 測試要求：
     - 判成 true：`$%&!`、`-$%&!`、`- $%&!`、`...`、`?!`、`#$%&!`
     - 判成 false：空字串、純空白、`Ugh!`、`3!`、`啊！`、`-Hello`、`♪ lyrics ♪`
   - `♪♪` 這類只有音符的句子，本身也符合這個條件，但它們仍歸在 `music_only`（見 AC #2）。
2. **聽寫過濾丟掉它：** `filterHallucinationsWith` 的 R0 新增丟掉原因 `symbol_only`（固定字串常數，跟其他原因放在一起，附註解說明來源）。
   - 判斷順序：提詞回音 → 只有音符 → 只有符號 → 其他規則。所以 `♪♪` 仍記成 `music_only`，log 才分得出兩種。
   - 被這條認定的句子，不計入「重複句」的計數（跟 music_only 一樣，因為 R2b 只算還沒被認定的句子）。
3. **測試：**
   - 用 NAS 上那 3 句原文（`-$%&!`、`$%&!`、`$%&!`），夾在真對白中間，跑 `filterHallucinations`：3 句都以 `symbol_only` 丟掉，真對白全部保留。
   - 再加一條：`♪♪` 仍是 `music_only`；`3!` 和 `Ugh!` 保留。
4. **不改片內字幕軌那條路**（`subtitle/sdh_filter.go`），在 Dev Notes 寫明理由。
5. **檢查：** `go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all` 全綠。
6. **正式機實測（部署後補，要花錢）：** 在丟棄容器重跑同一個 10 分鐘片段。上次花 US$0.19，這次會先給 Alexyu 看估價、他同意才跑。驗收標準：
   - 英文 SRT 沒有只有符號的句子。
   - log 的 `dropped_by_reason` 出現 `symbol_only`。

## Tasks / Subtasks

- [x] T1 `apps/api/internal/ai/music_only.go` 加 `IsSymbolOnlyText`，跟 `IsMusicOnlyText` 共用「這個字元可以略過嗎」的判斷（AC #1）
  - [x] 表驅動測試放 `music_only_test.go`
- [x] T2 `whisper_segments.go`：常數 `dropReasonSymbolOnly = "symbol_only"`，接到 R0（AC #2）
  - [x] 過濾測試（AC #3）
- [ ] T3 正式機實測（AC #6，部署後，先給估價、等 Alexyu 同意）

全部後端，一個檔案的規則加一個常數，不用拆單。

## Dev Notes

### 不要做的事

- 不要把 `$%&!` 翻成「啊！」。翻譯那邊要另外教模型；這張用最小、最省錢的做法：直接丟掉。
- 不要改 `IsMusicOnlyText` 的行為。片內字幕軌也用它（`sdh_filter.go:115`），把 `$` 加進音符會連帶改到那條路。
- 不要動片內字幕軌那條路（AC #4）。

### 已知陷阱

- `unicode.IsLetter` 對中文字會回 true，所以「啊！」會保留。這是對的。
- 純標點的 `...`、`?!` 也會被丟掉。這是刻意的：字幕上單獨一句「...」或「?!」本來就沒意義。這兩種沒有在實際檔案裡看過，所以只是規則順帶涵蓋到，不是這張要修的痛點。
- 空字串不能判成 true，不然會在 log 裡多出一堆莫名其妙的 `symbol_only`。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

### References

- 上一張同類：`disc-2026-10-asr-music-only-dash.md`（說話人破折號）、`disc-2026-10-asr-music-only-cues.md`（只有音符）。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-12。

### Debug Log References

- 紅：先寫 `TestIsSymbolOnlyText`／`TestFilterHallucinations_R0SymbolOnly`，編譯失敗（`dropReasonSymbolOnly` 未定義）→ 實作後綠。
- staticcheck ST1018：測試字串裡直接放了 U+200B，改成 `\u200b` 跳脫。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- `IsSymbolOnlyText` 跟 `IsMusicOnlyText` 共用新的 `isBlankRune`（空白＋零寬字元），`IsMusicOnlyText` 行為不變（原有測試全綠）。
- R0 順序：提詞回音 → `music_only` → `symbol_only`。測試用 NAS 那 3 句原文與原時間（263.886、264.546、284.566 秒），3 句都以 `symbol_only` 丟掉；`♪♪` 仍記 `music_only`；`3!`、`Ugh!`、`Face me!` 保留。
- 片內字幕軌那條路（`sdh_filter.go`）沒動。
- 檢查：`go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all`（0 errors）全綠。

### Senior Developer Review (AI) — 2026-10-12

結論：0 High／0 Medium／2 Low，皆接受。

- **L1（接受）** 斷點續跑的聽寫段落快取（`services/asr_chunk_store.go:73-78`）存的是**已過濾**的 SRT。30 天內「跑到一半中斷」的片，續跑時拿回來的舊段落仍可能帶 `$%&!`。這只會發生在中斷過的片、只有一次，而且整部跑完後快取就會刪掉。要擋住它，得把快取版本號升到 v3，但那會讓所有中斷過的片重新付聽寫費，划不來。接受。
- **L2（接受）** 單獨一個 `-`、`...`、`?!` 或只有表情符號的句子也會被丟掉。字幕上單獨出現這些沒有意義，故意這樣設計（Dev Notes 已寫）。

### Discovery Triage

- N/A — no out-of-scope work discovered（Bob 建檔時）。

### File List

- `apps/api/internal/ai/music_only.go`（`IsSymbolOnlyText`、`isBlankRune`）
- `apps/api/internal/ai/music_only_test.go`
- `apps/api/internal/ai/whisper_segments.go`（`dropReasonSymbolOnly`，接 R0）
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-symbol-only-cues.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-12 | Bob create-story（R0 加 `symbol_only`：沒有字母和數字的句子丟掉；片內字幕軌那條路不動）。 |
| 2026-10-12 | Amelia dev-story＋code review 0H/0M/2L（皆接受）；全套綠。T3 正式機實測待部署、要先給估價。 |
| 2026-10-12 | PR #745 CI 全綠合併；T3 正式機實測待部署（先給估價）。 |
