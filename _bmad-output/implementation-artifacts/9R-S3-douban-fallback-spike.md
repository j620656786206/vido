# Spike 9R-S3 — Douban Localization Fallback

Status: done

**Epic:** epic-9R-subtitle-route-c (Spike S3) · **Gates:** any 在地化 path leaning on Douban metadata
**Owner:** dev (Amelia) · **Date:** 2026-07-05 · **Effort:** S

## Question (from `subtitle-route-c-stories-2026-06.md` §Spikes/S3)

> Does vido's Douban scraper return real zh metadata live (it JS-renders)?
> **Pass:** end-to-end parse of a real result, else drop Douban from the chain.

## TL;DR verdict — **DROP Douban from the 在地化 chain** (with one salvageable crumb)

- ❌ **As-shipped scraper: FAIL live.** Probe `cmd/douban-spike` (committed, rerunnable):
  search parsed **0 results** (JS shell — none of our three goquery selector generations
  match); ALL subject-page paths (detail / review-summary / SearchByID) **blocked after 5
  retries each — 18/19 requests blocked**.
- 🔒 **Root cause (raw evidence):** `movie.douban.com/subject/*` now answers anonymous
  requests with **HTTP 302 → `sec.douban.com/c?r=…`** — an anti-bot challenge gate. Same
  external-uncontrollable failure class as Zimuku's Yunsuo WAF (ADR fragility finding
  confirmed again).
- 💎 **Salvageable crumb:** the search shell (`search.douban.com/movie/subject_search`)
  embeds **plain-JSON `window.__DATA__`** with real results — id, year, cover, genres,
  director+cast names, and **regional title variants incl. 台譯**(`潜行凶间(港) /
  全面启动(台)`) — parseable with a regex+`json.Unmarshal`, no JS engine. One GET,
  currently unchallenged. **No plot text**, Simplified only.
- Consequence for 9R-13 / Section E: **metadata localization must NOT depend on Douban.**
  zh source = TMDB zh-TW + LLM translation + glossary (already the keystone plan). Douban at
  most an OPTIONAL title-variant hint via `__DATA__` — never a required chain link.

## Probe results (2026-07-05, residential TW network, `go run ./cmd/douban-spike`)

| Probe | Path | Result |
|---|---|---|
| P1 | `Searcher.Search("全面啟動", movie)` | HTTP 200, **0 parsed** (selectors `.result-list .result` / `.item-root` / `.sc-bZQynM` all absent — page is a 13KB JS bootstrap) |
| P2 | `Scraper.ScrapeDetail("3541415")` (Inception) | **BLOCKED** ×5 retries (`douban: blocked - unexpected content type`) |
| P3 | `Scraper.ScrapeReviewSummary("3541415")` | **BLOCKED** ×5 retries |
| P4 | `Searcher.SearchByID("3541415")` | **BLOCKED** ×5 retries |
| metrics | — | total=19 ok=1 blocked=18 retries=15 |

Raw evidence: `curl -I movie.douban.com/subject/3541415/` (browser UA) → `302
location: https://sec.douban.com/c?r=…`; search HTML contains `window.__DATA__ = {"count":
15, "items": [{"id": 3541415, "abstract": "…全面启动(台)…", "abstract_2":
"克里斯托弗·诺兰 / 莱昂纳多·迪卡普里奥 / …"}]}`.

Caveat: single vantage point (TW residential IP). The NAS may see different treatment —
re-probe there is one command (`go run ./cmd/douban-spike`), but the architecture decision
should assume the gate exists (it is Douban policy, not our IP's bad luck).

## Fallout beyond 9R (Rule 24 triage)

The **shipped Epic 12 豆瓣短評 detail block** (F-6, `ScrapeReviewSummary`) rides the same
blocked path → live it degrades to its fail-soft state (per-section, by design) but is
functionally dead from this vantage. Filed **③ `bugfix-douban-sec-gate-liveness`**
(sprint-status, at discovery time): decide keep-as-failsoft / hide the block / add a
`__DATA__`-based partial degradation. NOT a 9R dependency.

## Implications

1. **9R-13 (metadata localization):** zh chain = TMDB zh-TW fields + 9R-7 LLM translation +
   9R-6 glossary. Douban removed as a planned source. (Doc updated expectation only — 9R-13's
   entry already lists no Douban dependency.)
2. **Optional future:** a `__DATA__` search parser could feed the glossary with 台/港 title
   variants (nice glossary seed). File only if a real story needs it — not tracked now beyond
   this note.
3. **Do NOT invest in challenge-solving** (sec.douban.com) — same arms-race class the ADR
   already rejected for Zimuku.

## Artifacts

- `apps/api/cmd/douban-spike/main.go` — committed live probe (route-c-poc precedent);
  rerunnable anywhere (`go run ./cmd/douban-spike`).

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | Spike executed (Amelia): live probe + raw-curl evidence. Verdict: as-shipped scraper fails live (search=JS shell, subject pages behind sec.douban.com challenge); Douban dropped from the 在地化 chain; `window.__DATA__` noted as optional title-variant crumb; ③ filed for the shipped 豆瓣短評 block's liveness. Status → done. |
