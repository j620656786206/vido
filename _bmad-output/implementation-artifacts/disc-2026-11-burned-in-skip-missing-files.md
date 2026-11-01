# Story disc-2026-11-burned-in-skip-missing-files：燒字幕檢查跳過已經不在的檔案

Status: done

**Source:** ⚖️ Alexyu 2026-11-02 選「1 先修仙逆」。-b2 上線後第一次片庫總檢查（2026-11-02 07:05，47 季、45 季有結果、US$0.11）。

## 問題

《仙逆》S1（151 集）沒判斷出來：這季第一個「沒有字幕軌」的集數指向 `/media/tv/Renegade Immortal/Special/[GM-Team]…劇場版….mp4`，檔案已經不在 NAS 上。抽它截不到畫面（"0 of 0"），整季就放棄；而且記了 `burned_in_checked_at`，7 天內不會再試。

## 做法

- `BurnedInSubtitleService` 挑集數前先過濾掉檔案不在的（`os.Stat` 是一般檔案才算，測試可注入）。第一集和第二集複查都從過濾後的清單挑。全部都不在 → `no_episode_to_check`（不寫時間戳）。
- migration 050（一次性資料修正）：清掉「有時間戳、沒有結果」的季的 `burned_in_checked_at`，下一輪馬上重試；有結果的季不動。

## Acceptance Criteria

1. [x] 第一個沒字幕軌的集數檔案不在 → 改抽下一個在的。
2. [x] 全部不在 → `no_episode_to_check`，不呼叫模型、不寫時間戳。
3. [x] migration 050 只清「沒結果但有時間戳」的列。
4. [x] `go test`（services、migrations）、`go vet`、`staticcheck`、`gofmt` 乾淨。

## Dev Agent Record

- `burned_in_subtitle_service.go`：`fileExists` 欄位（預設 `fileOnDisk`）＋`onDisk` 過濾；測試 `TestCheckSeason_SkipsEpisodesWhoseFileIsGone`、`TestCheckSeason_EveryFileGoneIsNoEpisodeToCheck`、`TestFileOnDisk`；既有測試的假路徑注入「都在」。
- `050_retry_undecided_burned_in_checks.go`＋測試。
- 《金色》（4K 杜比視界截不出畫面）重試後一樣會判斷不了，7 天後再試；不在本張範圍。

## Change Log

- 2026-11-02：建立＋實作。
