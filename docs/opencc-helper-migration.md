# OpenCC C++ Helper 遷移

Vido 的字幕與豆瓣 metadata 簡繁轉換保留 `s2twp` profile 與既有介面。為了移除目前 Go binding 的 GPL 傳遞相依，API 已加入官方 C++ OpenCC CLI backend。

## 啟用方式

根目錄 Dockerfile 會在 multi-stage build 中編譯並打包固定版本的官方 OpenCC（目前 `ver.1.4.2`），正式映像預設使用 C++ helper：

```bash
VIDO_OPENCC_BACKEND=cpp \
VIDO_OPENCC_BIN=/usr/bin/opencc \
VIDO_OPENCC_CONFIG=/usr/share/opencc/s2twp.json \
  ./api
```

helper 透過 stdin/stdout 傳遞內容，單次轉換逾時 30 秒；失敗時保留原文並回傳錯誤。未設定 `VIDO_OPENCC_BACKEND=cpp` 時，開發環境仍使用既有 binding，方便在尚未安裝 C++ OpenCC 的環境執行測試。

映像建置使用 Docker Buildx，會依目標平台編譯 helper；不應在 NAS 上自行以 host binary 覆蓋 `/usr/local/bin/opencc`。

## 上線門檻

1. 以相同 `s2twp` golden corpus 比對兩個 backend（字幕、豆瓣 metadata、BOM、冪等性與台灣詞彙）。
2. 用 Docker Buildx 驗證 `linux/amd64`、`linux/arm64`，並在 Unraid、Synology 或 QNAP 實機執行 smoke test。
3. 確認 `go list -deps ./cmd/api` 不再包含 `github.com/liuzl/cedar-go`，再移除 Go binding 與相關模組。
4. 產生 SBOM／license inventory，逐一確認 OpenCC 字典與 runtime package 的授權。

在上述門檻完成前，不宣稱 Vido 整體採 MIT 或 Apache-2.0，也不應將 GHCR image 標為 MIT。
