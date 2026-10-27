# disc-2026-10-bare-episode-filename-uses-folder-title：檔名只有 `S01E03.mkv` 的集數也收得進來，劇名用資料夾名

Status: done

<!-- SM Bob create-story 2026-10-27；行號為 main `25be6c4d`。Rule 24 lane ③，由 disc-2026-10-gemini-parse-model-retired-again 立案；Alexyu 10/27 選 A 接著做。 -->

**Source:** 2026-10-27 NAS 備份實測（臨時容器、`gemini-3.5-flash-lite`）：`/media/tv/The Lord of the Rings - The Rings of Power/Season 1/S01E01.mkv`～`S01E08.mkv` 只進了 E01、E02、E04、E06。另外 4 集的 log 是 `AI_INVALID_RESPONSE: Cannot parse AI response: missing required field: title` → `SCANNER_UNMATCHED: cannot determine episode number, file not imported`。

## 為什麼會這樣

- 掃描器只把**檔名**交給解析器（`scanner_service.go:282`，`filepath.Base`）。
- regex 的標準格式要求季集前面有劇名（`tvPatternStandard` 的 `title` 至少一個字、`parseStandardFormat` 檢查 `titleRaw != ""`），`S01E03.mkv` 不符合 → 交給 AI。
- AI 也看不到劇名，有時回 `title=S01E03`、有時回空的（空的就被判失敗）——每次掃描看運氣，還花錢。
- 其實掃描器根本不需要檔名裡的劇名：`IngestEpisodeFile`（`media_ingest_service.go:299`）用**劇資料夾**認是哪一部，片名是空的就用資料夾名（`TitleFromSeriesDir`）。缺的只是 regex 願意回「第幾季第幾集、劇名空白」。

## Acceptance Criteria

1. **純季集檔名直接解析**：檔名（去掉開頭的 `[群組]` 標籤後）以 `SxxEyy` 開頭——例如 `S01E03.mkv`、`s01e03.mkv`、`S01E03.Pilot.1080p.mkv`、`S01E01-E02.mkv`——regex 就回成功：影集、第 xx 季、第 yy 集（集數範圍照標準格式處理）、劇名空白，畫質等資訊照舊抽取。**不叫 AI**。
2. **劇名從資料夾來**：這樣的檔案進資料庫時掛在它的劇資料夾那一部（既有規則），新建的影集用資料夾名當片名（既有的 `TitleFromSeriesDir`）。
3. **有劇名的照舊**：`Show.Name.S01E03.mkv` 等既有格式的結果完全不變（標準格式先試）。
4. **一個檔案只算一集**：同一部影集裡，如果有別的集數列也記著同一個檔案（例如以前解析失敗塌成「第 0 集」的那一列——正式機《魔戒》第 1 季就有一列 `episode_number=0` 指著 `S01E08.mkv`），這次收進正確集數後，那一列的檔案路徑清掉（跟「檔案不見了」同一種狀態），不會一個檔出現兩次。只清同一部影集、同一個路徑、不是這次這一列的。
5. **測試**：
   - 解析器：上面四種純季集檔名、帶開頭標籤的、大小寫；有劇名的既有案例不變；`CanParse` 跟 `Parse` 一致。
   - 掃描器／匯入：`Show/Season 1/S01E03.mkv` 進資料庫是第 1 季第 3 集、掛在 `Show` 那一部；同一個檔以前在第 0 集那列 → 那列的路徑被清掉、第 3 集那列有路徑（真的 SQLite）。

## 不要做的事

- 不改 AI 解析、不改「AI 回了季集但沒有劇名」的處理（純季集檔名已經不會走到 AI）。
- 不刪任何一列。
- 不處理放在影集資料夾裡的電影（`Renegade Immortal/Special/…[Movie]…`，另一回事）。

## Tasks

- [x] Task 1 — `TVParser`：純季集格式（`Parse`＋`CanParse`）（AC 1、3）
- [x] Task 2 — 匯入：收進一集後，同影集同路徑的其他列清掉路徑（窄介面＋repository 方法）（AC 4）
- [x] Task 3 — 測試、vet、staticcheck；NAS 備份實測《魔戒》第 1 季 8 集（AC 5）

## Dev Notes

- 解析器：`parser/tv_parser.go` `Parse`（`:38`）依序試各格式；`CanParse`（`:83`）要跟 `Parse` 一致；標準格式在 `parser/patterns.go:55`。
- `ParserService.ParseFilenameWithContext`：TV regex 成功就直接回，不會叫 AI（`parser_service.go:94`）。
- 其他吃解析結果的地方都已處理空劇名：電影比對（`enrichment_service.go:667`，空的就判失敗）、影集比對（`:421`，只在非空時用）、字幕候選的顯示名（`generation_candidates.go:1363`，空的就用原名）。
- 匯入：`UpsertEpisode`（`media_ingest_service.go:275`）用「影集＋季＋集」找既有列；清路徑照 `EpisodeRepository.ClearMissingFile`（`:505`）的欄位（`file_path`、`subtitle_tracks`、`subtitle_tracks_file_sig`）。窄介面斷言照 `episodeRemovalRepo` 的做法，不擴大寬介面和它的 mock。
- 正式機唯讀（10/27）：`episode_number=0` 而且有路徑的只有 3 列——《魔戒》S01E08（塌陷殘留，會被 AC 4 清掉）、《仙逆》劇場版（不是季集檔名，不受影響）、《種地吧》`S01E00`（真的第 0 集，路徑不會跟別列重複，不受影響）。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- `parser/patterns.go`：新增 `tvPatternBareEpisode`（`SxxEyy` 開頭，可接集數範圍、集名、畫質、副檔名）。
- `parser/tv_parser.go`：`Parse` 在標準格式之後試 `parseBareEpisodeFormat`（劇名留空、季集＋範圍、畫質照抽）；`CanParse` 加上同一個樣式。
- `repository.EpisodeRepository.ReleaseFileFromOtherEpisodes`（同影集、同路徑、不是這一列 → 清路徑與字幕軌快取）；`MediaIngestService.UpsertEpisode` 寫入後呼叫（窄介面斷言 `episodeFileReleaser`，失敗只記 log）。
- 測試：`parser/tv_parser_bare_episode_test.go`（6 種純季集檔名、有劇名的照走標準格式、`S01.mkv`／`E03.mkv` 等不算）、`services/bare_episode_ingest_test.go`（真的 SQLite：資料夾當劇名、舊的第 0 集那列讓出檔案而別的檔不動、重掃不重複）。反向驗證：關掉解析 → 4 個測試紅；只關掉讓出檔案 → `TestIngest_FileLeavesTheOldEpisodeZeroRow` 紅。
- **NAS 實測（2026-10-27 14:29，正式資料庫備份＋臨時容器；拿掉所有 AI 金鑰，只留 TMDb，媒體唯讀）**：掃描後《魔戒：力量之戒》第 1 季 E01～E08 **8 集全進**（之前 0 集；換了 Gemini 模型後也只有 4 集），log 沒有任何「delegating to generic AI」；舊的第 0 集那列讓出 `S01E08.mkv`（`file moved to its real episode … released=1`）。掃描的未比對數 6 → 2。剩下兩個：《仙逆》劇場版（放在影集資料夾的電影）、《種地吧》`S01E00`（第 0 集特別篇，掃描器不收集數 0——既有規則，不在這張範圍）。測完容器與資料夾都刪了。
- **正式機（2026-10-27 14:5x，`main-f19cf73`）**：Alexyu 手動掃描後確認《魔戒：力量之戒》第 1 季 8 集都在。
- `go test ./internal/services/ ./internal/parser/ ./internal/repository/ ./internal/handlers/ ./cmd/...` 全過；`go vet`、`staticcheck` 乾淨。

### File List

- `apps/api/internal/parser/patterns.go`
- `apps/api/internal/parser/tv_parser.go`
- `apps/api/internal/parser/tv_parser_bare_episode_test.go`（新）
- `apps/api/internal/repository/episode_repository.go`
- `apps/api/internal/services/media_ingest_service.go`
- `apps/api/internal/services/bare_episode_ingest_test.go`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、`disc-2026-10-gemini-parse-model-retired-again.md`（標 done）
