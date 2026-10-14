# Vido

**自架在 NAS 上的影音管理工具，專門幫沒有中文字幕的影片做出繁體中文字幕。**
影片裡有英文字幕，就抽出來翻成繁中；完全沒有字幕，就先聽聲音、再翻。翻出來用的是台灣用語。

> **English**: Vido is a self-hosted media manager for your NAS, built around one problem: getting Traditional Chinese (Taiwan) subtitles for videos that don't have them. If a video carries an English subtitle track, Vido extracts and translates it; if it has no subtitles at all, Vido transcribes the audio first and then translates. Translations use Taiwanese vocabulary and keep character names consistent.

---

## 為什麼做這個

我在家裡架了 NAS 當家庭劇院，用了一陣子才發現真正卡住的地方：**很多影片只有英文字幕，或者根本沒有字幕。**

原本用的字幕工具對繁中處理得不好，線上字幕網站對冷門片或新片也常常找不到。翻出來的字幕一下「視頻」、一下「質量」，角色名每集都翻得不一樣。

所以 Vido 的字幕流程是這樣：

```
影片裡已經有中文字幕  →  直接用，不花錢
影片裡有英文字幕      →  抽出來，翻成繁中
完全沒有字幕          →  先聽聲音轉成英文，再翻成繁中
```

字幕之外，Vido 也幫你把媒體庫整理好：繁中的片名與介紹、海報、季集清單，讓這個家庭劇院真的好用。

## 目前能做什麼

**字幕（核心）**

- **翻譯影片裡的英文字幕** — 從 mkv／mp4 抽出英文字幕軌，翻成繁中。有好幾條英文軌時優先用一般版，畫面上的強制字幕也會一起翻。單部的「生成字幕」和批次都會先看片內有沒有字幕
- **聽聲音生字幕** — 沒有字幕的電影和影集，先用語音辨識（OpenAI Whisper，或你自架的相容服務）轉成英文，再翻成繁中。音檔切成兩分鐘一段，並會過濾常見的幻聽，例如只有音樂符號的句子、重複的假台詞、假署名
- **台灣用語** — 內建台灣用語詞庫，有三種在地化程度可以選（「設定 → 字幕設定」）。中國大陸、香港、澳門的片子只把字轉成繁體，保留原本的說法
- **角色名前後一致** — 比對到 TMDb 時，自動把演員與角色名放進詞彙表；也會從片庫裡已有的官方繁中字幕學人名譯法。詞彙表可以匯出、匯入
- **花多少錢先講清楚** — 會花錢的按鈕按下去之前，先顯示預估金額；每次都可以挑翻譯模型，看得到價格與品質等級。每次執行有花費上限，超過會暫停，已經做好的部分會保留，下次只做剩下的；AI 帳號沒錢時會整批停下並告訴你。活動頁看得到「本月 AI 花費」
- **字幕狀態一目瞭然** — 季清單標出每一集有沒有中文字幕，可以篩選「缺中文字幕」，首頁顯示繁中字幕覆蓋率
- **簡轉繁** — 用官方 OpenCC 轉換，可以批次處理整個媒體庫

**媒體庫**

- **繁中優先的資料** — 片名、介紹、海報以 TMDb（zh-TW）為主，可選擇開啟 Wikipedia 補充。豆瓣也可以開啟，但目前常被擋，效果有限
- **AI 檔名解析** — 看得懂格式很亂的檔名，比對出是哪一部片；認錯或沒認出來時，可以手動選對的那部
- **媒體庫管理** — 遞迴掃描、多個資料夾與磁碟區、換海報、修改資訊、備份與還原
- **瀏覽與探索** — 首頁媒體牆、進階搜尋、詳情頁（評分、季集列表、推薦、觀看平台、預告）
- **活動中心** — 看得到字幕做到第幾段、第幾句，以及每月 AI 花費
- **設定引導** — 首次啟動時一步步帶你完成設定
- **外觀** — 深色「夜行」與淺色「日巡」兩種主題，手機上也能完整操作

**選配整合（本來就有在用才需要）**

- **qBittorrent** — 在 Vido 裡看下載進度，以及檔案進媒體庫了沒
- **Sonarr／Radarr** — 連線後可以在 Vido 裡管理想看清單與缺集，入庫後自動處理字幕

**目前的限制**

- 字幕只能從**英文**翻成繁中，日文、韓文等其他語言還不支援
- 字幕翻譯預設用 **Claude**；填了 Gemini 金鑰可以改選 Gemini（用你自己的 Gemini 額度，免費方案 Vido 會自動放慢）
- 線上字幕網站（射手網、OpenSubtitles）的搜尋目前沒有開放設定，字幕主要靠片內字幕與語音辨識
- 畫面上的字（招牌、書名、字卡）還不會翻

> ⚠️ **專案狀態**：積極開發中，設計為**單人使用**，目前主要在作者自己的 NAS 上運行。歡迎試用與[回報問題](#回報問題)，但還不建議用於關鍵用途。
>
> 🔒 **存取控制**：設定環境變數 `VIDO_AUTH_PASSWORD` 即可為整個 App（UI + API）加上一道共用密碼登入（`latest` 映像才有，v0.1.1 還沒有）。**要把這個實例分享給別人、或讓它超出信任的家用網路前，強烈建議先設定。** 留空則維持舊的無登入行為——此時**請只在家用網路內使用**，遠端存取請走 VPN、Tailscale 或反向代理加驗證，不要把服務直接暴露到公開網路。未設定密碼時，任何能連到伺服器的人都能完全控制它（讀取／修改設定與 API 金鑰、下載資料庫備份、加入下載任務）。

## 安裝

Vido 用 Docker 安裝，跑在你自己的 NAS 上。請照你的 NAS 平台看安裝指南，不需要下載原始碼：

- [Synology DSM／Container Manager 安裝指南](docs/synology-installation-guide.zh-TW.md)
- [QNAP QTS／Container Station 安裝指南](docs/qnap-installation-guide.zh-TW.md)
- [Unraid 安裝指南](docs/unraid-installation-guide.zh-TW.md)（[English](docs/unraid-installation-guide.md)）

裝好之後打開網頁，第一次會有設定精靈帶你完成初始設定。

> 🧩 **要用 AI 字幕，請先確認這幾件事**（上面的安裝指南正在更新，有衝突時以這裡為準）：
>
> 1. **加上 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`**。沒加的話，Vido 不會讀片內的英文字幕，只會走「聽聲音」（之後會改成預設）。
> 2. **填入 `CLAUDE_API_KEY`**。如果想在網頁上（設定精靈或「設定 → 金鑰設定」）填金鑰，要先設 `ENCRYPTION_KEY`，不然網頁會存不了。
> 3. **媒體資料夾要能寫入**。產生的字幕會存在影片旁邊，所以掛載媒體資料夾時不要加 `:ro`，`PUID`／`PGID` 也要對那個資料夾有寫入權限。
>
> 完全沒有字幕的片要先聽聲音，還需要一把聽寫服務的金鑰：建議填 `ELEVENLABS_API_KEY`（Scribe v2，作者 2026-10 的考卷上三次都過、Whisper 三次都沒過，而且每小時 US$0.22 比 Whisper 的 0.36 便宜），或填 `OPENAI_API_KEY` 用 Whisper，或把 `ASR_BASE_URL` 指到自架的語音辨識服務。兩把雲端金鑰都填的話預設用 ElevenLabs，可以在「設定 → 金鑰設定」改。

### 會花多少錢

Vido 本身免費。AI 字幕的費用是付給你自己的 Claude（聽聲音的話再加 ElevenLabs 或 OpenAI）帳號，按下去之前 Vido 會先顯示預估金額。以預設的翻譯模型為例，作者自己 NAS 上的實測：

| 影片                                 | 大約花費                    |
| ------------------------------------ | --------------------------- |
| 影集一集（約一小時），片內有英文字幕 | US$0.3～0.5                 |
| 電影一部（約兩小時），片內有英文字幕 | US$0.9～1.2                 |
| 影集一集，沒有字幕、要先聽聲音       | 約 US$1.4（只有一集的紀錄） |

可以在畫面上改選比較便宜的模型，大約只要上面的四成，但翻壞的句子會比較多。

### 常用設定

這些都是 Docker 的環境變數，在安裝指南的 compose 設定裡填。

| 設定                                | 要填嗎                          | 用途                                                                                         |
| ----------------------------------- | ------------------------------- | -------------------------------------------------------------------------------------------- |
| `VIDO_SUBTITLE_PIPELINE_MODE`       | 要用 AI 字幕就填 `pipeline`     | 先讀片內字幕，有英文字幕就直接翻；不填只會聽聲音                                             |
| `CLAUDE_API_KEY`                    | 要用 AI 字幕就要填              | 把字幕翻成繁中（目前只支援 Claude）                                                          |
| `ELEVENLABS_API_KEY`                | 沒字幕的片要聽聲音時建議填      | ElevenLabs Scribe 語音辨識，每小時約 US$0.22；填了就預設用它                                 |
| `OPENAI_API_KEY`                    | 選填                            | OpenAI Whisper 語音辨識，每小時約 US$0.36；沒填 ElevenLabs 時用它                            |
| `VIDO_ASR_PROVIDER`                 | 選填，預設 `auto`               | 兩把都填時要用哪家：`auto`（有 ElevenLabs 就用它）、`elevenlabs`、`openai`；網頁設定可以蓋過 |
| `ASR_BASE_URL`、`ASR_MODEL`         | 選填                            | 改用自架的語音辨識服務（例如 Speaches、WhisperLive），不用 OpenAI key，也不花雲端費用        |
| `ENCRYPTION_KEY`                    | 想在網頁上填金鑰就要填          | 32 個字以上的隨機字串，用來加密存放你的金鑰                                                  |
| `AI_RUN_BUDGET_USD`                 | 選填，預設 `5`                  | 每次執行最多花幾美元。超過會暫停，已經做好的部分會保留，下次只做剩下的                       |
| `VIDO_AUTH_PASSWORD`                | 建議填                          | 整個 Vido 的登入密碼，見上面的「存取控制」                                                   |
| `PUID`、`PGID`                      | 依你的 NAS                      | 容器用哪個使用者身分執行，要能寫入媒體資料夾                                                 |
| `AI_PROVIDER`                       | 只有 Claude key 的話填 `claude` | 讓檔名解析也用 Claude（預設是 Gemini）                                                       |
| `TMDB_API_KEY`                      | 不用                            | 映像已經內建一把；想用自己的額度再填                                                         |
| `ENABLE_WIKIPEDIA`、`ENABLE_DOUBAN` | 選填                            | 開啟片名、介紹的補充來源                                                                     |

完整的設定清單見 [`.env.example`](.env.example)。

Vido 有一個預設關閉的「匿名使用回報」（開關在「設定 → 連線設定」最底下）：送什麼、不送什麼見 [匿名使用回報](docs/usage-report.zh-TW.md)（[English](docs/usage-report.md)）。

## 回報問題

裝不起來、功能壞掉、字幕轉出來不對——都歡迎開 issue：

**[→ 開一張 GitHub Issue](https://github.com/j620656786206/vido/issues/new/choose)**

回報時請盡量附上（issue 表單會引導你填）：

1. **執行環境** — NAS 平台（Unraid／Synology／QNAP／其他）與安裝方式
2. **版本** — image tag（例如 `latest`、`v0.1.1`），或執行 `docker inspect vido --format '{{index .Config.Labels "org.opencontainers.image.revision"}}'` 取得確切 commit
3. **重現步驟** — 做了什麼、預期看到什麼、實際看到什麼
4. **Logs** — `docker logs vido --tail 100` 的輸出（貼上前請檢查裡面沒有 API key）

不確定是不是 bug 也沒關係，先開再說。

## 想一起開發

歡迎！開發環境、建置、測試指令與技術架構，請看 [開發指南](docs/development.md)。

## 授權

**Vido 原始碼採 Apache-2.0**（見 [`LICENSE`](LICENSE)）；Docker image 內的第三方元件依各自授權散布。

## 第三方服務與資料來源

- **TMDB** — This application uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.
  （本應用程式使用 TMDB 與 TMDB API，但未經 TMDB 認可、認證或核准。）此為 [TMDB API 條款](https://www.themoviedb.org/api-terms-of-use) §3 要求的原句，不得改寫；App 內同樣顯示於「設定 → 金鑰設定」的 TMDB 欄位與各詳情頁頁尾（story sub-6-9）。
- **豆瓣** — 繁中 metadata 的補充來源之一，透過解析公開網頁取得（HTML scraping，非官方 API）。網站改版時可能失效，屬 best-effort 的 fallback。
- **Wikipedia** — metadata fallback 來源。
