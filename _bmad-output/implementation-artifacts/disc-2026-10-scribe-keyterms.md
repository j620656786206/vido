# Disc：ElevenLabs 把 Jerlamarel 聽成「Chilam Balam」——先把這部片的人名告訴它

Status: review

**Source:** 2026-10-14 v0.2 發版前 NAS 實測（See S01E02 前 10 分鐘、ElevenLabs Scribe）。7:42～7:52 三句「Jerlamarel!」被聽成「Chilam Balam」（一本馬雅古書），翻成「奇蘭巴倫」；詞彙表裡 Jerlamarel→謝拉馬威 對不上因為英文就錯了。Alexyu：「傑拉馬瑞跟奇蘭巴倫是同一人」，選 **A：做，預設開，餵詞彙表＋演員表的英文名，每小時多 4 美分**。

## 查到的事（2026-10-14，main `01ba0b5d`）

- ElevenLabs speech-to-text API 有 `keyterms`（list of string）：讓模型偏向聽出這些字；上限 1000 個、每個 50 字以內、最多 5 個詞、不能含 `< > { } [ ] \`；**加收 20%**；超過 100 個時每次請求至少計 20 秒（API reference 與 keyterm-prompting 指南，2026-10-14 查）。batch 的 `scribe_v2` 支援。
- `apps/api/internal/ai/scribe.go:45`（改前）明寫「NOT sent on purpose: keyterms」——當時的理由是 Whisper 的人名提示六次都沒修到人名、還會把名單唸出來；但那是 Whisper 的「提示文字」機制，Scribe 的 keyterms 是詞彙表，不會被唸出來。
- 人名來源已經有：`transcription_asr_prompt.go:121` `asrPromptFor` → TMDb 演員表的角色名＋可信來源的詞彙（排除 `subtitle` harvest，免得把上次聽錯的餵回去）→ `ai.BuildASRPrompt` 截到 12 個／200 字。Whisper 提示預設關（`VIDO_ASR_NAME_PROMPT=false`，`config.go:212`）。
- 名單怎麼到 client：`transcription_service.go:1038` 用 ctx（`ai.WithASRPrompt`）；Scribe 的 `TranscribeDetailed` 讀 `ASRPromptFromContext` 只拿來做幻聽過濾（`scribe.go:228`）。
- 計費：`scribe.go:218` `RecordASRWithRate(seconds, ScribePerMinuteUSD())`；估價 `budget.go:179` `ASRPerMinute("elevenlabs")`。`scribe_test.go:326` 鎖定 0.22/小時。
- 文件寫 0.22 的地方：README `:91`、`:114`；Unraid 指南兩版 `:66`；`docs/docker-compose.nas.yml:30`；`docker-compose.yml:68`；`unraid-template/vido.xml:168`（`:179` Whisper 那條誤寫 0.22–0.36）；`ApiKeysForm.tsx:97`＋spec `:106`；`deployment.md:169`。

## Acceptance Criteria

1. 用 ElevenLabs 聽寫時，請求帶 `keyterms`：這部片的角色名（TMDb 演員表，billing 順序）＋可信詞彙來源的英文名，去重、照 ElevenLabs 規則清理、最多 100 個；Whisper 不受影響。
2. 計費誠實：有送 keyterms 的請求以 0.22×1.2 記帳；沒送（沒任何名字）以 0.22 記。估價（選片清單、單部估價）用含 20% 的費率（偏高不偏低）。
3. 預設開；`VIDO_ASR_KEYTERMS=false` 關掉。Whisper 的 `VIDO_ASR_NAME_PROMPT` 維持預設關、各自獨立。
4. 文件與畫面的「每小時約 US$0.22」改成「約 US$0.26（含人名提示）」；`.env.example`、`docs/development.md` 補變數。
5. 測試：keyterms 由 ctx 進 multipart（一個名字一個 `keyterms` 欄位）、費率 ×1.2、沒名字不送且基本費率；清理規則（去重、<50 字、≤5 詞、去掉禁用字元、上限 100）；`asrNamesFor` 是完整名單（不受 Whisper 的 12 個上限）；估價費率含 20%。
6. 正式機驗證：同一段片段**重聽**（清掉 asr_chunk 快取，約 US$0.05），7:42～7:52 要變成「謝拉馬威」；log 有 `asr keyterms built names=N` 與 `keyterms=N`。

## Tasks / Subtasks

- [x] T1 `ai`：`WithASRKeyterms`／`ASRKeytermsFromContext`／`NormalizeScribeKeyterms`；`scribe.go` 送欄位、計費 ×1.2、log；`budget.go` `ScribeKeytermsSurcharge`／`ScribePerMinuteWithKeyterms`、`ASRPerMinute("elevenlabs")` 含附加費（AC 1、2）
- [x] T2 `services`：`asrPromptFor` 拆成 `asrPromptSources`＋`asrNamesFor`；call site 兩個開關各自處理；`SetASRKeyterms`；config `VIDO_ASR_KEYTERMS`；main 接線（AC 1、3）
- [x] T3 文件與畫面文案（AC 4）
- [x] T4 測試（AC 5）
- [ ] T5 NAS 重聽驗證（AC 6，合併後）

## Dev Notes

- 名單順序沿用 `asrPromptNames`（角色名在前），所以就算超過 100 個，被截掉的是後面的詞彙，不是主角。
- 不改 Whisper 路徑的任何行為；`scribe.go` 的幻聽過濾仍讀 Whisper 提示（有開才有）。
- `ElevenLabs` 說 keyterms 給 Scribe v2 batch 與 realtime 都支援；只用 batch。

### Time-dependent visual coverage

N/A — 後端＋一行設定頁文案（`settings-api-keys-form` visual 基準的文字會變：0.22→0.26，CI 會判 diff；若超過門檻就刪基準重拍）。

### References

- ElevenLabs API reference `speech-to-text/convert`（keyterms 規則、20% 附加費）、guides `keyterm-prompting`。
- `scribe.go`、`asr_prompt.go`、`budget.go`、`transcription_asr_prompt.go:121-185`、`transcription_service.go:1030-1050`、`config.go:135-142`。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- 見 Tasks。費率常數 `ScribePerMinute()` 仍是 0.22（既有測試鎖定），新增 `ScribePerMinuteWithKeyterms()`＝0.264；`ASRPerMinute("elevenlabs")` 回含附加費的費率，`asr_scribe_default_test` 對應改一行。

### Discovery Triage

N/A — no out-of-scope work discovered.

### File List

- `apps/api/internal/ai/{asr_prompt.go,scribe.go,scribe_test.go,budget.go}`、`apps/api/internal/services/{transcription_service.go,transcription_asr_prompt.go,transcription_asr_prompt_test.go,asr_scribe_default_test.go}`、`apps/api/internal/config/config.go`、`apps/api/cmd/api/main.go`
- `README.md`、`docs/{deployment.md,development.md,unraid-installation-guide.md,unraid-installation-guide.zh-TW.md,docker-compose.nas.yml}`、`docker-compose.yml`、`unraid-template/vido.xml`、`.env.example`、`apps/web/src/components/settings/{ApiKeysForm.tsx,ApiKeysForm.spec.tsx}`
