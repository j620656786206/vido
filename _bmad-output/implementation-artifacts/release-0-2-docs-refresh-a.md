# Release 0.2 文件翻新 (a)：安裝路徑——照著裝就裝得起來

Status: review

**Source:** 2026-10-14 party mode（founder-growth：貼文自己說「安裝說明還沒跟上，照著裝可能會卡住」，26 則留言 0 安裝，這句就是原因）。Alexyu 裁定 A：v0.2 第一件事是文件。負責：Paige（Technical Writer）。本單只管「新人照著裝會不會卡住」的檔案；`deployment.md`／`PRODUCT.md`／`ROADMAP` 的真相檔在 `release-0-2-docs-refresh-b`。

## 為什麼

一個 agent 把安裝文件逐行對程式，找到約 55 處過時。最傷的三處會讓**每一個**照指南裝的人失敗：

1. **媒體資料夾全部掛 `:ro`**。產生的字幕寫在影片旁邊（`apps/api/internal/subtitle/placer.go:73`「Place writes a subtitle file next to its media file」），唯讀掛載 → 每支字幕都失敗。README 第 78 行自己說「不要加 `:ro`」，但它連到的每份指南和 compose 都加了。
2. **「要 Claude 金鑰」已過時**：Claude 或 Gemini 擇一（#786）、TMDb 金鑰已內建進 image（`Dockerfile:86-91` 把 `bundledTMDbKey` 編進去）、ElevenLabs 選填（#783）、所有金鑰設定頁都能填。
3. **image tag 寫錯**：指南叫人用 `v0.1.0-beta.1`，CI 發的 tag 沒有 `v`（`.github/workflows/docker.yml:285-291`：`{{version}}` → `0.1.1`、`0.1`、`0`，分支 `main`，`sha-xxxx`，`latest`）。

## 查到的事（2026-10-14，main `ee51f9c7`，本人開檔確認的標 ✅；只有 agent 報告的標 ⚠️ dev 開工時先開檔）

### 掛載 `:ro`（✅ 全部本人 grep 確認）
- `docker-compose.yml:17`、`:39-40`（`${MEDIA_PATH:-./media}:/media:ro`，註解「read-only for security」）
- `docs/docker-compose.nas.yml:12`（`/path/to/media:/media:ro`）
- `docs/synology-installation-guide.zh-TW.md:24`「媒體資料夾只需要讀取權限」、`:37` `/volume1/video:/media:ro`
- `docs/qnap-installation-guide.zh-TW.md:23`、`:36` `/share/Movies:/media:ro`
- `docs/unraid-installation-guide.md:51`（Read Only）、`:94`、`:113`；`.zh-TW.md` 同行號（唯讀）
- `unraid-template/vido.xml:58` Description「Mounted read-only」；⚠️ agent 說 `:57` `Mode="ro"`
- `.env.example:304`「This folder will be mounted read-only」
- 注意：`--read-only --tmpfs /tmp`（`vido.xml:15`、Unraid 指南 `:30`、`:112`）是**容器根檔案系統**唯讀，跟媒體無關，**保留**。

### 金鑰與設定（✅ 程式側本人確認）
- `apps/api/internal/config/config.go:216-219`：`ENCRYPTION_KEY`、`VIDO_AUTH_PASSWORD`（「Empty = auth disabled」）。
- 精靈六步（`apps/web/src/components/setup/SetupWizard.tsx:35-54`）：歡迎・qBittorrent・媒體庫・API 金鑰・匿名回報・完成。
- ⚠️ 精靈的金鑰步只收 TMDb 與 Claude（`ApiKeysStep.tsx:27-52`）；設定頁有五把（`ApiKeysForm.tsx:61-110`：claude、gemini、tmdb、elevenlabs、openai）。
- ⚠️ 網頁存金鑰要 `ENCRYPTION_KEY`（`key_settings_service.go:35-44`）。
- ⚠️ 容器以 root 起、降到 `PUID`／`PGID`，預設 1000:1000（`docker-entrypoint.sh:22-50`）；Unraid 模板 99:100（`vido.xml:66-79`）。
- ⚠️ `GIN_MODE` 沒人讀，Gin 模式來自 `ENV`（`main.go:68-73`）。
- `docs/unraid-installation-guide.md:57-69`（✅）環境變數表只有 TMDb、語言、`GIN_MODE`、`AI_PROVIDER`、Gemini、Claude；缺 `ELEVENLABS_API_KEY`、`OPENAI_API_KEY`、`ASR_BASE_URL`、`ENCRYPTION_KEY`、`VIDO_AUTH_PASSWORD`、`PUID`／`PGID`、`VIDO_SUBTITLE_PIPELINE_MODE`（#788 改成預設 pipeline 後只需說「要關才設 legacy」）。
- ⚠️ Unraid 指南 `:13-17`「Method A：在 Community Apps 搜 Vido」——repo 裡只有 `unraid-template/vido.xml`，**查不到有上架**。Alexyu 確認前先改成「用模板 URL 安裝」。
- ⚠️ Unraid 指南 `:99-101`、`:111` 叫人 `chown 1000:1000`，跟模板的 99:100 衝突。

### 登入（✅）
- `docs/synology-installation-guide.zh-TW.md:61`、`docs/qnap-installation-guide.zh-TW.md:63`「Vido 目前沒有內建登入機制」→ 錯，`VIDO_AUTH_PASSWORD` 存在（`config.go:218`）。

### tag（✅）
- `docs/synology-installation-guide.zh-TW.md:50`、`docs/qnap-installation-guide.zh-TW.md:50`：`v0.1.0-beta.1` 不存在。
- `README.md:62`「`latest` 映像才有，v0.1.1 還沒有」、`:127` 例子 `v0.1.1` → 實際 tag `0.1.1`。

### README 其他（✅）
- `README.md:4`、`:6`、`:21` 寫「有英文字幕就翻；沒有字幕就先聽聲音再翻」，沒說聽聲音**只做英文對白**（只有 `:55` 提）。留言兩位的片庫是日文特攝片，裝了第一天就失望。第一段的定位改寫在 `release-0-2-readme-positioning`，本單只在這三處補「英文對白」。
- `README.md:84` 費用付給「你自己的 Claude」→ 也可能是 Gemini。
- `README.md:106` 跟 `docker-compose.yml:76`「`ENCRYPTION_KEY` Auto-derived if empty」衝突（⚠️ agent：只有 qBittorrent 密碼會自動推導，`crypto/key_derivation.go:57`；存 API 金鑰一定要設）。

### compose（✅ `docker-compose.yml:48-68` 本人讀）
- 根目錄 `docker-compose.yml:28-31` 是 `build:` 從原始碼——本機 build 沒有內建 TMDb 金鑰。要說清楚「這份是開發用；NAS 請用 `docs/docker-compose.nas.yml`」。
- `docs/docker-compose.nas.yml:13-28` 缺 `ELEVENLABS_API_KEY`、`VIDO_ASR_PROVIDER`、`ENCRYPTION_KEY`、`VIDO_GEMINI_MIN_INTERVAL`。

## Acceptance Criteria

1. **媒體掛載可寫**：上面列的每一處 `:ro`／「唯讀」／「只需要讀取權限」改成可寫，並在每份指南加一句「產生的字幕會存在影片旁邊，所以媒體資料夾要能寫入；`PUID`／`PGID` 要對它有寫入權限」。`unraid-template/vido.xml` 媒體路徑 `Mode="rw"`、Description 改。容器根檔案系統的 `--read-only` 保留。
2. **金鑰說明一致**：四份 NAS 指南＋兩份 compose＋`.env.example` 都說同一件事——TMDb 內建不用填；翻譯要 Claude **或** Gemini 擇一；沒字幕的片要聽聲音才需要 ElevenLabs（建議）或 OpenAI；網頁上填金鑰要先設 `ENCRYPTION_KEY`；`VIDO_AUTH_PASSWORD` 選填（分享給別人用才設）。Unraid 指南的環境變數表補齊、拿掉 `GIN_MODE`（改 `VIDO_LOG_LEVEL=debug`）。
3. **tag 正確**：所有版本例子用 CI 真的會發的格式（`0.1.1`、`latest`）；「`latest` 才有登入」那句拿掉（0.2 發版後兩者都有）。
4. **登入說法正確**：Synology／QNAP 指南改成「設 `VIDO_AUTH_PASSWORD` 就有共用密碼登入；沒設就不要暴露到網際網路」。
5. **精靈與設定頁**：每份指南的「裝好之後」一段說「打開網頁會進六步設定精靈；金鑰也可以之後在 設定 → 金鑰設定 填」。
6. **英文對白**：`README.md:4`、`:6`、`:21` 補「英文對白」；`:84` 改「Claude 或 Gemini」。
7. **PUID／PGID**：Unraid 指南的 `chown` 建議改成跟模板一致（99:100）或改成「跟你設的 PUID/PGID 一致」。
8. **Community Apps**：⚖️ Alexyu 2026-10-14 確認「Vido 沒上架 CA」。Unraid 指南 Method A（在 CA 搜 Vido）刪掉，改成「用模板網址安裝」（Docker 分頁 → Add Container → Template repositories 加 repo 的 `unraid-template/` 網址，或直接貼 `vido.xml` 的 raw 網址）。
9. **雙語**：Unraid 指南中英兩版都改；Synology／QNAP 現在只有 zh-TW，維持（不新增英文版，記進 Discovery Triage 不算缺）。
10. `pnpm run format:check` 綠；`docs/` 沒有新的壞連結（相對連結都指到存在的檔案）。

## Tasks / Subtasks

- [x] T1 `:ro` 全部改可寫＋加說明（AC 1）
- [x] T2 金鑰／環境變數段落統一（AC 2、5）
- [x] T3 tag、登入、PUID、Community Apps（AC 3、4、7、8）
- [x] T4 README 三處「英文對白」＋ `:84`（AC 6）
- [x] T5 Unraid 英文版同步（AC 9）、format:check（AC 10）

## Dev Notes

- 先等 #788（`release-0-2-pipeline-default`）合併再開分支，它改了 README 74-105、deployment.md、compose、模板，避免衝突。
- 不改 `docs/deployment.md`、`PRODUCT.md`、`ROADMAP.zh-TW.md`（-b）。不改程式。不改 README 第一段定位（`release-0-2-readme-positioning`）。
- 寫法照 CLAUDE.md：白話、台灣用語；metadata／AI 之類的名詞保留。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純文件）。

### References

- audit：session 2026-10-14 的 Explore agent 報告（本檔「查到的事」已把要用的行號抄進來）。
- `apps/api/internal/subtitle/placer.go:73`、`Dockerfile:86-91`、`.github/workflows/docker.yml:280-291`、`config.go:216-219`、`SetupWizard.tsx:35-54`。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14，以 Paige（Technical Writer）身分。

### Completion Notes List

- **四份 NAS 指南重寫**（Synology／QNAP zh-TW、Unraid en＋zh-TW）：媒體掛載改可寫並說明原因；金鑰段統一（TMDb 內建、Claude 或 Gemini 擇一、ElevenLabs 建議、`ENCRYPTION_KEY` 剛好 32 個字、`VIDO_AUTH_PASSWORD`）；六步精靈；tag 格式 `0.1.1`；登入說法；Unraid 環境變數表補齊、`GIN_MODE` 換 `VIDO_LOG_LEVEL`、權限段改成「root 起、降到 PUID/PGID（模板 99:100）」、Method A 改成「Template repositories 加 repo 網址」（⚠️ 這個安裝法我沒在 Unraid 上實測，Alexyu 下次用 NAS 時驗一次）。
- `docs/docker-compose.nas.yml` 重寫：拿掉 `:ro`、補 `ELEVENLABS_API_KEY`／`VIDO_ASR_PROVIDER`／`VIDO_GEMINI_MIN_INTERVAL`／`ENCRYPTION_KEY`，每個變數一句白話註解。
- 根目錄 `docker-compose.yml`：標明是從原始碼 build 的開發用檔、NAS 請用 nas 版；`:ro` 拿掉；`ENCRYPTION_KEY` 註解改正（只有 qBittorrent 密碼會自動推導）。
- `unraid-template/vido.xml`：媒體 `Mode="rw"`；`GIN_MODE` 換 `VIDO_LOG_LEVEL`；新增 `ELEVENLABS_API_KEY`；TMDb／Claude／Gemini／OpenAI 的說明改成現況（OpenAI 那條原本寫「要重啟」，現在熱載）。XML 用 minidom 解析過。
- `.env.example`：`GIN_MODE` 換 `VIDO_LOG_LEVEL`；媒體「mounted read-only」改可寫。
- README：`:4`、`:6`、`:21` 補「英文對白」；`:62` 拿掉「`latest` 才有登入」；`:83` Claude 或 Gemini；`ENCRYPTION_KEY` 改「剛好 32 個字」（`crypto.go:25` `KeySize = 32`，原本寫「以上」是錯的）；回報範例 tag `0.1.1`。
- grep 剩下的 `:ro`／唯讀／read-only 都是「不要加 `:ro`」「根檔案系統唯讀」這類正確說法。相對連結全部指到存在的檔案。prettier 全綠。

### Discovery Triage

- ③ Synology／QNAP 指南只有 zh-TW 沒有英文版 → 既有狀態，不開單（README 英文段已指向它們，之後有英文使用者再說）。
- ③ 根目錄 `docker-compose.yml` 用 named volume（`vido-data:`）跟 NAS 版用 host path 不一致 → 開發用檔，不動。
- ① Unraid 模板缺 `ELEVENLABS_API_KEY`、`GIN_MODE` 沒人讀 → 併入本單 T3 一起改（AC 2 的「金鑰說明一致」涵蓋模板）。

### File List

- `docs/{synology-installation-guide.zh-TW.md,qnap-installation-guide.zh-TW.md,unraid-installation-guide.md,unraid-installation-guide.zh-TW.md,docker-compose.nas.yml}`
- `docker-compose.yml`、`unraid-template/vido.xml`、`.env.example`、`README.md`
