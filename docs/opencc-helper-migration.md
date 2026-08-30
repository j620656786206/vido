# OpenCC C++ Helper 遷移

Vido 的字幕與豆瓣 metadata 簡繁轉換保留 `s2twp` profile 與既有介面。為了移除目前 Go binding 的 GPL 傳遞相依，API 已改為只呼叫官方 C++ OpenCC CLI，不再編譯或載入 Go binding。

## 啟用方式

根目錄 Dockerfile 會在 multi-stage build 中編譯並打包固定版本的官方 OpenCC（目前 `ver.1.4.2`），正式映像預設使用 C++ helper：

```bash
VIDO_OPENCC_BIN=/usr/local/bin/opencc \
VIDO_OPENCC_CONFIG=/usr/share/opencc/s2twp.json \
  ./api
```

helper 透過 stdin/stdout 傳遞內容，單次轉換逾時 30 秒；失敗時保留原文並回傳錯誤。未安裝 helper 時服務會以 degraded mode 啟動，不會偷偷改用另一套轉換器。

本機開發（macOS）請先 `brew install opencc`，並依安裝路徑設定 `VIDO_OPENCC_BIN` 與 `VIDO_OPENCC_CONFIG`（Homebrew 的設定檔位於 `$(brew --prefix)/share/opencc/`）。

映像建置使用 Docker Buildx，會依目標平台編譯 helper；不應在 NAS 上自行以 host binary 覆蓋 `/usr/local/bin/opencc`。

## 上線門檻

1. 以相同 `s2twp` golden corpus 比對兩個 backend（字幕、豆瓣 metadata、BOM、冪等性與台灣詞彙）。
2. 用 Docker Buildx 驗證 `linux/amd64`、`linux/arm64`，並在 Unraid、Synology 或 QNAP 實機執行 smoke test。
3. 確認 `go list -deps ./cmd/api` 不再包含 `github.com/longbridgeapp/opencc`、`github.com/liuzl/da` 或 `github.com/liuzl/cedar-go`。
4. 執行 `scripts/audit-licenses.sh` 產生 SBOM／license inventory，逐一確認 OpenCC 字典與 runtime package 的授權。

Vido 原始碼現在採 Apache-2.0；這不會改變 Docker image 中 FFmpeg、Alpine 等第三方元件的個別授權。GHCR image 應維持 `NOASSERTION` 或附完整第三方 notices，不應標為純 Apache-2.0。
