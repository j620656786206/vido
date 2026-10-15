# Disc：一批翻譯回應被截斷就整集失敗——找出原因、改成切半重送

Status: done

**Source:** 2026-10-15 正式機 S01E04（Sonnet 50 句一批）第 4 批回應 output_tokens=4096、沒有文字 → `AI_INVALID_RESPONSE` → 整集失敗（US$0.126）。當天 hotfix #804 把 Sonnet 退回 10 句。Alexyu：「繼續修」。

## 查到的事（2026-10-15，main `08e33fe6`）

- 錯誤來源：`apps/api/internal/ai/claude.go:515-517` 回應裡沒有 text block 就回 `ErrAIInvalidResponse`；`textFromMessage`（`:369-379`）只取第一個 text block。
- **根因：Sonnet 5 預設會「想」（adaptive thinking），想的 token 算在 max_tokens 裡。** 本機用同一批（S01E04 第 151～200 句、同樣的系統提示、max_tokens 4096）直接打 API，每次回應都有一個 thinking block：
  - 預設（不設 thinking）6 次：output 1,003／1,015／1,041／1,049／1,081／3,346／3,597 tokens——翻譯文字每次都是約 2,200 字（約 1,000 tokens），多出來的 0～2,600 tokens 全是「想」。正式機那次想超過 3,000 就把 4,096 用光，文字一個字都沒寫。
  - `thinking: disabled` 3 次：1,000／1,011／1,033，沒有 thinking block，50 句都在。
  - `effort: low` 3 次：996／996／1,016，有很短的 thinking block，50 句都在。
  - 10 句一批也會想：同一段 10 句，預設 205 對 921 tokens（想了約 700）。所以這不只是 50 句的問題，10 句一批時我們一直在為「想」付錢（S01E03 的 US$0.31 裡有一部分是這個）。
- 一批失敗的處理：
  - 片內字幕那條（`internal/subtitle/pipeline.go:965-990`）：非暫時性錯誤一律「整集停」，理由寫的是「下一批也會一樣失敗」——對截斷不成立（換小一點就好）。
  - 聽寫那條（`internal/services/translation_service.go:508-540`）：失敗的那批留英文、繼續下一批（不會整集停，但一批 50 句全留英文）。
- Gemini（OpenAI 相容端點）：`openai_compat.go:328-329` 內容空就回 `ErrAIInvalidResponse`，沒看 `finish_reason`。

## 事先登記的規則（量之前寫）

關掉 thinking 會不會讓翻譯變差，用 golden-v2 200 句考卷量 Sonnet 50 句一批：

- **過**：翻壞率 ≤ 2.0% 且自然率 ≥ 89.0%（基準＝Sonnet 10 句、預設 thinking：1.5%／90.0%）。考 2 次，兩次都過才算。
- 過 → 翻譯請求一律關 thinking，Sonnet 放回 50 句。
- 不過 → thinking 保留、把回應上限依批次放大（50 句 → 16,384），Sonnet 維持 10 句，另議。

## Acceptance Criteria

1. 新的錯誤 `ai.ErrAIResponseTruncated`：回應頂到上限、又沒有可用文字時回它（同時仍算 `ErrAIInvalidResponse`，舊的判斷不變）。Claude 看 `stop_reason == max_tokens`；OpenAI 相容端點看 `finish_reason == length`。
2. `ai.WithThinkingOff(ctx)`：Claude 收到時送 `thinking: {type: disabled}`；其他 provider 忽略。翻譯服務的兩個入口（片內字幕的 `TranslateChunk`、聽寫的 `TranslateWithGlossaryHarvest`）都帶它；評審、檔名解析等其他呼叫不受影響。
3. **切半重送**：片內字幕那條遇到 `ErrAIResponseTruncated` 時，把那批對半切、各自重送，一路切到 1 句；1 句還被截斷就留英文（算品質沒過，照既有的 5%／20% 上限）。不再整集失敗。聽寫那條同樣切半，不再整批留英文。
4. 量測照「事先登記的規則」做，結果寫進本檔；過了才把 `claude-sonnet-5` 放回 50。
5. 測試：Claude 帶 thinking-off 時請求 JSON 有 `"thinking":{"type":"disabled"}`、沒帶時沒有；stop_reason=max_tokens 且無文字 → `ErrAIResponseTruncated`；OpenAI 相容 finish_reason=length → 同；pipeline 一批截斷 → 切成兩半都翻好、1 句截斷 → 留英文不整集失敗；聽寫路同。
6. 正式機：末日光明 S01E04 重跑（同一集、同樣 Sonnet），成功、看請求數／費用／耗時／人名。

## Tasks / Subtasks

- [x] T1 `ai`：截斷錯誤、thinking-off（AC 1、2）
- [x] T2 pipeline 與聽寫路的切半重送（AC 3）
- [x] T3 測試（AC 5）
- [x] T4 量測與批次決定（AC 4）
- [x] T5 正式機（AC 6）：✅ 2026-10-15 12:07 image 011e122d、末日光明 S01E04、預設 Sonnet 50 句、不想：474 句、11 次請求、US$0.160、3 分 6 秒，0 句留英文、0 次截斷。人名跟舊官方字幕一致（謝拉馬威 8／8、安女爵 5／5、塔馬迪尊 4／4、瑪嘉拉 4／4、敦爵士 4／4），無簡體。對照 S01E03（10 句、會想）：439 句、46 次請求、US$0.31、5 分鐘 → 每句費用約省一半。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-15。

### Completion Notes List

- `internal/ai/thinking.go`：`ErrAIResponseTruncated`、`WithThinkingOff`／`ThinkingOffFromContext`。`claude.go` 收到 thinking-off 送 `thinking: {type: disabled}`；無文字且 `stop_reason=max_tokens` → 回 `ErrAIInvalidResponse`＋`ErrAIResponseTruncated`。`openai_compat.go` 解析 `finish_reason`，`length` 且無文字 → 同。四個 catalog Claude 模型（haiku-4-5、sonnet-4-6、sonnet-5、opus-4-8）都實測接受 `thinking: disabled`。
- `internal/subtitle/pipeline.go`：`translateSplitting` 取代直接呼叫 `TranslateChunk`；截斷就對半切、遞迴到 1 句，1 句還截斷就回空結果讓 gate 留英文；請求數照實際次數記。其他無效回應照舊整集停。
- `internal/services/translation_service.go`：`TranslateWithGlossaryHarvest` 與 `TranslateChunk` 帶 `WithThinkingOff`；聽寫路的單批請求改走 `completeSplitting`（同樣切半）。
- 測試：`internal/ai/thinking_test.go` 四條；pipeline 三條（切半、1 句留英文、其他無效回應照舊停）；翻譯服務三條（聽寫路切半＋每個請求都關 thinking、1 句留英文、TranslateChunk 關 thinking）。`go test ./...`、vet、staticcheck 綠。

**量測（照事先登記的規則）**：golden-v2 200 句、Sonnet 50 句一批、thinking 關：

| 次 | 翻壞率 | 自然率 | 過？ |
|---|---|---|---|
| 1 | 1.0% | 95.0% | ✅ |
| 2 | 0.5% | 95.0% | ✅ |

（對照：thinking 開、50 句 0.5%／96.5% 與 0.5%／96.0%；thinking 開、10 句 1.5%／90.0%。）兩次都過 → 翻譯一律關 thinking、Sonnet 放回 50 句。評審費兩次共 US$0.63。

### Discovery Triage

- ① 10 句一批也在為 thinking 付錢 → 本單 AC 2 一起解決（翻譯一律關 thinking）。
- ③ 估價費率已在 `disc-2026-10-translation-rates-recalibrate-after-batch`；關 thinking 後 Sonnet 實際費用會再降，同一張單一起校正。

### File List

- `apps/api/internal/ai/{thinking.go,thinking_test.go,claude.go,openai_compat.go}`
- `apps/api/internal/ai/prompts/{subtitle_translator.go,subtitle_translator_test.go}`
- `apps/api/internal/subtitle/{pipeline.go,pipeline_test.go}`
- `apps/api/internal/services/{translation_service.go,translation_service_test.go}`
- `eval/golden-v2-2026-10/claude-sonnet-5-batch50-nothink-run{1,2}.json`
