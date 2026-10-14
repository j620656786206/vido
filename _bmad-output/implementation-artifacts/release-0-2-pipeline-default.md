# Release 0.2：AI 字幕管線改成預設開啟（不用再手動設 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`）

Status: review

**Source:** 2026-10-14 party mode（founder-pm／John／Bob／Sally／founder-growth）。FB 招募文 26 則留言、0 個安裝；貼文自己寫「安裝說明有幾個地方還沒跟上」。Alexyu 裁定 A：這週不開新功能，先發 v0.2；第一張單就是把管線模式預設改成 `pipeline`（Alexyu 第二次選 A）。

## 為什麼

新裝的人照任何一份安裝指南、compose 或 Unraid 模板裝起來，`VIDO_SUBTITLE_PIPELINE_MODE` 都不會被設到，於是 Vido 停在 `legacy`：按「生成字幕」只會走「聽聲音」（付費 ASR），**片內的英文字幕不會被讀**。招募文主打的那句「有英文字幕就翻成繁中」在預設安裝下根本不會發生。README 第 76 行要人手動加這個變數，還寫著「之後會改成預設」——現在就改。

## 查到的事（2026-10-14，main `ee51f9c7`）

- 預設值在 `apps/api/internal/config/config.go:266`：`cfg.loadString("VIDO_SUBTITLE_PIPELINE_MODE", SubtitlePipelineModeLegacy)`；:262-265 的註解說明為什麼亂值要讓啟動失敗。
- 常數與驗證在 `apps/api/internal/config/subtitle_pipeline.go`：`:16` legacy、`:21` pipeline、`:28-31` 合法集合、`:34-40` `validateSubtitlePipelineMode`、`:46-48` `SubtitlePipelineEnabled()`（唯一的讀法）。`:13-15` 的註解寫「This is the default」——要跟著改。
- 測試 `apps/api/internal/config/subtitle_pipeline_test.go:22-26`「unset defaults to legacy」要改成 pipeline；`:27-31` legacy 明設仍接受、`:37-46` 亂值／大小寫失敗，維持。
- 整個 repo 讀這個值的地方只有 `apps/api/cmd/api/main.go:994`、`:1009`（開機 log 印 mode）和 `config.SubtitlePipelineEnabled()`（grep `SubtitlePipelineMode` 全 `apps/api`，本次 session；`batch.go`／`subtitle_pipeline_handler.go` 只用 enabled 布林，沒碰字串）。
- 文件提到的地方（grep `PIPELINE_MODE` 全 repo 的 `*.md`／`*.yml`／`*.xml`／`.env.example`，本次 session）：
  - `README.md:76`「加上 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`…（之後會改成預設）」、`:100` 設定表那列「要用 AI 字幕就填 `pipeline`」。
  - `docs/deployment.md:96`「This is **off by default**」、`:100` 表格 Default `legacy`、`:117-119` 亂值說明、`:120-122`「`pipeline` without `CLAUDE_API_KEY`…」、`:123-128` legacy／pipeline 的批次行為、`:222` 沒有設定頁開關要重啟。
  - `unraid-template/vido.xml:152-162`：`Default="legacy"`、Description 寫 legacy 是 default、值 `>legacy<`。
  - `.env.example`：**沒有這個變數**（grep `SUBTITLE` 零筆）。
  - `docker-compose.yml:54-62`、`docs/docker-compose.nas.yml:13-28`：環境變數區塊都沒有這個變數。
- 正式機（Alexyu 的 NAS）本來就明設 `pipeline`（容器 env 清單含 `VIDO_SUBTITLE_PIPELINE_MODE`，2026-10-14 ssh 查），改預設對它沒影響。

## Acceptance Criteria

1. 沒設 `VIDO_SUBTITLE_PIPELINE_MODE` 時，`config.Load()` 的 `SubtitlePipelineMode` 是 `pipeline`，`SubtitlePipelineEnabled()` 回 true。
2. 明設 `legacy` 仍是 `legacy`（舊行為還在，只是不再是預設）；設亂值（`pipelien`、`Pipeline`）仍啟動失敗，錯誤訊息含變數名。
3. `subtitle_pipeline.go` 與 `config.go` 的註解不再說 legacy 是預設；說明改成「預設 pipeline；設 `legacy` 回到只搜尋字幕的舊引擎」。
4. 文件同一個 PR 一起改，改完 grep 整個 repo（排除 `_bmad-output/`、`.claude/`）不再有「要加／要填 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline` 才會…」、「off by default」、「Default `legacy`」這類句子：
   - `README.md:76` 那條改成「預設就會先讀片內英文字幕；想回到只搜尋字幕的舊模式才設 `legacy`」（或直接拿掉，把「要先確認」清單剩下兩條保留）；`:100` 表格列改成「選填，預設 `pipeline`」。
   - `docs/deployment.md:96`、`:100`、`:117-128`、`:222` 對應改；`:120-122`「需要 Claude 金鑰」那句改成「Claude 或 Gemini 擇一」（這句本來就過時，順手；金鑰其他過時項留給 `release-0-2-docs-refresh`）。
   - `unraid-template/vido.xml:152-162`：`Default="pipeline"`、值 `>pipeline<`、Description 改。
   - `.env.example` 補一段：變數名、預設 `pipeline`、`legacy` 的意思。
   - `docker-compose.yml`、`docs/docker-compose.nas.yml`：加一行註解說明預設 pipeline、要關才設 `legacy`（不強制寫進 environment）。
5. `go test ./internal/config/...` 綠；`go test ./...`、vet、staticcheck、prettier 綠。
6. 開機 log（`main.go:994`／`:1009`）照舊印 mode，不用改。

## Tasks / Subtasks

- [x] T1 `config.go:266` 預設改 `SubtitlePipelineModePipeline`；`:262-265` 與 `subtitle_pipeline.go:5-21` 註解改（AC 1–3）
- [x] T2 `subtitle_pipeline_test.go:22-26` 期望改 pipeline，case 名改「unset defaults to pipeline」；補一條 `SubtitlePipelineEnabled()` 未設時為 true（AC 1–2）
- [x] T3 文件：README、deployment.md、vido.xml、.env.example、兩份 compose（AC 4）
- [x] T4 全套檢查（AC 5）

## Dev Notes

- 不做：拿掉 `legacy`；加設定頁開關（`deployment.md:222` 的「沒有 UI 開關」維持）；金鑰／`:ro` 掛載等其他文件過時項——那是 `release-0-2-docs-refresh`（Paige）的範圍，審計清單在該單。
- 文件同時有中英版的只有 Unraid 指南（`docs/unraid-installation-guide*.md`），它們沒提到這個變數，不用動；README 只有中文版。
- 提交訊息照 CLAUDE.md 的白話格式：「原本會怎樣：新裝的人沒設變數，按生成字幕只會聽聲音、英文字幕不會被讀」。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端設定＋文件）。

### References

- `apps/api/internal/config/config.go:262-270`、`subtitle_pipeline.go:1-48`、`subtitle_pipeline_test.go:15-60`
- `README.md:74-105`、`docs/deployment.md:94-130`、`:218-223`、`unraid-template/vido.xml:152-162`、`docker-compose.yml:48-62`、`docs/docker-compose.nas.yml:10-30`
- party mode 2026-10-14 裁定：v0.2 範圍五件事（1 本單、2 `release-0-2-docs-refresh`、3 README 定位段、4 #783／#786 NAS 實測、5 tag＋release notes＋About）

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- `config.go:266` 預設改 `SubtitlePipelineModePipeline`；`config.go`／`subtitle_pipeline.go` 的註解改成「預設 pipeline、`legacy` 是退回舊引擎的開關」，亂值的例子從 `pipelien` 換成 `legcay`（現在怕的是「關不掉」不是「開不了」）。
- 測試：`TestLoad_SubtitlePipelineMode` 的「unset」case 改期望 pipeline；新增 `TestSubtitlePipelineEnabled_DefaultOn`（沒設→true、設 legacy→false）。`legacy` 明設、亂值、大小寫三條維持。
- 文件：README 的「要先確認」清單從三條變兩條（拿掉手動設變數那條，金鑰那條改成 Claude 或 Gemini 擇一）、設定表多 Gemini 列、管線列改「選填，預設 pipeline」；`docs/deployment.md` 表格、「off by default」、亂值說明、「需要 Claude 金鑰」四處改；`unraid-template/vido.xml` 預設與說明改 pipeline；`.env.example` 補變數；兩份 compose 加註解。
- grep 全 repo（排除 `_bmad-output/`、`.claude/`）：不再有「加上 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`」「off by default」「Default="legacy"」「只會聽聲音」。
- `go test ./...`、vet、staticcheck、prettier 全綠。改動不碰前端，沒有 visual。
- 自審（adversarial）：讀這個值的只有 `main.go` 的 log 和 `SubtitlePipelineEnabled()`；`cost_consent_test.go`／`process_item_freeonly_test.go`／handler 測試都自己設 `Config{SubtitlePipelineMode: …}`，不吃預設，全套跑過沒紅。

### Discovery Triage

N/A — no out-of-scope work discovered（金鑰／`:ro` 等文件過時項是開工前就立好的 `release-0-2-docs-refresh`）。

### File List

- `apps/api/internal/config/{config.go,subtitle_pipeline.go,subtitle_pipeline_test.go}`
- `README.md`、`docs/deployment.md`、`.env.example`、`docker-compose.yml`、`docs/docker-compose.nas.yml`、`unraid-template/vido.xml`
