package eval

import (
	"context"
	"errors"
	"strings"
	"testing"
)

var items = []JudgeItem{
	{ID: "a", Text: "Hi", Refs: []string{"嗨", "你好"}, Candidate: "嗨"},
	{ID: "b", Text: "Bye", Refs: []string{"再見", "掰"}, Candidate: "拜拜"},
}

func TestParseJudgeResponse(t *testing.T) {
	good := "```json\n[{\"id\":\"a\",\"score\":2,\"why\":\"自然\"},{\"id\":\"b\",\"score\":1}]\n```"
	got, err := ParseJudgeResponse(good, items)
	if err != nil {
		t.Fatalf("good reply: %v", err)
	}
	if got["a"].Score != 2 || got["b"].Score != 1 || got["a"].Why != "自然" {
		t.Fatalf("scores = %+v", got)
	}
	bad := map[string]string{
		"not json":     "sure! a=2 b=1",
		"missing cue":  `[{"id":"a","score":2}]`,
		"out of range": `[{"id":"a","score":3},{"id":"b","score":1}]`,
		"negative":     `[{"id":"a","score":-1},{"id":"b","score":1}]`,
	}
	for name, raw := range bad {
		if _, err := ParseJudgeResponse(raw, items); !errors.Is(err, ErrJudgeResponse) {
			t.Errorf("%s: want ErrJudgeResponse, got %v", name, err)
		}
	}
}

type scriptedCompleter struct {
	replies []string
	calls   int
	lastSys string
	lastUsr string
}

func (s *scriptedCompleter) CompleteText(_ context.Context, sys, usr string, _ int) (string, error) {
	s.lastSys, s.lastUsr = sys, usr
	if s.calls >= len(s.replies) {
		return "", errors.New("no more replies")
	}
	r := s.replies[s.calls]
	s.calls++
	if r == "ERR" {
		return "", errors.New("upstream")
	}
	return r, nil
}

func TestJudgeBatch_RetriesOnceThenFails(t *testing.T) {
	c := &scriptedCompleter{replies: []string{"garbage", `[{"id":"a","score":2},{"id":"b","score":0}]`}}
	got, err := JudgeBatch(context.Background(), c, items)
	if err != nil || got["b"].Score != 0 || c.calls != 2 {
		t.Fatalf("retry path: err=%v got=%v calls=%d", err, got, c.calls)
	}
	if !strings.Contains(c.lastSys, "0 ＝") || !strings.Contains(c.lastUsr, "\"id\": \"b\"") {
		t.Fatalf("prompt not built from rubric/items")
	}
	c = &scriptedCompleter{replies: []string{"garbage", "still garbage"}}
	if _, err := JudgeBatch(context.Background(), c, items); !errors.Is(err, ErrJudgeResponse) {
		t.Fatalf("two bad replies should surface ErrJudgeResponse, got %v", err)
	}
	c = &scriptedCompleter{replies: []string{"ERR"}}
	if _, err := JudgeBatch(context.Background(), c, items); err == nil || errors.Is(err, ErrJudgeResponse) {
		t.Fatalf("transport error must pass through unchanged, got %v", err)
	}
}

func TestJudgeSystemPrompt_IsStableRubric(t *testing.T) {
	p := JudgeSystemPrompt()
	for _, must := range []string{"0 ＝", "1 ＝", "2 ＝", "JSON", "簡體"} {
		if !strings.Contains(p, must) {
			t.Errorf("rubric lost %q", must)
		}
	}
	if JudgeVersion != "judge-v1" {
		t.Errorf("bump JudgeVersion only with a rubric change")
	}
}
