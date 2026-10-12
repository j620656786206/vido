# Disc：AI 帳號沒錢時，Vido 當成「暫時塞車」一直重試

Status: done

**Source:** `disc-2026-09-asr-quota-429-treated-as-transient`（2026-09-19 由 `disc-2026-09-generation-resume-b-asr-chunk-store` 真機驗證時發現，P3；證據是 NAS log 2026-09-18 23:33）。2026-10-13 Alexyu：「接著做 disc-2026-09-asr-quota-429-treated-as-transient」。

## 查到的事（2026-10-13，Bob 逐一開檔確認，main `0efac393`）

### OpenAI 聽寫（`apps/api/internal/ai/whisper.go`）

帳號沒額度時，OpenAI 回 HTTP 429，內容是 `{"error":{"type":"insufficient_quota","code":"insufficient_quota",…}}`。429 這個號碼，同時也代表「一下子打太多次，等等再來」。Vido 只看號碼，不看內容，所以出了三個問題：

1. **白白重試。** `postTranscriptionWith`（`whisper.go:521-569`）把所有 429 都交給 `isTransientStatus`（`retry.go:27-29`，429 或 5xx 就算暫時性錯誤），結果一段音訊要打 3 次，中間等 1 秒＋2 秒。
2. **誤關幻聽過濾。** 重試完還是 429，就會走到 `transcribeVerbose`（`whisper.go:324-334`）。這裡把所有 4xx 都當成「這個引擎不支援 verbose_json」，呼叫 `markVerboseUnsupported`，造成兩個後果：
   - 這個連線之後**一直**改用最簡單的 SRT 格式，幻聽過濾整個關掉，log 寫 `falling back to plain SRT — hallucination filtering disabled`。
   - 接著再用 SRT 格式打一次，又是 3 次 429。

   結果是：一段音訊打 6 次請求才失敗，而且直到重開 Vido 之前，所有聽寫都沒有幻聽過濾。401／403（金鑰錯誤）也會走到同一條路。
3. **看不懂的錯誤。** 錯誤訊息是 `ErrWhisperAPIError: status 429 — <整包 JSON>`。管理字幕視窗的失敗訊息走 `SoloRunner.failureMessage`（`apps/api/internal/subtitle/solo_runner.go:441-457`），只寫「字幕生成失敗」，底下的詳細說明是整包 JSON。

### Claude 翻譯（`apps/api/internal/ai/claude.go`）

- Anthropic 帳號沒額度時回 HTTP 400，訊息是「Your credit balance is too low to access the Anthropic API…」。
- `classifyErr`（`claude.go:251-285`）對 400 不重試，這是對的。但它只回 `AI_PROVIDER_ERROR: status 400`，使用者看不出原因。
- Claude 的 429 是真的「打太快」，現在照樣重試，這個不改。

### 不會被影響的地方

- 字幕流程的「暫時錯誤就保留英文、繼續做」只看 `ErrAIRetriesExhausted`，而且明確排除 `ErrAIQuotaExceeded`（`apps/api/internal/subtitle/pipeline.go:1150`）。新的錯誤會被當成 `ErrAIQuotaExceeded` 處理，所以字幕流程會立刻停，不會硬做出半英文的字幕。
- Gemini 的 429（`RESOURCE_EXHAUSTED`）同時代表「打太快」和「額度用完」，從回應分不出來是哪一種。這張不動。

## Story

身為自己付 AI 費用的人，
我希望帳號沒錢時，Vido 馬上停下來，並直接告訴我「去哪裡儲值」，
這樣不用白等、不會偷偷關掉幻聽過濾，也不用去讀一包 JSON。

## Acceptance Criteria

1. **新的錯誤型別**：`ai` 套件新增 `CreditExhaustedError{Provider, Code, BillingURL}`。
   - `Error()` 是短短一行，例如 `AI_CREDIT_EXHAUSTED: openai account has no credit left (insufficient_quota)`，**不能**包含整包回應內容。
   - `errors.Is(err, ErrAIQuotaExceeded)` 要成立，這樣字幕流程原本的「立刻停」規則照樣適用。
   - `errors.As` 拿得到 Provider 和 BillingURL。
2. **OpenAI 聽寫**：回應內容（不分大小寫）含下面任一個詞時，判成沒額度，**不重試**，直接回 `CreditExhaustedError{Provider:"openai", BillingURL:"https://platform.openai.com/settings/organization/billing"}`：
   - `insufficient_quota`
   - `billing_hard_limit_reached`
   - `credit_balance_exhausted`

   這個判斷看的是回應內容，不是 HTTP 狀態號碼：OpenAI 用 429 和 400 回這種錯誤都看過。只有「打太快」的 429（內容沒有這三個詞）照樣重試。
3. **不再誤關幻聽過濾**：`transcribeVerbose` 只有在引擎**真的不支援這個格式**時，才改用最簡單的 SRT。下面這三種情況不改用，直接把錯誤往上丟：
   - 沒額度
   - 429（重試完還是 429）
   - 401／403（金鑰錯誤）

   這些都是帳號的問題，跟引擎支不支援 verbose_json 無關。
4. **Claude 翻譯**：400 的回應內容含 `credit balance`（不分大小寫）時，回 `CreditExhaustedError{Provider:"anthropic", BillingURL:"https://console.anthropic.com/settings/billing"}`，同樣不重試。其他 400 照舊。
5. **使用者看得懂**：`failureMessage` 碰到 `CreditExhaustedError` 時：
   - 訊息改成「OpenAI 帳號沒有額度了，儲值後再按一次生成」（Anthropic 則寫「Anthropic 帳號…」）。
   - 詳細說明放儲值網址。
6. **測試**（每一條都要有）：
   - OpenAI 回 429＋`insufficient_quota`：**只打 1 次**請求；錯誤是 `CreditExhaustedError`；錯誤字串裡沒有 JSON；之後的聽寫仍然走 verbose_json（沒有改用 SRT）。
   - OpenAI 回一般 429（打太快）：照樣重試 3 次；之後仍然走 verbose_json（沒有改用 SRT）。
   - OpenAI 回 401：不改用 SRT，錯誤照樣往上丟。
   - OpenAI 回 400＋`unsupported response_format`：照舊改用 SRT（原本的測試要保持綠燈）。
   - Claude 回 400＋credit balance：判成 `CreditExhaustedError`，不重試。
   - `failureMessage`：錯誤經過好幾層包裝後，仍然顯示上面那句中文和儲值網址。
7. **檢查**：下面這些全綠：
   - `go build／vet／test ./...`
   - `~/go/bin/staticcheck ./...`
   - `pnpm run lint:all`
8. **正式機實測：不做。** 要讓帳號真的沒錢才測得到，靠單元測試證明。

## Tasks / Subtasks

- [x] T1 新檔 `ai/credit_exhausted.go`：`CreditExhaustedError`、兩個儲值網址、`openAICreditExhausted`（AC #1）
- [x] T2 `ai/whisper.go`：判斷沒額度、不重試；`transcribeVerbose` 不再因帳號問題改用 SRT（AC #2、#3）
- [x] T3 `ai/claude.go`：400＋credit balance（AC #4）
- [x] T4 `subtitle/solo_runner.go`：`failureMessage`（AC #5）
- [x] T5 測試＋檢查（AC #6、#7）

全部是後端，動 4 個檔案，不用拆單。

## Dev Notes

### 不要做的事

- 不要把所有 429 都當成沒額度。真的「打太快」時，重試才是對的。
- 不要在錯誤字串裡放整包回應。整包回應原本就會寫進 log（`whisper.go:559-562` 的 `Whisper API error`）。
- 不要動 Gemini（理由見上）。
- 不要動批次。帳號沒錢時，批次會繼續跑下一部，但每一部都只打 1 次請求就失敗，很快就結束。要不要「沒錢就整批停下來」是產品決定，牽涉新的批次狀態和畫面文案，另外處理。

### Time-dependent visual coverage

N/A — 純後端，沒有碰到會讀現在時間的畫面元件。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Debug Log References

- 突變測試：把 `transcribeVerbose` 的 `!isAccountRejection(...)` 拿掉，3 條新的 Whisper 測試全紅：
  - 沒額度那條：改用 SRT 又打了一次。
  - 打太快那條：被誤關過濾。
  - 401 那條：多打了一次 SRT。

  還原後全綠。

### Completion Notes List

- 型別放在新檔 `credit_exhausted.go`，沒有放進 `types.go`，型別和判斷集中一處。它 `Unwrap` 成 `ErrAIQuotaExceeded`。Whisper 那邊再包一層 `ErrWhisperAPIError`，原本認這個錯誤的地方照樣認得到。
- `transcribeVerbose` 新增 `isAccountRejection`：沒額度、401、403、429 都不改用 SRT。原本「400＋unsupported response_format 改用 SRT」的測試仍然是綠的。
- Claude：400 而且內容含 `credit balance`，回 `CreditExhaustedError`（不重試）。其他 400 照舊回 `AI_PROVIDER_ERROR`。
- 管理字幕視窗：
  - 訊息改成「OpenAI 帳號沒有額度了，儲值後再按一次生成」。
  - 詳細說明改成同一句，加上儲值網址。開頭是中文，所以畫面會當一般句子顯示，不用等寬字體。
- **AC 外多做的兩處（同一件事的另外兩個出口，都很小）：**
  - 設定頁「測試金鑰」：帳號沒額度時，原本顯示「無法驗證金鑰」，現在顯示「金鑰有效，但帳戶沒有額度了」，並附儲值網址。錯誤代碼仍然是 `AI_QUOTA_EXCEEDED`，前端不用改。
  - 生成工作區的紀錄列（`generationEventCopy.ts`）：多一行對照，看到 `AI_CREDIT_EXHAUSTED` 或「帳號沒有額度」就顯示「AI 帳號沒有額度」。不加的話，詳細說明改成中文句子後，這一列只會寫「生成失敗」。
- 檢查全綠：
  - `go build ./...`、`go vet ./internal/...`、`go test ./...`
  - `~/go/bin/staticcheck ./...`
  - `pnpm run lint:all`（0 errors）
  - `tsc --noEmit`
  - `vitest src/components/subtitle`：21 檔、889 個測試

### Senior Developer Review (AI) — 2026-10-13

結論：0 High／0 Medium／2 Low，皆接受。

- **L1（接受）** 批次遇到帳號沒錢時，會繼續跑下一部。每一部都只打 1 次請求就失敗，很快結束，但還是一整排「失敗」。要不要「沒錢就整批停下來」是產品決定，見 Discovery Triage。
- **L2（接受）** 自架語音辨識的引擎如果用 429 表示別的意思，現在不會再改用 SRT。這是對的：429 本來就不是「格式不支援」。

### Discovery Triage

- **設計稿可能缺一行**：生成工作區紀錄列的失敗文案定義在 `.pen` 的 F11-SPEC-LOG（`generationEventCopy.ts` 開頭的註解這樣寫）。新的「AI 帳號沒有額度」要不要補進設計稿，交給 Sally。這張沒有動 `.pen`。
- **批次沒錢就停**：同 L1。只有真的遇到才值得做，不立新單，記在這裡。

### File List

- `apps/api/internal/ai/credit_exhausted.go`（新）
- `apps/api/internal/ai/credit_exhausted_test.go`（新）
- `apps/api/internal/ai/whisper.go`
- `apps/api/internal/ai/claude.go`
- `apps/api/internal/subtitle/solo_runner.go`
- `apps/api/internal/subtitle/solo_runner_test.go`
- `apps/api/internal/handlers/key_settings_handler.go`
- `apps/api/internal/handlers/key_settings_handler_test.go`
- `apps/web/src/components/subtitle/generationEventCopy.ts`
- `apps/web/src/components/subtitle/generationEventCopy.spec.ts`
- `_bmad-output/implementation-artifacts/disc-2026-09-asr-quota-429-treated-as-transient.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
