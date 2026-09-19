# Story disc-2026-09-generation-resume-b-asr-chunk-store：語音辨識到一半被預算擋下，下次只辨識沒做過的段——每段辨識完就存文字，音訊不存

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person whose batch hit the budget ceiling at chunk 9 of 16,
I want the next 下次繼續 to re-extract the audio (free) but send only chunks 9–16 to speech recognition,
so that the $0.48 already paid for chunks 1–8 is not paid again, and a 4K remux on a small NAS never needs 300 MB of audio kept around to make that possible.

## Context

`disc-2026-09-generation-resume-from-checkpoint` 拆出的**第二張**（A 翻譯續跑 `…-a-translation-cache` 同時建單；C 估價扣減依賴 A＋B）。與 A **互不相依**，可先後任一順序出貨。

### 🔴 建單時查到的事（main `ef224134`）

1. **辨識結果只在記憶體。** `transcribeAudio`（`transcription_service.go:1240-1258`）把 16 段的 `filtered`／`unfiltered` 放進兩個陣列，最後 `ai.MergeSRTChunks(srtChunks, chunkSeconds)` 合併（時間軸在**合併時**才位移，所以段落 SRT 是段內時間）；任一段錯誤就 `return`（`:1249`），陣列丟掉。預算是在**每段開始前**由 governor 預檢的（`governor.go:67-72`，`whisper.go:432` `governed(...)`）→ 擋在第 N 段時，1..N-1 段**已付費、已在陣列裡**、然後丟掉。
2. **費用記錄在 `TranscribeDetailed` 裡**（`whisper.go:244-250` → `Budget.RecordASRWithRate`）——**沒送去辨識的段自然記 $0**，本張不用碰預算。
3. **被擋在辨識階段時，DB 狀態**：pipeline 模式留 `extracting`（`process_item.go:579-600` `pauseASRItem` 刻意不動 media row），legacy 模式什麼都不寫（`transcription_service.go:805-813` 只 `failJob`）。兩者 `subtitle_language` 都不是 `zh-Hant` → `FindMissingZhHantSubtitle`（`movie_repository.go:929-938`，key 在 `subtitle_language`）**仍會撿到** → 「下次繼續」（scope=missing）會再跑。✅ 不需要改狀態。
4. **段的身分**：`SplitAudioChunks`（`whisper.go:509-580`）`start = i × chunkSeconds`，`chunkSeconds` 由 `WhisperChunkDuration=600` 與 byte rate 夾出（16 kHz mono 固定 600）；`lang` 由 `WhisperLanguageFromTrack(selectedTrack.Language)`；音軌 index 由 `SelectEnglishTrack`。ASR 端點：`ASRProviderHolder` 的 fingerprint 是 `key|baseURL|model`（`asr_provider_holder.go:96`），**沒有排除金鑰的公開 accessor**；`h.model` 可能是空字串（用 `ai.WhisperModel` 預設）。
5. **來源檔身分已有先例**：`services/route_cache.go:92-128`——`osFileIdentity(path) (size, mtimeUnix)`＋ key `subtitle:route:v{n}:{mediaID}:{size}:{mtime}`，秒級精度、與掃描器同一組（`scanner_service.go:477-478`）。`subtitle/auto_generation.go:257-265` 的 `fileChangedAt`（取 mtime／ctime 較晚者）在 `subtitle`，`services` 不能 import。
6. **暫存放哪**：`cache_entries` 表（`key TEXT PK, value TEXT, type, expires_at`；`Set` 只驗 key 非空與 ttl>0，`cache_repository.go:113-136`；`GetMany` 每 500 key 一句；`Delete(key)` 有、無 prefix 刪除）。**不需要 migration**。每段 SRT 約 6 KB（118 KB／16），一部片 ≈ 100 KB；辨識完成後刪掉，穩態幾乎為零。
7. **⛔ 不存 WAV**：整片 WAV ≈ 300 MB（16 kHz mono s16）＋切段再 300 MB；小 NAS 存不起、`/tmp` 可能是 RAM。續跑時**重抽音訊**（這部片 5 分 20 秒，只花磁碟／CPU，不花錢），跳過已付費的段。
8. **`transcribeOne`**（`:1262-1278`）回 `(filtered, unfiltered)`，`guardAgainstEmptyTranscript`（`:1294`）要整片的 unfiltered 當退路——所以每段要**兩個字串都存**。
9. **測試縫**：`fakeCacheRepo`（`route_cache_test.go:415-447`，`services` 套件內）；上一張留下的 `installFakeMediaTools`／`writeHeaderOnlyWAV`／`slowASR`（`transcription_run_budget_test.go`）；`ai/whisper_test.go:501-507` 的 `execCommandContext` 縫。

### ⚖️ 設計裁定（SM，2026-09-18）

1. **存段的文字，不存音訊。** 每段辨識成功後立刻 `Set`：key ＝ `asrchunk:v1:` ＋ sha256(mediaID ｜ 檔案 size ｜ mtime ｜ 音軌 index ｜ lang ｜ start ｜ chunkSeconds ｜ ASR 端點指紋)，value ＝ JSON `{"filtered":…,"unfiltered":…}`，type `asr_chunk`，TTL 30 天（同 segment cache）。
2. **端點指紋排除金鑰**：`ASRProviderHolder.EndpointFingerprint() string` ＝ `baseURL|model`（model 空時用 `ai.WhisperModel`）。換供應商／模型 → 全部 miss（刻意）。
3. **來源檔身分沿用 `osFileIdentity`**（size＋mtime 秒級），與 route cache 同一套；不引入 ctime（`fileChangedAt` 在 `subtitle`，且 route cache 已接受同一精度）。
4. **一份 manifest**：key ＝ `asrchunk:v1:manifest:` ＋ sha256(mediaID ｜ size ｜ mtime ｜ 端點指紋)，value ＝ JSON `{"track":1,"lang":"en","chunk_seconds":600,"done":[0,600,…]}`，同 TTL；每段寫入後更新。用途：(a) **story C 的估價**只要一次 `Get` 就知道已有幾段；(b) log 一行「asr chunk cache hits=N/16」。
5. **流程**：`SplitAudioChunks` 之後先算全部 key → **一次 `GetMany`** → 逐段：命中 → 用存的兩個字串、不呼叫 ASR（$0 自然成立）；miss → `transcribeOne` → `Set`＋manifest。任何錯誤照舊 `return`（已寫的段留著）。
6. **成功寫出 `.en.srt` 後刪掉該片的段與 manifest**（16＋1 次 `Delete`，best-effort、失敗只 Warn；TTL 是第二道清理）。
7. **快取失敗只 Warn，絕不讓付費 run 失敗**；讀到壞 JSON 或空字串 → 當 miss；命中的 filtered 為空但 unfiltered 非空 → 照 `guardAgainstEmptyTranscript` 既有語意處理。
8. **不改 DB 狀態、不改 SSE／API 形狀**；**不動預算／governor**。
9. **批次先跑「有進度」的片**（⚖️ Alexyu 2026-09-18 追問後補）：預算上限是**每批一個信封**（`generation_batch.go:423-429`，整批共用一個 `ai.Budget`），所以一批 1,000 部、上限 $1 時，第 1 部花完 $1 整批就停，後面 999 部沒開始；但「下次繼續」（scope=missing）的順序是**片名字母序**（`movie_repository.go:939`），選片（scope=selected）則照使用者送來的順序——若每次選的片不同，就會**一次留下一部半成品**，累積成幾百部半成品，而且每部都永遠做不完（今天的行為：每批重付 $1 從頭來，$3.65 的片永遠到不了終點）。修法：`collectItems` 之後**穩定排序**，有續跑進度的片排最前面（辨識段 manifest 存在，或 `subtitle_status = untranslated`），其餘維持原順序。這樣每一批都先把上一批沒做完的做完，半成品在任何時刻最多一部；就算使用者硬是每次選不同的片，暫存也只是 TTL 30 天內的 ≤ 0.6 MB／部。

## Acceptance Criteria

1. **`ASRProviderHolder.EndpointFingerprint()`**：回 `baseURL + "|" + model`，model 空 → `ai.WhisperModel`；**測試斷言不含金鑰**（用一個看得出來的假 key）。

2. **`services.ASRChunkStore`**：介面 `GetMany`／`Set`／`Delete`；`NewASRChunkStore(repository.CacheRepositoryInterface)`（type `asr_chunk`）；純函式 `asrChunkKey(id ASRChunkIdentity) string` 與 `asrManifestKey(...)`，identity 結構 `{MediaID, FileSize, FileMTime, TrackIndex, Lang, Start, ChunkSeconds, Endpoint}`；表格測試：任一欄位變 → key 變；同輸入 → 同 key；key 不含任何原始欄位明文（只有前綴＋hash）。`TranscriptionService.SetASRChunkStore(...)`，`main.go` 用 `repos.Cache` 接上（legacy／pipeline 皆是）。

3. **`transcribeAudio` 走快取**（AC 裁定 5）：
   - 需要的身分資料由 `runPipeline` 傳入（mediaID、`osFileIdentity(filePath)`、`selectedTrack.Index`、lang、端點指紋）；`osFileIdentity` 失敗 → **不用快取**（Warn），照舊全部辨識。
   - 開始前一次 `GetMany`；命中段不呼叫 ASR；miss 段辨識後立刻 `Set`＋更新 manifest；log 一行 `asr chunk cache` 含 `hits`／`total`／`media_id`。
   - 不需要切段的短檔（`NeedsChunking=false`）**也走同一套**（一段、start=0）——短片被擋的機率低但語意一致；或明確裁定跳過並寫註解（dev 二擇一，記在 Completion Notes）。
   - 成功寫出 `.en.srt` 後刪段＋manifest（裁定 6）。

4. **可觀察行為（測試）**：
   - 16 段、fake ASR 第 9 次呼叫回 `ai.ErrBudgetExceeded` → store 內 8 段＋manifest `done` 長度 8、run 失敗（既有語意）；再跑一次（同檔、同端點）→ **ASR 只被呼叫 8 次**、合併後 SRT 與一次跑完位元相同（時間軸位移正確）、log `hits=8/16`；成功後 store 為空。
   - 檔案 mtime 變了 → 0 命中；端點指紋變了 → 0 命中；lang 變了 → 0 命中。
   - `GetMany` 回錯 → 16 次呼叫、Warn、成功；`Set` 回錯 → 成功、Warn；value 是壞 JSON → 當 miss。
   - 命中段的 `unfiltered` 進入 `guardAgainstEmptyTranscript`（fake 讓 filtered 全空、unfiltered 有字 → 輸出用 unfiltered）。
   - `NeedsChunking=false` 那條路的行為與裁定一致。
   - 規模：`fakeCacheRepo` 塞 10,000 個 `asr_chunk` key（≈ 600 部片同時卡住的極端）→ 一部片的 `GetMany` 仍只帶 16 個 key（斷言 `manyCalls` 的 key 數）。

5. **真機驗證（done 的門檻）**：隔離容器食譜，同一部 157 分鐘的片：`AI_RUN_BUDGET_USD=0.30` 跑（5 段 × $0.06 後被擋，約 8 分鐘）→ log 有 `Set`＋manifest → 改 5.0 再跑 → log `hits=5/16`、`asr_calls=11`、`spent_usd` ≈ $0.66＋翻譯 ≈ $3.4 總計；成功後 `cache_entries` 內該片的 `asr_chunk` 列為 0（`sqlite3` 查 type 計數）。正式 Vido 不動、跑完清理。

6. **批次先跑有進度的片（裁定 9）。** `GenerationBatchProcessor` 加一個窄的 port（Rule 11）`ResumeProgressFinder{ HasResumeProgress(ctx, mediaType, mediaID string) bool }`，`TranscriptionService` 實作：manifest 存在（一次 `Get`）**或** row `subtitle_status == untranslated`（既有 reader）。`collectItems` 回來後 `sort.SliceStable`，有進度的在前、其餘原序；nil port ＝ 不排序（既有行為）。**測試**：3 部片、第 2 部有 manifest → 佇列順序 2,1,3；第 3 部 `untranslated` → 3,1,2（穩定）；port 回錯／nil → 原序；port 的查詢次數 ＝ 片數（不准 N+1 以外的額外查詢，1,000 部 ＝ 1,000 次 `Get`，可接受，記錄實測毫秒）。202／status 的 `items[]` 順序跟著變（SSE `current_index` 語意不變）。

7. **不做的事**：不存 WAV（裁定 7）；不改 DB `subtitle_status`；不改 SSE／202／status；不做估價扣減（story C 用 manifest）；不動 governor／budget；不動 `subtitle` 套件；不加 migration。

8. **CI 全綠**：`pnpm nx test api`、`pnpm nx test web`、`pnpm run lint:all`、`pnpm run format:check`。⛔ 測試不 `run_in_background`；跑完 `pnpm run test:cleanup`。

## Tasks / Subtasks

- [x] **Task 1 — 端點指紋（AC: #1）**：紅測試（不含金鑰、空 model 用預設）→ 實作
- [x] **Task 2 — store、key、manifest（AC: #2）**：紅測試（key 表格、adapter type／TTL／miss-by-absence、manifest round-trip）→ 實作＋`main.go` 接線
- [x] **Task 3 — `transcribeAudio` 走快取＋成功後清理（AC: #3, #4）**：紅測試（8/16 被擋→再跑呼叫 8 次、位元相同、身分變則 miss、讀寫失敗、壞 JSON、unfiltered 退路、短檔、規模）→ 實作
- [x] **Task 4 — 批次先跑有進度的片（AC: #6）**：紅測試（順序、穩定、nil port、查詢次數）→ `ResumeProgressFinder`＋`sort.SliceStable`＋`main.go` 接線
- [x] **Task 5 — 收尾（AC: #7, #8）**：全套閘門；mutation check（拿掉 `GetMany`／拿掉 `Set`／key 少一欄／成功後不刪——各至少一紅）
- [x] **Task 6 — 真機驗證（AC: #5）**

## Dev Notes

### 這張的重點

- **不存音訊**是整張的核心取捨：續跑多花 5 分鐘重抽，換來小 NAS 不用放 600 MB 暫存。
- 段 SRT 是**段內時間**（合併時才位移）——存的就是 `transcribeOne` 回來的原樣，命中後照舊丟進 `MergeSRTChunks`。
- 預算完全不用碰：命中段沒有 `TranscribeDetailed` 呼叫，就沒有記帳。

### 上游契約（Rule 20）

- 無 contract 變動：SSE／API 形狀不變；`cache_entries` 表新增一個 type 是加寬；`EndpointFingerprint` 是新 accessor。

### 已知陷阱

- `fingerprint` 含金鑰（`asr_provider_holder.go:96`）——**絕不能**拿它當 key 材料；只用 `baseURL|model`。
- `SplitAudioChunks` 的 `chunkSeconds` 可能小於 600（高 byte rate）——key 要用實際的 `chunkSeconds`，不要寫死。
- `GetMany` 的 miss ＝ 不在 map 裡；過期由讀取端過濾（`cache_repository.go:80`）。
- 成功後 `Delete` 是 best-effort；TTL 30 天是保底。`ClearCacheByType("metadata")` 會整表清（既有）。
- mtime 秒級：同一秒內換檔會撞 key——route cache 已接受此精度；記錄即可。

### Source tree

```
apps/api/internal/services/asr_provider_holder.go(+test)   ← Task 1
apps/api/internal/services/asr_chunk_store.go(+test)        ← Task 2（新）
apps/api/internal/services/transcription_service.go(+test)  ← Task 3
apps/api/cmd/api/main.go                                    ← Task 2
```

### Cross-Stack Split Check

後端 task 6、前端 0 → 不觸發。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.**

### References

- [Source: `apps/api/internal/services/transcription_service.go:805-818, 1240-1300`、`asr_provider_holder.go:36-48, 76-78, 96-107`、`route_cache.go:78-128`、`route_cache_test.go:415-462`、`transcription_run_budget_test.go`]
- [Source: `apps/api/internal/ai/whisper.go:32-34, 244-250, 432, 509-580`、`ai/budget.go:154-159, 248-272`、`ai/governor.go:67-72`]
- [Source: `apps/api/internal/repository/cache_repository.go:35-136, 150`、`interfaces.go:349-385`]
- [Source: `apps/api/internal/repository/movie_repository.go:929-938`、`episode_repository.go:153-162`、`subtitle/process_item.go:579-600`]
- [Source: project-context.md#Rule 11 / #Rule 13 / #Rule 19 / #Rule 20]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Dev Amelia，2026-09-18）

### Debug Log References

- 紅→綠：`go test ./internal/services/ -run 'TestASRChunk|TestASRManifest|TestASRProviderHolder_EndpointFingerprint|TestTranscribeAudio_|TestRunTranscription_StoresChunks|TestHasResumeProgress|TestGenerationBatch_(FilmsWithProgress|OrderIsUntouched|ResumeLookup)'`
- Mutation check（六刀全部被殺）：拿掉 `GetMany` → 6 紅；拿掉 `Set` → 4 紅；key 少 mtime → 2 紅；key 少端點 → 2 紅；成功後不刪 → 1 紅（`TestRunTranscription_StoresChunksAndClearsThemOnSuccess`）；批次不排序 → 2 紅。
- 閘門：`pnpm nx test api` 綠；`pnpm nx test web` 270 files 綠；`pnpm run lint:all` 0 errors（127/129 warnings 既有浮動）；Prettier 全過；`pnpm run test:cleanup` 已跑。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-18）。
- **短檔（`NeedsChunking=false`）走同一套**（AC #3 二擇一選前者）：整檔當一段、`start=0`、`chunk_seconds=0`。`transcribeAudio` 現在只有一條迴圈，短檔與長檔差別只在有沒有切段——語意一致、少一條分支。
- **一次讀取含 manifest**：`GetMany` 帶 N 段 key ＋ 1 個 manifest key（157 分鐘片 ＝ 17 個 key），manifest 在記憶體更新後每段寫一次；沒有每段的 read-modify-write 往返（`TestTranscribeAudio_LookupIsBoundedByTheFilmNotTheTable` 斷言 `manyCalls=1`、key 數 ＝ 段數＋1）。舊 manifest 的 track／lang／chunk_seconds 與本次不同時視為不存在（重新開始一份），段 key 本身仍各自獨立命中。
- **成功後清理用「本次 key ∪ manifest 列出的 key ∪ manifest」**（去重）：manifest 寫入曾失敗也清得到本次寫的段；失敗只 Warn，TTL 30 天保底。
- **`HasResumeProgress(ctx, mediaType, mediaID, filePath)`** 比 AC #6 多一個 `filePath` 參數：manifest key 需要檔案身分（size＋mtime），而 `GenerationBatchItem` 已帶 `filePath`，不必再查 DB。port 回 `bool`（查詢失敗 ＝ 沒進度），批次排序失敗只影響順序，不影響批次。
- **`transcribeAudio` 多一個 `*asrChunkScope` 參數**（nil ＝ 舊行為）：`runPipeline` 由 `mediaID`＋`osFileIdentity(filePath)`＋`selectedTrack.Index`＋lang＋`asrEndpointOf(s.asr)` 建 scope；`osFileIdentity` 失敗 → Warn、scope=nil、全部辨識。
- **端點指紋退路**：ASR provider 沒實作 `EndpointFingerprint()`（測試 fake、直接注入的 `*ai.WhisperClient`）→ 用 `"|" + ai.WhisperModel`，與 holder 的 hosted 預設同值。
- 沒動：DB `subtitle_status`、SSE／202／status 形狀、governor／budget、`subtitle` 套件、migration（AC #7）。
- ✅ **Task 6 真機驗證（2026-09-18 23:12 → 2026-09-19 13:19，NAS 隔離容器 `vido-resume-test`：正式 image ＋ 交叉編譯二進位覆蓋 `/usr/local/bin/api`，DB 副本、scratch symlink；同一部片 `ceb9fec6`，157 min／66.8 GB；正式 Vido 全程未動、跑完 `docker rm -f`＋`rm -rf`、正式片庫資料夾 0 個 srt）**：

  | 回合 | 設定 | 結果 |
  | --- | --- | --- |
  | 1 | `AI_RUN_BUDGET_USD=0.30` | 抽音訊 9:20 → `hits=0 total=16` → 第 1–5 段各 $0.06（`run_spent_usd=0.3`）→ 第 6 段 `AI budget exhausted — skipping call` → `transcription failed … AI_BUDGET_EXCEEDED`。**`cache_entries` 6 列**：5 段（15–24 KB／段，`{"filtered":…,"unfiltered":…}`）＋ manifest `{"track":1,"lang":"en","chunk_seconds":600,"done":[0,600,1200,1800,2400]}`。 |
  | 2a | `AI_RUN_BUDGET_USD=5.0` | `hits=5 total=16`、從第 6 段開始 → OpenAI 回 **429 `insufficient_quota`**（帳號沒額度）→ run 失敗、$0、**6 列原封不動**（失敗不清暫存 ✅）。 |
  | 2b | 同上（儲值後） | 家裡網路斷線：`lookup api.openai.com … i/o timeout` → run 失敗、$0、6 列仍在。 |
  | 2c | 同上（網路恢復） | **`hits=5 total=16`、`asr_calls=11`、`asr_seconds=6425`**（＝11 × 600 − 尾段）、`spent_usd=3.13`（辨識 $0.66 ＋ 翻譯 $2.47）；`.en.srt` 寫出後 **`asr_chunk` 列 ＝ 0**（翻譯還在跑時查已是 0）；`.en.srt`／`.zh-Hant.srt` 各 2,014 句；`transcription complete` `duration≈25m38s`。 |

  結論：AC #5 的三個數字（`hits=5/16`、`asr_calls=11`、成功後列數 0）全部命中；兩次意外中斷（沒額度、斷網）額外證明「失敗不清暫存」。
- 真機順帶看到（非本單）：2c 翻譯階段 Claude 有一批（20–30）逾時 3 次 → 31 句保留英文 → row 記 `untranslated`（既有的 partial 語意，`backlog-translate-budget-partial-progress`／story A 範圍）。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES**
- **① expand-scope-in-place**：`ASRProviderHolder` 沒有排除金鑰的端點 accessor → AC #1；manifest（為 story C 準備）→ AC #2。
- **② spawn-blocking-story**：無。
- **③ backlog-with-carry-forward-link**：`disc-2026-09-generation-resume-c-estimate-deduction`（sprint 條目）；`disc-2026-09-generation-resume-a-translation-cache`（同時建單）。
- **③（dev 真機驗證時新增）**：`disc-2026-09-asr-quota-429-treated-as-transient`——OpenAI 429 `insufficient_quota`（沒額度）被當成暫時性錯誤重試 3 次，還被 `DetailedTranscriber` 誤判成「引擎不支援 verbose_json」而關掉幻覺過濾；`disc-2026-09-asr-base-url-assumed-free`——`ASR_BASE_URL` 有設就記 $0，接 Groq（$0.04/hr，OpenAI 相容）這類**付費**相容端點時預算完全失效；順帶：Groq 換過去每部片辨識 $0.94 → $0.10。
- Reference: `project-context.md` Rule 24

### File List

- `apps/api/internal/services/asr_chunk_store.go`（新）— identity／key／manifest／port／adapter／端點退路
- `apps/api/internal/services/asr_chunk_store_test.go`（新）
- `apps/api/internal/services/asr_provider_holder.go` — `EndpointFingerprint()`
- `apps/api/internal/services/asr_provider_holder_test.go` — 兩條指紋測試
- `apps/api/internal/services/transcription_service.go` — `chunkStore` 欄位＋`SetASRChunkStore`、`asrChunkScope`／`chunkScope`、`transcribeAudio` 走快取、`loadStoredChunks`／`storeChunk`／`clearChunkCache`、`HasResumeProgress`、`runPipeline` 接 scope＋成功後清理
- `apps/api/internal/services/transcription_chunk_resume_test.go`（新）— AC #3／#4 行為測試
- `apps/api/internal/services/transcription_hallucination_test.go` — 既有呼叫補 `nil` scope
- `apps/api/internal/services/generation_batch.go` — `ResumeProgressFinder` port、`SetResumeProgressFinder`、`resumableFirst`（`sort.SliceStable`）
- `apps/api/internal/services/generation_batch_resume_order_test.go`（新）— AC #6
- `apps/api/internal/services/route_cache_test.go` — `fakeCacheRepo` 加失敗注入與寫／刪帳本
- `apps/api/cmd/api/main.go` — `SetASRChunkStore(NewASRChunkStore(repos.Cache))`、`SetResumeProgressFinder(transcriptionService)`

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-19 | 🚧 **REVIEW**（Dev Amelia）。Task 6 真機驗證通過：$0.30 回合存 5 段＋manifest；$5 回合 `hits=5/16`、`asr_calls=11`、$3.13、成功後列數 0；中途兩次外部中斷（OpenAI 沒額度、家裡斷網）暫存都沒掉。NAS 清理完畢、正式 Vido 未動。兩條新 disc 立案（429 沒額度被當暫時性錯誤；`ASR_BASE_URL` 一律記 $0）。 |
| 2026-09-18 | Dev Amelia：Task 1–5 完成（紅測試→實作→mutation 六刀全殺→全套閘門綠）。Task 6 真機驗證待跑。 |
| 2026-09-18 | ⚖️ Alexyu 追問「幾百幾千部卡在半路」：查證預算是每批一個信封、missing 順序是字母序、selected 照使用者順序——每次換選片就會累積半成品且永遠做不完。補裁定 9／AC #6：批次先跑有進度的片。 |
| 2026-09-18 | Story 建立（SM Bob, create-story；main `ef224134`）。拆三張的第二張。裁定：存段的文字不存音訊（續跑重抽 5 分鐘換小 NAS 不用放 600 MB）；key 排除金鑰；沿用 `osFileIdentity`；一份 manifest 給估價用；成功後刪；快取失敗只 Warn；不改 DB 狀態、不動預算。 |
