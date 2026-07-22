---
name: project-nas-test-instance
description: "Live Vido test deployment on the user's Unraid NAS — URL, how it's run, and the P0 data-integrity bug found there on 2026-07-13"
metadata: 
  node_type: memory
  type: project
  originSessionId: 6c31910b-68fa-40da-9e5a-50fdfbc94571
---

Alex runs a live Vido instance on his Unraid NAS at `http://192.168.50.52:8088/`
(image `ghcr.io/j620656786206/vido:main`, `--read-only`, `-v /mnt/user/appdata/vido:/vido-data`,
`-v /mnt/user/data/media:/media:ro`). It is reachable from this Mac via Bash `curl` — use it
for real smoke sweeps instead of guessing. `/health` and `/api/v1/*` are the API surface
(83 GET routes; `DB_PATH=/vido-data/vido.db`, modernc SQLite, WAL).

**On 2026-07-13 a smoke sweep found the library is full of ghost records**: 5922 movie rows
but the scanner only sees 57 files; 222/222 sampled movies have `file_path: null` and
`year: null`; TV series sit in the `movies` table duplicated once per episode
(雪中悍刀行 ×20, 最後生還者 ×16) while `/api/v1/series` returns 0 and `tv_count: 0`.
This one defect is very likely the source of most of the "many bugs" Alex sees in the UI.

**Why:** Alex reports bugs from real NAS usage, not from local dev — symptoms he describes
are downstream of this deployment's data state, so reproduce against the NAS, not localhost.

**How to apply:** Before triaging any Vido UI bug Alex reports, check whether it is a costume
of the ghost-library defect. Also note the image has **no PUID/PGID support** (runs as uid 1000
while Unraid appdata is nobody:users 99:100) and SQLite lives on `/mnt/user` (FUSE/shfs) —
both are latent causes of the DB going permanently unhealthy. See
[[feedback-let-user-demo-before-proposing]] and [[project-qbt-state-mapping]].
