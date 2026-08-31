---
name: GitHub Account
description: This repo pushes/PRs with the j620656786206 (personal) gh account, not alexyu-tvbs — switch before any PR/CI op
type: project
---

`vido`'s remote is `j620656786206/vido.git`. All `git push` / `gh` PR / CI operations must run under the **`j620656786206`** GitHub account.

**Why:** The machine is also logged into `alexyu-tvbs` (work account, used by the `web-*` work repos). When the wrong account is active, PR creation/merge fails with permission errors — this was a recurring friction point. Work repos like `web-health-nextjs` / `web-health-mvp` use `tvbstw` instead, so the active account differs per repo.

**How to apply:** Before any PR/CI op, run `gh auth status`; if the active account is not `j620656786206`, run `gh auth switch --user j620656786206`. Also enforced in this repo's `CLAUDE.md` and the `/ship` skill (`.claude/skills/ship/SKILL.md`).

**Concurrent-session flip-flop (2026-08-24):** when another Claude session works a tvbs repo on this machine at the same time, it switches the shared gh credential back to `alexyu-tvbs` mid-pipeline — the switch is process-global, not per-session. Symptom: `must be a collaborator (createPullRequest)` or `does not have the correct permissions to execute MergePullRequest` on a repo you were just green on. It happened 3× in one afternoon (PRs #269/#270). Don't debug — re-run `gh auth switch --user j620656786206` immediately before EVERY `gh pr create`/`merge` call (not just once at pipeline start), and expect the other session to flip it back.
