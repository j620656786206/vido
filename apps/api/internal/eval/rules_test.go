package eval

import (
	"reflect"
	"testing"
)

func has(reasons []string, want string) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

// One positive and one negative case per rule class (AC #2 / #5).
func TestRuleCheck_EachClass(t *testing.T) {
	plain := &Cue{ID: "x", Text: "You're a jerk, Thom.", Names: map[string][]string{"Thom": {"湯姆"}}}
	cases := []struct {
		name   string
		prev   *Cue
		cue    *Cue
		next   *Cue
		zh     string
		want   string
		absent bool
	}{
		{name: "missing → empty", cue: plain, zh: "   ", want: "empty"},
		{name: "echoed", cue: plain, zh: "You're a jerk, Thom.", want: "echoed"},
		{name: "not echoed", cue: plain, zh: "湯姆，你這個混蛋。", want: "echoed", absent: true},
		{name: "simplified leak", cue: plain, zh: "汤姆，你这个混蛋。", want: "simplified_leak"},
		{name: "no leak", cue: plain, zh: "湯姆，你這個混蛋。", want: "simplified_leak", absent: true},
		{name: "name mismatch", cue: plain, zh: "托姆，你這個混蛋。", want: ReasonNameMismatch},
		{name: "name present (variant)", cue: &Cue{Text: "Shiva", Names: map[string][]string{"Shiva": {"溼婆", "濕婆"}}}, zh: "濕婆神", want: ReasonNameMismatch, absent: true},
		{name: "forbidden term", cue: &Cue{Text: "The software broke.", Forbid: []string{"軟件", "软件"}}, zh: "軟件壞了。", want: ReasonForbiddenTerm},
		{name: "allowed term", cue: &Cue{Text: "The software broke.", Forbid: []string{"軟件", "软件"}}, zh: "軟體壞了。", want: ReasonForbiddenTerm, absent: true},
		{
			name: "time shift: neighbour's anchor on this cue",
			prev: &Cue{Text: "I was a boxer.", Anchors: []string{"拳擊"}},
			cue:  &Cue{Text: "Killed a man in the ring.", Anchors: []string{"擂台"}},
			zh:   "我以前是拳擊手。", want: ReasonTimeShift,
		},
		{
			name: "time shift absent: own anchor present",
			prev: &Cue{Text: "I was a boxer.", Anchors: []string{"拳擊"}},
			cue:  &Cue{Text: "Killed a man in the ring.", Anchors: []string{"擂台"}},
			zh:   "在擂台上打死過人。", want: ReasonTimeShift, absent: true,
		},
		{
			name: "time shift absent: neither anchor (just a bad line, not a shift)",
			prev: &Cue{Text: "I was a boxer.", Anchors: []string{"拳擊"}},
			cue:  &Cue{Text: "Killed a man in the ring.", Anchors: []string{"擂台"}},
			zh:   "我很餓。", want: ReasonTimeShift, absent: true,
		},
	}
	for _, tc := range cases {
		got := RuleCheck(tc.prev, tc.cue, tc.next, tc.zh)
		if has(got, tc.want) == tc.absent {
			t.Errorf("%s: reasons=%v (want %q absent=%v)", tc.name, got, tc.want, tc.absent)
		}
	}
}

func TestRuleCheck_CleanCueHasNoReasons(t *testing.T) {
	cue := &Cue{Text: "Marcus took the money.", Names: map[string][]string{"Marcus": {"馬可斯"}}}
	next := &Cue{Text: "Elena took the blame.", Names: map[string][]string{"Elena": {"艾蓮娜"}}}
	if got := RuleCheck(nil, cue, next, "錢是馬可斯拿的。"); len(got) != 0 {
		t.Fatalf("clean cue flagged: %v", got)
	}
	// Derived anchors: names' first rendering when no explicit Anchors.
	if got := anchorsOf(next); !reflect.DeepEqual(got, []string{"艾蓮娜"}) {
		t.Fatalf("anchorsOf = %v", got)
	}
}
