---
name: Three-Gate Verification Before User Review
description: Dev must pass Bob (SM) and Sally (UX) verification before requesting user review — never skip internal gates
type: feedback
---

After Dev (Amelia) completes UI work, the verification flow is strictly: Amelia → Bob (SM confirms AC met) → Sally (UX compares against design screenshots) → User (Alexyu) final review.

**Why:** In Story 5-0, Amelia repeatedly asked Alexyu to review the UI directly, skipping Bob and Sally verification. This wasted user time reviewing broken output that internal review would have caught.

**How to apply:**
1. Amelia completes work → self-verify CSS rules are applied in DevTools
2. Bob reviews against acceptance criteria
3. Sally compares against design screenshots in `_bmad-output/screenshots/`
4. Only after both approve → present to user for final confirmation
5. NEVER skip gates 2 and 3, even under time pressure
