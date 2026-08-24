# Story bugfix-i-2: the image ignores PUID/PGID, so a first install cannot start

Status: review

## Story

As someone installing vido on Unraid for the first time,
I want the container to run as the uid/gid my appdata is owned by,
so that it starts instead of dying on a permission error that never mentions ownership.

## Scope note — this is a split, not the whole of `bugfix-i`

`bugfix-i-unraid-deployment-hardening` is a **six-item bundle** filed from the original deployment. Shipping it as one story would violate the story-splitting rule and mix a release blocker with P3 polish. This story takes **item (2)** — the one that stops a new install dead — plus **item (6)**, a one-line URL fix on the same deployment surface. The remaining four are filed as their own entries at story-creation time (Rule 24 ③) and named in Discovery Triage.

## Evidence — reproduced first-hand 2026-08-24

While setting up an isolated verification container for `bugfix-scanner-bracket-prefix-filenames-dropped`:

```
$ mkdir -p /mnt/cache/appdata/vido-verify        # created root:root, as Docker also does
$ docker run ... -v /mnt/cache/appdata/vido-verify:/vido-data ghcr.io/.../vido:main
Exited (1)
ERROR Configuration validation failed error="configuration validation failed:
  - VIDO_DATA_DIR: directory '/vido-data' is not writable:
    open /vido-data/.vido_write_test: permission denied"
```

- `Dockerfile` ends with `USER vido` (uid 1000 via `adduser -u 1000`). Unraid's appdata is `nobody:users` = **99:100**, and Docker creates a missing bind-mount source as **root:root 0755**. Either way uid 1000 cannot write.
- The message never says "chown" or names an owner, so the installer has no path forward. `chown -R 1000:1000` fixes it, which is not discoverable.
- Community-Apps images are expected to honour `PUID`/`PGID`; vido exposes neither.
- The existing install only works because its appdata happens to be owned by uid 1000.

### The constraint that rules out the usual fix

`unraid-template/vido.xml:15` sets `ExtraParams="--read-only --tmpfs /tmp:size=64M,mode=1777"`. The linuxserver.io approach — `usermod`/`groupmod` the baked-in user at startup — writes to `/etc/passwd` and `/etc/group` and therefore **fails under `--read-only`**. Bind mounts stay writable, and `su-exec` accepts a numeric `uid:gid` with no passwd entry, so the workable shape is: chown the mounts, drop privileges numerically.

## Acceptance Criteria

1. `PUID` / `PGID` env vars select the runtime uid/gid. Defaults are **1000:1000** — the identity the image has always used — so an existing deployment that sets neither behaves exactly as before.
2. An entrypoint runs as root **only** long enough to take ownership of the writable bind mounts (`/vido-data`, `/vido-backups`), then `exec`s the API as `PUID:PGID` via `su-exec`. `USER vido` is removed from the Dockerfile, because without a root step there is no way to repair a mount Docker created as `root:root`.
3. `/app/public` is **not** chowned — it is baked into the image and read-only at runtime; touching it would fail under `--read-only`.
4. The recursive chown is **skipped** when a directory already has the target owner, so the steady-state restart does not walk the poster cache. Verified by observing no ownership message on a second start.
5. A chown that cannot succeed (host-exported read-only mount, read-only rootfs) logs a warning and **continues** — the app's own writability check is the real gate and produces a clear error. Startup must not abort on a best-effort step.
6. A non-numeric `PUID` or `PGID` fails fast with exit code 1 and a message naming the offending value, rather than being passed to `su-exec` to produce an opaque error.
7. `unraid-template/vido.xml` exposes `PUID`/`PGID` with Unraid's conventional **99 / 100** defaults, described as safe to change (the entrypoint migrates ownership), placed before the API-key blocks so they read as install-time identity rather than credentials.
8. **Item (6):** the template `<Icon>` URL points at `j620656786206/vido`, not the 404-ing `alexyu/vido`.
9. Verified by actually running the built image, not by reading the Dockerfile: a `root:root` mount is adopted and the process runs as the requested ids; `99:100` works; `--read-only` degrades gracefully; a non-numeric value exits 1; a second start skips the chown; **an already-1000:1000 install is left untouched**.
10. Template XML stays well-formed (`xml.etree` parses it). Zero Go changes, zero frontend changes.

## Tasks / Subtasks

- [x] Task 1 — Entrypoint (AC: #1–#6)
  - [x] 1.1 `docker-entrypoint.sh`: validate ids, conditional recursive chown of the two mounts, `exec su-exec`
  - [x] 1.2 Dockerfile: add `su-exec`, `ENV PUID/PGID`, `COPY` + `ENTRYPOINT`, drop `USER vido`
- [x] Task 2 — Template (AC: #7, #8, #10)
  - [x] 2.1 `PUID`/`PGID` Config blocks with 99/100 defaults
  - [x] 2.2 Fix the `<Icon>` URL
  - [x] 2.3 XML well-formedness check
- [x] Task 3 — Verify against a real build (AC: #9)
  - [x] 3.1 Build on the NAS and exercise all six scenarios, recording actual output

## Dev Notes

- **Do not use `usermod`.** It is the common recipe and it does not work here — see the `--read-only` constraint above. `su-exec` with numeric ids needs no passwd entry.
- **`ENTRYPOINT` + existing `CMD ["api"]`** means the entrypoint receives `api` as `$1` and `exec`s it; `api` is on `PATH` at `/usr/local/bin/api`. Do not change `CMD`.
- **Default 1000:1000, template default 99:100.** These differ on purpose: the image default preserves history for anyone running it raw, while the template teaches new Unraid installs the correct convention. Because the entrypoint chowns, moving an existing install from 1000 to 99 is self-healing rather than breaking.
- **Best-effort chown is deliberate, not sloppy.** Aborting startup because one directory could not be chowned would convert a warning into an outage; the app already validates writability and reports it clearly.
- Rule 7 / 10 / 20: **N/A** — no error codes, no routes, no wire contract. Rule 23: N/A (no frontend).
- Not in scope, tracked separately: the DB-dead-but-200 state, the permanently-unhealthy DB, the health-ping timeout, and in-container diagnosis. See Discovery Triage.

### Time-dependent visual coverage

- N/A — container packaging only, no `apps/web/src/components/**` touched.

### References

- [Source: Dockerfile — `adduser -u 1000`, `USER vido`, runtime stage]
- [Source: unraid-template/vido.xml:14-15 — the 404 icon URL and the `--read-only` ExtraParams]
- [Source: sprint-status.yaml `bugfix-i-unraid-deployment-hardening` — the six-item bundle and the first-hand reproduction]

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Claude Fable 5)

### Debug Log References

Built a minimal image on the NAS (alpine + su-exec + the entrypoint, `CMD ["id"]`) so the entrypoint could be exercised without a 12-minute full app build. Actual output:

| scenario | result |
|---|---|
| `root:root` mount, default PUID | `taking ownership of /vido-data (0:0 -> 1000:1000)` → `uid=1000 gid=1000` |
| `PUID=99 PGID=100` | `uid=99 gid=100(users)` |
| `--read-only --tmpfs /tmp` | warned on the unchownable dir, still reached `uid=99 gid=100(users)` |
| `PUID=abc` | `PUID/PGID must be numeric (got PUID='abc' PGID='1000')`, **exit 1** |
| `PGID=1x` | same message, **exit 1** |
| second start, mount already `99:100` | no ownership message — chown skipped |
| existing `1000:1000` install, default PUID | no ownership message — untouched |

**Two of my own tests lied on the first pass and were re-run.** The exit-code check read `$?` from a `\| tail -2` pipeline, so it reported 0 for a script that had correctly exited 1. The skip check counted a message from `/vido-backups`, which in the *test* image is a rootfs directory rather than a bind mount and so is legitimately re-chowned every run — `/vido-data` was being skipped correctly all along. Both were re-verified in isolation before any of the above was believed.

### Completion Notes List

- **The blocker is closed against a real build, not against a reading of the Dockerfile.** The exact failure that motivated this story — `root:root` bind mount → `Exited (1)` — now starts cleanly.
- **`--read-only` is why this is not the linuxserver recipe.** `usermod` writes `/etc/passwd`; the template runs `--read-only`. Chowning the (writable) bind mounts and handing numeric ids to `su-exec` is the shape that actually works, and scenario 3 above proves it degrades gracefully rather than dying when a target cannot be chowned.
- **Zero impact on the existing install** (scenario 7): appdata is already `1000:1000` and the image default is `1000:1000`, so the conditional chown does nothing. Ownership only moves if the owner explicitly sets `PUID`/`PGID` — and because the entrypoint chowns, that move is self-healing rather than a manual migration.
- **`USER vido` removal is the deliberate trade.** Running the entrypoint as root is required to repair a mount Docker created as `root:root`; privileges are dropped via `exec su-exec` before the API ever starts, so the long-lived process is still unprivileged. Every Unraid/Community-Apps image makes the same trade.
- **Item (6) folded in** because it is one line on the same file: the template `<Icon>` pointed at `alexyu/vido`, which 404s — the account is `j620656786206`.
- Template XML re-parsed with `xml.etree` after editing (AC #10).
- 🔗 AC Drift: **NONE** — no prior story specifies container identity or entrypoint behaviour; this is the first. Checked `Dockerfile` / `unraid-template/` references across `_bmad-output/implementation-artifacts/*.md`.
- 📎 Contract Stamps: NONE (no `[@contract-v*]` in scope; packaging change, no wire contract).
- 🎭 A11y Pre-Flight: N/A (no `apps/web/` files touched).
- 🎨 UX Verification: SKIPPED — no UI changes.

### Discovery Triage

- **YES — the parent bundle was split. Four sibling entries filed at story-creation time (Rule 24 ③, bidirectional):**
  - `bugfix-i-1-sqlite-permanent-unhealthy` — the P0-latent DB death that only a reinstall cleared; evidence was destroyed, so it needs a reproduction path before it can be fixed.
  - `bugfix-i-3-db-dead-returns-200` — P1: the server answers 200 on `/` and `/health/services` while every DB-backed call fails differently. This is what made one defect look like ten.
  - `bugfix-i-4-health-ping-ctx-below-busy-timeout` — P2. **The parent entry's numbers were wrong and are corrected in the new entry:** the ping contexts are 5s / 5s / 2s (`database/health.go:77,85,112`) against a 5s `DB_BUSY_TIMEOUT`, not the "3s" recorded. `QuickHealth`'s 2s is the real offender. Fixing it means giving the health path access to the configured timeout, so it is not the one-liner the parent implied.
  - `bugfix-i-5-no-in-container-diagnosis` — P3: `--read-only` + non-root blocks `apk add sqlite`; wants a `vido db` subcommand or a documented throwaway-alpine recipe.
- `bugfix-i-unraid-deployment-hardening` is annotated with the split and left open as the tracking parent until its last child closes.

### File List

- Dockerfile
- docker-entrypoint.sh (new)
- unraid-template/vido.xml
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/bugfix-i-2-puid-pgid.md

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | Task 1: `docker-entrypoint.sh` (numeric-id validation, conditional recursive chown of the two writable mounts, best-effort on failure, `exec su-exec`); Dockerfile gains `su-exec` + `ENV PUID/PGID` + `ENTRYPOINT` and drops `USER vido`. |
| 2026-08-24 | Task 2: template exposes `PUID`/`PGID` at Unraid's 99/100 before the API-key blocks; `<Icon>` URL corrected from the 404-ing `alexyu/vido` to `j620656786206/vido`; XML re-parsed. |
| 2026-08-24 | Task 3: all six scenarios exercised against a real build on the NAS. Two flawed test harnesses caught and re-run before their results were trusted. |
| 2026-08-24 | Split `bugfix-i` — four sibling entries filed for items (1), (3), (4), (5); item (4)'s recorded timeout values corrected in the process. |
