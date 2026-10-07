# Disc：官方字幕兩季寫法不同時，名詞表學到的是兩種寫法的共同片段（Jerlamarel → 「拉馬」）

Status: done

**Source:** 2026-10-07 第七節：See 官方 zh-TW 第 1 季「謝拉馬威」、第 2 季「傑拉馬瑞爾」，miner 學到「拉馬」；同類「Princess Maghra→拉公主」「Trivantians→斯人」。

## Story

身為 Vido 的使用者，我希望名詞表學到的是完整的人名寫法（多數那一種），不是兩種寫法的共同片段。

## 設計

- `Mine` 選出次數最多的 Han 子字串後，若它是**片段**（從不單獨出現在標點之間，`bounded == 0`），就往上長：找「能解釋它至少一半出現次數、同樣通過專一性門檻」的超字串，優先選有單獨出現過的（是名字）、再比次數、再比長度；重複到長不動為止（`growToFullRendering`）。
- 已會單獨出現的（真名字如「托比」）不動，所以「去問托比」不會被長進去。

## Acceptance Criteria

1. 7 段「謝拉馬威」＋4 段「傑拉馬瑞爾」→ 學到「謝拉馬威」（support 7），不是「拉馬」。
2. 「托比」在「去問托比」×3＋單獨出現 ×3 的樣本裡仍是「托比」。
3. 既有 mine／miner 測試不變；`go test ./...`、staticcheck、lint:all 全綠。
4. **實測：** 正式機重學 See，`Jerlamarel` 變成「謝拉馬威」（official_subtitle）。

## Dev Agent Record

- **AC #4 ✅（2026-10-07）：** 英文側來源修好（#709）後正式機重學：Jerlamarel → 謝拉馬威 52/83、Trivantians → 崔凡特斯人、Princess Maghra → 瑪格拉公主；與本機 15 集重現一致。
- **Adversarial CR（2026-10-07，fresh agent，在 scratchpad 複製套件實跑攻擊用例）2H/2M/1L，全部修掉：**
  - H1 「至少一半」原本對上一層算、且沒套 MinSupport，會一路爬到只出現 1 次的片語、結果隨機 → 門檻改成對**原始片段**算（`origin`）＋ `n ≥ MinSupport`。
  - H2 用「有沒有單獨站過」判片段，沒呼格的小樣本會把「托比」長成「問托比」 → 改用 **run 邊界**：在這個詞的所有行裡從沒出現在 Han run 開頭（左邊永遠黏東西）或結尾才算片段，而且只往黏著的那一側延伸；「托比在哪」（開頭）＋「去問托比」（結尾）就是完整詞。`hanSubstringsWithEdges` 回傳 Whole／AtStart／AtEnd。
  - M1 光桿「瑪格拉」被長成「瑪格拉公主」後再被 `dropSubTerms` 刪掉 → 邊界規則順便解掉（瑪格拉兩側都看過）；加測試。
  - M2 兩季各半、同長度時平手隨機 → 兩處平手補字串決勝；加 20 次決定性測試。
  - L 長不出來的雙側片段（2+2 太薄）→ 直接不學（「拉馬」這種詞條沒用）；單側片段（如「公主」）保留。
  - 本機真實 S01E02：仍學到 Jerlamarel→謝拉馬威 23/23 等 10 條。
- Claude Fable 5.1。🔗 AC Drift: NONE（sub-7-5a 的選法補一條成長規則）。📎 Contract Stamps: NONE。🎭 A11y: N/A。🎨 UX: SKIPPED。

### File List

- `apps/api/internal/subtitle/mine/mine.go`、`mine_test.go`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | 正式機驗證通過 → done。 |
| 2026-10-07 | CR 2H/2M/1L 修掉：origin 門檻、run 邊界判片段、方向性成長、決定性、雙側片段不學。 |
| 2026-10-07 | create-story＋dev 同日；全綠，狀態 review，待正式機重學驗證。 |
