---
name: project-nas-test-instance
description: "Live Vido deployment on the user's Unraid NAS — URL, container config, SSH access, the 2026-08-01 FUSE/SQLite outage and its fix, and open ops gaps"
metadata: 
  node_type: memory
  type: project
  originSessionId: d88e2713-bc45-4966-9c72-7fd390c97f59
  modified: 2026-08-24T07:12:59.039Z
---

Alex runs a live Vido instance on his **Unraid** NAS (host `Tower`) at `http://192.168.50.52:8088/`,
image `ghcr.io/j620656786206/vido:main`. Reachable from this Mac by `curl`, and **SSH works
key-based as `root@192.168.50.52`** (`ssh -o BatchMode=yes` succeeds — no password needed).
Use both for real diagnosis instead of guessing. `DB_PATH=/vido-data/vido.db`, modernc SQLite, WAL.
Note the NAS is **Unraid, not the DS920+** that the subtitle-pipeline M1 pilot bars target.

**`/health` is the fastest triage tool**: it returns DB `status` / `latency` / `open_connections`.
`open_connections: 0` + ping timeout **while the file is readable from inside the container**
means the Go connection pool is wedged — not a disk, mount, or lock-contention problem.
`/api/v1/scanner/status` answers without touching the DB, so "that one works but everything
else hangs" is the signature of a DB-layer failure rather than a dead process.

## The FUSE/SQLite outage — predicted 2026-07-13, happened 2026-08-01, fixed 2026-08-04

The container used to mount `/mnt/user/appdata/vido` (Unraid's **shfs FUSE** user-share layer).
SQLite WAL needs an mmap'd `-shm` file plus POSIX locks, which FUSE does not handle reliably.
On **2026-08-01 07:17 local** the DB wedged: process alive at 0% CPU, `/health` 503 with a 3s
ping timeout forever, every DB-backed endpoint hanging, **for three days** before Alex noticed.
Root trigger unknowable — Unraid's syslog had already rotated past it.

**Fix applied 2026-08-04 (verified):** back up db+wal+shm → `docker restart` (WAL replayed
cleanly, zero data loss) → recreate the container with **`/mnt/cache/appdata/vido`** mounted
directly, removing FUSE from the DB I/O path. Confirmed by `/proc/*/fd` scan: `shfs` no longer
holds the DB, only the `api` process does. Safe because the `appdata` share is
`shareUseCache="only"` on the `cache` (nvme) pool — nothing lived on the array.
The media mount **stays** `/mnt/user/data/media:ro` (a genuine multi-disk share).
Old container kept as `vido-prefusefix` (Exited 0) as a rollback point.

**There now IS an Unraid template** — as of 2026-08-24 a container recreate/redeploy should go
through `/usr/local/emhttp/plugins/dynamix.docker.manager/scripts/rebuild_container Vido`
(same code path the Unraid WebUI "Update"/"Apply" button uses; reads
`/boot/config/plugins/dockerMan/templates-user/my-Vido.xml` — the **host's own stored template,
distinct from the repo's `unraid-template/vido.xml`**, which only seeds a *new* install/Community
Apps import and does not auto-sync to an existing one). Verified working three times this session
(pulls the new image, tears down + recreates the container, cleans the orphaned old image). Do
NOT hand-roll `docker run` for a redeploy on this box anymore — use the template script.

**Deploy checklist that's now proven to work:** `sqlite3 vido.db ".backup <dir>/vido.db"` before
touching anything → `docker pull ghcr.io/j620656786206/vido:main` → compare
`docker inspect Vido --format '{{.Image}}'` vs `docker inspect <img> --format '{{.Id}}'` (a plain
`docker restart` does NOT pick up a newly-pulled image, only `rebuild_container`/recreate does) →
`rebuild_container Vido` → poll `/health` and re-verify row counts.

## Resolved gaps (as of 2026-08-24 — do not cite the 2026-08-04 "open gaps" list below as current)

1. **Backups now work and have real history.** `/mnt/cache/appdata/vido/backups/` holds multiple
   dated backups from this session's fixes (`pre-wipe-*`, `pre-scanner-fix-*`, `pre-puid-fix-*`),
   each taken via `sqlite3 ".backup"` before a risky change and left in place afterward. The
   in-app scheduled-backup feature's own health is still unverified — these are all manual.
2. **PUID/PGID is now supported** (`bugfix-i-2-puid-pgid`, PR #276, merged 2026-08-24) —
   `docker-entrypoint.sh` chowns the writable bind mounts and drops to `PUID:PGID` via `su-exec`
   before exec'ing the API, because the template runs `--read-only` so the usual `usermod`
   startup trick doesn't work. Defaults stay `1000:1000`; this box's appdata already was
   `1000:1000` so the deploy was a verified no-op on ownership. New installs should use `99:100`
   (Unraid's `nobody:users`) via the template's now-exposed `PUID`/`PGID` Config fields — but that
   only applies to a *fresh* install using the repo template; this box's stored host template
   was deliberately left untouched since its defaults already match.
3. **The data-integrity story from 2026-07-13 is resolved, not just re-verify-worthy.** The
   "5922 ghost movie rows" state was fixed 2026-08-24 by wiping and rescanning
   (`bugfix-c-data-migration`, done via reset not migration — see that story for why) plus a real
   scanner parser bug found in the process (`bugfix-scanner-bracket-prefix-filenames-dropped`,
   PR #272: bracket-prefixed release-group filenames were silently collapsing ~17% of TV episodes
   onto one placeholder row). Current clean state: movies=55, episodes=2406 (matches disk within
   a documented ~62-file residual of legitimate duplicate releases). Re-verify row counts before
   citing this as "clean" in a *future* session, but do not re-cite the old 5922/57 numbers.
4. **Still open, not touched this session:** no restart policy (`RestartPolicy=no` — a dead
   process stays dead unnoticed; this was the reason the 2026-08-01 outage ran three days).
5. **Also live on this box, not a defect being tracked here:** `AI_PROVIDER=gemini` with an
   **empty `GEMINI_API_KEY`** — AI-assisted parsing is silently off. Owner decision (set a key,
   or switch to `AI_PROVIDER=claude`), not a bug — see `bugfix-gemini-default-model-retired`.

See [[feedback-let-user-demo-before-proposing]], [[project-qbt-state-mapping]],
[[feedback-user-may-merge-prs-manually]].
