# Backlog：照安裝指南裝 Vido，拿到的是每次合併就變的開發版

Status: ready-for-dev

**Source:** 2026-10-19 founder-growth 討論 Unraid Community Apps 時查到：`unraid-template/vido.xml` 的 Repository 指向 `:main`。2026-10-21 party mode（founder-pm、founder-growth、BMAD）盤點「新使用者第一次裝會撞到的問題」時再查，問題比原本記的大：

- 我查了 ghcr.io 的 tag：`:latest` 和 `:main` 的 digest 一模一樣（`sha256:ccf317eb…`，2026-10-21）；`:0.2` 是另一個（`sha256:86086d90…`）。
- 原因：`.github/workflows/docker.yml` 兩段 metadata-action 都有 `type=raw,value=latest,enable={{is_default_branch}}`，每次合併進 main 都會把 `:latest` 推到新的 main build。`docs/deployment.md:415` 也明寫「`latest` follows the main branch」。
- 所以安裝指南寫的 `:latest`（Unraid、Synology、QNAP、`docker-compose.nas.yml`）跟模板的 `:main` 其實是同一個東西：**外部測試者每次重開或更新，都會拿到最新的開發版**，一天可能變好幾次，也可能拿到還沒測完的版本。

⚖️ **Alexyu 2026-10-21：** party mode 選 A「首次安裝修正」，本單是第 1 張（10-27 檢查外部安裝之前要合）。目標是「只有發正式版，外部使用者才會更新」。

**做法（Winston，2026-10-21）：** 讓 `:latest` 只跟正式版，不再跟 main；模板改成 `:latest`。不選「指南全部改寫成 `:0.2`」——那樣發 v0.3 時，照指南裝的人永遠停在 0.2，每次發版還要改五份文件。`:latest` 改成「最新正式版」之後，指南現在寫的 `:latest` 就是對的，只有模板和描述文字要改。

Alexyu 自己的正式機用的是 `:main`（`docker inspect Vido` → `ghcr.io/j620656786206/vido:main`，模板 `my-Vido.xml` 也是 `:main`），**不受影響**；部署流程照舊。

## Story

身為照安裝指南把 Vido 裝在 NAS 上的外部使用者，
我想要我裝的版本只在作者發正式版時才會變，
這樣我不會某天重開 NAS 就拿到一個還沒測完、可能壞掉的版本。

## Acceptance Criteria

1. **CI：`:latest` 只在推正式版 tag 時移動。** `.github/workflows/docker.yml` 兩段 metadata-action（約 124 行的 per-arch、約 284 行的 merge job；檔內註解要求兩段逐字相同）：
   - `:latest` 只在推 `v*.*.*` 而且**不是預發佈版**（tag 裡沒有 `-`，例如 `v0.3.0-rc.1` 不算）時產生。
   - push 到 main 時，照舊產生 `:main` 和 `:sha-xxxxxxx`，**不**產生 `:latest`。
   - 動手前先讀 `docker/metadata-action` v6 README 的 `flavor` → `latest` 那段。預設的 `latest=auto` 在 `type=semver` 遇到正式版 tag 時，可能已經會自動加 `latest`。選一個做法，並在 workflow 註解寫清楚 `:latest` 從哪裡來：
     - (a) 刪掉 raw latest 那行，靠 `latest=auto`。
     - (b) 設 `flavor: latest=false`，raw 那行改成只在正式版 tag 啟用。
     建議 (b)，因為規則寫在一個地方，看得到。
   - 那行註解「`docker pull ghcr.io/.../vido` 預設抓 :latest — 沒有這行就會失敗」改寫成現況：已經有正式版，`:latest` 一定存在；它現在代表「最新正式版」。
   - 不動其他 tag 規則：`{{version}}`、`{{major}}.{{minor}}`、`{{major}}`、`sha-`、`ref,event=branch`。也不動 labels。`org.opencontainers.image.version` 那段註解說明的陷阱還是成立，不要省掉 tags 規則。
2. **Unraid 模板：** `unraid-template/vido.xml` 的 `<Repository>` 改成 `ghcr.io/j620656786206/vido:latest`。其他欄位不動。
3. **文件跟著改（中英文都要改，照 bilingual docs 規則）：**
   - `docs/deployment.md:415`：「`latest` follows the main branch」改成 `latest` 是最新正式版、`main` 是開發版（每次合併都會變，不建議外部使用者用）。列出的 tag 更新到 0.2，例如 `0.2.0`、`0.2`、`0`、`latest`。回滾範例的版本號跟著更新。
   - `docs/synology-installation-guide.zh-TW.md:56`、`docs/qnap-installation-guide.zh-TW.md:55`：「`latest` 會隨主線更新」改成「`latest` 是最新的正式版，作者發新版才會變」。指定版本的範例改成 `0.2.0`／`0.2`。
   - `docs/unraid-installation-guide.md`、`.zh-TW.md`：第 32 行已經是 `:latest`，不用改。確認第 22 行「下面每個欄位都跟它一致」在改完模板後是真的。如果這兩份有「怎麼更新」的段落，補一句：作者發新版後，在 Unraid 按 Check for Updates 才會拿到。
   - 開工時跑 `git grep -nE "vido:(main|latest)|follows the main|隨主線"`，排除 `_bmad-output` 和 `.claude`。其他對外文件（README、ROADMAP、`docs/*.md`）只要說到 `latest` 的意思，就一起改。只改說法，不改無關內容。
4. **不動正式機：** 不碰 NAS 上的 `my-Vido.xml`，也不碰 `.claude/memory` 裡寫 `:main` 的記憶（那是作者自己的部署方式，是對的）。
5. **驗證：**
   - 本機：如果有 `actionlint`，對 `docker.yml` 跑一次。沒有的話，用 `python3 -c 'import yaml; yaml.safe_load(open(...))'` 至少確認 YAML 合法。
   - PR 的 CI 全綠。PR 事件不推 image，看 `Docker` 那個 job 有沒有過。
   - **合併後一定要查**（合併本身就是一次 push 到 main）：等 main 的 Docker workflow 跑完，用 ghcr.io 的匿名 token 查 `:main`、`:latest`、`:0.2` 三個 tag 的 `docker-content-digest`。合格的樣子：
     - `:main` 換成新 digest。
     - `:latest` **還是合併前的值**（`sha256:ccf317eb…`，或合併前最後一次 main build 的 digest）。
     
     查法：`curl -s "https://ghcr.io/token?scope=repository:j620656786206/vido:pull"` 拿 token，再用 `curl -sI` 打 `/v2/j620656786206/vido/manifests/<tag>`，帶 `Accept: application/vnd.oci.image.index.v1+json`。數字寫進 Dev Agent Record。`:latest` 也跟著動了，就是 AC #1 沒做對，要開修正 PR。
6. **全綠：** `lint:all`（看「✖ N problems (X errors」那行）、`format:check`（Prettier 會檢查 md 和 yml）。

## Tasks / Subtasks

- [ ] T1：讀 metadata-action `flavor.latest` 文件，改 `docker.yml` 兩段，兩段要逐字相同（AC #1）
- [ ] T2：改模板（AC #2）
- [ ] T3：文件 grep，中英文一起改（AC #3）
- [ ] T4：本機驗 YAML、lint、format（AC #5、#6）
- [ ] T5：PR、CI 綠、合併，再查三個 tag 的 digest（AC #5）

## Dev Notes

- **目前 `:latest` 的內容：** 改完之後，`:latest` 會停在合併前最後一次 main build（約 75a21b32 之後那版），直到下一次發正式版。這版比 0.2.0 新，包含 #854「生成字幕不覆寫使用者的 .en.srt」。這是可以接受的，因為現有使用者不會被退版。下一次發 v0.2.1 或 v0.3.0 時，`:latest` 就會對齊正式版。**不要**手動 retag `:latest`，本機沒有 packages 寫入權限，而且那是發版的事。
- **為什麼不是只改模板：** 只改模板的話，指南的 `:latest` 還是每次合併都會變，外部使用者還是會拿到開發版。
- **為什麼不改成 `:0.2`：** 見上方 Winston 的說明。
- **「發版」流程不用改：** 推 `v0.x.y` tag 時，semver 規則照舊產生 `0.x.y`、`0.x`、`0`，再加 `latest`。
- Unraid 上架 Community Apps 另外需要 `ca_profile.xml`（https://ca.unraid.net/submit）。⚖️ Alexyu 2026-10-19：先不上架，本單不做。

### 不要做的事

- 不手動推、不手動改 registry 的 tag。
- 不動正式機、不部署（本單只改 CI 和文件，合併後也不用部署）。
- 不改 `type=ref,event=branch`：作者的正式機靠 `:main`。

### References

- [Source: .github/workflows/docker.yml:118-135（per-arch metadata，註解說兩段要相同）、:278-292（merge job metadata）]
- [Source: unraid-template/vido.xml:4]
- [Source: docs/deployment.md:405-422]
- [Source: docs/synology-installation-guide.zh-TW.md:55-56、docs/qnap-installation-guide.zh-TW.md:55]
- [Source: docs/unraid-installation-guide.zh-TW.md:12-32]
- [Source: ghcr.io tags/list 2026-10-21：main, latest, 0.1.0, 0.1, 0, 0.1.1, 0.2.0, 0.2；main 與 latest digest 相同]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-21 | 建單（Bob）。party mode 盤點時查到 `:latest` 等於 `:main`，所以範圍從「模板改 `:latest`」擴大成「`:latest` 只跟正式版＋模板改 `:latest`＋文件說法」。Winston 選「`:latest` 跟正式版」，不選「指南改成 `:0.2`」。 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
