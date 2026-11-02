# Story disc-2026-11-secret-log-password：伺服器紀錄不再露出密碼的頭尾 8 個字

Status: done

**Source:** 2026-11-02 檢查 OpenSubtitles 設定紀錄時發現。

## 問題

存密鑰時的紀錄、開機時的設定紀錄，都用「前 4 字＋****＋後 4 字」遮罩。API Key 這樣沒關係（看得出是哪把、拿不去用），但密碼這樣等於露出 8 個字元。實際已出現在 NAS 紀錄：OpenSubtitles 密碼、`VIDO_AUTH_PASSWORD`（登入密碼）；qBittorrent 密碼存的時候也一樣。

## 做法

- 密鑰名稱含 `password` 的，存的時候紀錄只寫 `****`。
- 開機設定紀錄的 `VIDO_AUTH_PASSWORD` 只寫 `****` 或 `(not set)`。

## Acceptance Criteria

1. [x] `opensubtitles_password`、`qbittorrent.password` 存的時候紀錄只有 `****`；API Key 照舊顯示頭尾。
2. [x] 開機紀錄的登入密碼只顯示 `****`。
3. [x] 測試；go 檢查全過。

## Dev Agent Record

- 已經寫進 NAS 紀錄的舊行不會消失，要等 Docker 紀錄輪替；擔心的話可以改密碼。
