# Disc（後端）：生成字幕再也不會蓋掉使用者自己放的英文字幕檔

Status: review

**Source:** `backlog-movie-generate-overwrites-existing-en-srt`（P1）。2026-10-19 準備 FB 新貼文截圖時在正式機重現；⚖️ Alexyu 同日「請 Bob 把 P1 開成 story」。依 Story Splitting Rule 拆兩張：
- **本單 `-a`**：安全網——不管走哪條路，**絕不覆寫使用者原本的 `.en.srt`**（純後端、小）。
- **`-b`**：省錢——旁邊有英文字幕檔時直接拿來翻，不再聽寫（推翻 sub-1-4 的「片旁字幕檔歸搜尋引擎管」，要 Alexyu 裁定）。

**跟 FB 發文的關係：** founder-growth 建議本單（至少 `-a`）合併後再發新貼文——來試的人片子多半旁邊就有英文字幕，第一次用就被蓋掉，信任就沒了。

## 重現（2026-10-19 20:53，正式機 image `583b5a3e`）

1. 《Tears of Steel》（CC BY，Blender）remux 成 `.mkv`（片內**沒有**字幕軌）＋官方 `TOS-en.srt` 改名 `Tears of Steel (2012).en.srt` 放旁邊，掃描入庫。
2. 詳細頁 →「管理字幕」：上方「現有字幕」列出「英文・本地檔案・Tears of Steel (2012).en.srt」；按鈕「生成字幕 $0.19」＋「語音辨識＋AI 翻譯，約需數分鐘」。
3. 按下去 → ElevenLabs 聽寫（log `asr provider (re)built … provider=elevenlabs`、`AI usage recorded (ASR)`）→ 翻譯 → `.zh-Hant.srt`。
4. **`.en.srt` 從 4,764 bytes 變 5,193 bytes**，內容換成聽寫稿（開頭多出 `Roger, Roger`、時間軸全變）。官方檔已手動還原；被蓋掉的版本留在 NAS `/mnt/user/data/tmp-tos/asr-overwrote-en.srt` 當證據。

## 查到的事（2026-10-19，Bob 以 `git show origin/main:` 逐一確認）

- **按鈕走哪裡**：`ManageSubtitleDialogV2.tsx:420-421` → `transcriptionService.ts:116-118` `POST /movies/:id/transcribe?translate=true`（單集 `:129` `POST /episodes/:id/transcribe`）。預設 pipeline 模式下，`transcription_handler.go:184-189` 交給 `startSolo` → `SoloRunner.Start` → `Pipeline.ProcessItem`（`disc-2026-10-single-generate-ignores-embedded-english-a` 搬過來的）。
- **為什麼聽寫**：`router.go:94` `SelectAndRoute` 只看片內字幕軌（`extractor.go:141-148` `if t.External { continue }`；`verdictWithoutTrack` `router.go:190`「M1 is embedded-only; sidecars stay the search engine's domain」）。片內沒英文軌 → `RouteNoTextSource`（`SoloRunner` 另把 `RouteSkip` 也送聽寫，`solo_runner.go:249-253`）→ `transcribeFallback`（`process_item.go:542-573`）→ `pipelineASRAdapter.Transcribe`（`cmd/api/asr_adapter.go:30-39`）→ `RunTranscription` → `runPipeline`。
- **蓋掉的那一行**：`transcription_service.go:1080-1083`
  ```go
  srtPath = filepath.Join(mediaDir, baseName+".en.srt")
  if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
  ```
  沒有檢查檔案是否已存在。這是全專案**唯一**寫 `<base>.en.srt` 的地方，電影和單集共用。
- **續跑**：`tryTranslateOnlyResume`（`:946`）／`resumeSource`（`:1173-1187`）只在媒體列 `subtitle_status == untranslated` 時才把 `.en.srt` 當續跑點（`sub-2-2a-untranslated-status-backend.md:59`，Winston：「使用者自己放的字幕不能被誤當續跑點」）。**本單不能破壞這條**：Vido 自己聽寫到一半的稿子要續得到，使用者的檔案不能被當續跑點。
- **對照**：繁中那邊的 placer 是原子寫入＋`.bak` 備份（`placer.go:224`），英文聽寫稿沒有走 placer。`:2200` 的 zh-Hant 後備直寫只在沒接 placer 時發生，不在本單範圍。
- **沒有測試**檢查「既有 `.en.srt` 跑完後還在」。相關既有測試：`router_test.go:546` `TestSelectAndRoute_ExternalSidecarsAreIgnored`、`solo_runner_test.go:136`／`:165`、`transcription_solo_handler_test.go:76`／`:97`、`transcription_episode_handler_test.go:138-167`。

## Story

身為把英文字幕檔放在影片旁邊的 NAS 使用者，
我想要 Vido 不管怎麼生成字幕，都不會動到我原本的英文字幕檔，
這樣我的播放器照樣讀得到它，我也不用擔心試用 Vido 會弄壞我的片庫。

## Acceptance Criteria

1. **⚖️ 聽寫稿的檔名——Alexyu 2026-10-19 選 A。** 當 `<base>.en.srt` 已存在、而且**不是**這次要續跑的 Vido 稿（`subtitle_status != untranslated`）時：
   - **✅ A（採用）**：聽寫稿寫成 `<base>.vido.en.srt`。Jellyfin／Kodi 會把它認成另一條英文字幕（最後一段 `en` 是語言），使用者原本的檔完全不動；Vido 的續跑也讀這個名字。
   - ~~B：聽寫稿不落地（只留在暫存＋chunk 快取），使用者看不到英文稿；中斷後要靠 chunk 快取續跑。~~（不採用）
   - 不考慮：先備份成 `.bak` 再覆寫——使用者的播放器就讀不到原檔了，等於還是蓋掉。
2. **絕不覆寫。** `runPipeline` 寫英文聽寫稿前先判斷目標檔是否存在；照 AC #1 的決定換檔名（或不落地）。寫入用「先寫暫存檔再 rename」，跟 placer 一樣原子。**`os.WriteFile` 直接寫 `<base>.en.srt` 只剩一種情況：檔案原本不存在。**
3. **續跑不壞。** `untranslated` 續跑能找到 Vido 自己的稿子（A 案：新檔名；舊名 `<base>.en.srt` 也要相容——升級前留下來的續跑點）；使用者自己的 `<base>.en.srt` 仍然**不會**被當續跑點（`sub-2-2a` 的守則）。
4. **測試（Rule 28 Real-Shape：真的檔案系統，`t.TempDir()`）。**
   - 旁邊已有 `.en.srt`（內容 X）→ 跑完整條（假 ASR 回內容 Y）→ 原檔仍是 X、Vido 稿在 AC #1 決定的位置是 Y、`.zh-Hant.srt` 照常產生。
   - 旁邊沒有 `.en.srt` → 行為跟現在一樣（寫 `<base>.en.srt`）。
   - `untranslated` 續跑：新檔名、舊檔名各一個案例都能續；使用者的 `.en.srt` 在非 `untranslated` 狀態下不會被讀成續跑點。
   - 電影與單集各一（共用 `runPipeline`，但 `mediaDir`／`baseName` 來源不同）。
5. **活動記錄／事件**：`transcription_complete` 的 `srt_path` 指向實際寫入的檔（不是被跳過的那個）。
6. **正式機驗證。** 合併、NAS 更新映像後，用同一部《Tears of Steel》（官方 `.en.srt` 已還原）再按一次「生成字幕」：官方 `.en.srt` bytes 不變（4,764）、Vido 稿在新位置、`.zh-Hant.srt` 有產生。Completion Notes 貼 `ls -la` 與 `sha256sum`。
7. `go test ./...`、`~/go/bin/staticcheck ./...`、`lint:all`（看「✖ N problems (X errors」那行）綠。

## 不做

- 旁邊有英文字幕檔就直接翻、不聽寫（省錢）→ `-b`。
- 「管理字幕」對話框的估價與說明文字（仍會寫「語音辨識＋AI 翻譯」）→ `-b`。
- `.mov` 掃不到、簡體片名、「你／妳」誤判 → 各自的 backlog 單。

## Dev Agent Record

### Completion Notes

- **做法：** 新增 `apps/api/internal/services/transcript_sidecar.go` 的 `saveEnglishTranscript`。`<base>.en.srt` 那個名字上**有任何東西**（使用者的檔、甚至讀不到狀態的東西）就改寫 `<base>.vido.en.srt`；只有那個名字完全空著才寫 `<base>.en.srt`。寫入是「同資料夾暫存檔（檔名含 `.tmp.`，字幕清單和 miner 都會跳過）→ chmod 0644 → rename」，失敗會刪暫存檔。`runPipeline` Phase 3 改呼叫它，`srtPath` 就是實際寫入的檔，所以 `transcription_complete` 的 `srt_path`、`untranslated` 寫回的路徑、帳本輸出都跟著對（AC #5）。
- **續跑（AC #3）不用改：** `resumeSource` 本來就讀媒體列**記錄的路徑**，不是用檔名去猜。新稿記成 `.vido.en.srt` 就從它續；升級前記成 `.en.srt` 的照舊續；`untranslated` 以外的狀態一律不續（既有 `TestRunTranscription_NoResumeWhenStatusNotUntranslated` 仍綠）。
- **連帶修一處：** `subtitle/inventory.go` 的 `sidecarLanguage` 原本取檔名標籤第一段當語言，`.vido.en.srt` 會被標成語言「vido」。現在先去掉開頭的 `vido.`，讀成英文；管理字幕對話框的「現有字幕」和單集的 subtitle_tracks 都走這裡。
- **檢查過、不用改：** miner 的英文片旁字幕依檔名排序，`X.en.srt` 排在 `X.vido.en.srt` 前面，兩個都在時仍先用使用者的；`.en.srt` 全專案只有這一個寫入點。
- **給 `-b2` 的提醒：** 路由把片旁英文字幕當翻譯來源時，`.vido.en.srt` 是 Vido 自己的聽寫稿，不是使用者的字幕，要另外決定要不要用。
- **測試（AC #4，真檔案系統 `t.TempDir()`）：** `transcript_sidecar_test.go`——
  - `saveEnglishTranscript` 五個：空著寫原名＋權限 0644＋沒留暫存檔；使用者檔不動、稿子進 `.vido.en.srt`；第二次重寫自己的稿；名字被資料夾占住也算占用；rename 失敗不留暫存檔、使用者檔不動。
  - 續跑：`untranslated` 記錄 `.vido.en.srt` → 只翻譯、送去翻的是聽寫稿不是使用者檔、使用者檔不動。
  - **整條跑一次（真 ffmpeg 做兩秒的片＋假聽寫＋假翻譯）**：電影、單集各一，使用者 `.en.srt` 一個 byte 不變、聽寫稿在 `.vido.en.srt`、`.zh-Hant.srt` 有產生、完成事件的 `srt_path` 指向 `.vido.en.srt`；旁邊沒字幕時照舊寫 `.en.srt`。沒有 ffmpeg 的環境（CI）這三個會 skip，前面六個照跑。
  - 把修正拿掉重跑：電影、單集兩個整條測試都 FAIL（證明測得到原本的 bug）。
- **檢查：** `go test ./...` 全綠、`go vet` 乾淨、`~/go/bin/staticcheck ./...` 乾淨、`lint:all`「✖ 170 problems (0 errors」、`format:check` 通過。
- **跨模型 CR（Sonnet，2026-10-20）：** 沒有 blocker。
  - 修了一條 should-fix：片庫的「有沒有中文字幕」判斷（`models/chinese_subtitle.go` 的 `classifyTag`）會把 `vido.en` 的第一段「vido」當成看不懂的標籤，判成「不確定」而不是「沒有中文」。現在把 `vido` 加進 `nonLanguageSegments`，並補了測試案例。
  - 沒改的 nit：(1) 檢查檔案和 rename 之間如果使用者剛好放進 `.en.srt`，還是會蓋掉。這個空檔只有幾毫秒；改用 hardlink 防的話，NAS 的 SMB／shfs 不一定支援，所以不做。(2) 以前 Vido 自己寫的 `.en.srt` 分不出是不是使用者的檔，重新生成時會留著，另外寫 `.vido.en.srt`。這是安全的那一邊。(3) CI 沒有 ffmpeg，三個整條測試在 CI 會 skip，本機有跑過。
- **正式機驗證（AC #6）：** 合併、NAS 更新映像後補在下面。
