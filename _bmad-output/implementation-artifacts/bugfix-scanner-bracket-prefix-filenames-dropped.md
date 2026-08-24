# Story bugfix: scanner drops bracket-prefixed episode filenames

Status: done

## Story

As someone whose TV library uses fansub / release-group filenames (`[Grp] Show S01E05.mkv`, `[劇名].Some.Drama.S01E48.mp4`),
I want those files to land as real episodes instead of silently collapsing into a single placeholder row,
so that my library shows what is actually on disk instead of ~17% of it vanishing with no error.

## Evidence (measured 2026-08-24 against the live NAS, not inferred)

After the `bugfix-c` wipe+rescan of `192.168.50.52`:

- Disk: **2467** TV video files. DB: **2056** episodes. Gap: **412** files.
- Those 412 files did **not** error — they collapsed into **20 rows with `episode_number = 0`** (the worst-hit series: 96 files → 1 row, 69 → 1, 47 → 1). Season was also wrong: every collapsed row says `season_number = 1`, including files under `Season02/` named `...S02E08...`.
- Scanner reported `files_found=2522, files_created=2522, error_count=0` — a clean bill of health for a scan that created 2111 rows.
- Running the real 412 filenames through the real `TVParser` (diagnostic harness): **408 → `needs_ai`** (parser gives up entirely), 4 → parse fine but are genuine duplicate-episode files.
- Prototype of the fix below on the same 412: **386 rescued, 22 left, 94.7% coverage**.

### Root cause 1 — leading bracket is an unconditional give-up

`apps/api/internal/parser/tv_parser.go:46-49`

```go
// Skip anime fansub bracket format - these need AI
if animePattern.MatchString(filename) {
    return result
}
```

`animePattern = ^\[.+?\]` (`patterns.go:29`). **Any** filename starting with `[` is abandoned before a single pattern is tried — including names carrying a perfectly standard `SxxExx`. Confirmed live: `[劇名].Some.Drama.S01E48...`, `[劇名].Some.Detective.S01E24...`, `[劇名].Some.Period.Drama.2026.S01E25...`, `[Grp] ... S01E05 ...`.

The design intent was "hand these to AI". That hand-off is dead: `AI_PROVIDER=gemini` and the default model was retired by Google on 2026-06-01 (`bugfix-gemini-default-model-retired`); NAS shows `ai_cache=0` and `parse_jobs=0`. So the give-up is terminal.

`movie_parser.go:31,56` carries the identical give-up — same defect, movie path.

### Root cause 2 — video resolution false-matches the `1x05` guard

`tvShowPattern = (?i)[Ss]\d{1,2}[Ee]\d{1,3}|\d{1,2}x\d{1,3}` (`patterns.go:26`).

`1920x1080` contains `20x108`, which matches `\d{1,2}x\d{1,3}`. Verified:

| filename | `tvShowPattern` hit |
|---|---|
| `Some Anime Title - 82 (B-Global Donghua 1920x1080 HEVC AAC MKV).mkv` | `20x108` |
| `Show - 05 (1280x720).mkv` | `80x720` |

`parseAnimeDashFormat` (`tv_parser.go:274`) bails whenever that guard hits, so **every anime-dash file whose name states its resolution can never parse**. `tvPatternAlt` itself does not misfire (it anchors after the title separator), so this is a false-negative guard, not data corruption. The same guard also gates `movie_parser.go:26,51`.

### Why the damage is invisible

`IngestEpisodeFile` (`media_ingest_service.go`) defaults an unparsed file to `season=1, episode=0`, and `EpisodeRepository.Upsert` (`episode_repository.go:387`) resolves identity via `FindBySeriesSeasonEpisode(series_id, season_number, episode_number)`. Every unparsed file in a series therefore targets the same `(series, 1, 0)` row and overwrites its `file_path`. No constraint is violated, so no error is raised. `processTVFile` increments `FilesCreated` per *file*, not per row created — which is why the counter reports 2522.

**Not a regression.** The pre-wipe backup shows `old_episodes=2060` ≈ `new 2056`; the same files never made it. Before `bugfix-b`, ~620 of them were mis-filed into `movies` (wrong, but visible). `bugfix-b` correctly stopped polluting `movies` and, without a parser fix, converted "wrong data" into "missing data".

## Acceptance Criteria

1. A leading release-group tag no longer aborts parsing. `TVParser.Parse` strips one or more leading `[...]` tags (plus any trailing `.`/`_`/`-`/space separators) and runs the existing pattern chain against the remainder. `ParseResult.OriginalFilename` still carries the untouched filename.
2. Behaviour for names WITHOUT a leading tag is bit-for-bit unchanged — the strip is a no-op there, and no currently-passing filename changes its parse result. **⚠️ AC amended during implementation:** the original wording ("existing suites stay green with **zero edits to their expectations**") was wrong and had to be relaxed. Six fixtures across three files asserted the *old* give-up behaviour and legitimately had to flip — see Completion Notes → "Intentional test-expectation changes". The invariant that actually holds, and that the suites now guard, is: *no filename lacking a leading tag changed its result*.
3. If the stripped remainder still does not parse, the result stays `ParseStatusNeedsAI` exactly as today (no fabricated season/episode).
4. If a filename is nothing but tags (strip yields empty), the result stays `ParseStatusNeedsAI` and does not panic.
5. `MovieParser.Parse` / `CanParse` receive the same symmetric treatment — fixing only the TV path would leave the identical defect live one file away (the `bugfix-settings-design-ref-6uctx-sweep` "fix one, create inconsistency" lesson).
6. `tvShowPattern`'s `NxNN` alternative requires separator/anchor boundaries so a resolution token (`1920x1080`, `1280x720`) no longer matches, while a genuine `1x05` / `Show.12x05.mkv` still does.
7. `TVParser.CanParse` and `MovieParser.CanParse` agree with their `Parse` counterparts — a filename `Parse` now succeeds on must not have `CanParse` return false.
8. Regression corpus: the 412 real NAS filenames are committed as `testdata` and asserted at **≥ 94%** parse coverage, with the exact rescued count pinned so a future change that silently lowers it turns the test red.
9. Gates: `pnpm nx test api` green, `gofmt` clean, `pnpm run format:check` green. Zero frontend changes.
10. Verified on the live NAS: rescan produces episode rows matching disk file count within the documented residual, and `episode_number = 0` collapse rows drop to ~0.

## Tasks / Subtasks

- [x] Task 1 — RED: pin the defect with the real corpus (AC: #8)
  - [x] 1.1 Commit the 412 real filenames as `apps/api/internal/parser/testdata/nas-tv-filenames-2026-08.txt`
  - [x] 1.2 Write a table test asserting current coverage is broken, then flip it to the post-fix threshold
- [x] Task 2 — Fix root cause 1: leading-tag strip (AC: #1, #2, #3, #4, #5, #7)
  - [x] 2.1 Add `leadingTagPattern` + `StripLeadingTags` helper
  - [x] 2.2 `TVParser.Parse` / `CanParse`: match against the stripped name instead of aborting
  - [x] 2.3 `MovieParser.Parse` / `CanParse`: same symmetric change
- [x] Task 3 — Fix root cause 2: resolution false-positive (AC: #6)
  - [x] 3.1 Anchor `tvShowPattern`'s `NxNN` alternative with separator boundaries
  - [x] 3.2 Test both directions: `1920x1080` must NOT match, `Show.1x05.mkv` must
- [x] Task 4 — Gates + NAS verification (AC: #9, #10)
  - [x] 4.1 `pnpm nx test api`, `gofmt -l`, `format:check`
  - [x] 4.2 Deploy to NAS, rescan, compare disk-vs-DB counts and `episode_number=0` rows

## Dev Notes

- **Do NOT delete the `needs_ai` fallback.** It stays the terminal state for genuinely unparseable names (AC #3). This story widens what gets *tried*, it does not remove the safety net.
- **Do NOT thread directory context into the parser in this story.** A `Show/SeasonNN/` fallback would cover 402/412 (98%) on its own and is tempting, but it is a scanner-layer change with a much larger blast radius. Filed separately if the residual after this fix still matters — the strip alone already reaches 94.7%.
- **Known residual (~22 files), deliberately out of scope:**
  - `Become.a.Farmer.2023.S01E00...` — `E00`. Parses, but `episode = 0` is the same slot the unparsed default uses, so a real "episode 0" special still collides. Distinguishing parsed-zero from unparsed-zero is a `media_ingest_service` change, not a parser one.
  - `Some Anime Title - 8x (...1920x1080...)` — should be rescued by Task 3; re-measure after and update the pinned number.
- Quality extraction runs on the stripped name. Markers (`1080p`, `WEB-DL`, `HEVC`) always follow the title so they survive; only a leading `[Group]` tag is lost as a release-group source, and `releaseGroupPattern` reads the trailing `-GROUP` form anyway. Acceptable, noted.
- Rule 7 / 10 / 20: **N/A** — no error codes, no routes, no wire contract. Rule 23: N/A (no frontend).
- The counter lie (`files_created` counting files, not rows) is tracked separately as `bugfix-scanner-counter-reports-phantom-creates`; **do not fix it here**, but be aware it is why this bug went unseen.

### Time-dependent visual coverage

- N/A — backend only, no `apps/web/src/components/**` touched.

### References

- [Source: apps/api/internal/parser/tv_parser.go:46-49, 274 — the two give-up sites]
- [Source: apps/api/internal/parser/patterns.go:26, 29 — `tvShowPattern`, `animePattern`]
- [Source: apps/api/internal/parser/movie_parser.go:26-33, 51-58 — symmetric defect]
- [Source: apps/api/internal/services/media_ingest_service.go — `IngestEpisodeFile` season=1/episode=0 default]
- [Source: apps/api/internal/repository/episode_repository.go:387 — `Upsert` identity key]
- [Source: sprint-status.yaml `bugfix-scanner-bracket-prefix-filenames-dropped` — corrected diagnosis]

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Claude Fable 5)

### Debug Log References

- Diagnostic harness over the 412-name corpus, run against the unmodified parser: **408 → `needs_ai`**, 4 → parse but are duplicate-episode files.
- Bracket-strip prototype alone: 386 rescued (94.7%). With the `tvShowPattern` resolution fix added: **404/412 (98.1%)**, pinned by `rescuedFloor = 404`.
- Live NAS verification ran in a throwaway container (`vido-verify`, own DB dir, own port 8099, media mounted read-only) so the production instance was never at risk; container and temp files removed afterwards.
- Gates: `pnpm nx test api` green · `go vet ./internal/...` clean · `gofmt -l` clean on every touched file · `pnpm run format:check` green.

### Completion Notes List

- **Live NAS result (AC #10)** — same disk, same libraries, rescan with the fixed binary:

  | metric | before | after |
  |---|---|---|
  | episodes | 2056 | **2406** (+350) |
  | disk-vs-DB gap | 411 | **62** |
  | `episode_number = 0` collapse rows | 20 | **3** |
  | seasons | 130 | 133 |
  | worst-hit series A | 1 row | 51 |
  | worst-hit series B | 1 row | 70 |
  | worst-hit series C | collapsed | 93 |
  | movies | 55 | 55 (unchanged — the movie path was symmetric but this library has no tagged movies) |

- **Residual 62, characterised (not hand-waved):** mostly legitimate duplicates — a second release of an episode the DB already holds (`an alternate release` alongside the existing Season01 rows; `[Grp] Some Donghua - 155` where that series already has that episode). One episode row per episode is the intended identity; those files are not lost data, they are alternate copies. The rest are bare `S01E01.mkv` names whose title exists only in the directory — the documented Root-cause-2 residual, out of scope by design.
- **Intentional test-expectation changes (6 fixtures, 3 files)** — every one asserted that a bracket-prefixed name must fail, which is the defect:
  - `tv_parser_test.go` — `TestTVParser_CannotParse` fixture changed to a tagged name with *no* episode marker; `TestTVParser_CanParse` split into with-/without-episode-number cases (one now `true`).
  - `parser_service_test.go` (5 sites) + `parser_handler_test.go` (2 sites) — these test the **AI-fallback path**, not the parser; they merely borrowed `[Grp] Some Anime - 26…` as a convenient "unparseable" fixture. Swapped for `[Grp] Some Anime (BD 1920x1080).mkv` and `[TestGroup] Unknown Anime.mkv` / `[Group] Anime.mkv`, which are genuinely unparseable, so each test keeps testing what it means to test.
- **Dead code removed:** `animePattern` had exactly four call sites, all four of which this story replaces; it is gone rather than left as a decorative regex.
- **Pre-existing gofmt drift:** `internal/parser/patterns.go` was already gofmt-dirty before this change (one stray blank line) — fixed in passing since the file was being edited anyway. `internal/services/parser_service_test.go` is also dirty on lines this story never touches; deliberately NOT reformatted (diff noise), and already tracked by `backlog-repo-wide-gofmt-drift` / `backlog-go-gofmt-not-enforced`.
- 🔗 AC Drift: **FOUND** — this story's own AC #2, amended above. No prior story's AC is contradicted: `bugfix-b` (scanner library routing) stays correct; this fixes the parser gap it exposed.
- 📎 Contract Stamps: NONE (no `[@contract-v*]` in this story or its upstream refs — no wire contract defined or consumed; parser-internal change).
- 🎭 A11y Pre-Flight: N/A (100% backend — no `apps/web/` files touched).
- 🎨 UX Verification: SKIPPED — no UI changes.

### Discovery Triage

- **YES — one discovery, lane ③ (already-filed, bidirectional):** attempting the isolated NAS verification reproduced `bugfix-i-unraid-deployment-hardening` first-hand — the container refused to start with `VIDO_DATA_DIR: directory '/vido-data' is not writable: permission denied` because the image runs as uid 1000 while a root-created appdata dir is `root:root`. Exactly the PUID/PGID defect that entry describes, now with a reproducible one-line trigger. No new entry needed; the existing `bugfix-i-unraid-deployment-hardening` is annotated with this reproduction.
- No other out-of-scope work discovered.

### File List

- apps/api/internal/parser/patterns.go
- apps/api/internal/parser/tv_parser.go
- apps/api/internal/parser/movie_parser.go
- apps/api/internal/parser/bracket_prefix_test.go (new)
- apps/api/internal/parser/testdata/nas-tv-filenames-2026-08.txt (new — 412-name regression corpus)
- apps/api/internal/parser/tv_parser_test.go
- apps/api/internal/services/parser_service_test.go
- apps/api/internal/handlers/parser_handler_test.go
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/bugfix-scanner-bracket-prefix-filenames-dropped.md

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | Task 1: committed the 412 real NAS filenames as a regression corpus; RED test pinned at `rescuedFloor = 404`. |
| 2026-08-24 | Task 2: `StripLeadingTags` + `leadingTagPattern`; TV and Movie `Parse`/`CanParse` now judge the stripped remainder instead of giving up. `animePattern` deleted (all four call sites removed). |
| 2026-08-24 | Task 3: `tvShowPattern` NxNN alternative fenced by separators so `1920x1080` / `1280x720` stop reading as episode markers; this alone recovered the anime-dash releases, lifting corpus coverage 94.7% → 98.1%. |
| 2026-08-24 | Task 4: gates green; live NAS rescan in an isolated container — episodes 2056 → 2406, disk-vs-DB gap 411 → 62, collapse rows 20 → 3. AC #2 amended (6 test fixtures legitimately flipped). |
