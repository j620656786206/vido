#!/bin/sh
# Vido container entrypoint — PUID/PGID support (story bugfix-i-2-puid-pgid).
#
# WHY THIS EXISTS
# The image used to end with `USER vido` (uid 1000). Unraid's appdata is
# nobody:users (99:100), and Docker creates a missing bind-mount source as
# root:root 0755 — so a first-time install died at startup with
#   VIDO_DATA_DIR: directory '/vido-data' is not writable: permission denied
# and nothing in that message tells you to chown. Community-Apps images are
# expected to honour PUID/PGID; this is that contract.
#
# WHY NOT usermod
# The Unraid template runs the container with `--read-only`, so the linuxserver
# approach of rewriting /etc/passwd fails. Bind mounts stay writable under
# --read-only, so we chown the mounts instead and hand the uid:gid straight to
# su-exec, which takes numeric ids and needs no passwd entry.
#
# Defaults are 1000:1000 — the uid the image has always run as — so pulling this
# image without setting PUID/PGID behaves exactly as before.
set -e

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

case "$PUID$PGID" in
*[!0-9]*)
	echo "vido: PUID/PGID must be numeric (got PUID='$PUID' PGID='$PGID')" >&2
	exit 1
	;;
esac

# Only the writable bind mounts. /app/public is baked into the image and is
# read-only at runtime; touching it would fail under --read-only.
for dir in /vido-data /vido-backups; do
	[ -d "$dir" ] || continue

	# Recursive chown is skipped when the directory already belongs to the target
	# owner, which is the steady state on every restart after the first. Doing it
	# unconditionally would walk the whole poster cache on each boot.
	current="$(stat -c '%u:%g' "$dir" 2>/dev/null || echo '')"
	if [ "$current" != "$PUID:$PGID" ]; then
		echo "vido: taking ownership of $dir ($current -> $PUID:$PGID)"
		# Best-effort: a mount the host exports read-only, or one already owned
		# correctly one level down, must not abort startup. The app's own
		# writability check is the real gate and reports a clear error.
		chown -R "$PUID:$PGID" "$dir" 2>/dev/null ||
			echo "vido: could not chown $dir — continuing; the startup writability check will report if this matters" >&2
	fi
done

exec su-exec "$PUID:$PGID" "$@"
