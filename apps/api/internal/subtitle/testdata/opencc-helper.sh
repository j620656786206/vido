#!/bin/sh
input=$(cat)
case "$*" in
  *t2s.json*) printf '%s' "$input" | sed -e 's/簡/简/g' -e 's/體/体/g' -e 's/測/测/g' ;;
  *) printf '%s' "$input" | sed -e 's/简体/簡體/g' -e 's/测试/測試/g' -e 's/软件/軟體/g' -e 's/内存/記憶體/g' -e 's/网络/網路/g' -e 's/信息/資訊/g' -e 's/程序/程式/g' -e 's/打印机/印表機/g' -e 's/硬盘/硬碟/g' -e 's/国家/國家/g' -e 's/发展/發展/g' -e 's/寄生虫/寄生蟲/g' -e 's/基泽/基澤/g' -e 's/无业游民/無業遊民/g' -e 's/台灣/臺灣/g' -e 's/这/這/g' -e 's/导演/導演/g' -e 's/电影/電影/g' -e 's/叙事/敘事/g' -e 's/张力/張力/g' -e 's/霸王别姬/霸王別姬/g' -e 's/花样年华/花樣年華/g' -e 's/卧虎藏龙/臥虎藏龍/g' ;;
esac
