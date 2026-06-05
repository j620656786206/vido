---
name: CSS Changes Must Be Verified Before Iterating
description: After modifying CSS/Tailwind classes, always verify in DevTools that the CSS rules are actually being generated and applied before making further changes
type: feedback
---

After any CSS or Tailwind class modification, the FIRST step is to verify in browser DevTools that the CSS rules actually appear in the Styles panel for the target element. If rules are missing, investigate the build pipeline (Tailwind config, PostCSS, version mismatch) instead of blindly changing class names.

**Why:** In Story 5-0, Amelia made 10+ commits changing Tailwind classes that never took effect because `styles.css` used Tailwind v3 syntax (`@tailwind base/components/utilities`) while `@tailwindcss/postcss` v4 plugin was installed. Hours of iteration were wasted changing classes when the root issue was CSS rules not being generated at all.

**How to apply:**
1. Change CSS/class → Check DevTools Styles panel → Rules missing? → Investigate build pipeline first
2. Never iterate on class names if the previous change had no visible effect — that's a signal the rules aren't being generated
3. When Tailwind classes don't work, check: version match (v3 vs v4 syntax), content paths, PostCSS config, and `@import` vs `@tailwind` directives
