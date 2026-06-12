---
name: project-gh-account-for-prs
description: Which gh CLI account to use when opening PRs on the vido repo
metadata: 
  node_type: memory
  type: project
  originSessionId: 3a92a316-89f4-4760-a17f-f75656d85df0
---

Opening PRs on `j620656786206/vido` requires the `gh` CLI active account to be **`j620656786206`** (the repo owner). The machine has TWO gh accounts logged in; the default-active one is **`alexyu-tvbs`** (a work account that is NOT a collaborator on vido) — using it makes `gh pr create` fail with `GraphQL: must be a collaborator (createPullRequest)`.

**Fix before any `gh pr`/`gh api` write op on vido:**
`gh auth switch --hostname github.com --user j620656786206`

⚠️ **The switch does NOT reliably persist across commands (observed 2026-06-12, PR #60).** The active account reverted to `alexyu-tvbs` between `gh pr create` (succeeded) and `gh pr merge` (failed `must be a collaborator`). So switch **atomically, immediately before each write op, in the same command**:
`gh auth switch --user j620656786206 && gh pr merge <N> --squash --delete-branch`

⚠️ **`gh auth status` LIES about the active account.** It showed `j620656786206` as `Active account: true` while `gh api user` returned `alexyu-tvbs`. **Trust `gh api user --jq .login`, not `gh auth status`,** to verify identity.

Note: SSH `git push` works regardless (the SSH key resolves to the owner); only the `gh` OAuth identity needs switching. First hit: 2026-06-07, PR #32. Re-confirmed + nuances added 2026-06-12, PR #60. Re-confirmed again 2026-06-12, PR #61 (same revert: `gh pr create` failed `must be a collaborator` because `gh api user` was `alexyu-tvbs` despite an earlier switch; atomic switch-in-same-command worked for both create and merge).
