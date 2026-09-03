#!/bin/sh
# scan-partial-zh.sh — 量「同一部影集裡，有些集有官方繁中、有些集沒有」的比例。
#
# 這是 party-mode 2026-09-03 的裁定：加速器②（用片庫裡現成的官方繁中字幕
# 餵同影集其他集的詞彙表）只有在「部分集有、部分集沒有」時才成立，所以先量
# 再決定要不要做。
#
# 在 Vido 容器裡跑（host 沒有 ffprobe）：
#   scp eval/scan-partial-zh.sh root@192.168.50.52:/mnt/user/appdata/vido/eval/
#   ssh root@192.168.50.52 'docker exec -i Vido sh -s < /mnt/user/appdata/vido/eval/scan-partial-zh.sh' > eval/partial-zh.csv
#
# 每一集的判定（跟 SelectCandidates 的 deliver/translate/ASR 路由對齊）：
#   has_zh   = 有外掛 zh-TW/zh-Hant/cht/tc/繁 字幕，或內嵌 chi/zho 文字軌 → deliver，$0
#   translate= 沒有中文、但有內嵌英文文字軌                                   → 走翻譯
#   asr      = 兩者皆無（PGS 圖片軌或無字幕）                                  → 走 ASR
# Vido 自己產出的 `.zh-Hant.srt`（含 .bak）一律不算官方字幕。
#
# 輸出 CSV：kind,name,total,has_zh,translate,asr,partial
#   partial=1 表示 has_zh>0 且 translate>0 —— 加速器②派得上用場的影集。

MEDIA=${MEDIA:-/media}
command -v ffprobe >/dev/null 2>&1 || { echo "ffprobe not found" >&2; exit 1; }
command -v awk >/dev/null 2>&1 || { echo "awk not found" >&2; exit 1; }

# classify <video-path> → 印出 has_zh|translate|asr 其中之一
classify() {
  f=$1
  dir=$(dirname "$f")
  stem=$(basename "$f"); stem=${stem%.*}

  # 外掛字幕：同 stem 開頭、副檔名 srt/ass/ssa、語言 tag 是繁中。排除 Vido 自產。
  ext_zh=0
  for s in "$dir/$stem".*; do
    [ -f "$s" ] || continue
    case "$s" in
      *.zh-Hant.srt|*.zh-Hant.srt.bak|*.bak|*.tmp.*) continue ;;
      *.srt|*.ass|*.ssa) ;;
      *) continue ;;
    esac
    case "$s" in
      *.zh-TW.*|*.zh-tw.*|*.cht.*|*.CHT.*|*.tc.*|*.TC.*|*繁*|*.zh-Hant.hi.*|*.zh-Hant.sdh.*) ext_zh=1; break ;;
    esac
  done

  emb_zh=0; emb_en=0
  # 每行：codec_name,language（tag 缺時為空）
  probe=$(ffprobe -v error -select_streams s -show_entries stream=codec_name:stream_tags=language -of csv=p=0 "$f" 2>/dev/null)
  if [ -n "$probe" ]; then
    emb_zh=$(printf '%s\n' "$probe" | awk -F, 'tolower($1)~/^(subrip|ass|ssa|mov_text|webvtt|text|srt)$/ && tolower($2)~/^(chi|zho|zh)/ {c=1} END{print c+0}')
    emb_en=$(printf '%s\n' "$probe" | awk -F, 'tolower($1)~/^(subrip|ass|ssa|mov_text|webvtt|text|srt)$/ && tolower($2)~/^(eng|en)/ {c=1} END{print c+0}')
  fi

  if [ "$ext_zh" = 1 ] || [ "$emb_zh" = 1 ]; then echo has_zh
  elif [ "$emb_en" = 1 ]; then echo translate
  else echo asr
  fi
}

echo "kind,name,total,has_zh,translate,asr,partial"

# 影集：以 tv/<series> 資料夾為單位彙總
for series in "$MEDIA"/tv/*/; do
  [ -d "$series" ] || continue
  name=$(basename "$series")
  total=0; hz=0; tr=0; as=0
  find "$series" -type f \( -iname '*.mkv' -o -iname '*.mp4' -o -iname '*.avi' -o -iname '*.m4v' \) 2>/dev/null | sort > /tmp/_eps.txt
  while IFS= read -r v; do
    [ -n "$v" ] || continue
    total=$((total+1))
    case $(classify "$v") in
      has_zh) hz=$((hz+1)) ;;
      translate) tr=$((tr+1)) ;;
      asr) as=$((as+1)) ;;
    esac
  done < /tmp/_eps.txt
  [ "$total" -gt 0 ] || continue
  partial=0; [ "$hz" -gt 0 ] && [ "$tr" -gt 0 ] && partial=1
  printf 'tv,"%s",%d,%d,%d,%d,%d\n' "$name" "$total" "$hz" "$tr" "$as" "$partial"
done

# 電影：逐部列出（系列電影要靠 TMDb collection 才能分組，這裡先不分）
find "$MEDIA"/movies -type f \( -iname '*.mkv' -o -iname '*.mp4' -o -iname '*.avi' -o -iname '*.m4v' \) 2>/dev/null | sort > /tmp/_movies.txt
while IFS= read -r v; do
  [ -n "$v" ] || continue
  name=$(basename "$(dirname "$v")")
  case $(classify "$v") in
    has_zh) printf 'movie,"%s",1,1,0,0,0\n' "$name" ;;
    translate) printf 'movie,"%s",1,0,1,0,0\n' "$name" ;;
    asr) printf 'movie,"%s",1,0,0,1,0\n' "$name" ;;
  esac
done < /tmp/_movies.txt
