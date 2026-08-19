# Story: Bugfix E — qBT 錯誤/暫停種子被計成「排隊中」（Activity 誠實分桶）

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Source:** NAS BUG HUNT（2026-07-13, party-mode）P1；2026-08-19 What'Sub 對抗重審列為無悔四項之 N2。
**Risk: 🟢 LOW** —— 單點算式修正＋additive 欄位；比照 `project_qbt_state_mapping`（Sonarr/Radarr 業界標準）。

---

## Story

As a NAS owner opening the Activity hub every day,
I want errored and paused torrents counted in their own buckets instead of being swept into「排隊中」,
so that 3,068 個壞掉的種子不再偽裝成正常排隊 —— Activity 與 `/downloads/counts` 對同一批種子說同一句話。

---

## Context —— 缺陷機制（live NAS 實證）

`ActivityService.downloadsSection`（`activity_service.go:190-193`）：

```go
queued := c.All - c.Downloading - c.Completed - c.Seeding
```

減式漏掉 `Error` 與 `Paused` → ERRORED / PAUSED 全數落入 queued。live NAS 實測：activity 回報 queued **3102**（= 3068 error + 34 paused），而 `/downloads/counts` 同時正確回報 `error: 3068` —— **兩個端點對同一批 3,641 顆種子自相矛盾**。`DownloadCounts` 結構早就有 `Paused`/`Error` 欄位（`qbittorrent/torrent.go:130-137`），資料源齊備，純算式與呈現層缺陷。

## Acceptance Criteria

### AC #1 — BE 分桶修正
- `queued := c.All - c.Downloading - c.Completed - c.Seeding - c.Error - c.Paused`（clamp ≥ 0 照舊）。
- `DownloadsSection`（`activity_service.go:93-99`）新增 additive 欄位 `Errored int \`json:"errored"\``、`Paused int \`json:"paused"\``。既有 key（`status/downloading/queued/total/error`）**一個不動**（`error` 是字串錯誤訊息欄位，新計數欄位取名 `errored` 避開撞名）。additive、不 bump；wire-shape 測試釘 snake_case key。

### AC #2 — FE Activity hub 誠實呈現
- `ActivityHub.tsx:198` detail 字串擴為：`{downloading} 個進行中 · {queued} 個排隊` ＋ **當 `errored > 0` 時**追加 ` · {errored} 個錯誤`（`--error` 色調，與既有 badge token 一致；`paused > 0` 同理顯示「暫停」，`--text-muted`）。零值不顯示 —— 健康系統的畫面 byte-unchanged。
- 側欄狀態 strip（Epic 0 F2）若同源消費此 section，同步繼承；若另有算式，本 story 僅修 ActivityHub，strip 差異記入 Dev Notes 供 follow-up。

### AC #3 — 測試
- BE：table-driven 分桶測試 —— error/paused/混合/全零四案；queued 不再含 error/paused；additive 欄位序列化 key 斷言。
- FE：`ActivityHub.spec.tsx` —— errored>0 顯示、=0 不顯示（回歸釘）。
- 既有 activity/downloads 測試全綠。

### AC #4 — 範圍紅線
- 不動 `/downloads/counts`（它本來就是對的）。
- 不動 qBT state→bucket 對映本身（`project_qbt_state_mapping` 已定，對映層非本案）。
- 效能問題（>1.1s）屬 bugfix-f，不在此案。

## Tasks / Subtasks

- [ ] Task 1: AC #1 算式＋additive 欄位（activity_service.go）
- [ ] Task 2: AC #2 ActivityHub 呈現
- [ ] Task 3: AC #3 測試
