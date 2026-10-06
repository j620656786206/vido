# Disc：「里」不再被當成簡體字——帕里斯、英里這類句子不會再整句被退回英文

Status: review

**Source:** 2026-10-06 See S01E02 第二次 NAS 實測（`eval-see-s01e02-asr-vs-official.md`「第二次實測」）：568 句裡 **15 句留成英文**，log 全是 `reason=simplified_leak`，14 句含「帕里斯」、1 句含「英里」。

## Story

身為 Vido 的使用者，
我希望「帕里斯」「數千英里」這種正常的繁體句子不會被品質檢查當成簡體字而整句退回英文，
這樣看片時不會突然冒出一句英文。

## 查到的事（2026-10-06，逐一開檔確認）

- 品質閘門 `quality_gate.go:97-102`：`Detect([]byte(text)).SimplifiedCount > 0` 就把該句標 `simplified_leak`，註解寫明「STRICT by design: any simplified-only character fails the cue」。這條嚴格規則本身是對的（FR16），問題出在字表。
- `detector.go:168-200` 的 `simplifiedOnlySet` 註解說是「只存在於簡體的字」，但表裡有幾個字**同時是合法的繁體字**：`里`（公里、鄰里、帕里斯；只有「裡」才對應 里）、`几`（茶几）、`云`（人云亦云）、`丰`（丰采）、`余`（余光中、余＝我）、`占`（占卜）、`划`（划船）。表頭已經排除過一批同類字（走、面、后、着…），這幾個是漏網。
- `Detect` 同時用這張表算整份文件的簡繁比例（`detector.go:61-100`，門檻 70%／30%）；拿掉 7 個字對分類影響極小——簡體文件還會踩到其餘 190 多個字（`TestDetect_SimplifiedChinese` 保護）。
- NAS 實跑證據：`subtitle_runs` 該筆 `stubborn_count=15`；15 句 cue index：29、53、100、117、128、152、171、250、351、428、434、438、464、466、516。這些句子被排除在 segment cache 外（log「excluded from the segment cache」），所以修好後 Regenerate 只會重翻這 15 句，其他從快取來。

## 設計

從 `simplifiedOnlySet` 拿掉 `里 几 云 丰 余 占 划 准 么 佣`（後三個是 CR 補的：不准／老么／佣金），註解寫明原因與例子。閘門多一張**簡體「詞」**名單（`quality_gate.go` `simplifiedWords`：哪里／家里／心里／几乎／几个／多余／其余／丰富／什么／怎么／准备／标准／佣人…），把這些字「當簡體用」時的漏洞補回來。不動 OpenCC、不動 traditionalOnlySet。

## Acceptance Criteria

1. `Detect` 對「帕里斯」「數千英里」「公里」「茶几」「人云亦云」「丰采」「余光中」「占卜」「划船」「不准動」「老么」「佣金」的 `SimplifiedCount` 都是 0；「这里」「几个」仍 > 0。
2. 品質閘門：`去找帕里斯談。`、`能跨越數千英里說話的機器`、`不准動！老么拿走了佣金。` 通過；`在这里`（有 这）、`家里沒人`、`几乎什么都不剩，准备多余的丰富晚餐`（簡體詞）仍是 `simplified_leak`；既有的 `说点什麼`／`这个软件很好用` 案例不變。
3. 既有 detector 測試（繁／簡／混合／門檻）全部不改期望值照常通過。
4. `go test ./...`、`go vet`、`staticcheck`、`lint:all` 全綠。
5. **NAS 實測（部署後補）：** 對 See S01E02 再 Regenerate 一次，`stubborn_cues` 15 → 0，且 Paris 有一致的中文寫法。

## Tasks / Subtasks

- [x] T1 拿掉 7 個雙用字＋註解（AC #1）
- [x] T2 detector／quality gate 測試（AC #1、#2、#3）
- [x] T3 檢查（AC #4）

## Dev Notes

- 不要把閘門改成比例制或容許值：一個真的簡體字（这／个）就是 FR16 要抓的瑕疵。
- 不要用 OpenCC 事後補救代替修字表：閘門刻意在 OpenCC 之前看模型原始輸出。
- 其他可疑的雙用字（么、冲、种、复、够）這次**不動**：么／冲／复／够 在台灣標準寫法裡不算合法繁體，种 當姓氏太罕見；誤殺風險低，等再有實例再加。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- **對抗式 CR（另開 agent，2026-10-06）：1 Medium＋1 High 建議，都採納**
  - **Medium** 拿掉 里／几／云／丰／余 之後，「家里」「几乎」「多余」「丰富」這種把它們當簡體用的常見漏字，閘門抓不到了（實測 `家里沒人` simp=0）→ 閘門加 `simplifiedWords` 詞表（`containsSimplifiedWord`），這些詞照樣退回；加兩個測試。
  - **High** 集合裡還有 `准`（不准、批准）、`么`（老么）、`佣`（佣金）同樣會誤殺——「不准動！」是對白最常見句型之一 → 一併拿掉，加測試。
  - 其餘仍在集合的疑似字（坏、复、够、冲、与、无、体）在 zh-TW 字幕幾乎不出現，保留。
- **測試夾具調整（刻意）：** `TestDetect_BoundaryExact30` 原本用 70 個簡體字湊 30%，其中含 几、云；拿掉後只剩 68 個、比例變 30.6% → 判成 zh。把夾具的 几／云 換成 难／雾（仍是簡體專用字），**期望值不變**。
- 🔗 AC Drift: FOUND — sub-1-4 AC #4／sub-1-5a FR16「任何簡體專用字就退回」的規則不變；變的是「哪些字算簡體專用」（detector 字表），`detector_test.go` 既有期望值未動。
- 📎 Contract Stamps: NONE（本單不定義也不消費 wire contract）。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

N/A — no out-of-scope work discovered

### File List

- `apps/api/internal/subtitle/detector.go`（改）
- `apps/api/internal/subtitle/detector_test.go`（改）
- `apps/api/internal/subtitle/quality_gate.go`（改：`simplifiedWords`／`containsSimplifiedWord`）
- `apps/api/internal/subtitle/quality_gate_test.go`（改）
- `_bmad-output/implementation-artifacts/disc-2026-10-simplified-leak-false-positive-li.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-06 | create-story＋dev 同日：拿掉 里／几／云／丰／余／占／划；detector 與 gate 各加測試；全綠，狀態 review。 |
| 2026-10-06 | CR：再拿掉 准／么／佣；閘門加簡體詞表補回召回率；測試補齊，全綠。 |
