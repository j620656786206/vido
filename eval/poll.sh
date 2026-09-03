#!/bin/bash
# usage: poll.sh <want_done_count> <max_loops(30s)>
DB=/mnt/user/appdata/vido/vido.db
want=${1:-5}; loops=${2:-17}
for i in $(seq 1 $loops); do
  n=$(sqlite3 $DB "select count(*) from subtitle_runs where started_at > datetime('now','-3 hours') and status in ('completed','failed','skipped');")
  [ "$n" -ge "$want" ] && break
  sleep 30
done
echo "---runs (done=$n)"
sqlite3 -header $DB "select substr(media_id,1,8) m, status, model_id, prompt_version, glossary_version, cue_count, round(spent_usd,3) usd, source_language, substr(error_message,1,160) err, substr(started_at,12,8) st, substr(completed_at,12,8) done from subtitle_runs order by started_at desc limit 12;"
echo "---warn/err"
docker logs --since 12m Vido 2>&1 | grep -E "level=(WARN|ERROR)" | grep -v "GET /api" | cut -c1-220 | tail -15
