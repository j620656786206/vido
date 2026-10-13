# Disc：批次選片清單的價錢，扣掉已經聽寫過的部分（c2：後端）

Status: done

**Source:** 2026-10-12 從 `disc-2026-09-generation-resume-c-estimate-deduction` 拆出。2026-10-13 Alexyu 從 backlog 選 1。

這張拆成兩半：

- **c2（這張）**：後端。選片清單每一部的價錢、總計、每個模型的總計，都扣掉已經聽寫存起來的分鐘；每列多一個欄位 `resumed_asr_minutes`。
- **c2b**：`disc-2026-10-consent-list-resume-note`。清單上那一行說明，要等設計稿。

c2 可以單獨部署：價錢只會變低，前端不用改也能用。

## 查到的事（2026-10-13，Bob，main `0b39f799`）

- 選片清單的價錢來自 `GenerationCandidateService.Analyze`（`apps/api/internal/services/generation_candidates.go:739`）：
  - 每部片 `estimateUSD(route, minutes, asrRate, model)`。
  - 語音辨識路線 = 分鐘 × (聽寫費率＋翻譯費率)。
  - 完全不看之前做到哪。
- 單部估價（c，#728 一帶）已經會扣：`TranscriptionService.resumedASRMinutes`（`transcription_resume_credit.go`）。
  - 讀 manifest，再確認列出的段還在，每部 2 次查詢。
- 剛上線的「沒錢就整批停」（#759、#760）會讓更多片停在一半。儲值後按「下次繼續」會回到這張清單，現在看到的是全價。
- 一個 1,200 集的片庫，如果每部各查 2 次，就是 2,400 次查詢。所以這張要**整批查**：
  - 所有 manifest 一次 `GetMany`。
  - 所有段一次 `GetMany`。
  - `GetMany` 本身每 500 個 key 切一批（`cache_repository.go:61`）。
- 只算語音辨識路線的片。抽取路線的片不讀磁碟（`planRouteCache` 對有存軌道資料的片完全不 stat），這張也不能讓它們多 stat。

## Story

身為被預算或帳號額度擋下過的使用者，
我希望批次選片清單上的價錢只算還沒做的部分，
這樣我按「下次繼續」看到的數字，接近實際會花的錢。

## Acceptance Criteria

1. **整批查**：`TranscriptionService` 新增整批版本 `resumedASRMinutesMany`，一次回答很多部片。
   - 所有 manifest 一次 `GetMany`，所有段一次 `GetMany`。
   - 單部版本 `resumedASRMinutes` 改成呼叫整批版本，算法只有一份。
   - 規則跟 c 一樣：段要還在才算；`chunk_seconds` 為 0 時，有存就等於整部；上限是片長。讀取失敗就當 0。
2. **只算語音辨識路線**：`Analyze` 先把每部片分好路線，再一次查語音辨識路線那些片的已聽寫分鐘。抽取路線的片不會多讀磁碟或快取。
3. **價錢**：語音辨識路線的片 = (片長 − 已聽寫分鐘) × 聽寫費率 ＋ 片長 × 翻譯費率。翻譯那半不扣（9/18 裁定：清單不算翻譯命中）。
   - 每列 `estimated_usd`、`summary.estimated_total_usd`、每個模型的 `per_candidate` 和 `total_usd` 都用扣過的價錢。列加起來仍然等於總計。
   - 預估時間（`estimated_minutes_by_model`）不動。
4. **新欄位**：`GenerationCandidate` 加 `resumed_asr_minutes`（float，沒有就省略）。只加不改。
5. **沒接上就跟現在一樣**：沒有接上聽寫快取時（測試、舊設定），價錢和現在完全相同。
6. **測試**：
   - 一部語音辨識的片有 2 段存著（每段 2 分鐘）：價錢少 4 分鐘的聽寫費；`resumed_asr_minutes` = 4；總計和每個模型的總計都對得上。
   - 抽取路線的片就算有 manifest，也不扣。
   - 查詢次數：3 部語音辨識的片，manifest 只查 1 次、段只查 1 次。
   - 快取讀取失敗：報全價，清單照樣出來。
   - 單部估價原本的測試保持綠燈。
7. **檢查**：`go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all` 全綠。

## Tasks / Subtasks

- [x] T1 `transcription_resume_credit.go`：整批版本，單部改呼叫它（AC #1）
- [x] T2 `generation_candidates.go`：先分路線、再整批查、再算錢；新欄位；`SetResumeCredit` 接線（AC #2–#5）
- [x] T3 `cmd/api/main.go` 接上
- [x] T4 測試＋檢查（AC #6、#7）

## Dev Notes

- 前端不動，那是 c2b。
- 估價時不切音檔，相信切點會重複（同檔、同靜音偵測）。跟 c 一樣，切點變了最多就是付全價。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Debug Log References

- 突變測試：把 `Analyze` 的 `estimateUSDWithCredit` 換回 `estimateUSD`，`TestAnalyze_StoredTranscriptionComesOffTheASRQuote` 變紅。還原後全綠。

### Completion Notes List

- `resumedASRMinutesMany`：先 stat 每部片、算 manifest key，一次 `GetMany`；解出 manifest 後把所有段的 key 合起來，再一次 `GetMany`。單部的 `resumedASRMinutes` 改成呼叫它，c 原本的測試全部綠。
- `Analyze` 拆成兩輪：
  - 第一輪分路線，進度回報照舊。
  - 中間只對語音辨識的片一次查已聽寫分鐘。
  - 第二輪算錢，每列、總計、每個模型的總計都用 `estimateUSDWithCredit`。
- `estimateUSDWithCredit`：只扣聽寫那半；抽取路線或 0 分鐘時，結果跟 `estimateUSD` 完全一樣。
- 新欄位 `resumed_asr_minutes`（omitempty）。`SetResumeCredit(*TranscriptionService)` 在 `main.go` 接上；傳 nil 不會變成「有值的 nil 介面」。
- 檢查全綠：`go build ./...`、`go vet`、`go test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all`（0 errors）、`format:check`。

### Senior Developer Review (AI) — 2026-10-13

結論：0 High／0 Medium／2 Low，皆接受。

- **L1（接受）**：有存軌道資料的語音辨識片，原本不 stat 檔案，現在會多 stat 一次。這些片本來就要被處理，多一次 stat 不會多叫醒硬碟太多；抽取路線完全不受影響。
- **L2（接受）**：pipeline 模式的批次聽寫跟單部用同一個 chunk store 和 manifest key，所以這裡沿用 c 的假設。切點變了最多就是付全價。

### File List

- `apps/api/internal/services/transcription_resume_credit.go`
- `apps/api/internal/services/generation_candidates.go`
- `apps/api/internal/services/generation_candidates_resume_test.go`（新）
- `apps/api/cmd/api/main.go`
- `_bmad-output/implementation-artifacts/disc-2026-09-generation-resume-c2-consent-list-deduction.md`
- `_bmad-output/implementation-artifacts/disc-2026-10-consent-list-resume-note.md`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-consent-list-resume-note-pen-prompt.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
