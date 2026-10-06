---
name: next-story
description: vido 的「下一個 story」流水線：從 sprint-status 找下一張 → Bob create-story → dev-story → code-review → /ship，合併後自動接下一張。帶明確的停止規則（設計決策、動到 .pen、CI 非偶發紅燈就停）。當 Alexyu 說「下一個 story」「繼續做下一張」「做 <story-id>」「把 epic N 做完」時觸發。「從 backlog 挑一張」不在這裡——那要先列候選給他選。
---

# /next-story — vido 的 story 流水線

把「選定的下一張 story」從建檔一路帶到 PR 合併，然後接下一張，**中間不問**。
只在碰到真正屬於 Alexyu 的決定時停下來。

這份 skill 是 vido 專用，疊在 `/ship` 之上：`/ship` 管「改動 → 合併」那半段，本 skill 管「哪一張 → 建檔 → 實作 → 審查」那半段，並決定何時該停。

## 先分清楚：這是「做下一張」還是「挑一張」

| Alexyu 說的 | 意思 | 你做什麼 |
|---|---|---|
| 「下一個 story」「繼續做下一張」「做 <id>」「把 epic N 做完」 | 順序已經排好 | 走本 skill，直接做 |
| 「從 backlog 挑一張」「挑下一張」「接下來做什麼」（沒指定哪張） | 排優先序，是他的決定 | **不要做**。回 2～3 張候選（各一句：痛點、大小、要不要設計稿）＋你的推薦，然後停下來等他選。選完才進本 skill |

這條是 PR #652 的教訓（2026-10-02）：他要候選，我直接做完開 PR。「順序排好」和「要我排序」是兩件事。

## 0. 開工前

1. 確認 gh 帳號與分支規則照 `CLAUDE.md`（`j620656786206`、不碰 `main`、不用 worktree）。
2. 讀 `_bmad-output/implementation-artifacts/sprint-status.yaml`。
3. 看 `.claude/memory/MEMORY.md` 的「Development Workflow Feedback」一節；那裡的規則（先建 story、該拆就拆、拿掉 testid 先 grep E2E、push 前跑 staticcheck）每一張都適用。

## 1. 找下一張

照這個順序找，找到第一張就停：

1. 狀態 `in-progress` 的 story（上次沒做完的，先收尾）。
2. 狀態 `ready-for-dev` 的 story，依 epic 內的順序。
3. 都沒有 → 當前 `in-progress` epic 裡狀態 `backlog` 的下一張（還沒建檔，先走 §2）。
4. 連這個都沒有 → **停**。回報「這個 epic 做完了」，列出 retro 是否該跑，然後等 Alexyu 說下一個 epic 是哪個。不要自己跳到別的 epic。

Alexyu 有指定 `<id>` 就直接用那張，跳過搜尋。

## 2. 建 story（沒有檔就一定要建，再小也是）

story 檔不存在（`_bmad-output/implementation-artifacts/<id>.md`）→ 先跑 `bmad:bmm:workflows:create-story`（Bob）。
P3 一行修正也要，2026-09-29 被攔過（「應先請SM CS story?」）。

建完檢查：AC 寫得出可驗證的句子、規模沒超過上一張同類 story。超過就拆，拆完回報怎麼拆的，然後繼續。

## 3. 實作與審查

1. `bmad:bmm:workflows:dev-story`（Amelia）。
2. 設計相關的 story：對照 `_bmad-output/screenshots/` 的設計稿逐張驗，照 `feedback_design_verification.md`。
3. `bmad:bmm:workflows:code-review`：在範圍內的問題全修、各補測試；範圍外的寫進 story 的 Dev Agent Record，不修。
4. 回報 bug 之前，先確認資料是新的：NAS 要讀 `/mnt/cache` 路徑或直接打 API，隔幾分鐘再讀一次；本機 DB 要確認是這次跑出來的，不是上次留下的。讀到過期資料誤報過兩次。

## 4. 交付

呼叫 `/ship`。它會負責 adversarial review、開分支、commit、PR、CI 自癒、合併。
`/ship` 的規則全部沿用，特別是：

- 每個 check 都 `pass`／`skipping` 才合併，輪詢到沒有 pending（#574 教訓）。
- CI 修正可以放同一個 PR（`chore(ci): …`），不另開。
- CI 紅了先看歷史（`gh run list --workflow <name> --limit 30`）：長期綠、剛紅、跨分支同時紅 = 外部故障，重跑最多 2 次；斷斷續續紅很久 = 慢性病，立 Rule 24 條目。

合併後：sprint-status 標 `done` 並附 PR 編號與一句摘要、story 檔的 Dev Agent Record 補齊。

## 5. 接下一張

給一行摘要（story id、PR 連結、跑綠了哪些套件），**直接回 §1**。不要問「要不要繼續」。
Alexyu 說過「以後的項目不用問我 時間到就執行」。

## 停止規則（碰到就停，講清楚卡在哪，等他）

| 情況 | 為什麼要停 |
|---|---|
| story 需要產品或架構決定（範圍、UX 行為、新依賴、資料模型） | 這是他的決定，猜錯成本高 |
| 會動到 `ux-design.pen`，或設計稿跟 AC 對不上 | 設計是 Sally／Alexyu 的，不是 dev 自己補 |
| CI 紅燈不是偶發：重跑 2 次還紅，且歷史顯示不是外部故障 | 繼續硬修會把 CI 修正和功能混到看不懂 |
| Visual Regression 出現**真的 diff**（不是缺 baseline） | 要他確認是不是故意的 |
| 這個 epic 做完了 | 下一個 epic 是排序決定 |
| 新功能、不在任何 epic 裡的點子 | 先 party mode／founder-pm，他拍板才開單（`feedback_new_feature_needs_party_mode_first.md`） |

停下來時的格式：

```
停在：<story id> 的 <哪一步>
原因：<一句話>
需要你決定：<A 還是 B；我會選哪個、為什麼>
已完成的部分：<哪些已合併／已寫好但沒 push>
```

不算停止理由：lint 失敗、缺 `-linux` baseline、gh 帳號被切走、偶發的 runner 斷線。這些自己處理。

## 不要做的事

- 不要「從 backlog 挑一張」然後直接做。
- 不要跳過 create-story。
- 不要為了趕進度把兩張 story 塞進一個 PR。
- 不要在 story 檔沒更新的情況下合併。
- 不要把 `/ship` 的「只在產品／架構決定時停」解讀成「設計 diff 也不用停」。
