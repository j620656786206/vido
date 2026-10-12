# Disc：片尾配樂被聽寫成「Subs by www.zeoranger.co.uk」

Status: review

**Source:** `disc-2026-09-whisper-credit-line-hallucination`（2026-09-19 由 `disc-2026-09-generation-resume-b-asr-chunk-store` 真機驗證時抽查字幕發現，P3）。2026-10-12 Alexyu 從 backlog 選 A。

## 查到的事（2026-10-12，Bob 逐一開檔／唯讀查 NAS 確認，main `c45a9e69`）

### 實際看到的

- 2026-09-18 NAS 實測《哈利波特：火盃的考驗》：全片對白 2:23:02 就結束，片尾只有配樂。Whisper 在 2:29:54 編出一句 `Subs by www.zeoranger.co.uk`，翻譯照付錢處理，進了 `.zh-Hant.srt`。
- 這句是 Whisper 訓練資料裡字幕組的署名。網路上的字幕檔常在片尾放這種署名，Whisper 就學會「沒人說話時寫一句署名」。社群早就記錄過這類幻聽，例如 `Subtitles by the Amara.org community`、中文的「字幕由 Amara.org 社区提供」「请不吝点赞 订阅 转发 打赏支持明镜与点点栏目」「中文字幕志愿者 李宗盛」。
- 2026-10-12 唯讀複查：
  - 那份 SRT 已經不在了，NAS 上 HP4 資料夾只剩 mkv。
  - 正式 DB 的 `cache_entries` 用 `sqlite3 -readonly` 查 `zeoranger／subs by／amara／for watching`，0 筆，因為逐句快取 30 天就過期。
  - `/mnt/user/data/media` 全部 `.srt` 用 grep 掃同樣的字，也是 0 筆。
  - 所以證據只剩 9/19 立案時的紀錄。這張用單元測試證明。

### 為什麼現在的過濾沒擋到

- 聽寫的每一段都會先過 `filterHallucinationsWith`（呼叫處：`apps/api/internal/ai/whisper.go:366`）。現有規則看的是下面這些：
  - `no_speech_prob`／`avg_logprob` 分數：R1 `silence`
  - 壓縮比：R2 `repetition`
  - 重複句：R2b `repeat_run`／`repeat_spaced`
  - 片尾連續 3 句以上的低信心句子：R3 `tail`
  - 長相：只有音符 `music_only`、只有符號 `symbol_only`、提詞回音 `prompt_echo`
- 這句署名的特性是信心很高、只有一句、句子正常。分數規則看不出來；片尾規則要 3 句連在一起才會丟。所以它一路活下來。

### 規則範圍（Bob 決定，Alexyu 可推翻）

立案時列的例子還有 `Thanks for watching`。這張**不收**這種句子：
- 「Thanks for watching.」是真人會講的台詞，例如「謝謝你幫我顧小孩」。
- 片尾那種「感謝收看、記得訂閱」連續好幾句的情況，已經由 R3 `tail` 處理（`whisper_segments.go:211-219` 的註解寫的就是這個 POC 案例）。
- 規則看的是「誰做的字幕」這種**署名**，以及只會出現在署名裡的**品牌字**。這兩種在電影對白裡幾乎不會出現。

## Story

身為看 AI 字幕的人，
我希望 Whisper 憑空寫出的字幕組署名（例如 `Subs by www.zeoranger.co.uk`）不要出現在字幕上，
這樣片尾不會冒出一句奇怪的網址，也不用花錢翻譯它。

## Acceptance Criteria

1. **新判斷**：`ai` 套件新增 `isCreditLineHallucination(text string) bool`（不匯出，只有聽寫這條路用）。先用 `normalizeForEcho` 正規化（轉小寫、只留字母數字、標點變空白），然後兩條規則任一成立就回 true：
   - **(a) 署名開頭**：正規化後**整句以署名開頭**。開頭要剛好在字的邊界上，也就是整句就是那個詞，或後面接著空白。署名開頭有這些：`subs by`、`subtitles by`、`subtitled by`、`captions by`、`captioning by`、`closed captioning by`、`transcription by`、`transcribed by`、`字幕由`、`中文字幕志愿者`、`中文字幕志願者`。
   - **(b) 品牌字**：正規化後**含有**只在署名裡出現的字：`zeoranger`、`amara org`、`明镜与点点`、`明鏡與點點`。
   - 每一條都要附註解，說明它是哪裡來的（實測過，或是社群記錄的 Whisper 幻聽）。
2. **聽寫過濾丟掉它**：在 `filterHallucinationsWith` 的 R0 新增丟掉原因 `credit_line`。
   - 它是固定字串常數，跟其他原因放在一起，附註解說明來源。
   - 判斷順序：提詞回音 → 只有音符 → 只有符號 → **署名** → 分數規則。
3. **測試**：
   - 判成 true：
     - `Subs by www.zeoranger.co.uk`（HP4 原句）
     - `Subtitles by the Amara.org community`
     - `subs by`、`- Subs by someone`
     - `字幕由Amara.org社区提供`
     - `请不吝点赞 订阅 转发 打赏支持明镜与点点栏目`
     - `中文字幕志愿者 李宗盛`
     - `Captioning by CaptionMax`
   - 判成 false（真對白不能誤殺）：
     - `Thanks for watching.`
     - `Who subs by the river?`
     - `He subtitles books for a living.`
     - `Subsidies by the state are gone.`
     - `字幕呢？`
     - 空字串
   - 過濾測試：把 HP4 原句放在片尾的最後一句，前面是真對白。用 `filterHallucinationsWith(segs, false, "")` 跑，也就是中段那一段的設定、片尾規則關閉。結果要是署名以 `credit_line` 丟掉、真對白全部保留。
   - 再加一條：分數完全正常（`no_speech_prob` 很低、`avg_logprob` 很高）的署名也要丟。這正是現有規則擋不住的情況。
4. **不改片內字幕軌那條路**（`subtitle/sdh_filter.go`）：人工字幕檔裡的署名是另一回事，這張只處理 Whisper 自己編的。
5. **檢查**：下面這些全綠：
   - `go build／vet／test ./...`
   - `~/go/bin/staticcheck ./...`
   - `pnpm run lint:all`
6. **正式機實測：不做（不花錢）**。10 分鐘的 See 片段沒有片尾配樂，跑了也觸發不了這條規則。要整部片才有片尾，但禁止在正式機按整集生成。部署後只要確認 log 正常，等下次真的生成整部片時，看 `dropped_by_reason` 有沒有出現 `credit_line`。

## Tasks / Subtasks

- [x] T1 寫（放在新檔 `credit_line.go`，規則和清單集中一處；常數與 R0 接線在 `whisper_segments.go`） `isCreditLineHallucination` 和兩份清單，加常數 `dropReasonCreditLine = "credit_line"`，接到 R0（AC #1、#2）
- [x] T2 測試（AC #3）：表驅動判斷＋過濾整合測試
- [x] T3 檢查全綠（AC #5）

全部是後端，只改一個檔案，加一條規則和一個常數，不用拆單。

## Dev Notes

### 不要做的事

- 不要收「Thanks for watching」這類一般台詞（理由見「規則範圍」）。
- 不要用相似度比對（例如編輯距離）。整句開頭和品牌字已經涵蓋看過的例子；用模糊比對容易誤殺真的對白。
- 不要動 `IsMusicOnlyText`／`IsSymbolOnlyText`（片內字幕軌也用 `IsMusicOnlyText`）。

### 已知限制

- 有一種退路（plain SRT fallback）完全不過濾：引擎不支援 `verbose_json` 時，Vido 改用一般 SRT，這時所有幻聽規則都不跑。這是原本就有的限制，不在這張範圍。
- 斷點續跑會用「已過濾」的段落快取。30 天內「跑到一半中斷」的片，續跑時拿回來的舊段落仍可能帶署名。這跟 symbol_only 那張的 L1 一樣，接受。

### Time-dependent visual coverage

N/A — 純後端，沒有碰到會讀現在時間的畫面元件。

### References

- 同類前例：`disc-2026-10-asr-symbol-only-cues.md`、`disc-2026-10-asr-music-only-cues.md`。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-12。

### Debug Log References

- 紅：先寫 `credit_line_test.go`，編譯失敗（`dropReasonCreditLine` 未定義）→ 實作後綠。

### Completion Notes List

- `isCreditLineHallucination` 先用 `normalizeForEcho` 正規化，再跑兩條規則：
  - 署名開頭：英文要在字的邊界上，也就是整句就是那個詞，或後面接空白，所以 `Subsidies by…` 不會中。中文沒有空白可以當邊界，直接比開頭（`字幕由Amara…`）。
  - 品牌字：句子裡出現就算。
- R0 順序：提詞回音 → `music_only` → `symbol_only` → `credit_line` → 分數規則。
- 過濾測試把 HP4 原句設成高信心（`no_speech_prob` 0.02、`avg_logprob` -0.1），並關掉片尾規則跑，證明只有新規則會丟它。真對白 `It's not over.`、`He subtitles books for a living.` 都保留。
- 片內字幕軌那條路（`sdh_filter.go`）沒動。
- 檢查全綠：
  - `go build ./...`
  - `go vet ./internal/ai/`
  - `go test ./...`
  - `~/go/bin/staticcheck ./...`
  - `pnpm run lint:all`（0 errors）

### Senior Developer Review (AI) — 2026-10-12

結論：0 High／0 Medium／2 Low，皆接受。

- **L1（接受）**：英文句子如果以 `Subtitles by…` 開頭，不管後面接什麼都會被丟掉，例如「Subtitles by the river」。這種開頭在對白裡實際上不會出現；放寬成「後面要接網址才算」，又會漏掉 Amara 那句。
- **L2（接受）**：斷點續跑的段落快取存的是已過濾的 SRT，跟 symbol_only 那張的 L1 一樣，接受。

### Discovery Triage

- N/A — no out-of-scope work discovered.

### File List

- `apps/api/internal/ai/credit_line.go`（新）
- `apps/api/internal/ai/credit_line_test.go`（新）
- `apps/api/internal/ai/whisper_segments.go`
- `_bmad-output/implementation-artifacts/disc-2026-09-whisper-credit-line-hallucination.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
