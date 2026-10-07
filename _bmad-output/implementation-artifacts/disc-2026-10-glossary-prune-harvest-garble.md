# Disc：有官方字幕可對照時，Vido 自己聽寫猜出來、官方英文裡根本沒有這個詞的名詞表列會被清掉

Status: done

**Source:** `disc-2026-10-asr-harvest-pollutes-glossary` 的第二半。See 名詞表 29 條 `subtitle`（聽寫收成）列，用 15 集官方英文字幕（8,365 句）一查：**20 條在官方英文裡一次都沒出現**——Algini（Alkenny）、Aniwa（Haniwa）、Chalamarel／Chola Morel／Durla Morel／Jerla Morel／Jorna Morel／Trilla Morel／Gilles Morel／Morel（Jerlamarel）、Cofoun（Kofun）、Mara（Maghra）、Paeon（Payan）、Papa Voss、Timat Dijon（Tamacti Jun）、Sungrave、gatherpacks、pseudopacks、Jeremiah、Jill。這些都會進翻譯提示，聽寫再聽錯一次就被強制翻成那個錯字。留下的 9 條（Chet-chet、Shadow、Witchfinder、Maghra、Lord Carne、Souter Bax、Zee、Sak、rose fruit）都是真的。

## Story

身為 Vido 的使用者，我希望旁邊有官方字幕的劇，名詞表不會留著 Vido 自己聽錯的人名；翻譯提示只放真的存在的詞。

## 設計

- 官方字幕學習跑完一部劇後，對這部劇**未確認**的 `subtitle` 來源列逐條檢查：若在這次用到的所有官方英文句子裡**一次都沒出現**（整字比對，`mine.MentionsWord`）→ 刪除。
- 只在覆蓋夠的時候做：用到的集數 ≥ 2 且 ≥ 有檔案集數的一半（怕一兩集的官方字幕就判整部劇）。
- 已確認的、TMDb／手動／官方來源的一律不碰。結果多 `terms_pruned` 與清單；設定頁句尾「其中 N 個是聽寫猜錯的詞，已清掉」。
- 聽寫收成下次會不會又寫回來？會，但收成只在翻譯時發生、而官方字幕存在的集數不會走聽寫路，所以只剩「沒有官方字幕的那幾集」；再學一次又會清掉。

## Acceptance Criteria

1. See 重學後 `terms_pruned=20`，上列 20 條消失、9 條留下；TMDb／官方／已確認列不動。
2. 覆蓋不足（用到 1 集、或不到一半）→ 不清。
3. `go test ./...`、lint:all 全綠。

## Dev Agent Record

- **AC #1 ✅（2026-10-07，正式機 #713）：** 重學 See `terms_pruned=20`，剩 9 條全是真詞（eval 第九節）。
- **Adversarial CR（2026-10-07，fresh agent）1H/1M/2L，全部修掉：**
  - H1 新一季還沒有官方字幕時，剛收成的正確新角色名會在下次掃描被清掉（正是收成功能要解決的場景）→ 加「每一季覆蓋」門檻：有檔案但一集都沒被學到的季存在，整部劇就不清；加測試。
  - M2 判定語料原本只有「對齊成功」的英文句，中文檔沒有的句子（recap、歌詞）不在裡面 → 改用每集**全部**英文 cue；加「只在未對齊句出現仍保留」測試。
  - L3 補斷言：被官方取代的 Jerlamarel 是 replaced 不是 pruned。
  - L4 面板同時刪掉同一列時不再記成失敗（`ErrGlossaryTermNotFound` 視為已清）。
  - 知悉：`local:` 範圍（沒 TMDb id）的劇也會清，與收成寫入同一範圍一致；翻譯暫停中的工作在清掉後 cue 快取會換鍵重付——這是「名詞表改了要換鍵」的既定行為。
- Claude Fable 5.1。🔗 AC Drift: FOUND — sub-5-5「收成只增不減」→ 有官方對照時可刪未確認收成列。📎 Contract Stamps: NONE（`terms_pruned` additive）。🎭 A11y: N/A。🎨 UX: 只多一句文字。

### File List

- `apps/api/internal/subtitle/miner/miner.go`、`miner_test.go`（改）
- `apps/web/src/services/glossaryMineService.ts`、`apps/web/src/components/settings/OfficialSubtitleMiningCard.tsx`、`.spec.tsx`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | 正式機驗證通過（20 清、9 留）→ done。 |
| 2026-10-07 | CR 1H/1M/2L：每季覆蓋門檻、全部英文 cue 當語料、斷言與容錯。 |
| 2026-10-07 | create-story＋dev 同日；本機 See 20/29 驗證；狀態 review，待正式機重學。 |
