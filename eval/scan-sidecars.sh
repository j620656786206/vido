#!/bin/bash
# List videos that have an external Chinese subtitle sidecar, with their embedded sub/audio langs.
cd /mnt/user/data/media
find tv movies -type f \( -iname "*.srt" -o -iname "*.ass" -o -iname "*.ssa" \) ! -iname "*.zh-Hant.srt" ! -iname "*.bak" ! -iname "*.tmp.*" 2>/dev/null | sort | while IFS= read -r sub; do
  dir=$(dirname "$sub"); base=$(basename "$sub")
  stem=$(echo "$base" | sed -E 's/\.(srt|ass|ssa)$//; s/\.(hi|sdh|forced)$//; s/\.(zh-TW|zh-Hant|zh-HK|zh-CN|zh-Hans|zh|cht|chs|chi|zho|tw|cn|tc|sc|繁體|简体|繁|简|eng|en|ja|ko|default)$//I; s/\.(hi|sdh|forced)$//')
  vid=$(ls "$dir" 2>/dev/null | grep -iE "\.(mkv|mp4|avi|m4v)$" | grep -F "$stem" | head -1)
  [ -z "$vid" ] && vid=$(ls "$dir" 2>/dev/null | grep -iE "\.(mkv|mp4|avi|m4v)$" | head -1)
  [ -z "$vid" ] && continue
  echo "$dir/$vid|$base"
done | awk -F'|' '{a[$1]=a[$1] (a[$1]?"; ":"") $2} END{for(k in a) print k "|" a[k]}' | sort
