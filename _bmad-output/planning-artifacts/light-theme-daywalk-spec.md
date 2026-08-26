The regex works and isolates the blocks cleanly. Writing the brief.

---

# 日巡 (Daywalk) — Light Theme, Final Spec

**Design-system architect · 2026-08-26 · Status: ready for story, pending 5 Alexyu decisions (§9)**

---

---

## ⚖️ ALEXYU RULINGS — 2026-08-26 (these override the recommendations below)

| # | Question | Ruling |
|---|---|---|
| **A1** | gold's hex vs the focus-ring alias | **(a) darken gold to `#886208` 泥金.** Focus ring stays aliased to the accent. Accepted cost: light gold reads as deep antique gold, not gold leaf. |
| **A2** | 靛青 in one theme or both | **BOTH.** `--info` / `--info-text` change in DARK too — the bright cyan is the last signal-blue-era leftover and has no wuxia justification. Costs a dark `--info-text` re-measure + a visual sweep of every 想要 pill, done in the light PR. |
| **A3** | follow `prefers-color-scheme` on first load | **YES — follow the OS.** Overrides the spec's recommendation. This REQUIRES a `matchMedia` stub in `test-setup.ts` (absent today; without it a large share of the jsdom suite crashes), a media-query change listener, and the inline boot script must read the media query. The stored user choice still wins over the OS. |
| **A4** | card-edge strength | not asked; spec's recommendation stands — **keep `#cdbe9b`**, matching 夜行 on both ΔL\* and ratio. Revisit only if the paper edge proves too faint on a real screen. |
| **A5** | when the 266 sites get fixed | **BEFORE, as its own PR.** They are a live dark-theme defect on their own merits; keeping them out makes the light PR reviewable. |

---

## 0. What I actually did

I read all three palettes and the pass-1 synthesis, then **rebuilt the gate's arithmetic from scratch** — `channel`/`luminance`/`flatten`/`ratio` transcribed from `apps/web/src/styles-contrast.spec.ts:23-57`, including JS `Math.round`'s round-half-up and `Number(r.toFixed(2))`'s decimal rounding — and ran **all three palettes plus the incumbent dark theme** through one 60-case matrix. Scratch harness: `/private/tmp/claude-502/-Users-alexyu-projects-personal-vido/595fa519-155e-414d-938f-5b2aa5c5fe8e/scratchpad/verify.py`. No repo file was modified.

**Nobody lied.** I spot-checked 29 individual claims across all three proposers and both audit documents — every single one reproduces to the second decimal, including the audit's `#ab8a40 → 3.02`, `#876d32 → 4.56`, `--error on #f3efe3 → 4.73`, and each palette's self-reported tightest pairs. The pass-1 synthesis's verification of palette 1 is sound and I reproduce it. **The comparison it could not do is below, and it changes two of its conclusions.**

---

## 1. Verdict: 日巡 (palette 1), with two grafts from palette 2

| | **P1 日巡** | **P2 晝行** | **P3 晝行** |
|---|---|---|---|
| Failures / 60 cases | **0** | 1 | 1 |
| Minimum AA case | **5.02** | 4.27 | 4.25 |
| AA cases under 5.00:1 | **0 of 54** | 2 of 54 | **8 of 54** |
| Focus ring vs 3 grounds | 5.12 / 4.50 / 3.96 | 5.55 / 4.87 / **4.67** | 5.26 / 4.60 / 3.96 |
| `--accent-hover` as text (6 live sites) | 6.65 / 5.84 / 5.14 | 7.74 / 6.80 / 5.86 | 6.26 / 5.48 / **4.72** |
| Ground ladder ΔL\* (夜行: 6.31 / 5.76) | 5.04 / 4.75 | 5.04 / 5.55 | 5.17 / 5.63 |

All three are competent and all three independently reached the same three conclusions (gold's hex cannot survive, cinnabar `#c0392b` is the one true invariant, `--text-on-accent` must flip to paper). They are separated by **engineering margin**, not by taste.

**P3 is disqualified by margin.** Its six semantic `*-text` tokens were each solved backwards from the worst surface targeting ≈4.9:1, so *every one of them* sits within 0.43 of the cliff: 4.75, 4.88, 4.89, 4.89, 4.91, 4.93. Eight of its 54 AA cases are under 5.00. That is a palette with no room for the next tint-alpha nudge — and its `--error-tint` at `0x1d` has only **17 steps** of alpha headroom before AA breaks, the least of any candidate including dark. Its `--accent-hover` as text lands at 4.72 on `--bg-tertiary`, which is a live pairing (`ActivityHub.tsx:183`, `:246`, `SidebarGroupParent.tsx:46`, `GlossaryRowV2.tsx:110`, `:133`, `BatchSubtitleDialog.tsx:236`). P3's stronger `--border-subtle` is its one genuinely better idea — carried forward as decision **A4**.

**P2 is the honourable runner-up** and beats P1 on two axes: its focus ring clears 4.67 on `--bg-tertiary` (P1: 3.96) and it holds every tint alpha byte-identical to dark, which is an elegant invariant. But it fails `--text-muted` on `--accent-subtle` over `--bg-tertiary` at 4.27, and its ground ladder ΔL\* 5.04/5.55 is less even than it claims. Two of its individual decisions are better than P1's and I grafted both.

**P1 wins because it is the only palette with zero AA cases under 5.00:1** in a matrix four times larger than the one the gate runs today — in a codebase whose own spec docstring records being bitten by late-discovered sub-AA values three times.

### Corrections I had to make

| # | Claim | Reality |
|---|---|---|
| 1 | **pass-1 §5 family 3** — add `--accent-subtle` composited vs `--accent-text` *and* `--text-muted`, × 3 surfaces × 2 themes, "light measures 5.31 / 5.02" | **This family fails DARK.** `--text-muted` on `--accent-subtle` over `--bg-tertiary` = **3.92** (and 4.57 on `--bg-secondary`). Adding it as written turns the light PR red on the dark theme. The pairing is also **not live**: `SidebarNavItem.tsx:78` overrides to `--accent-hover` when active. Family must be scoped — see §5. |
| 2 | **pass-1 §5 family 4** — add `--text-on-accent` vs all 9 solids × 2 themes | **This family fails DARK on two solids:** `--error` = **3.33**, `--error-pressed` = **2.63**. One of them is live: `GlossaryRowV2.tsx:185` paints `text-[var(--text-on-accent)]` on `bg-[var(--error)]` — a **live dark-theme AA failure today**. Family must be scoped and the violator fixed — see §5 and must-fix #3c. |
| 3 | **P1's `--error-pressed: #952a1f`** | Wrong direction-rule application. Measured: `--accent-hover` is the **only** state token that lightens in dark (L\* +9.8). `--accent-pressed` (−11.1), `--error-pressed` (−6.4), `--warning-pressed` (−0.6) already darken. P1 darkened cinnabar's press a **second** time, splitting the family across themes for no gain. **Grafted P2/P3's `#9c3a2b`** — see §2. |
| 4 | **audit: 227 white/black utilities; 260 base-semantic-as-text across 91 files** | **230** and **266 across 98 files.** pass-1's recount is correct; I reproduce it exactly (198 `text-white`, 13 `bg-black/60`, …; 85 `--error`, 59 `--accent-primary`, 55 `--success`, 50 `--warning`, 9 `--info`, 8 `--accent-hover`). |
| 5 | **P2: "`--accent-pressed` wired to `:hover` in 35 of its 36 call sites"** | **34 hover, 2 active** (36 total). Close enough to be the right finding, and it is load-bearing — see must-fix #6. |
| 6 | **pass-1 §2.1: the bright-gold window is "5× wider than claimed"** | True but immaterial, and I confirm its own correction: requiring 3:1 on **all three** grounds returns the **empty set** across the full lightness sweep of `#c9a24b`'s hue. `--bg-tertiary` is the binding constraint. Decision **A1** stands. |

**Zero hex corrections were needed in palette 1 itself.** All 55 of its measurements reproduce.

---

## 2. The grafts

Both are from **palette 2**, both are measurement-justified, neither changes a token name or role.

**Graft 1 — `--error-pressed: #9c3a2b`** (P2/P3's value, byte-identical to 夜行), replacing P1's `#952a1f`.
*Why:* P1's stated rule is "on paper, pressure drives pigment deeper." Measured against dark, `--error-pressed` **already** darkens (L\* 38.3 vs `--error`'s 44.7; 1.26:1 between them). P1 applied the inversion to a token that never needed inverting, producing a 1.47:1 press delta where dark has 1.26:1. Keeping `#9c3a2b` preserves the *exact* dark press-delta, makes cinnabar's story in the brief a single sentence — **硃砂 does not invert, in either direction, ever** — and reduces the light theme's novel-value count by one. Cost: nothing. It measures 6.36 / 5.59 / 4.93 against the three grounds and carries the paper label at 6.59:1. Re-audited: the palette still returns **0 failures / 60**, min AA still 5.02.

**Graft 2 — P2's `--accent-pressed` finding, as a must-fix.** P2 was the only proposer to check *how the state tokens are actually wired*. Verified: `--accent-pressed` appears at **34 `hover:` sites and 2 `active:` sites**. So in light, the token named "pressed" is what 34 components use for hover, and it is the *darkest* rung of the ladder. That is not a value change — it is why must-fix #6 exists and why the CSS block carries a direction warning comment.

**Not grafted, deliberately:** P3's `--border-subtle` (better ratio, wrong hue family — offered as decision **A4** with a hue-corrected hex); P2's unchanged tint alphas (P1's argument that 12% of a pigment over paper is invisible is correct, and its raised alphas still leave 20–121 steps of headroom); P3's 65% scrim (P1's 70% gives `--text-inverse` 6.44:1 over a white poster vs P3's 5.35:1).

---

## 3. The token block

Drop in **after** the `:root` block's closing brace at `apps/web/src/styles.css:107`, before `@keyframes fadeIn`. Note the reserved placeholder at `styles.css:105-106` is *inside* `:root` — delete those two comment lines and place the real block outside.

```css
/* =============================================================================
 * 日巡 (Daywalk) — the light theme, 2026-08-26.
 * The same wuxia world at noon: three weights of 宣紙 ground, 松煙墨 pine-soot
 * ink for text, GOLD unchanged in ROLE (「你在這裡」/the action), 硃砂 cinnabar
 * still faults-and-destruction only. Dark remains the default; this is opt-in.
 *
 * TWO RULES INVERT AND THE TOKEN NAMES DO NOT SAY SO — read before editing:
 *   1. --bg-tertiary is now the DARKEST ground, so it is the worst case for
 *      every text token and it binds the whole palette.
 *   2. --accent-hover goes DARKER, not lighter. It is the ONLY state token
 *      whose sense inverts: --accent-pressed / --error-pressed /
 *      --warning-pressed already darken in 夜行 and keep that direction here.
 *      Any component that lightens on its own (filter: brightness(),
 *      hover:brightness-110, color-mix toward white) moves the WRONG WAY.
 *
 * Every value below is gated by styles-contrast.spec.ts, which parses this
 * block SEPARATELY from :root. Do not add a token here without a light
 * counterpart case in that spec — the parity assertion will fail you.
 * ------------------------------------------------------------------------- */
[data-theme='light'] {
  /* Background — 宣紙: raw sheet → second sheet → pressed sheet.
   * ΔL* 5.04 / 4.75 against 夜行's own 6.31 / 5.76, so cards and hover rows
   * stay as findable in light as in dark. Never #ffffff — a clinical white
   * breaks the world instantly. */
  --bg-primary: #faf6ea;
  --bg-secondary: #f0e7d3;
  --bg-tertiary: #e5d9c3;
  /* The deckle edge. Matched to 夜行 by ΔL* (19.51 vs 18.99) AND by ratio
   * (1.70 vs 1.66) — both axes agree, which is why this value is defensible.
   * Ungated in BOTH themes; decorative, not informational. See §8.1 / A4. */
  --border-subtle: #cdbe9b;

  /* Accent / Brand — 泥金, gold ground in glue, which dries deep.
   * #c9a24b is 2.22:1 on this paper. An exhaustive sweep of its hue returns
   * the EMPTY SET for a value clearing 3:1 on all three grounds while
   * carrying an ink label — --bg-tertiary is the binding constraint (A1).
   * The state ladder DARKENS and the label contrast is monotone increasing
   * 5.30 → 6.88 → 9.14, so no interaction state can break AA. */
  --accent-primary: #886208;
  --accent-hover: #725205;
  --accent-pressed: #5b4103;
  /* UNCHANGED from 夜行. Cinnabar's press already darkens there (1.26:1);
   * inverting it a second time would split the family across themes. */
  --error-pressed: #9c3a2b;

  /* Semantic solids. INVARIANT: every solid stays at Y ≤ 0.1737 so
   * --text-on-accent clears 4.5:1 on it. Lightening any of these by eye
   * silently drops every button label on that fill below AA. */
  --success: #0b7352; /* 青碧 jade, deepened */
  --error: #c0392b; /* 硃砂 — UNCHANGED, the only colour that survives inversion */
  --warning: #a6510c; /* 赭 burnt ochre (#d4763f is 2.97:1 here — fails the 3:1 fill floor) */
  --info: #0b657d; /* 靛青 — see A2; the one token that reads as a different hue */
  --warning-pressed: #8b4208;

  /* Text — 松煙墨 pine-soot ink with a green cast, the inversion of 夜行's
   * ink-green grounds. Deliberately not a neutral black. */
  --text-primary: #16231d; /* 15.04 / 13.21 / 11.64 */
  --text-secondary: #32493e; /* 9.01 / 7.91 / 6.97 */
  --text-muted: #41554c; /* 7.39 / 6.50 / 5.72 */
  /* Flips to paper, preserving 夜行's property that this EQUALS --bg-primary.
   * TRAP: a call site that used this to mean "the dark ground" inverts wrongly
   * and silently. Audit every usage — must-fix #4. */
  --text-inverse: #faf6ea;

  /* Semantic tints — alphas RISE (0x1f → 0x33; --accent-subtle 0x26 → 0x40).
   * 12% of a pigment over paper is nearly invisible; the alpha must rise for
   * the chip to exist at all. These are CONTRAST INPUTS, not decoration.
   * The PIGMENT stays 夜行's bright value on purpose: a thin wash of the dark
   * gold goes muddy khaki AND drops --accent-text to exactly 4.50:1, one
   * rounding step from failing (verified). --error-tint is the tightest in
   * the palette: AA breaks past 0x47 (20 steps of headroom). */
  --accent-subtle: #c9a24b40; /* ~25% — active nav wash, selected-row tint */
  --accent-tint: #c9a24b33; /* ~20% — accent badge / chip background */
  --accent-text: #654804; /* 7.83 / 6.88 / 6.06; worst 5.46 on own tint over --bg-tertiary */
  --success-tint: #6fbfa833;
  --success-text: #06583e; /* worst 5.56 */
  --error-tint: #c0392b33;
  --error-text: #87251b; /* worst 5.02 — the binding pair of the whole palette */
  --warning-tint: #d4763f33;
  --warning-text: #7b3a06; /* worst 5.22 */
  --info-tint: #06b6d433;
  --info-text: #075469; /* worst 5.27 */

  /* INVERTS from ink to paper — the direct consequence of darkening the gold.
   * Clears 4.5:1 on every solid it legitimately lands on (worst 5.21 on
   * cinnabar). Also retires 夜行's #14161a, a cool near-black left over from
   * the signal-blue era. */
  --text-on-accent: #fdfaf2;
  /* The one documented sub-AA exemption — intentionally sub-AA (TC-1).
   * Deliberately tuned to 3.40:1 on --bg-primary, matching 夜行's 3.44:1.
   * A naive inversion of #5e6e66 measures 4.69:1 on paper, which would PASS
   * AA and quietly break the exemption assertion. Do not "fix" this value. */
  --text-disabled: #7a8980;
  /* DOES NOT INVERT. The scrim covers poster ARTWORK, not the theme, and the
   * artwork is identical in both themes. A paper-coloured scrim bleaches key
   * art to pastel and leaves a paper modal with no boundary. Keeps 70%,
   * re-tinted from neutral #000 to 夜行's own ink-green. Carries must-fix #3. */
  --overlay-scrim: #0c1512b3;
  --focus-ring: #886208; /* = --accent-primary. 5.12 / 4.50 / 3.96 — clears WCAG 1.4.11 on all three */

  /* Shadows are INK, not soot: rgba of --text-primary at low alpha, because
   * black at 30–60% on warm paper reads as a bruise. Two layers from md up —
   * on paper, elevation is a crisp near-edge plus a soft cast; a single blur
   * at low alpha simply disappears. Total ink ≤15.4% where 夜行 used 60%. */
  --shadow-sm: 0 1px 2px rgba(22, 35, 29, 0.06);
  --shadow-md:
    0 1px 2px rgba(22, 35, 29, 0.05), 0 4px 8px rgba(22, 35, 29, 0.06);
  --shadow-lg:
    0 2px 4px rgba(22, 35, 29, 0.06), 0 8px 16px rgba(22, 35, 29, 0.07);
  --shadow-xl:
    0 4px 8px rgba(22, 35, 29, 0.07), 0 16px 32px rgba(22, 35, 29, 0.09);
}
```

**36 theme-dependent tokens, all present.** The 4 radius and 6 gap tokens correctly do not appear — they are theme-independent and inherit from `:root`.

Also update the stale comment at `styles.css:120` (`/* Global dark theme baseline */`) — the rule paints from tokens and is theme-agnostic. Comment only, no CSS change.

### Verified measurements

| Token | on `--bg-primary` | `--bg-secondary` | `--bg-tertiary` | on own tint over the three grounds |
|---|---|---|---|---|
| `--text-primary` | 15.04 | 13.21 | 11.64 | — |
| `--text-secondary` | 9.01 | 7.91 | 6.97 | — |
| `--text-muted` | 7.39 | 6.50 | 5.72 | — |
| `--accent-text` | 7.83 | 6.88 | 6.06 | 6.75 / 6.06 / **5.46** |
| `--error-text` | 8.41 | 7.39 | 6.51 | 6.26 / 5.58 / **5.02** |
| `--warning-text` | 7.93 | 6.97 | 6.14 | 6.47 / 5.79 / **5.22** |
| `--success-text` | 7.86 | 6.90 | 6.08 | 6.89 / 6.17 / **5.56** |
| `--info-text` | 7.83 | 6.88 | 6.06 | 6.56 / 5.86 / **5.27** |
| `--accent-hover` (as text, 6 sites) | 6.65 | 5.84 | 5.14 | — |
| `--text-disabled` (must stay <4.5) | **3.40** | 2.99 | 2.63 | — |
| `--accent-primary` = `--focus-ring` (3:1) | 5.12 | 4.50 | **3.96** | — |

`--text-on-accent` on the solids: `accent-primary` 5.30 · `accent-hover` 6.88 · `accent-pressed` 9.14 · `error-pressed` 6.59 · `warning-pressed` 6.99 · `success` 5.61 · `error` 5.21 · `warning` 5.29 · `info` 6.35.

**Tint-alpha headroom before AA breaks** (the danger zone for a future editor): `error-tint` `0x33`→**`0x47`** (20 steps, the tightest) · `warning-tint` →`0x61` · `info-tint` →`0x72` · `accent-subtle` →`0x92` · `accent-tint` →`0x92` · `success-tint` →`0xac`.

---

## 4. The gate is broken today — fix it first

`styles-contrast.spec.ts:18` and `:48` call `CSS.match(...)` **without `/g`**, which returns the first occurrence in the whole file. Append a `[data-theme="light"]` block below `:root` and all 30 assertions keep measuring the **dark** values. The pass-1 synthesis proved this empirically — it appended a 1.10:1 light block and the suite reported **30/30 green** — and restored the file. I accept that evidence and did not re-run the destructive test.

I did validate the **replacement** parser against the real `styles.css` with a simulated light block appended: `dark --bg-primary` → `#0c1512`, `light --bg-primary` → `#faf6ea`, neither block leaking into the other, `token8` resolving `--error-tint` to `#c0392b1f` and `#c0392b33` respectively. It works.

---

## 5. Contrast-gate restructuring

Replace the parsing head of `apps/web/src/styles-contrast.spec.ts`. Lines 23–57 (`channel`/`luminance`/`flatten`/`ratio`) stay untouched — do not "improve" them; the whole point is that the spec's arithmetic is the definition.

```ts
const CSS = readFileSync(join(__dirname, 'styles.css'), 'utf8');

/**
 * Split styles.css into per-theme blocks BEFORE matching any token.
 *
 * The bug this replaces: `CSS.match(/--x:\s*(#[0-9a-f]{6})/)` has no /g and
 * returns the FIRST occurrence in the whole FILE, so every assertion measured
 * the :root (dark) values regardless of theme. Verified 2026-08-26: appending
 * a light block with #eae4d6 text on #faf6ea ground (1.10:1 — literally
 * invisible) left this suite 30/30 GREEN. That would have been the fourth
 * instance of the defect class the docstring above warns about.
 */
function themeBlock(label: string, head: RegExp): string {
  const m = CSS.match(head);
  if (!m) throw new Error(`theme block "${label}" not found in styles.css`);
  return m[1];
}

const THEMES = {
  dark: themeBlock('dark (:root)', /^:root\s*\{([\s\S]*?)^\}/m),
  light: themeBlock('light', /^\[data-theme=['"]light['"]\]\s*\{([\s\S]*?)^\}/m),
} as const;
type Theme = keyof typeof THEMES;
const ALL_THEMES = Object.keys(THEMES) as Theme[];

/**
 * Literal 6-digit hex ONLY, per theme. The literal requirement is a FEATURE:
 * color-mix(), oklch(), or a var() alias would make this throw rather than
 * silently measure nothing. Do not "modernise" it.
 */
function token(theme: Theme, name: string): string {
  const m = THEMES[theme].match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})\\b`));
  if (!m) throw new Error(`token --${name} not found in ${theme} theme`);
  return m[1];
}

function token8(theme: Theme, name: string): string {
  const m = THEMES[theme].match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{8})\\b`));
  if (!m) throw new Error(`8-digit token --${name} not found in ${theme} theme`);
  return m[1];
}
```

Then thread `theme` through every case. **30 cases → ~125.**

| # | Family | Cases | Rationale |
|---|---|---|---|
| 1 | `BODY_TEXT_TOKENS` × `SURFACES` × themes | 8×3×2 = **48** | the original gate, now actually measuring both themes |
| 2 | `TINT_PAIRS` × **all three surfaces** × themes | 5×3×2 = **30** | today only `--bg-tertiary`. It is the worst ground in both themes (lightest in dark, darkest in light) — assert all three rather than rely on that coincidence holding |
| 3 | **NEW** — `--accent-subtle` composited over 3 grounds, vs `--accent-text` and `--text-primary`, × themes | 2×3×2 = **12** | `--accent-subtle` is ungated today and is the PR #287 surface. **Scoped to these two foregrounds only** — see the exclusion note below |
| 3b | **NEW** — `--accent-hover` on `--accent-subtle`, 3:1 **non-text** floor, × themes | 3×2 = **6** | `SidebarNavItem.tsx:78` and `:113` render an *icon* in `--accent-hover` on the active wash. Light: 5.53 / 5.00 / 4.51. Dark: 8.20 / 7.04 / 6.04 |
| 3c | **NEW** — `--accent-hover` as **text** on 3 grounds × themes | 3×2 = **6** | 6 live text sites (`ActivityHub.tsx:183`, `:246`, `SidebarGroupParent.tsx:46`, `GlossaryRowV2.tsx:110`, `:133`, `BatchSubtitleDialog.tsx:236`). Nothing gates this today in **either** theme |
| 4 | **NEW** — `--text-on-accent` vs 7 solids × themes | 7×2 = **14** | the ungated Y ≤ 0.1737 invariant. **`--error` and `--error-pressed` are EXCLUDED** — see below |
| 5 | **NEW** — 3:1 non-text floor: `--accent-primary`, `--focus-ring` vs 3 grounds × themes | 2×3×2 = **12** | WCAG 1.4.11. `--focus-ring` is applied app-wide from one rule at `styles.css:151-154`. This is the assertion that makes decision A1 *enforceable* rather than a comment |
| 6 | `--text-disabled` exemption, **per theme** | **2** | today it asserts `< 4.5` on dark values only. Light needs its own — it measures 3.40 and must **stay** below 4.5 |
| 7 | **NEW** — token-set parity | **1** | assert both blocks declare the same theme-dependent token names. This is what catches a token added to `:root` and forgotten in light, which would otherwise silently inherit dark |

### Three exclusions, each with a comment in the spec

These are the corrections to the pass-1 synthesis's gate design. Adding them as it proposed would turn the light PR red **on the dark theme**.

- **`--text-muted` on `--accent-subtle` (family 3).** Dark measures **3.92 / 4.57 / 5.32** — sub-AA on `--bg-tertiary`. The pairing is **not live**: `SidebarNavItem.tsx:78` sets `text-[var(--text-muted)]` as the *base* state and overrides to `--accent-hover` under `data-[status=active]`, which is the only state that paints `--accent-subtle`. Gating it would fail dark for a combination the app never renders. Exclude with a comment naming this line.
- **`--text-on-accent` on `--error` and `--error-pressed` (family 4).** Dark measures **3.33** and **2.63**. `--error-pressed` is not live (its 8 sites use `text-white` — see must-fix #5). `--error` **is** live at `GlossaryRowV2.tsx:185` and is therefore a real dark-theme defect **today** — must-fix #3c. Exclude from the gate now, re-include in the same PR that fixes that line.
- **`--border-subtle`.** 1.66:1 in dark, 1.70:1 in light. Gating it at 3:1 fails both themes on day one. Pre-existing product decision, not something light introduces — §8.1 and decision **A4**.

---

## 6. Switch wiring

### 6.1 Where `data-theme` goes

**`document.documentElement` (`<html>`), and nowhere lower.** `styles.css:8` defines every token on bare `:root`, and `@layer base` (`styles.css:121-127`) paints `html, body, #root` from those tokens. Attaching to the `__root.tsx` wrapper `<div>` leaves the `html`/`body` background rule on dark values. `data-theme` is net-new: there is currently **zero** `documentElement` / `body.classList` / `setAttribute('data-…')` usage anywhere in `apps/web/src`.

### 6.2 Persistence — the precedent is `useDownloadsView.ts`

**Named precedent: `apps/web/src/hooks/useDownloadsView.ts`** — 32 lines, the whole file, and the cleanest instance of the house pattern: module-level `STORAGE_KEY`, a `readStored()` that try/catches and falls back to a hardcoded default, `useState(readStored)` as a *synchronous* initialiser with no effect, a `useCallback` setter that writes through inside try/catch, returns a tuple. I read it to confirm.

**Secondary precedent, and the more directly analogous one: `AppShellV2.tsx:18-45`** (`vido:sidebar:collapsed`) — the same shape written inline, and the *only* place the shell already reads a per-user display preference at boot. That is the precedent being extended.

Copy `useDownloadsView.ts` verbatim into `apps/web/src/hooks/useTheme.ts`:

- **Key: `vido:theme`** — the current `vido:ns:key` convention. (`vido-library-preferences` and `vido-recent-searches` are the legacy `vido-kebab` era; do not use it.)
- **Values: `'dark' | 'light'`.** Absent or garbage → `'dark'`.
- The setter writes `localStorage` **and** stamps/removes the attribute on `document.documentElement`. **Stamp `data-theme="light"` for light; *remove* the attribute for dark** rather than writing `data-theme="dark"` — so `:root` stays the single source of the default and the two can never disagree.
- **Do not use the server settings store.** `apps/api/internal/handlers/settings_handler.go:155-163` would need zero backend work, but no `apps/web/src/services/*` consumes it, it puts the theme behind a network round-trip (the flash problem in its worst form), and `main.go:946-954` makes a new sub-route there a live route-ordering hazard (`/settings/:key` shadowing the specific handlers).

### 6.3 Default behaviour and `prefers-color-scheme`

**v1: two-state toggle, dark default, `prefers-color-scheme` NOT consulted.** Recommended, pending decision **A3**.

- **Product:** the constraint is *"DARK STAYS THE DEFAULT."* Honouring `prefers-color-scheme` makes the default *conditional* — a different product statement, and Alexyu's call.
- **Mechanical:** I verified `apps/web/src/test-setup.ts` stubs `IntersectionObserver` (line 6) and `localStorage` (line 42) — **not `matchMedia`**. That absence is exactly why `DownloadsBrowseV2.tsx:59-64` guards `typeof window.matchMedia !== 'function'` and `ScanProgress.spec.tsx:59-73` mocks it per file. A theme hook calling `matchMedia` at module scope or on mount would crash a large share of the jsdom suite. System-follow costs a `test-setup.ts` stub **plus** a change listener — small, but it is scope, and it should be bought deliberately rather than assumed.

### 6.4 Flash of wrong theme

Real, and **one-directional**. The CSS `<link>` in `apps/web/index.html` is render-blocking, so tokens exist at first paint — but they are the *dark* values, since dark is `:root`. A light-preferring user gets a full-viewport ink-green paint until the module graph loads and React mounts, then a snap to 宣紙白 — easily hundreds of ms on a cold NAS load over LAN. Dark users see nothing.

Add to `apps/web/index.html`, **inside `<head>`, after the stylesheet link and before the module script**:

```html
<script>
  // Stamp the theme before first paint. Without this, a light-preferring user
  // gets a full-viewport dark flash until React mounts, because :root is dark.
  // Dark writes NO attribute — :root is the default and stays its only source.
  try {
    if (localStorage.getItem('vido:theme') === 'light') {
      document.documentElement.setAttribute('data-theme', 'light');
    }
  } catch (e) {
    /* private mode / storage disabled — fall through to the dark default */
  }
</script>
```

Safe against staleness: `apps/web/nginx.conf:101` already disables caching for `index.html`, so this cannot go stale against a new hashed build. Plain inline JS, so Vite leaves it untouched in dev and in the production output.

### 6.5 Where the control lives

**Settings — `/settings`, in a new 外觀 (Appearance) section.** Not System.

Per `_bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md` (lines verified):

- **line 340** — *"**D4-3 Settings split: SETTINGS + SYSTEM (option 甲, \*arr model).** Settings = preferences (連線/掃描/首頁/qBT). System = ops dashboards."*
- **line 373** — *"7. **設定 Settings** — preferences"*
- **line 410** — route table: `| Settings | /settings | preferences only |`
- **line 572** — `| settings.tsx | ✏️ preferences only (連線/掃描/首頁/qBT) | D4-3 |`

A theme choice is a user preference, not an ops dashboard. Files: `apps/web/src/routes/settings.tsx` and a new child under `apps/web/src/routes/settings/`.

**Doc amendment required in the same PR:** the same ADR at **line 90** lists *"dark theme only"* among the non-functional constraints driving nav IA. Shipping this amends that line. Update it so the ADR does not silently contradict the product. This is a doc edit, not a decision.

---

## 7. Must-fix list, ordered

| # | What | Where | Severity |
|---|---|---|---|
| **1** | **Fix the contrast gate first.** `String.match` without `/g` returns the first file-wide occurrence; a light block below `:root` leaves all 30 assertions measuring dark. Proven 30/30 green on a 1.10:1 theme. Restructure per §5 — including the three exclusions, or the PR turns red on dark. **Must land before or with the palette, never after.** | `styles-contrast.spec.ts:18`, `:48`; flat lists `:60`, `:67-76`; dark-only exemption `:112-115` | **Blocker** |
| **2** | **266 base semantic colours used as text, across 98 files.** In light, on `--bg-tertiary`: `--error` **3.90**, `--warning` **3.96**, `--accent-primary` **3.96**, `--success` **4.19** — all sub-AA, most also failing `--bg-secondary`. Must move to the `*-text` tokens. **This is a dark-theme bug too** — the same defect class the four token-debt PRs removed, re-created inside the token vocabulary where neither the lint rule nor the gate looks. Breakdown: 85 `--error`, 59 `--accent-primary`, 55 `--success`, 50 `--warning`, 9 `--info`, 8 `--accent-hover`. | 98 files under `apps/web/src` | **Blocker** — see A5 for sequencing |
| **3** | **8 call sites paint `--text-primary` directly on `--overlay-scrim`.** The scrim stays dark by design, so in light this is ink on ink: **2.34:1** over a white poster, **1.19:1** over a black one. All must move to `--text-inverse` (6.44 / 17.92). | `HeroBanner.tsx:161`, `:187`; `PosterCard.tsx:231`, `:237`, `:242`, `:291`; `PosterCardV2.tsx:144`; `AvailabilityBadge.tsx:22` | **Blocker** |
| **3b** | **Live dark-theme bug.** `text-[var(--text-on-accent)]` (`#14161a`) on `bg-[var(--overlay-scrim)]`: **2.11:1** over a white poster, **1.16:1** over a black one — broken in production, in dark, today. It ironically *self-heals* in light (6.67:1) because `--text-on-accent` flips to paper. Fix to `--text-inverse` in the same sweep. | `components/media/DetailHeroV2.tsx:64` | **High — pre-existing** |
| **3c** | **Live dark-theme bug, newly found.** `text-[var(--text-on-accent)]` on `bg-[var(--error)]`: **3.33:1** in dark. The other 8 error-button sites use `text-white` on the same fill and are fine (cinnabar does not change). Light measures 5.21:1. Fix, then re-include `--error` in gate family 4. | `components/subtitle/GlossaryRowV2.tsx:185` | **High — pre-existing** |
| **4** | **`--text-inverse` is a trap.** In 夜行 it is byte-identical to `--bg-primary` (1.00:1 against it), so call sites may have used it to mean "the dark ground". In light it flips to paper and inverts wrongly *and silently*. Audit every usage for intent. | `grep -rn 'text-inverse' apps/web/src` | High |
| **5** | **230 `white`/`black` utilities the eslint rule structurally cannot see.** `no-hardcoded-palette.js:37` requires a numeric hue segment (`{prefix}-{hue}-{50\|[1-9]00}`); `white`/`black` have none. 198 of the 230 are `text-white`, measuring **1.08 / 1.23 / 1.43** on the three paper grounds — invisible. In dark the same utilities are 13.96–18.56 and look fine. | `eslint-rules/no-hardcoded-palette.js:37`; scope `eslint.config.mjs:263-272` | High |
| **6** | **Hover's direction inverts for gold, and the token that carries it is misnamed.** `--accent-pressed` is wired to **34 `hover:` sites and only 2 `active:` sites** — so in light, 34 components hover to the *darkest* rung. That is correct behaviour, but it must be understood before anyone "fixes" it. Separately: grep for components that lighten on their own — `filter: brightness()`, `hover:brightness-*`, `color-mix` toward white — which move the wrong way in light while every token measurement stays green. (Current scan: 7 `group-hover:opacity-*` sites, no `brightness` utilities. Low but re-check at build time.) | `grep -rnE 'brightness-\|filter:\|color-mix' apps/web/src` | High |
| **7** | **`--warning-pressed` is a no-op press in dark.** `#d97706` vs `--warning` `#d4763f` measures **1.02:1** between them — the press state is invisible. Light fixes it incidentally (`#8b4208`, 1.4:1). Pre-existing; flag, don't necessarily fix here. | `styles.css:77` | Medium |
| **8** | **`.css` files are not linted at all** — the palette rule runs only on `.ts`/`.tsx`, so `styles.css` itself is unguarded. | `eslint.config.mjs:263-272` | Medium |
| **9** | **98 `.ts`/`.tsx` files sit outside the rule's glob** (`lib/`, `hooks/`, `utils/`, `services/`, `main.tsx`, `router.tsx`, `visual-harness/`). At least one carries classnames: `utils/libraryStatus.ts:41-46`. | `eslint.config.mjs:263-272` | Medium |
| **10** | 8 raw hex values in `style={{}}` objects and 1 `hsl()` template literal — the rule is class-shaped, not colour-shaped. | `ColorPlaceholder.tsx:33` + 8 others | Low |

---

## 8. Known, accepted gaps

### 8.1 `--border-subtle` is decorative, not informational, in both themes

Measures **1.70:1** in light and **1.66:1** in dark, against WCAG 1.4.11's 3:1 for meaningful UI boundaries. The palette matches 夜行 on **both** ΔL\* (19.51 vs 18.99) **and** ratio (1.70 vs 1.66), which is why the value is defensible rather than arbitrary. But card edges genuinely are decorative in both themes. Clearing 3:1 on paper needs roughly `#9c8c6b` — ΔL\* 38, visibly a hard rule, not a deckle edge. **Not in scope for v1**; it is a pre-existing dark-theme property and changing it alters the visual language in *both* themes. Palette 3 argued for a stronger border and deserves credit — surfaced as decision **A4** with a hue-corrected hex ready.

---

## 9. Decisions that need Alexyu

Five product calls. **A builder may not pick any of them.**

### A1 — Gold's hex changes, or the focus ring stops being gold. Pick one.

`--accent-primary: #c9a24b` is **2.22:1** on 宣紙. The *role* survives inversion; the *value* cannot. I swept the full lightness range of that exact hue and there are exactly two ways out:

- **(a) Darken the gold to `#886208` 泥金** *— what this spec assumes.* Everything stays aliased, the focus ring clears 1.4.11 on all three grounds (5.12 / 4.50 / 3.96), hover and pressed get a real monotone ladder with label contrast only ever improving. **Cost: light gold reads as deep antique gold / bronze, not gold leaf catching light. The two themes' accents will not look like the same pigment side by side.**
- **(b) Keep a brighter gold (≈`#a58131`) and un-alias `--focus-ring` to its own darker value.** **Cost: the focus ring stops being the accent — one of 夜行's cleanest ideas.** And even then the gold button's own boundary sits at **2.60:1** on `--bg-tertiary`, so raised and hover surfaces get buttons that are hard to find. Verified: requiring 3:1 on all three grounds *while* carrying an ink label returns the **empty set** — `--bg-tertiary` is the binding constraint, not the state ladder.

**Recommendation: (a).** Note the original proposer's stated reason for (a) was wrong (they blamed the state ladder); the real reason is `--bg-tertiary`. Same answer, sounder footing.

### A2 — Does 靛青 replace cyan in *both* themes, or only in light?

`--info: #06b6d4` is bright cyan at Y=0.38 — impossible on paper, so light necessarily uses `#0b657d` 靛青 (deep teal-blue). **They read as different colours.** The audit already flags cyan as the one hue with no wuxia justification in either theme.

- **Light-only:** ships now, zero extra risk, but `--info` becomes the one token where the two themes are not the same world.
- **Both:** costs a re-measure of `--info-text` in dark and a visual-regression sweep of every 想要 pill, but closes the last signal-blue-era leftover. Light is the cheap moment to do it — the sweep is happening anyway.

### A3 — Does a light-preferring OS user get light on first load?

- **No** *(recommended; what §6.3 assumes)*: dark is literally the default, light is opt-in from Settings. Zero test-infra cost.
- **Yes:** requires stubbing `matchMedia` in `test-setup.ts` (currently absent — it would otherwise crash a large share of the jsdom suite), a media-query change listener, and extending the inline boot script. Also makes the default *conditional*, which is a different product statement than "dark stays the default."

### A4 — Is a 1.70:1 card edge good enough on paper? (raised by palette 3)

Palette 3 argued the light theme should have visibly stronger borders than dark (its `#a9b4af` measures 1.98 / 1.73 / 1.49). I did not graft it because its hue is a sage grey that leaves the warm-paper family. If Alexyu wants the stronger edge, the hue-corrected value is ready:

- **Keep `#cdbe9b`** *(recommended; what this spec assumes)*: 1.70 / 1.49 / 1.32, matching 夜行 on both ΔL\* and ratio. The two themes' card edges feel identical. **Cost: on a bright screen, a card edge on paper is very faint.**
- **Use `#c2af85`**: 1.99 / 1.75 / 1.54 — palette 3's strength, palette 1's hue (42°, same saturation), ΔL\* 24.8 from the page, still well above `--text-disabled` (L\* 55.7) so it can never read as ink. Re-audited: **0 failures / 60**. **Cost: light gets more structural drawing than dark, so the two themes stop being the same design at the edges.**

Either way this does **not** reach 3:1 — that is §8.1's separate, out-of-scope argument for both themes.

### A5 — Do the 266 base-semantic-as-text sites get fixed *before* light, or *with* it?

They are a **dark-theme bug today** (`--warning` as text measures 4.31:1 on its own tint over `--bg-tertiary` — the exact reason `--warning-text` exists). Light makes them worse and far more visible (3.90–4.19:1 on `--bg-tertiary`).

- **Before, as its own PR** *(recommended)*: keeps the light-theme PR reviewable, and fixes a live accessibility defect on its own merits, in the theme that ships today.
- **With:** one PR, but ~266 edits across 98 files land on top of a global palette change, and any visual-regression failure becomes ambiguous between the two causes.

---

## 10. Explicitly out of scope for v1

- **Any change to token names or roles.** A component that needs editing to work in light is a **finding** (§7), never a licence to add a token.
- **`--border-subtle` at 3:1** (§8.1) — separate decision, both themes.
- **System / `prefers-color-scheme` following**, unless A3 says otherwise.
- **Server-side theme persistence.** Client-only, per §6.2.
- **The `-linux` visual baselines.** Per `CLAUDE.md`, they cannot be generated on this darwin machine. The CI `Visual Regression` workflow auto-opens a `chore(visual): bootstrap N missing -linux baselines` PR — **merge that; never run `test:visual:update` locally.** Expect a large batch: a theme is a global repaint.
- **Light-theme design screens in `ux-design.pen`.** Its own story. Note `CLAUDE.md`'s screenshot workflow triggers on *any* `.pen` change, and a full regen is non-deterministic — only stage genuinely-changed screens.

## 11. Things a builder must NOT invent

1. **Any hex not in §3.** Every value there is gated arithmetic, not taste. A colour that looks wrong on screen is a §9 conversation, not an edit.
2. **Tint alphas.** They are contrast inputs. `--error-tint` breaks AA past `0x47`; `--accent-subtle` past `0x92`. Changing one without re-running the gate is the defect class this repo has been bitten by three times, with a fourth pre-installed.
3. **The tint *pigments*.** They stay 夜行's bright values on purpose — swapping `--accent-subtle` to the dark gold drops `--accent-text` to **exactly 4.50:1** (verified), one rounding step from failing.
4. **Whether the scrim inverts.** It does not. See §3's comment and must-fix #3.
5. **`--text-disabled`'s value.** It is tuned to *fail* AA at 3.40:1. A naive inversion of `#5e6e66` measures **4.69:1** on paper, passes, and silently breaks the exemption assertion.
6. **The three gate exclusions in §5.** They are not oversights — each one fails the **dark** theme, and two of them are excluded because the pairing is not live. Removing an exclusion without first fixing the named file turns the PR red on a theme it did not touch.
7. **The storage key or its shape.** `vido:theme`, values `'dark' | 'light'`, dark writes no attribute, per `useDownloadsView.ts`.