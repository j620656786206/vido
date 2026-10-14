# Gemini 免費方案翻譯考卷 2026-10

觸發：FB 招募文底下 Ming Jwo（2026-10-14）：「不需要 Claude，建議用 gemini 免費的，比 claude、openai、xai 付費的都快很多而且效果很好」。Alexyu 要求拿同一份 200 句考卷考一次，把數字貼回去。

## 考法

- 考卷：golden-v2（200 句，附角色名對照表，跟正式版一樣），同一個評審 claude-sonnet-5，跟 `eval/golden-v2-2026-10/claude-*.json` 完全同一套。
- 走 Gemini 的 OpenAI 相容端點（`https://generativelanguage.googleapis.com/v1beta/openai`），用 `cmd/grade --openai-base-url`，production 提示方式、每批 10 句。
- 金鑰：Alexyu 在 AI Studio 申請的免費方案。

## 結果（2026-10-14）

| 模型 | 0 分率 | 2 分率（自然） | 等級 | 第二組（人名／簡體改由評審評） | 翻 200 句 | 費用 |
|---|---|---|---|---|---|---|
| claude-sonnet-5（預設） | 1.5% | 90.0% | A | — | 1.6 分 | US$0.53（含評審） |
| claude-haiku-4-5 | 1.0% | 87.5% | A | — | 0.9 分 | US$0.34（含評審） |
| **gemini-3.5-flash-lite（免費）** | **2.0%** | **89.5%** | **A** | 1.5%／90.0% | 4.1 分（每 13 秒一批） | 模型 $0；評審 US$0.27 |
| gemini-3.8-flash | — | — | 考不成 | — | — | — |

- Gemini 3.5 Flash-Lite 跟 Sonnet 在誤差內。唯一沒過規則的 1 句是人名對不上，三家都一樣。
- **gemini-3.8-flash 考不成**：免費方案**每天只給 20 次請求**（`generate_content_free_tier_requests, limit: 20`），200 句 ÷ 10 句一批 = 20 批，加上重試就超過。兩次都 429，150～190 句被擋回來保留英文（0 分率 76%／95%，那是額度不是品質）。要考它要付費方案或分天跑。
- **免費方案的速率**：每分鐘 5 次。第一次沒控速（翻譯程式的重試只等 1～2 秒），一樣被擋。`cmd/grade` 新增 `--pace`（兩次呼叫最少隔多久），用 13 秒才考成；一集 835 句 ≈ 84 批 ≈ 18 分鐘。
- `gemini-2.5-flash` 對新帳號已經關閉（404「no longer available to new users」）。

## 結論

⚖️ **Alexyu 2026-10-14 裁定：Gemini（翻譯）和 ElevenLabs（聽寫）都做成可選項，各用使用者自己的免費額度；預設不動（Sonnet）。** 對外的說法是「可以用你自己的免費額度」，不是「Vido 免費」。

→ 開單 `gemini-translation-dispatch`（原 `backlog-gemini-translation-dispatch`）。

## 檔案

- `eval/golden-v2-2026-10/gemini-3.5-flash-lite.json`（含 trace）
- `apps/api/cmd/grade/paced.go`（`--pace`）
- 3.8-flash 的兩份失敗報告沒有留（量到的是額度，不是模型）。
