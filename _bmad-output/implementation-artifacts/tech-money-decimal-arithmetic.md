# Tech: 金額一律走十進位套件，前後端都是

Status: review

**No ticket exists for this** — Alexyu 於 2026-09-04 在 sub-6-8b 的 /ship 過程中提出，
並在我提出「用整數分就好」的替代方案後**重申要用套件**。這份規格記錄的是他的裁定，
以及我原方案為什麼被推翻。

## 裁定

> 「就金錢而言，數字的準確度要絕對準確，不能夠有 0.幾加 0.幾就跑出另外一個數字的
> 可能性。所以不管怎樣，我還是希望用套件去解決浮點數的問題，前端和後端都一樣。
> 因為我們串接的這些 AI 的 API key，背後模型的收費計算也都是以美金計價。」
> —— Alexyu, 2026-09-04

## 我原本提的方案，以及它為什麼錯

我先提議用**整數分**（`int64` / JS 整數）而不是十進位套件，理由是分在兩邊都精確可
表示、不必加依賴。

**那個方案在計費路徑上是壞的。** AI 定價不是分為單位的：Sonnet 是
`$3.00 / 1M input tokens`，也就是每個 token `$0.000003`。用整數分承載，
`Budget.RecordLLM` 在第一次呼叫就得把它捨成 `$0.00` —— 跑一百萬次之後帳本還是報
「花了 0 元」。Alexyu 指出的正是這一點。

整數分只夠用在「已經捨成分的顯示合計」，不夠用在「每 token 的計費累加」。
十進位套件兩者都涵蓋。

## 事實查核（2026-09-04）

- 原本 repo **沒有任何 decimal 套件**：`package.json` 與 `apps/api/go.mod` 都沒有。
- 兩邊都是 float64 + `math.Round(v*100)/100` / `toFixed(2)`。
- 真正的錢在 `ai.Budget`：`RecordLLM` 每次 LLM 呼叫累加一次 `spentUSD`，
  `Exceeded()` 拿它跟使用者同意的上限比 —— 一次翻譯批次會累加**數千次**。

## 選用的套件

| | 套件 | 為什麼 |
| --- | --- | --- |
| Go | `shopspring/decimal` v1.4.0 | Go 生態最通行的任意精度十進位；`Shift` 可做無損的 10 冪次縮放 |
| Web | `decimal.js` ^10.6.0 | Alexyu 指名；成熟、無依賴 |

（`cockroachdb/apd` 更嚴謹但 API 較繁；`Rhymond/go-money` 內部是整數最小單位，
正好踩到上面說的 sub-cent 問題。）

## 已完成 — 後端

1. **`ai.ModelPricing` 改成 `decimal.Decimal`**，且由**十進位字串**建構
   （`price("3.00", "15.00")`），不是 float 字面量。`0.30` 沒有精確的二進位表示，
   用 float 寫價目表等於還沒開始算就偏了 ~1e-17。
2. **`Budget.RecordLLM` 的單次費用完全精確**：
   `(in × pIn + out × pOut).Shift(-6)` —— `Shift` 是移動指數，**不是除以 1e6**，
   所以沒有除法、沒有捨入。
3. **`Budget.spentUSD` / `maxUSD` 改成 decimal**，`Exceeded()` 用 decimal 比較。
   新增 `Spent() decimal.Decimal`；`SpentUSD() float64` 留給 wire 與 log。
4. **估價側**（`services/generation_candidates.go`）：eval-1 實測費率改成十進位字串、
   `estimateUSD` 全程 decimal、整輪掃描的合計以 decimal 累加，只在寫進 wire 時narrow 一次。

### 三個證明這不是理論問題的測試

- `TestBudget_SpendIsExactAcrossManyCalls` — 三筆 $0.10 的結果是 `"0.3"`，
  不是 float64 的 `0.30000000000000004`。
- `TestBudget_CeilingIsNotSkippedByFloatDrift` — **這條是真的會花到錢**：
  `$0.10 + $0.70` 在 float64 是 `0.7999999999999999`，對 `$0.80` 的上限判定成
  **false**，於是又多跑一次使用者沒同意的呼叫。decimal 下剛好觸頂。
- `TestBudget_SubCentPerTokenCostIsNotRoundedAway` — Sonnet 一個 token 是
  `"0.000003"`，這正是整數分方案會歸零的那個數。

## 已完成 — 前端

5. `lib/currency.ts` 走 decimal.js，提供 `usd` / `sumUsd` / `addUsd` / `subUsd`
   / `gtUsd` / `ltUsd` / `percentOfUsd` / `roundUsd` —— 金額的算術與格式化同源。
   12 個 spec。
6. `consentSelection.ts` 的每一處金額運算改走那組 helper：`computeTotals` 的
   兩段累加、可行數的走訪、上限判定、`sumSelected`、`modelChoices` 的價差與
   百分比。檔案裡不再有任何裸的 `+ - < >` 用在錢上。
7. gallery fixture 的 `confirmTotals` 改用 `addUsd`。

### 寫測試才發現的第二層問題

**精確加法本身不夠。** `computeTotals` 原本累加的是「未捨入的原始列值」，捨入發生
在 `usd()` 顯示的那一刻。於是兩筆 0.005：

```
抽取 0.005 → 顯示 $0.01
語音 0.005 → 顯示 $0.01
合計 0.010 → 顯示 $0.01      ← 算術完全正確，畫面還是加不起來
```

修法是把捨入移到**進入加總之前**：`candidateUsd` 現在回傳已捨成分的值。這樣
列、群組小計、兩段路線、總計每一層都由整數分構成，一致性是結構保證而不是巧合。
實務上這是 no-op（`estimateUSD` 已經把每個 wire 值捨成整數分）—— 一個守衛本來
就該長這樣。

## 已知未解 — wire 仍是 JSON float

decimal 保護的是**兩端各自的算術**。金額跨 HTTP 時仍是 JSON number，
`JSON.parse` 一定給 IEEE-754 double —— 這是最後一個有損的環節。

要端到端精確，wire 上的金額得是**字串**（`"0.53"`）。那是跨
`sub-4-1 AC #7` / `sub-4-2 [@contract-v3]` / `sub-6-8a AC #3` 三個契約的
**Rule 20 bump**，牽動每一個消費端 → `backlog-money-string-on-the-wire`。

實務影響有限：目前每一筆跨 wire 的金額都已由 `estimateUSD` 捨成整數分，
而整數分在 double 裡的往返是無損的（值遠小於 2^53）。真正會被截斷的是
sub-cent 的 `spent_usd`，那條路徑目前只用於顯示。

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context)

### File List

- `apps/api/go.mod`、`go.sum`（+ shopspring/decimal v1.4.0）
- `apps/api/internal/ai/budget.go`、`budget_test.go`
- `apps/api/internal/ai/catalog.go`、`catalog_test.go`、`gemini_test.go`、`gemini_model_test.go`
- `apps/api/internal/services/generation_candidates.go`、`generation_candidates_test.go`
- `package.json`、`pnpm-lock.yaml`（+ decimal.js）
- `apps/web/src/lib/currency.ts`、`currency.spec.ts`
- `apps/web/src/components/subtitle/consent/consentSelection.ts`、`consentSelection.spec.ts`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
