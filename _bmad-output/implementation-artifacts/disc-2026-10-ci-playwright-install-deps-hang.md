# disc-2026-10-ci-playwright-install-deps-hang：CI 的 E2E 不再卡在「安裝系統套件」20 分鐘被取消

Status: review

<!-- SM Bob create-story 2026-10-27；行號為 main `f19cf73d` 之後。Rule 24 lane ③，Alexyu 10/27 指定做這張。 -->

**Source:** 2026-10 一週 6 次（#912、#914 兩次、#917、#918、#920）：某個 E2E 分片卡在 `Install Playwright system deps (browsers from cache)` 直到 job 的 20 分鐘上限被取消，測試一條都沒跑，每次重跑就過。#917 那次還因為合併閘門的 bug 被放行（`feedback_ship_merge_gate_poll_until_settled`）。

## 查到的事（#918 那次，job 122009583717）

- 不是「卡死」，是**很慢**：`npx playwright install-deps chromium webkit` 要從 `azure.archive.ubuntu.com` 抓 **181 個套件、119 MB**（webkit 要的 gstreamer／ffmpeg 那一串）。
- 那次每秒只有幾十 KB：`libavcodec60`（5.8 MB）抓了 1 分 38 秒、`libflite1`（13.6 MB）抓了 3 分 40 秒。04:27 開始，04:47 抓到第 70 個時 job 被取消。
- 瀏覽器本身已經有快取（`~/.cache/ms-playwright`），系統套件沒有，所以每次都要重抓這 119 MB。

## Acceptance Criteria

1. **系統套件快取**：E2E 分片把 install-deps 抓到的 `.deb` 存進 Actions 快取（key 綁 OS＋Playwright 版本＋瀏覽器組合，跟瀏覽器快取同一套規則）；下次先把快取的 `.deb` 放進 apt 的下載資料夾，apt 只會去鏡像抓快取裡沒有的新版套件。
2. **抓過的檔留下來**：明確設定 apt 保留下載的套件，存快取才有東西。
3. **卡住時快一點失敗、試第二次**：安裝那一步每次最多 6 分鐘、試兩次（第二次接著抓，中途被打斷的 dpkg 先收尾），整步最多 13 分鐘；不會再拖到 job 的 20 分鐘上限才知道。
4. **快取只存一次**：快取沒命中時才存；四個分片搶同一個 key，只有一個會存成功，其他只是警告，不算失敗。
5. **其他不變**：瀏覽器快取、測試本身、其他 job 都不動。

## 不要做的事

- 不換成 Playwright 官方容器映像（整個 job 要搬進容器，後端、靜態伺服器都要跟著驗，改動大）。
- 不改 `visual-regression.yml`（它只裝 chromium，套件少，這週沒卡過）。

## Tasks

- [x] Task 1 — `test.yml` E2E 分片：還原／種入／保留／存 apt 套件快取（AC 1、2、4）
- [x] Task 2 — 安裝步驟加每次 6 分鐘、兩次的重試與步驟上限（AC 3）
- [x] Task 3 — actionlint；PR 的 CI 跑一次（快取沒命中、存起來），再重跑 E2E 確認命中（AC 1、5）

## Dev Notes

- 工作流程：`.github/workflows/test.yml` 的 `test-e2e-sharded`（`timeout-minutes: 20`），瀏覽器快取在 `Cache Playwright browsers`。
- `actions/cache/restore@v4` 的 `cache-primary-key` 輸出給 `actions/cache/save@v4` 用；分開還原與存，是為了裝完套件就存，不必等測試成功。
- `timeout` 對整個 process group 送訊號，`sudo apt-get` 也會一起停；被打斷的安裝用 `dpkg --configure -a` 收尾再試。

### Time-dependent visual coverage

- N/A — 只有 CI。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- `test.yml`：`Restore apt packages for Playwright`（cache/restore）→ `Seed apt with the cached packages`（`APT::Keep-Downloaded-Packages "true"`、刪 `docker-clean`、把快取的 `.deb` 複製進 `/var/cache/apt/archives/`）→ 兩個安裝步驟改成 `timeout 6m` × 2 次＋`timeout-minutes: 13` → 沒命中時 `Keep the apt packages for the next run`（複製回 `~/apt-cache`）＋`Save apt packages`（cache/save）。
- `actionlint`：新加的部分沒有警告（檔案裡原有的 SC2064／SC2034 不是這次的）。

### File List

- `.github/workflows/test.yml`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
