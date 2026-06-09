---
name: GitHub Account
description: This repo pushes/PRs with the j620656786206 (personal) gh account, not alexyu-tvbs — switch before any PR/CI op
type: project
---

`vido`'s remote is `j620656786206/vido.git`. All `git push` / `gh` PR / CI operations must run under the **`j620656786206`** GitHub account.

**Why:** The machine is also logged into `alexyu-tvbs` (work account, used by the `web-*` work repos). When the wrong account is active, PR creation/merge fails with permission errors — this was a recurring friction point. Work repos like `web-health-nextjs` / `web-health-mvp` use `tvbstw` instead, so the active account differs per repo.

**How to apply:** Before any PR/CI op, run `gh auth status`; if the active account is not `j620656786206`, run `gh auth switch --user j620656786206`. Also enforced in this repo's `CLAUDE.md` and the `/ship` skill (`.claude/skills/ship/SKILL.md`).
