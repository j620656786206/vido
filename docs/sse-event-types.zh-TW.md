# SSE 事件類型參考文件

> **端點：** `GET /api/v1/events`
> **協定：** Server-Sent Events (text/event-stream)
> **最後更新：** 2026-03-26

## 概述

Vido 使用 Server-Sent Events 在三個功能中提供即時進度更新：媒體庫掃描、字幕搜尋/下載、批次字幕處理。SSE Hub 使用扇出模式，非阻塞式廣播，每個客戶端有獨立的緩衝通道（容量 100）。

## 連線方式

使用瀏覽器 `EventSource` API 連線至 SSE 端點：

```typescript
const es = new EventSource('/api/v1/events');

es.addEventListener('scan_progress', (e: MessageEvent) => {
  const event = JSON.parse(e.data);
  console.log(event.data);
});
```

連線成功後，伺服器會發送 `connected` 事件：

```json
{
  "type": "connected",
  "data": {
    "clientId": "uuid",
    "message": "Connected to event stream"
  }
}
```

每 30 秒發送一次 `ping` 保活訊號：

```json
{
  "type": "ping",
  "data": {
    "timestamp": 1711468800
  }
}
```

## 事件類型

### scan_progress

媒體庫掃描進度更新。掃描期間每處理 10 個檔案廣播一次，完成時再廣播一次。

**發布者：** `ScannerService.broadcastProgress()`

| 欄位           | 型別     | 說明                       |
| -------------- | -------- | -------------------------- |
| `filesFound`   | `int`    | 目前已發現的檔案總數       |
| `filesCreated` | `int`    | 新建的媒體項目數           |
| `filesUpdated` | `int`    | 已更新的項目數             |
| `filesSkipped` | `int`    | 略過的檔案數（已建立索引） |
| `filesRemoved` | `int`    | 移除的孤立項目數           |
| `errorCount`   | `int`    | 遇到的錯誤數               |
| `currentFile`  | `string` | 目前正在處理的檔案         |
| `percentDone`  | `int`    | 進度百分比（0-100）        |
| `isActive`     | `bool`   | 掃描是否仍在進行           |
| `startedAt`    | `string` | 掃描開始的 ISO 8601 時間戳 |

**範例 payload：**

```json
{
  "id": "uuid",
  "type": "scan_progress",
  "data": {
    "filesFound": 142,
    "filesCreated": 38,
    "filesUpdated": 2,
    "filesSkipped": 100,
    "filesRemoved": 0,
    "errorCount": 2,
    "currentFile": "/media/movies/Inception.2010.BluRay.mkv",
    "percentDone": 67,
    "isActive": true,
    "startedAt": "2026-03-26T14:30:00Z"
  }
}
```

---

### scan_complete

媒體庫掃描成功完成時發送一次。包含最終統計數據。

**發布者：** `ScannerService.broadcastScanComplete()`

| 欄位            | 型別     | 說明                  |
| --------------- | -------- | --------------------- |
| `files_found`   | `int`    | 發現的檔案總數        |
| `files_created` | `int`    | 新建的媒體項目數      |
| `files_updated` | `int`    | 已更新的項目數        |
| `files_skipped` | `int`    | 略過的檔案數          |
| `files_removed` | `int`    | 移除的孤立項目數      |
| `error_count`   | `int`    | 遇到的錯誤數          |
| `duration`      | `string` | 掃描耗時（如 "2.5s"） |

**範例 payload：**

```json
{
  "id": "uuid",
  "type": "scan_complete",
  "data": {
    "files_found": 142,
    "files_created": 38,
    "files_updated": 2,
    "files_skipped": 100,
    "files_removed": 2,
    "error_count": 0,
    "duration": "4.231s"
  }
}
```

---

### scan_cancelled

使用者取消掃描時發送。包含取消時的部分統計數據。

**發布者：** `ScannerService.broadcastScanCancelled()`

| 欄位          | 型別  | 說明                 |
| ------------- | ----- | -------------------- |
| `files_found` | `int` | 取消前已發現的檔案數 |
| `error_count` | `int` | 取消前遇到的錯誤數   |

**範例 payload：**

```json
{
  "id": "uuid",
  "type": "scan_cancelled",
  "data": {
    "files_found": 42,
    "error_count": 1
  }
}
```

---

### subtitle_progress

單一媒體項目的字幕管線進度。在搜尋和下載過程中，每個管線階段都會廣播。

**發布者：** `Engine.broadcastStatus()`、`SubtitleHandler.broadcastStatus()`

| 欄位         | 型別     | 說明                    |
| ------------ | -------- | ----------------------- |
| `media_id`   | `string` | 媒體項目 UUID           |
| `media_type` | `string` | `"movie"` 或 `"series"` |
| `stage`      | `string` | 目前管線階段（見下表）  |
| `message`    | `string` | 人類可讀的狀態訊息      |

**管線階段（依序）：**

| 階段          | 說明                                         |
| ------------- | -------------------------------------------- |
| `searching`   | 查詢字幕來源（Assrt、Zimuku、OpenSubtitles） |
| `scoring`     | 依語言、解析度、來源信任度排名結果           |
| `downloading` | 下載字幕檔案                                 |
| `converting`  | OpenCC 語言轉換（簡體 → 繁體）               |
| `placing`     | 將字幕檔案寫入媒體旁的磁碟位置               |
| `complete`    | 字幕已成功放置                               |
| `failed`      | 任何階段發生錯誤                             |

**範例 payload：**

```json
{
  "type": "subtitle_progress",
  "data": {
    "media_id": "550e8400-e29b-41d4-a716-446655440000",
    "media_type": "movie",
    "stage": "converting",
    "message": "Converting simplified to traditional Chinese..."
  }
}
```

---

### subtitle_batch_progress

批次字幕處理進度。每處理完一個項目後廣播，批次完成時再廣播一次。

**發布者：** `BatchProcessor.broadcastProgress()`

| 欄位            | 型別     | 說明                      |
| --------------- | -------- | ------------------------- |
| `batch_id`      | `string` | 批次操作 UUID             |
| `total_items`   | `int`    | 批次中的項目總數          |
| `current_index` | `int`    | 目前項目索引（從 0 開始） |
| `current_item`  | `string` | 正在處理的項目名稱        |
| `success_count` | `int`    | 成功處理的項目數          |
| `fail_count`    | `int`    | 失敗的項目數              |
| `status`        | `string` | 批次狀態（見下表）        |

**狀態值：**

| 狀態        | 說明                 |
| ----------- | -------------------- |
| `running`   | 批次正在處理中       |
| `complete`  | 所有項目已處理完成   |
| `cancelled` | 使用者取消了批次     |
| `error`     | 嚴重錯誤導致批次停止 |

**範例 payload：**

```json
{
  "type": "subtitle_batch_progress",
  "data": {
    "batch_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "total_items": 42,
    "current_index": 15,
    "current_item": "Inception (2010)",
    "success_count": 12,
    "fail_count": 3,
    "status": "running"
  }
}
```

---

### notification

通用通知事件。已定義但目前尚未使用。

| 欄位      | 型別     | 說明                               |
| --------- | -------- | ---------------------------------- |
| `level`   | `string` | `"error"`、`"warning"` 或 `"info"` |
| `title`   | `string` | 通知標題                           |
| `message` | `string` | 人類可讀的訊息                     |

---

## 實作備註

### Hub 架構

- **廣播通道：** 256 事件緩衝，非阻塞式發送（滿時丟棄）
- **客戶端通道：** 每客戶端 100 事件緩衝
- **保活機制：** 每 30 秒發送 ping
- **Gzip 排除：** SSE 端點從 gzip 中介軟體排除以維持串流

### 前端連線模式

**延遲連線**（建議）：僅在需要即時更新時連線。閒置時斷開，避免阻塞 Playwright `networkidle` 偵測。

```typescript
useEffect(() => {
  if (!isActive) {
    eventSource?.close();
    return;
  }
  const es = new EventSource('/api/v1/events');
  // ... 監聽事件
  return () => es.close();
}, [isActive]);
```

**輪詢備援：** SSE 連線失敗時，回退至輪詢對應的 REST 端點。

### 原始碼位置

| 元件             | 路徑                                             |
| ---------------- | ------------------------------------------------ |
| Hub + 事件類型   | `apps/api/internal/sse/hub.go`                   |
| SSE 處理器       | `apps/api/internal/sse/handler.go`               |
| Hub 測試         | `apps/api/internal/sse/hub_test.go`              |
| 掃描發布者       | `apps/api/internal/services/scanner_service.go`  |
| 字幕發布者       | `apps/api/internal/subtitle/engine.go`           |
| 批次發布者       | `apps/api/internal/subtitle/batch.go`            |
| 字幕處理器發布者 | `apps/api/internal/handlers/subtitle_handler.go` |
| 掃描進度 Hook    | `apps/web/src/hooks/useScanProgress.ts`          |
| 字幕搜尋 Hook    | `apps/web/src/hooks/useSubtitleSearch.ts`        |
