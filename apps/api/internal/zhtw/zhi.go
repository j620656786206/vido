package zhtw

import "strings"

// OpenCC (1.4.2, both s2tw and s2twp) reads 只 after 是／多／別／有 as the
// measure word 隻 — 「还是只有她知道」 comes out 「還是隻有她知道」, 「最多只能」
// comes out 「最多隻能」. It does it to Traditional input too, so a translation
// the model wrote correctly is broken by the final pass (issue #939: 9 of 83
// subtitles generated in the 60 days before 2026-11-01 carried it).
//
// The fix keeps OpenCC from touching those 只 at all: before the script step,
// each 只 that reads as the adverb "only" is swapped for a private-use
// placeholder OpenCC passes through untouched, and swapped back afterwards. It
// never rewrites an existing 隻, so a Traditional 「那是隻會說話的鳥」 stays as
// written; the only guess is Simplified input such as 「是只会说话的鸟」, where
// the adverb reading is the far more common one in dialogue.

// zhiPlaceholder is a private-use code point: OpenCC has no mapping for it and
// leaves the characters around it converting as they would on their own.
const zhiPlaceholder = ''

// zhiTriggers are the words OpenCC misreads before 只 that also precede the
// adverb in everyday speech (还是只有, 最多只能, 别只顾着, 没有只想, 不单只是).
// Numbers and demonstratives (一只, 这只, 哪只) are left alone: there the
// measure word is the usual reading.
var zhiTriggers = []string{"是", "最多", "頂多", "顶多", "至多", "別", "别", "沒有", "没有", "不單", "不单"}

// zhiAdverbFollowers are the characters that, right after 只, mark it as "only"
// rather than a measure word (a measure word is followed by its noun: 隻狗,
// 隻手). Simplified and Traditional forms are both listed.
var zhiAdverbFollowers = map[rune]bool{}

func init() {
	for _, r := range "有要能會会是想剩在得不因怕顧顾差需用給给對对為为限管聽听看說说做知求拿好靠跟和把讓让等當当去來来敢肯記记懂准許许吃喝帶带叫關关擔担愛爱怪認认覺觉夠够過过" {
		zhiAdverbFollowers[r] = true
	}
}

// protectAdverbZhi swaps every adverb 只 (see zhiTriggers, zhiAdverbFollowers)
// for zhiPlaceholder. Text that already holds the placeholder is returned
// unchanged, so restoreZhi can never turn a character the input really had
// into 只.
func protectAdverbZhi(s string) string {
	if !strings.ContainsRune(s, '只') || strings.ContainsRune(s, zhiPlaceholder) {
		return s
	}
	runes := []rune(s)
	changed := false
	for i, r := range runes {
		if r != '只' || i+1 >= len(runes) || !zhiAdverbFollowers[runes[i+1]] {
			continue
		}
		for _, t := range zhiTriggers {
			tr := []rune(t)
			if i >= len(tr) && string(runes[i-len(tr):i]) == t {
				runes[i] = zhiPlaceholder
				changed = true
				break
			}
		}
	}
	if !changed {
		return s
	}
	return string(runes)
}

func restoreZhi(s string) string {
	return strings.ReplaceAll(s, string(zhiPlaceholder), "只")
}
