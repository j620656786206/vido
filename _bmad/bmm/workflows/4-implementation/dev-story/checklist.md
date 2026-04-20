---
title: 'Enhanced Dev Story Definition of Done Checklist'
validation-target: 'Story markdown ({{story_path}})'
validation-criticality: 'HIGHEST'
required-inputs:
  - 'Story markdown file with enhanced Dev Notes containing comprehensive implementation context'
  - 'Completed Tasks/Subtasks section with all items marked [x]'
  - 'Updated File List section with all changed files'
  - 'Updated Dev Agent Record with implementation notes'
optional-inputs:
  - 'Test results output'
  - 'CI logs'
  - 'Linting reports'
validation-rules:
  - 'Only permitted story sections modified: Tasks/Subtasks checkboxes, Dev Agent Record, File List, Change Log, Status'
  - 'All implementation requirements from story Dev Notes must be satisfied'
  - 'Definition of Done checklist must pass completely'
  - 'Enhanced story context must contain sufficient technical guidance'
---

# 🎯 Enhanced Definition of Done Checklist

**Critical validation:** Story is truly ready for review only when ALL items below are satisfied

## 📋 Context & Requirements Validation

- [ ] **Story Context Completeness:** Dev Notes contains ALL necessary technical requirements, architecture patterns, and implementation guidance
- [ ] **Architecture Compliance:** Implementation follows all architectural requirements specified in Dev Notes
- [ ] **Technical Specifications:** All technical specifications (libraries, frameworks, versions) from Dev Notes are implemented correctly
- [ ] **Previous Story Learnings:** Previous story insights incorporated (if applicable) and build upon appropriately

## ✅ Implementation Completion

- [ ] **All Tasks Complete:** Every task and subtask marked complete with [x]
- [ ] **MANDATORY CHECKBOX AUDIT (Agreement 5, Epic 8 Retro):** Re-read the ENTIRE Tasks/Subtasks section of the story file from top to bottom. Count total `[ ]` (unchecked) items. If ANY unchecked items remain, either complete them or explicitly document why they are deferred. A story with unchecked tasks CANNOT be marked as done. This prevents the Story 8-8 failure where tasks were left unchecked but the story was declared complete.
- [ ] **Acceptance Criteria Satisfaction:** Implementation satisfies EVERY Acceptance Criterion in the story
- [ ] **No Ambiguous Implementation:** Clear, unambiguous implementation that meets story requirements
- [ ] **Edge Cases Handled:** Error conditions and edge cases appropriately addressed
- [ ] **Dependencies Within Scope:** Only uses dependencies specified in story or project-context.md

## 🧪 Testing & Quality Assurance

- [ ] **Unit Tests:** Unit tests added/updated for ALL core functionality introduced/changed by this story
- [ ] **Integration Tests:** Integration tests added/updated for component interactions when story requirements demand them
- [ ] **End-to-End Tests:** End-to-end tests created for critical user flows when story requirements specify them
- [ ] **Test Coverage:** Tests cover acceptance criteria and edge cases from story Dev Notes
- [ ] **Regression Prevention:** ALL existing tests pass (no regressions introduced)
- [ ] **Code Quality:** Linting and static checks pass when configured in project
- [ ] **Test Framework Compliance:** Tests use project's testing frameworks and patterns from Dev Notes

## 🔒 Security Validation (Epic 9 Retro AI-6)

- [ ] **Bounded I/O:** All file reads, HTTP response bodies, and buffer allocations have explicit size limits. No unbounded `io.ReadAll` or unlimited `[]byte` growth.
- [ ] **Context Lifecycle:** All goroutines, HTTP requests, and database queries accept and honor `context.Context`. No fire-and-forget goroutines without cancellation support.
- [ ] **Error Wrapping:** Errors are wrapped with `fmt.Errorf("%w", err)` or sentinel errors. No raw `err.Error()` string comparisons. Internal details are not leaked to API responses.

## 🎭 Frontend Performance + Accessibility Pre-Flight (Epic 10 Retro AI-1)

**When this applies:** ANY story that touches `apps/web/` React components, hooks, or routes. Skip only if the story is 100% backend (no files under `apps/web/`).

**Why this exists:** Epic 10 retro (2026-04-20) found that 4 of 4 frontend stories shipped first-pass implementations with perf or a11y HIGH/MEDIUM findings that CR caught and fixed. The skill is present — the checklist isn't. Each of the four items below maps to a specific Epic 10 story CR finding; citing the precedent makes the issue concrete rather than abstract.

- [ ] **Responsive image sizing:** Any `<img>` fed from TMDb (or other CDN with size variants) MUST use `srcSet` + `sizes` with a sub-`original` baseline in `src`. On mobile the browser must NOT download a 3–5 MB `original` backdrop. Reference: Story 10-2 CR finding H1 — `getImageUrl(..., 'original')` shipped 3–5 MB backdrops to mobile; fixed by `getBackdropSrcSet` (w780/w1280/original) with `w1280` baseline in `src`. Helpers: `apps/web/src/lib/image.ts::getBackdropSrcSet`, `getBackdropSizes`.

- [ ] **Modal focus management:** Any component declaring `aria-modal="true"` MUST trap focus (Tab/Shift+Tab cycles within modal), move focus on open to the modal's first focusable element (usually close button), and restore focus to the invoking trigger on close. Inactive siblings/slides in carousels MUST use React 19 `inert={!active}` to remove from focus order, hit testing, and a11y tree. Reference: Story 10-2 CR findings H2 (TrailerModal had `aria-modal` but no focus trap) + M1 (HeroBanner inactive slides were still tabbable).

- [ ] **aria-live on async-revealed content:** Badges, status pills, or any content that appears asynchronously after load (e.g., ownership badges, download-status pills, availability indicators) MUST carry `role="status"` + `aria-live="polite"` so screen-reader users are notified when the state changes. Reference: Story 10-4 CR finding L1 — `AvailabilityBadge` rendered async after a TanStack Query resolve but had no live-region announcement.

- [ ] **Lazy-load contract accuracy:** Any lazy-load / Intersection-Observer / pagination mechanism MUST have its network contract documented accurately in both (a) the AC text and (b) the code comment at the request site. If lazy-load means "≤N requests" rather than "1 request", say so — do NOT label it "single request". When adding lazy-load to a feature that already has a batching AC from a prior story, grep that prior story for contract references and update in lockstep (pairs with retro-10-AI2 AC contract drift check). Reference: Story 10-5 CR finding H1 — ownership POST claimed "single POST per homepage" but IntersectionObserver made it ≤N POSTs; comment was corrected and an `anyEnabledInflight` stability gate was added.

**Verification steps before marking this section [x]:**

1. Open DevTools Network panel, load the feature at 390px viewport width. Confirm no image request exceeds 1 MB.
2. Tab through any modal/dialog introduced by the story. Confirm focus stays inside and restores correctly on close.
3. Use macOS VoiceOver (or NVDA on Windows) to confirm any async-revealed status announces itself.
4. If the story introduces or modifies lazy-load/pagination: confirm the comment at the request site and the AC text agree on the actual request count semantics.

## 📝 Documentation & Tracking

- [ ] **File List Complete:** File List includes EVERY new, modified, or deleted file (paths relative to repo root)
- [ ] **Dev Agent Record Updated:** Contains relevant Implementation Notes and/or Debug Log for this work
- [ ] **Change Log Updated:** Change Log includes clear summary of what changed and why. Each implemented task should have a corresponding Change Log entry with date. (Epic 8 Retro: TD3/TD4 stories were implemented correctly but Change Log was empty — this check prevents that pattern.)
- [ ] **Review Follow-ups:** All review follow-up tasks (marked [AI-Review]) completed and corresponding review items marked resolved (if applicable)
- [ ] **Story Structure Compliance:** Only permitted sections of story file were modified

## 🎨 UX Design Verification (UI Stories Only)

- [ ] **Design Screenshots Compared:** All relevant design screenshots from `_bmad-output/screenshots/` reviewed against implementation
- [ ] **Layout Structure Match:** Element positioning, flex/grid arrangement, sidebar/overlay patterns match design
- [ ] **Spacing & Sizing Match:** Padding, margins, gaps, widths, heights match design specifications
- [ ] **Typography Match:** Font sizes, weights, colors for all text elements match design
- [ ] **Color Scheme Match:** Background, text, border, and accent colors match design palette
- [ ] **Component Styling Match:** Border radius, shadows, hover states, transitions match design
- [ ] **Content Labels Match:** Chinese text labels, button text, placeholders match design exactly
- [ ] **Discrepancies Fixed:** All identified gaps between design and implementation have been resolved
- [ ] **UX Verification Recorded:** Result (PASS/SKIPPED) recorded in Dev Agent Record

## 🔚 Final Status Verification

- [ ] **Story Status Updated:** Story Status set to "review"
- [ ] **Sprint Status Updated:** Sprint status updated to "review" (when sprint tracking is used)
- [ ] **Quality Gates Passed:** All quality checks and validations completed successfully
- [ ] **No HALT Conditions:** No blocking issues or incomplete work remaining
- [ ] **User Communication Ready:** Implementation summary prepared for user review

## 🎯 Final Validation Output

```
Definition of Done: {{PASS/FAIL}}

✅ **Story Ready for Review:** {{story_key}}
📊 **Completion Score:** {{completed_items}}/{{total_items}} items passed
🔍 **Quality Gates:** {{quality_gates_status}}
📋 **Test Results:** {{test_results_summary}}
📝 **Documentation:** {{documentation_status}}
```

**If FAIL:** List specific failures and required actions before story can be marked Ready for Review

**If PASS:** Story is fully ready for code review and production consideration
