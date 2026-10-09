# Disc：「缺中文」徽章照設計稿改成橘色

Status: done

**Source:** filed 2026-10-06 by `disc-2026-10-subtitle-filter-disagrees-with-badges`（Sally T4 時發現，Rule 24 lane ③）。⚖️ 2026-10-09 **Alexyu 裁定 A：設計稿對，改程式**。

## Story

身為片庫的使用者，
我要「缺中文」標籤用代表「可以處理」的橘色，
這樣一眼分得出「缺中文（可以去產生字幕）」和「無字幕源／已略過／未翻譯」（灰色）。

## 依據

- 設計稿 `flow-j-specs/j2-d`：J2-3 與 J2-6 的「缺中文」都是 warning 橘；J2-6 註明「沿用原『缺字幕』樣式・改字」。
- J2-2 原則：顏色是急迫度，不是結果。
- 程式 `apps/web/src/utils/libraryStatus.ts` 原本用 `TINT.neutral`（灰），與設計稿不一致。

## Acceptance Criteria

1. `deriveSubtitleStatus` 對 `chineseSubtitle === 'none'` 回傳 `TINT.warning`（`--warning-tint` 底＋`--warning-text` 字；`*-text` 版本已由 `styles-contrast.spec.ts` 守 AA）。
2. 無字幕源／已略過／未翻譯維持灰色；繁中／中文（success）、簡中（info）不變。
3. 所有用同一函式的地方（海報、清單、詳情、首頁 Hero）一起變色，不另改。
4. 不動 `ux-design.pen`。
5. 單元測試改成檢查橘色；lint／format／vitest 綠；畫面比對若有真的差異屬預期（裁定即確認）。

## Dev Agent Record

**Agent:** Opus 5.5。改動一行＋測試，沒有另派換模型 CR（純樣式 token 對齊設計稿）。

- `libraryStatus.ts`：`缺中文` → `TINT.warning`，註解寫明依據。
- `libraryStatus.spec.ts`：斷言 `--warning-tint`／`--warning-text`、不再是 `--bg-tertiary`。
- 檢查：`vitest`（libraryStatus 48、PosterCardV2、homepage 共 161）綠；eslint／prettier 綠；E2E 沒有斷言此顏色；畫面比對 gallery 固定資料未找到 `chineseSubtitle: 'none'` 的樣本，預期無差異，以 CI 為準。
- PR #739。

### File List

- `apps/web/src/utils/libraryStatus.ts`
- `apps/web/src/utils/libraryStatus.spec.ts`
- `_bmad-output/implementation-artifacts/disc-2026-10-missing-chinese-badge-tint-drift.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
