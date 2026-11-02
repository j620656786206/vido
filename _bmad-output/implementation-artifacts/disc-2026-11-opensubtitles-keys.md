# Story disc-2026-11-opensubtitles-keys：設定頁可以填 OpenSubtitles 的 API Key 和帳號密碼

Status: done

**Source:** ⚖️ Alexyu 2026-11-02「Open Subtitles 我有帳號」。PR #971 時因為「下載要帳號＋密碼、使用者還沒帳號」而延後。

## 問題

Vido 有 OpenSubtitles 字幕來源，但它只在開機時從密鑰庫讀 `opensubtitles_api_key`／`_username`／`_password`，設定頁沒欄位、環境變數不讀；而且搜尋也強制先登入，只有 Key 沒帳號時連搜尋都會失敗。

## 做法

1. 金鑰清單加三項：`opensubtitles`、`opensubtitles_username`、`opensubtitles_password`，密鑰名稱沿用來源原本讀的名字；環境變數 `OPENSUBTITLES_API_KEY`／`_USERNAME`／`_PASSWORD` 當退路。
2. 密碼在清單裡永遠只顯示 `••••••••`（連開頭結尾都不露）；帳號欄位用一般文字框。
3. 來源改成每次搜尋／下載時讀最新設定；帳密變了就丟掉舊的登入，下次重新登入。
4. 只有 Key 時可以搜尋（不登入）；下載時沒帳號 → 清楚的錯誤「需要帳號和密碼」。

## Acceptance Criteria

1. [x] GET／PUT `/settings/keys` 支援三個新欄位；密碼遮罩固定 8 個點。
2. [x] 儲存後下一次搜尋就用新設定；改密碼會重新登入。
3. [x] 只有 Key 能搜尋；下載缺帳號回清楚錯誤。
4. [x] 設定頁三列（帳號是文字框、密碼是密碼框）；視覺基準更新。
5. [x] go／web 測試、staticcheck、lint、format 全過。

## Dev Agent Record

- 沒有真的 OpenSubtitles Key，無法對正式 API 實測；行為以假伺服器測試涵蓋（每次讀設定、只有 Key 不登入、換密碼重登、下載缺帳號報錯）。上線後等 Alexyu 貼上 Key 再從 NAS log 確認。
- 順手移除測試裡已經沒人用的假密鑰庫（staticcheck U1000）。
