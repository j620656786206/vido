#!/bin/bash
# usage: recreate.sh [CLAUDE_MODEL] [--dry]   (empty model = remove override)
set -e
MODEL="$1"; DRY="$2"
ENV_ARGS=""
while IFS= read -r kv; do
  [ -z "$kv" ] && continue
  case "$kv" in CLAUDE_MODEL=*|HOSTNAME=*|PATH=*|HOME=*) continue;; esac
  case "$kv" in *'$'*|*'`'*|*'"'*) echo "unsafe env value in $kv" >&2; exit 1;; esac
  ENV_ARGS="$ENV_ARGS -e \"$kv\""
done < <(docker inspect Vido --format '{{range .Config.Env}}{{println .}}{{end}}')
[ -n "$MODEL" ] && ENV_ARGS="$ENV_ARGS -e CLAUDE_MODEL=$MODEL"
LABEL_ARGS=$(docker inspect Vido --format '{{range $k,$v := .Config.Labels}} -l {{printf "%q" $k}}={{printf "%q" $v}}{{end}}')
IMAGE=$(docker inspect Vido --format '{{.Config.Image}}')
CMD="docker run -d --name Vido $ENV_ARGS $LABEL_ARGS \
  -p 8088:8080/tcp \
  -v /mnt/cache/appdata/vido:/vido-data:rw \
  -v /mnt/cache/appdata/vido/backups:/vido-backups:rw \
  -v /mnt/user/data/media:/media:rw \
  --log-driver json-file --log-opt max-file=1 --log-opt max-size=50m \
  $IMAGE api"
if [ "$DRY" = "--dry" ]; then echo "$CMD" | sed -E 's/(KEY=[^ ]{8})[^"]*/\1.../g'; exit 0; fi
docker stop Vido >/dev/null && docker rm Vido >/dev/null
eval "$CMD"
sleep 6
docker ps --format '{{.Names}} {{.Status}}' | grep Vido
docker logs Vido 2>&1 | grep -E "Subtitle generation pipeline enabled|claude provider" | cut -c1-200
