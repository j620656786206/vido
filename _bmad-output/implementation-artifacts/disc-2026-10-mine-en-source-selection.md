# Disc：官方字幕學習的英文那一側拿錯來源——拿到 Vido 自己聽寫的 .en.srt、或拿到只有幾句的「強制字幕」軌

Status: done

**Source:** 2026-10-07 正式機重學 See 的逐集報告：S01E02 用了旁邊的 `…en.srt`（Vido 10/5 聽寫產出，人名全錯）只對上 287 段（嵌入的官方英文軌可對 568 段）；S01E07／S02E04／S02E05 選到 stream 2「English (Forced)」只有 1～3 句，對上 0～1 段。結果 Jerlamarel 的第 1 季票數掉到比第 2 季少，學成「傑拉馬瑞」。

## Story

身為 Vido 的使用者，我希望官方字幕學習用的英文是片子自帶的官方英文軌，不是 Vido 自己聽寫出來的檔、也不是只有幾句的強制字幕軌。

## 設計

- `Classify`：英文軌**跳過 Forced**；完整軌排在 SDH 軌前面。
- `mineEpisode`：**一律 ffprobe**（原本旁邊有 `.en.srt` 就不探，所以嵌入軌從不被考慮）。
- 英文側改成 `loadEnglish`：先嵌入軌（依序試，少於 20 句的跳過），沒有可用的嵌入軌才用 sidecar。中文側不變（官方 sidecar 優先）。

## Acceptance Criteria

1. 有 Vido `.en.srt`＋嵌入英文軌的片 → 用嵌入軌（`en_source=embedded stream N`），強制軌連抽都不抽。
2. 沒有嵌入英文軌 → 仍用 sidecar。
3. `go test ./...`、staticcheck、lint:all 全綠。
4. **實測：** 正式機重學 See：S01E02 `segs` 287 → ≈568、S01E07／S02E04／S02E05 不再是 0～1；`Jerlamarel` = 「謝拉馬威」。

## Dev Agent Record

- **AC #4 ✅（2026-10-07）：** 正式機逐集報告 15 集全用嵌入英文軌（S01E02 287 → 568 段；S01E07／S02E04／S02E05 0～1 → 483／470／643 段）；`terms_found=61`、Jerlamarel → 謝拉馬威。
- **Adversarial CR（2026-10-07，fresh agent）2M/2L＋留意，處理如下：**
  - M1 每個候選軌各跑一次全檔 ffmpeg → 改成一次 `Extract` 抽全部候選再依序挑；miner 的 extractor 沒共用字幕生成的 IO 閘門（既有）→ 記著，未處理。
  - M2 probe 失敗被靜默吞掉、會原封不動重演這個 bug → Warn log＋報告的 `en_source` 標 `(probe failed; embedded tracks unknown)`；加測試。
  - L1 `Classify` 只看 disposition 不看 Title（「English (Forced)」沒旗標的 release 仍會排第一）→ 匯出 router 的 `TrackIsForced`／`TrackIsSDH`（含 title 規則）給 miner 用；加 title-only 測試。
  - L2 跳過原因只留最後一個、extractor nil 訊息不一致 → 收集全部原因 `; ` 相連；加「只有短軌、沒 sidecar」測試。
  - 留意：以前「只有字幕組 zh、有 en sidecar」的集數不 probe，現在會 probe、片內若有嵌入 chi 軌（常是簡體）會拿來挖；與單一 show 端點既有行為一致，但掃庫會自動對全庫做——記在 `disc-2026-10-asr-harvest-pollutes-glossary` 第二半一起想。
- Claude Fable 5.1。🔗 AC Drift: FOUND — sub-7-5b「sidecar 優先」對英文側改為「嵌入優先」。📎 Contract Stamps: NONE。🎭 A11y: N/A。🎨 UX: SKIPPED。

### File List

- `apps/api/internal/subtitle/mine/sources.go`、`mine_test.go`、`apps/api/internal/subtitle/miner/miner.go`、`miner_test.go`、`apps/api/internal/subtitle/track_choice.go`（匯出 TrackIsForced／TrackIsSDH）（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | 正式機驗證通過 → done。 |
| 2026-10-07 | CR 2M/2L：一次抽全部候選、probe 失敗記 log＋標註、title 判斷 Forced／SDH、原因全列。 |
| 2026-10-07 | create-story＋dev 同日；全綠，狀態 review，待正式機重學驗證。 |
