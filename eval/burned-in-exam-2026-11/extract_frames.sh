#!/bin/sh
# runs inside the Vido container: extract 6 frames per file
OUT=/tmp/burn; mkdir -p $OUT
while IFS="$(printf '\t')" read -r n path; do
  d=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$path" </dev/null 2>/dev/null | cut -d. -f1)
  [ -z "$d" ] && { echo "$n nodur"; continue; }
  for p in 15 30 45 60 75 90; do
    t=$(( d * p / 100 ))
    ffmpeg -nostdin -v error -ss $t -i "$path" -frames:v 1 -vf scale=480:-2 -q:v 4 -y "$OUT/${n}_${p}.jpg" 2>/dev/null
  done
  echo "$n done $d"
done < /tmp/burn_list.tsv
