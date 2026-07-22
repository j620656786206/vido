---
name: infra-gpg-signed-commits
description: vido commits are GPG-signed; git commit hangs in the non-interactive tool shell waiting for pinentry
metadata: 
  node_type: memory
  type: project
  originSessionId: d66c1a6c-4a7f-4bfb-b2c1-9291bf340855
---

This repo has `commit.gpgsign = true` (key `3FF4F2E9543676BC`). `git commit` run from the agent's
non-interactive Bash tool **hangs** (times out at 2 min) because GPG's pinentry can't prompt for the
passphrase there — `--no-verify` does NOT help (it's signing, not the husky hook; the `.husky/pre-commit`
is a no-op and `.git/hooks/post-commit` is branch-gated to main).

**How to commit cleanly:**
- `export GPG_TTY=$(tty 2>/dev/null || echo "")` before `git commit`, and give it a ~90s timeout.
- If the GPG agent is locked, the first commit pops a **pinentry-mac GUI dialog** — the user enters the
  passphrase once (suggest they retry, or run the commit via `!` in their terminal). After that the agent
  caches the passphrase and subsequent agent-run commits succeed instantly.
- Alternative (only if unsigned is acceptable to the user — ASK, since they deliberately sign): `git commit --no-gpg-sign`.

**Always exclude `.claude/github-star-reminder.txt`** from these commits — it shows as modified at session
start but is unrelated; stage only the intended files explicitly. See [[project-gh-account]].
