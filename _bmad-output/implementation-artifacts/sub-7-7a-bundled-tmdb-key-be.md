# Story sub-7-7a: 內建 TMDb 金鑰——後端、打包、文件

Status: review

<!-- SM Bob create-story 2026-09-30，由 sub-7-7 拆出（依 feedback_split_oversized_stories：單獨後端就比上一張同類單大）。
     ⚖️ Alexyu 2026-09-30 對設計裁定 A：金鑰順序 settings→env→bundled、TMDb client 每次請求才問金鑰（不抄 holder）、
     BuildKit secret 不用 build-arg、順手堵 log 漏金鑰、拆成 7-7a（BE＋CI＋docs）與 7-7b（FE 文案）。
     行號為 main `3e6e354a`。原單 sub-7-7 的行號有錯（key_resolver.go:99 是 secretNameFor、source 沒有 "settings" 值、`sanitizeAttr` 不存在），本張以實際碼為準。 -->

## Story

身為第一次裝 Vido 的人，
我不用去申請 TMDb 開發者帳號，海報和中文片名就直接能用；
想用自己的金鑰時，設定頁填了就馬上生效，不用重開。

## 背景（查到的事）

- `services/key_resolver.go:27-36`：`KeySource` 只有 `secret`／`env`／`none`；`:150-154` env 之後直接回 `none`——內建層要插在這裡。
- `cmd/api/main.go:213-219`：TMDb client 直接吃 `cfg.TMDbAPIKey`（env），**從不經過 resolver**；resolver 在 `:608` 才建。所以精靈（dsr-13）和設定頁存的 TMDb 金鑰**連重啟都不會生效**。
- `tmdb/client.go:99-105, 213`：金鑰在 `NewClient` 時凍進 struct，`buildURL` 每次拿 `c.apiKey`。`SearchClient()`／`CreditsClient()`／`VideosProvider()` 把同一個裸 client 交給十幾個消費者（`main.go:223-922`），換 holder 會漏。
- `tmdb/client.go:174-181`：網路層失敗時 `*url.Error` 帶完整網址（含 `api_key=`），原樣進 stdout log 與 `Ping → RecordError → /settings/services` 的錯誤訊息。內建之後等於把 Alexyu 的金鑰印在每個使用者的 log。
- `tmdb/client.go:188-195`：429 只 `slog.Warn`，沒有任何 health hook；`health/monitor.go` 只靠 5 分鐘一次的 Ping。`models/degradation.go:159-171`：`rate_limited` 只在 degraded 顯示，錯 3 次變 down 就顯示「disconnected」。
- `Dockerfile:77-80`／`.github/workflows/docker.yml:139-171`：`go build -ldflags="-s -w"`，沒有任何 `-X`、沒有 build-args／secrets；provenance `mode=max`（build-arg 會進 attestation）。repo 已有 `TMDB_API_KEY` secret（test.yml:428 用）。
- 前置 `sub-6-9-tmdb-attribution` 已 done（PR #385／#386）。

## Acceptance Criteria

1. **內建方式。** `config.bundledTMDbKey` 只能由 `-ldflags "-X github.com/vido/api/internal/config.bundledTMDbKey=…"` 注入；原始碼、`go run`、CI 測試 binary 一律空字串（測試鎖住）。Dockerfile 以 **BuildKit secret mount** 讀 `/run/secrets/tmdb_bundled_key`，沒有就空；`docker.yml` 以 `secrets:` 傳 `TMDB_BUNDLED_KEY || TMDB_API_KEY`。**原始碼與 git 歷史零 key。**
2. **解析順序。** `KeyResolver`：settings（secret）→ env → bundled；新 `KeySourceBundled = "bundled"`（[@contract-v1→v2] 加值）。只有 TMDb 有內建層；空白視為沒有。
3. **免重啟生效。** TMDb client 每次請求向 resolver 問金鑰（`KeyProvider`），同一個 client 物件連續三次請求可以分別送出內建→自填→（清除後）內建的金鑰；精靈與設定頁存的 TMDb 金鑰不用重開就生效（吸收 `backlog-tmdb-runtime-key-resolution`）。
4. **沒金鑰不打網路。** resolver 回空 → client 直接回 `TMDB_UNAUTHORIZED: … not configured`，不消耗 rate-limit token；狀態頁因此顯示「未設定」而不是故障。
5. **log 不漏金鑰。** 網路層錯誤的 `*url.Error.URL` 把金鑰換成 `REDACTED`，錯誤鏈保留（`errors.Is(err, context.Canceled)` 仍成立）；設定頁 `List` 對 bundled 不回 `Masked`；boot log 只印 `TMDB_BUNDLED_KEY_present=true/false`。
6. **限流降級。** client 有 `RequestObserver`；`main.go` 接到 `healthMonitor`：429 → `UpdateServiceHealth(tmdb, err)`，成功 → 清掉；`ToServiceStatus` 在 down 且訊息是 rate-limit 時仍顯示 `rate_limited`（不是 disconnected）。
7. **文件。** `docs/deployment.md` 新增「The bundled TMDb key」小節（非商業授權、attribution 已內建、限流時自填、原始碼編譯沒有內建）並改寫 hot-reload 段；`README.md` 環境變數表與 `.env` 註解；`.env.example` 註解。
8. **測試。** resolver 六條（bundled 生效／env 勝／secret 勝／無 bundled 仍 none／只有 TMDb／空白視為無）；config 兩條；client 八條（provider 每次問、無 provider 沿用靜態、空金鑰不打網路＋通知 observer、provider 錯誤上拋、transport 錯誤不漏金鑰、錯誤鏈保留、observer 看到成功與 429、nil observer 安全）；models 一條（down＋rate-limit → rate_limited）；services 真加密整合一條（同一 client：內建→自填→清除→內建，List 回 bundled 且無 Masked）。`go build ./...`、`go vet`、api 全量綠。

## Tasks / Subtasks

- [x] Task 1 — `config/bundled.go` ＋ resolver 內建層 ＋ `main.go` resolver 前移（AC #1, #2）
- [x] Task 2 — `tmdb.Client` `KeyProvider`／`RequestObserver`／redaction／not-configured 短路（AC #3, #4, #5）
- [x] Task 3 — 429 → health（observer 接線、`ToServiceStatus` down 分支）（AC #6）
- [x] Task 4 — Dockerfile secret mount ＋ docker.yml `secrets:`（AC #1）
- [x] Task 5 — 文件（AC #7）
- [x] Task 6 — 測試、`go vet`、api 全量、lint:all（AC #8）

## Dev Notes

- **為什麼不抄 `ClaudeProviderHolder`**：TMDb 的裸 client 被十幾個消費者直接持有（`SearchClient()` 等），holder 換物件會漏；改成 client 內部每請求解析金鑰，一個地方改完全部生效。`asr_provider_holder.go:28-35` 留給「第三個消費者」的 `providerHolder[T]` 抽取問題：TMDb 不需要，維持現狀。
- **為什麼用 secret mount 不用 build-arg**：build-arg 會進 `docker history` 與 `mode=max` 的 provenance attestation；secret mount 只進編好的 binary（`strings` 仍找得到——Jellyfin／Radarr 同樣接受）。
- **cache 陷阱（自審抓到）**：BuildKit 的 layer cache **不看 secret 內容**——本機實測「有金鑰」建過一次後，「沒金鑰」再建直接命中舊 layer，binary 裡還是有金鑰；反過來換了金鑰也會一直用舊的。解法：CI 先算金鑰的 sha256 前 16 碼當 `TMDB_BUNDLED_KEY_FINGERPRINT` build-arg（指紋不是金鑰，128-bit 金鑰的 sha256 不可逆），Dockerfile 在 `RUN` 裡引用它，金鑰一換 cache 就失效。本機三次 build 驗證：有 secret＋指紋 → binary 含金鑰；無 secret（`--no-cache-filter api-builder`）→ 不含；無 secret 沿用預設指紋 → 命中無金鑰的 layer、不含；`docker history --no-trunc` 零外洩。
- **每請求解析的成本**：`resolver.Get` 一次 DB 讀＋AES 解密；TMDb client 本身限 4 req/s，可忽略。
- **FE 缺口留給 7-7b**：`KeySource` TS 型別加 `'bundled'`、`ApiKeysForm` 的「內建」標籤與「儲存後需重啟」文案、精靈跳過警語、`rate_limited` 時的「改用自己的金鑰」提示。在 7-7b 合併前，設定頁對 bundled 會顯示「尚未設定」（`stateLabel` default 分支）——同日出，可接受。
- `apps/api/Dockerfile`（沒有任何 workflow 引用）刻意不動。

### Time-dependent visual coverage

- N/A — 純後端。

### References

- party-mode 2026-09-03 TMDb 條款查證；Party Mode 2026-09-30 設計裁定；`services/key_resolver.go`、`tmdb/client.go`、`models/degradation.go`、`Dockerfile`、`.github/workflows/docker.yml`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 見 Change Log；驗證數字記在 PR。

### Discovery Triage

- 見 PR「沒做的事」。

### File List

- apps/api/internal/config/bundled.go、bundled_test.go、config.go
- apps/api/internal/services/key_resolver.go、key_resolver_test.go、key_resolution_integration_test.go、tmdb_service.go
- apps/api/internal/tmdb/client.go、client_test.go、client_key_test.go
- apps/api/internal/models/degradation.go、service_status_test.go
- apps/api/cmd/api/main.go
- Dockerfile、.github/workflows/docker.yml
- docs/deployment.md、README.md、.env.example
- _bmad-output/implementation-artifacts/sub-7-7a-bundled-tmdb-key-be.md、sub-7-7b-bundled-tmdb-key-fe.md、sub-7-7-bundled-tmdb-key.md、sprint-status.yaml

## Change Log

| Date       | Change                                                       |
| ---------- | ------------------------------------------------------------ |
| 2026-09-30 | create-story（SM Bob，自 sub-7-7 拆出）；dev-story（Amelia）→ review。 |
