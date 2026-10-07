# Disc：官方字幕差超過 30 秒就學不到名詞——See 的中文字幕多了 51.7 秒前情提要，整季一條人名都沒學到

Status: review

**Source:** 2026-10-05／06 See S01E02 實測：名詞表 33 條裡 `source=official_subtitle` 是 0 條，雖然官方 zh-TW 字幕就在旁邊。2026-10-07 聽寫側人名提示證實無效後升 P1——人名一致只能靠這裡。

## Story

身為 Vido 的使用者，
我希望片子旁邊有官方中文字幕時，Vido 能從裡面學到「Jerlamarel → 謝拉馬威」這種對照，
這樣翻譯時一集只會有一種寫法，不會七種。

## 查到的事（2026-10-07，用真實字幕時間驗證）

- `mine/align.go` `EstimateShift` 只在 ±30 秒內找位移。See 的官方 zh-TW 多了 51.7 秒前情提要，真正的位移是 **−51.5 秒**，找不到。
- 更糟：±30 秒內有個 −13.75 秒的巧合（73 次命中 vs 原位 38 次）過了舊的門檻（`base+base/5+5`＝50），所以每一句都被對到差 38 秒的地方 → 0 條。
- 用真實時間軸重算：±180 秒找到 −51.5 秒、命中 485／585；本機跑完整流程：對齊 568 段、學到 10 條（Jerlamarel→謝拉馬威 23/23、Baba Voss→巴巴禾斯、Paris→芭麗絲、Maghra→瑪嘉拉、Haniwa→哈妮娃、Kofun→高豐、Tamacti Jun→塔馬迪尊…）。

## 設計

- `shiftSearchMS` 30 000 → **180 000**（±3 分鐘，蓋得住前情提要與片頭差）。
- 新增強度門檻 `shiftMinHitPercent = 35`：選中的位移要讓較少那邊 ≥35% 的句子落在英文句起點上，否則不動（真的常數位移 60～90%、±180 秒內的雜訊峰約 10～15%）。舊門檻保留。
- 真實形狀測試：`shift_fixture_test.go` 只放兩份字幕的**時間數字**（公開 repo，不放任何一句字幕文字）。

## Acceptance Criteria

1. See S01E02 真實時間軸 → 位移 −51.5 秒（±0.5）。
2. 常數 4 秒位移（Scorpion 字幕組案例）仍找得到；已對齊的不動；無關的檔案不動；句數太少不動。
3. `go test ./...`、vet、staticcheck、lint:all 全綠。
4. **實測：** 正式機更新後對 See 重跑一次官方字幕學習（`official-subtitle mining finished` 的 `terms_found` 由 0 → ≥8），`show_glossary` 出現 `source=official_subtitle` 的 Jerlamarel→謝拉馬威；之後翻譯一集，Jerlamarel 只有一種寫法。

## Tasks / Subtasks

- [x] T1 視窗＋強度門檻（AC #1、#2）
- [x] T2 真實時間夾具＋5 個測試（AC #1、#2）
- [x] T3 檢查（AC #3）
- [ ] T4 正式機重學＋驗證（AC #4）

## Dev Notes

- 位移搜尋 1441 步 × 每集約 600～1500 句，每集一次，毫秒級。
- 沒有做「非常數漂移」（不同剪輯版、幀率差）——那是另一題。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- **Adversarial CR（2026-10-07，fresh agent，用真實夾具跑了 300 次隨機檔）1M/3L：**
  - M1 「無關檔不位移」測試其實是被舊門檻擋下的、沒測到 35% 規則 → 換成「真位移剛好在視窗外（+181 秒）」與「幀率漂移 25/23.976」兩個會過舊門檻、只被 35% 擋的案例。
  - L1 20～25 句的小檔（只有強制字幕那種）35% 只是 7 次命中，雜訊一半機率湊得到 → 加絕對門檻 `shiftMinHits = 12`（實測 20／25／30 句的誤接受 49%／16%／2% → 0／0／0.3%；真的 4 秒位移 20 句仍全中）；加測試。
  - L2 幀率漂移檔從「配到片頭幾分鐘」變「配到 0」——兩邊都學不到詞，不算回歸，記著。
  - L3 建議每集 log `shift_ms` 與命中率，方便下次排查「0 條」是對齊了還是被擋 → 沒做（`Align` 內部呼叫，要改簽名），留給下一次碰 miner 時順手。
  - 其他確認：zh 只涵蓋半集／SDH 多 40% 音效句／兩句併一句等情境 35% 都過（80～100%）；1441 步 × 每集一次毫秒級；夾具純數字。
- 🔗 AC Drift: FOUND — sub-7-5a「±30 秒」→ ±180 秒＋強度門檻。
- 📎 Contract Stamps: NONE。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

N/A — no out-of-scope work discovered

### File List

- `apps/api/internal/subtitle/mine/align.go`（改）、`shift_test.go`、`shift_fixture_test.go`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-mine-shift-range-too-narrow.md`（新）、`sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | CR 1M/3L：測試換成真正考 35% 的案例、加 12 次絕對門檻。 |
| 2026-10-07 | create-story＋dev 同日：±180 秒＋35% 強度門檻；真實時間夾具；本機端到端學到 10 條；狀態 review，待正式機重學。 |
