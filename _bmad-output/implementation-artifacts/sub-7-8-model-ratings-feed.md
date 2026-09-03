# Story 7.8: 模型品質等級由 Vido 集中評測、隨 App 發（黃金樣本 + ratings feed + CI）

Status: ready-for-dev

## Story

As a BYOK NAS owner choosing a model,
I want Vido to have measured each model on the same 200 cues and to tell me the grade,
so that I never have to buy three providers' keys to find out which one is good — and a new model gets graded the week it ships.

## Context

party-mode Murat／Alexyu：使用者不可能申請每家 key 實測；未評測不能寫問號（追不上新模型）。eval-1 的評測是手工活（9 部、38 份表、$9.14、兩天）。sub-6-8a 已留 `quality_grade?` 欄位並只填 sonnet=A／haiku=B。

## Acceptance Criteria

1. **黃金樣本。** `eval/golden/` 200 句：**不得**用受版權保護的片（eval 逐字稿已從 repo 移除且 gitignored）；來源用 CC／公有領域字幕（Blender 開源電影 Sintel／Big Buck Bunny／Tears of Steel 的官方字幕、TED CC-BY 演講）＋ 自寫的 40 句「陷阱句」（俚語、雙關、人名、時間位移誘餌、簡體漏網誘餌）。每句附 2 個可接受參考譯法與評分準則；README 註明授權。
2. **評分器。** `eval/grade.py`：跑指定 model（走 Vido 自己的 translate 路徑，`force:true`，對本機 API），用 **規則＋AI 判**混合：規則抓 echoed／簡體／人名不一致／時間位移（同 quality gate 語彙），AI 判 0/1/2 用固定 judge model 與固定 prompt（版本號進輸出）。輸出 `{model_id, prompt_version, lexicon_version, zero_rate, natural_rate, cost_usd, minutes, grade}`；grade 門檻沿用 eval-1 AC #4（0 分率 ≤ 5% 且 2 分率 ≥ 60% = A；只過其一 = B；否則 C）。單模型成本 ≤ $0.05。
3. **feed。** `apps/api/internal/ai/model_ratings.json`（embed）+ 遠端 `https://…/model-ratings.json`（可選，設定關閉；24h 快取，Rule 27 五柱）；`GET /settings/models`（sub-6-8a）改讀 feed 填 `quality_grade`／`quality_note`（「Vido 實測 2026-09，黃金樣本 v1」）。
4. **CI。** `.github/workflows/model-grade.yml`：`workflow_dispatch` 帶 `model_id`；用 repo secret 的評測 key 跑 `grade.py`，結果開 PR 更新 `model_ratings.json`。文件說明「加一個模型 = 跑一次 workflow」。
5. **未評測體驗。** 未在 feed 的模型：UI 顯示「尚未評測」＋「試跑 20 句（約 $0.01）」按鈕 → 後端 `POST /settings/models/:id/preview` 用黃金樣本前 20 句跑一次、回 zero_rate／natural_rate 與實際花費（記入 budget）；結果只存於該機 settings（`local_grade`），UI 標「你的實測」。
6. **測試。** 樣本 schema 驗證；規則評分器單元測試（每類陷阱一例）；feed 載入與遠端失敗降級（embed 值）；preview 端點成本上限；FE 未評測態 spec。

## Tasks / Subtasks

- [ ] **Task 1 — 黃金樣本 + 授權 README（AC: #1）**
- [ ] **Task 2 — `grade.py` 規則 + AI 判（AC: #2）**
- [ ] **Task 3 — feed embed/遠端 + models 端點接線（AC: #3）**
- [ ] **Task 4 — CI workflow（AC: #4）**
- [ ] **Task 5 — 試跑 20 句端點 + FE（AC: #5）**
- [ ] **Task 6 — 測試（AC: #6）**

（後端 4、腳本／CI 2、前端 1 —— 不觸發拆分；若 FE 超過 3 task 拆 b。）

## Dev Notes

- 這支是 sub-7-4 AC #5 的評測掛鉤來源；順序：7-8 Task 1–2 先於 7-4 Task 5。
- AI 判的偏誤（Claude 評 Claude）要在 `quality_note` 明示；人評校準（eval-1 待人評）落地後可換 judge 或加權。
- 遠端 feed 若做，網域與簽章（防竄改：feed 帶 sha256，App 內建公鑰）是新的外部整合——Rule 27，且屬 security posture 範圍。

### Time-dependent visual coverage

- N/A。

### References

- eval-1 AC #4 門檻與 T4/T5 三版；`eval/aggregate-full.py`（評分邏輯可搬）；sub-6-8a AC #2

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
