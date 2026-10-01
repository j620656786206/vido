# Story sub-7-8a: 黃金樣本 200 句 ＋ 評分器（規則＋AI 判，Go 子命令）— 後端／腳本

Status: done

<!-- SM Bob create-story 2026-10-01，由 sub-7-8 拆出（原單 BE 4＋腳本 2＋FE 1 的門檻被「樣本授權」「CI 跑不到本機 API」「試跑要設計稿」三件事撐破 → 拆成 7-8a 樣本＋評分器／7-8b 等級表＋CI／7-8c 試跑 20 句）。
     ⚖️ Alexyu 2026-10-01 裁定：①樣本＝Tears of Steel 76 句＋Sita Sings the Blues 84 句對白＋自寫 40 句陷阱（TED 是 CC BY-NC-ND，翻譯＝改作，不得用）；②評分器做成 Go 子命令 `cmd/grade`，不做 Python 打本機 API；③等級表只做內建 JSON＋CI 開 PR，不做遠端 feed；④單模型成本上限由 ≤$0.05 改為 ≤$0.50（Sonnet 翻 200 句本身就約 $0.11，再加 AI 判分）。
     行號為 main `4004c7af`。原單 6 處與程式碼不符（見「背景」）。 -->

## Story

身為要選翻譯模型的 BYOK 使用者，
我要 Vido 用同一份 200 句考卷考過每一個模型、告訴我等級，
而不是自己去申請三家的 key 一一試；新模型出來那一週就有分數。

## 背景（查到的事）

- 等級目前寫死在 `ai/catalog.go:66-90`（`modelMetadata` 只填 sonnet=A／haiku=B，`evalNote` 指 eval-1 人評 10,304 句）。其他模型一律空白 → FE 顯示「尚未評測」。
- 翻譯路徑**只能到 Claude**：`services/model_catalog.go:57-73 Available()` 刻意只列 Claude（CR H2），`ai.GeminiProvider` 也沒實作 `TextCompleter`（只有 `Parse`）。所以評分器也只評 Claude 模型；Gemini 等 `backlog-gemini-translation-dispatch`。
- 原單「grade.py 打本機 API 的 translate 路徑（`force:true`）」要片庫裡真的有一部片、要起伺服器、要 TMDb 比對——CI 做不到。既有先例 `cmd/route-c-poc/main.go:158-170` 已示範在程式內直接組 `ai.NewClaudeProvider` → `services.NewTranslationService(completer, nil)` → `Translate`，同一套 prompt／詞庫／版本號自動帶出。
- 版本三元組：`prompts.PromptVersionFor(level)`（`ai/prompts/localization.go:101`，形如 `m1-v3+zh-tw-lex-1+standard`，詞庫版本已含在內）、glossary 版本 `subtitle.GlossaryVersionHash`（考卷不帶詞彙表 → 空）、`model_id`。
- 規則層可直接重用 `subtitle.CheckChunk`（`subtitle/quality_gate.go:50`：missing／empty／echoed／simplified_leak）。「人名不一致」「時間位移」閘門沒有，要新寫（sub-7-9 之後會把 misaligned 加進閘門，先在評分器裡做）。
- 預算：`ai.WithBudget(ctx, b)` 掛上後 `governor.go:70` 超額回 `ai.ErrBudgetExceeded`；`Budget.SpentUSD()` 給實際花費。
- `.gitignore:113` 忽略 `eval/**/*.srt`——考卷不能是 .srt。改用 JSONL，放在 `apps/api/internal/eval/golden/`（`go:embed`，7-8c 的試跑端點要從 binary 讀前 20 句）。
- 授權：Tears of Steel 影片與字幕 CC BY 3.0（Blender Foundation；繁中檔 `download.blender.org/demo/movies/ToS/subtitles/TOS-CH-traditional.srt`，台灣腔、76 句時間碼 75 句對得上）。Sita Sings the Blues 影片 CC0（2013-01-18 起）；英文與繁中字幕取自 Wikimedia Commons TimedText，該站文字貢獻為 **CC BY-SA 4.0** → 考卷資料檔整體以 **CC BY-SA 4.0** 發布並逐條標來源。Sita 歌詞（Annette Hanshaw 錄音仍有版權至 2030）**一律不選**，只選對白。

## Acceptance Criteria

1. **黃金樣本 v1。** `apps/api/internal/eval/golden/golden-v1.jsonl` 恰 200 行，每行 `{id, source: tos|sita|trap, start, end, text, refs: [≥2 句可接受繁中], traps: [], names: {}, note}`；`source=tos` 76 句、`sita` 84 句（全對白、不含歌詞）、`trap` 40 句自寫（俚語／雙關／人名一致／時間位移誘餌（相鄰兩句語意可互換）／簡體漏網誘餌／台灣用語（原單 sub-7-4 詞庫）／數字單位／語氣）。`names` 標該句應固定的人名譯法（給規則層）。`README.md` 寫授權（CC BY-SA 4.0；逐來源 attribution）、來源 URL、怎麼重建（`scripts/build-golden.py` 從兩份 .srt 對齊）、陷阱分類表。
2. **規則層。** `internal/eval` 套件 `RuleCheck(cue, zh) []string`：重用 `subtitle.CheckChunk` 的四類；加 `name_mismatch`（譯文含 `names` 任一鍵的非指定譯法，或指定譯法缺席）；加 `time_shift`（譯文含相鄰 cue 的錨點（數字／人名／`names` 值）而不含自己的）。每類陷阱至少一條單元測試正反各一。
3. **AI 判。** 固定裁判 `claude-sonnet-5`、固定 prompt `judge-v1`（版本號進輸出；rubric 寫明 0／1／2 定義沿 eval-1、台灣口語優先、短句、不翻譯腔；給原文、兩個參考譯法、待評譯文；一次 20 句、回 JSON）。裁判偏誤（Claude 評 Claude）寫進輸出 `judge_note` 與 7-8b 的 `quality_note`。
4. **`cmd/grade`。** `go run ./cmd/grade --model <id> [--sample path] [--limit N] [--budget-usd 0.50] [--judge-model claude-sonnet-5] [--out file.json] [--level standard]`：key 讀 `ANTHROPIC_API_KEY`（或 `CLAUDE_API_KEY`）；用 `services.NewTranslationService` 走真翻譯（`WithLocalizationLevel`，無詞彙表、無 metadata）；一個 `ai.Budget` 蓋住翻譯＋裁判，超額即停並輸出 `grade: "incomplete"`。輸出 JSON `{model_id, sample_version, prompt_version, lexicon_version, glossary_version, judge_model, judge_version, cues, zero_rate, natural_rate, rule_fail_rate, rule_fails_by_reason, cost_usd, minutes, grade, judge_note, graded_at}`。等級：0 分率 ≤5% 且 2 分率 ≥60% → A；只過其一 → B；否則 C（沿 eval-1 AC #4；規則層任何一條失敗該句計 0 分）。
5. **測試。** 樣本 schema 測試（200 行、id 唯一、refs≥2、source 計數 76/84/40、無歌詞來源、每個 trap 類別 ≥3 句）；規則層每類正反例；等級門檻邊界；裁判回應解析（壞 JSON／缺句／越界分數）；`cmd/grade` 用假 completer 跑完整流程（含預算超額 → incomplete）。不打真 API。

## Tasks / Subtasks

- [ ] Task 1 — 建考卷：`scripts/build-golden.py` 對齊兩份字幕 → 160 句＋手寫 40 句陷阱＋每句第二參考譯法 → `golden-v1.jsonl`＋`README.md`（AC #1）
- [ ] Task 2 — `internal/eval`：載入／schema 驗證、規則層、等級函式（AC #2, #4 門檻）
- [ ] Task 3 — 裁判 prompt `judge-v1`＋解析（AC #3）
- [ ] Task 4 — `cmd/grade` 接線＋預算（AC #4）
- [ ] Task 5 — 測試全套（AC #5）

## Dev Notes

- 這支是 sub-7-4 AC #5 評測掛鉤的來源；7-4 的 `standard` vs `ott` 比較等本單合併後由 Alexyu 在本機跑（本機無 key 的 session 不跑）。
- 不碰 `catalog.go`、不碰 FE、不碰 CI——那是 7-8b／7-8c。
- 試算成本（eval-1 費率）：Haiku 200 句 ≈ $0.04、Sonnet ≈ $0.11、Opus 4.8 ≈ $0.2；裁判 Sonnet 讀 200 組 ≈ $0.05。$0.50 上限足夠，Opus 也進得去。
- 歌詞排除：Sita 英文字幕沒有 ♪ 標記，`build-golden.py` 以手挑 index 白名單為準，不靠偵測。

### Time-dependent visual coverage

- N/A（無 UI）。

### References

- eval-1 AC #4；`eval/aggregate-full.py`（門檻常數）；sub-6-8a AC #2；`cmd/route-c-poc/main.go`；`subtitle/quality_gate.go`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 5 task 全數交付，全 BE／腳本，不打真 API（本機無 key；真跑由 Alexyu 在本機或 7-8b 的 workflow 進行）。
- **考卷**：`golden-v1.jsonl` 200 行（tos 76／sita 84／trap 40），每句兩個參考譯法（第一句來源字幕原譯、第二句手寫），`names`／`anchors`／`forbid` 給規則層；`authored-v1.json` 是手寫半邊的唯一來源，`scripts/build-golden.py` 下載兩對字幕、以時間碼對齊（ToS 第 9 句時間碼差 0.8 秒，退回同序號）、合併輸出。Sita 84 句是手挑 index 白名單（英文檔沒有 ♪ 標記，偵測不可靠）。整份 CC BY-SA 4.0，每行帶 `attribution`。
- **規則層**：重用 `subtitle.CheckChunk`（missing／empty／echoed／simplified_leak）＋新寫 `name_mismatch`／`time_shift`／`forbidden_term`。🔗 AC Drift：`forbidden_term` 是 AC #2 沒列的加項——`simplified_leak` 只抓簡體字元，抓不到「軟件／視頻」這種繁體寫法的陸港用語，少了它 `simplified_bait`／`tw_lexicon` 兩類陷阱等於沒考。
- **裁判**：`judge-v1` rubric（0／1／2 沿 eval-1；台灣口語、短、不翻譯腔是原則不是任何受版權文字）、一批 20 句、壞回覆重試一次、缺句／越界分數一律 `ErrJudgeResponse`；兩次都壞的批次計 `unjudged` 並以 0 分入帳，不會憑空消失。`judge_note` 寫自評偏誤。
- **`cmd/grade`**：真 `services.TranslationService`（同 prompt／詞庫／`PromptVersionFor(level)`），一個 `ai.Budget` 蓋翻譯＋裁判，預設 ≤$0.50；**分 20 句一段送翻譯**——`TranslateWithGlossary` 在預算哨兵觸發時整段回 nil（它的續跑靠 segment cache，這裡沒有），不分段的話第 180 句停會丟掉前 179 句的錢；代價只是段界的 5 句上下文重置。服務對失敗批次「保留英文」的輸出照樣送進規則層 → 記成 `echoed`，不假裝漏翻。交付文字（OpenCC s2twp → 詞庫）給裁判看，原始輸出給規則層看，與管線閘門位置一致。只收 Claude 模型（翻譯路徑只到 Claude，`GeminiProvider` 沒實作 `TextCompleter`）。預算打到上限 exit 3、`grade: incomplete`。
- **測試** 24 條：樣本 schema／組成／每類陷阱 ≥3／參考譯法無簡體；`Validate` 六種壞樣本；規則層每類正反例；等級門檻九組邊界（含 eval-1 兩個實測點）；裁判解析四種壞回覆＋重試路徑；`Run` 六條（規則 0 分與裁判並行、預算停＝incomplete 非 error、硬失敗＝error、裁判預算停、壞裁判計 unjudged、無裁判）；`cmd/grade` 四條（flag 驗證含拒 Gemini、假 provider 端到端驗版本三元組與成本、預算上限、自訂樣本＋無裁判＋無 key）。
- 全量：`go build／vet` 綠、`go test ./...` 全綠、prettier 綠。staticcheck 本機 binary 是 go1.25 建的、吃不下 go1.26 stdlib（`fips140only_go1.26.go`），對新套件掃過乾淨；兩條 U1000（`config.go:357`、`processor.go:198`）在 main 上就有、本單未碰。

### Discovery Triage

- 無新單。`backlog-gemini-translation-dispatch` 既有條目補一句：解掉後 `cmd/grade` 的 Claude-only 檢查要一併放開。

### File List

- apps/api/internal/eval/golden.go、rules.go、grade.go、judge.go、run.go（＋ 5 份 _test.go）
- apps/api/internal/eval/golden/golden-v1.jsonl、authored-v1.json、README.md
- apps/api/cmd/grade/main.go、main_test.go
- scripts/build-golden.py
- _bmad-output/implementation-artifacts/sub-7-8a-golden-sample-and-grader.md、sub-7-8b-model-ratings-embed-ci.md、sub-7-8c-local-preview-run.md、sub-7-8-model-ratings-feed.md、sprint-status.yaml

## Change Log

| Date       | Change                                        |
| ---------- | --------------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-8 拆出）；四項裁定入檔。 |
| 2026-10-01 | dev-story（Amelia）→ review。 |
| 2026-10-01 | PR #640 合併（Alexyu）→ done。 |
