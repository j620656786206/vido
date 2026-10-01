package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vido/api/internal/ai"
)

func sampleCues() []Cue {
	return []Cue{
		{ID: "c1", Source: "trap", Text: "Hi Thom.", Refs: []string{"嗨，湯姆。", "湯姆你好。"}, Names: map[string][]string{"Thom": {"湯姆"}}, Traps: []string{"x"}},
		{ID: "c2", Source: "trap", Text: "Bye.", Refs: []string{"再見。", "掰。"}, Traps: []string{"x"}},
		{ID: "c3", Source: "trap", Text: "The software broke.", Refs: []string{"軟體壞了。", "程式壞了。"}, Forbid: []string{"軟件"}, Traps: []string{"x"}},
		{ID: "c4", Source: "trap", Text: "Run!", Refs: []string{"跑！", "快跑！"}, Traps: []string{"x"}},
	}
}

// judgeAll answers every batch with the given score for every id.
func judgeAll(score int) ai.TextCompleter {
	return completerFunc(func(_ context.Context, _, usr string, _ int) (string, error) {
		var items []JudgeItem
		if err := json.Unmarshal([]byte(usr[strings.Index(usr, "["):]), &items); err != nil {
			return "", err
		}
		var out []JudgeScore
		for _, it := range items {
			out = append(out, JudgeScore{ID: it.ID, Score: score})
		}
		b, _ := json.Marshal(out)
		return string(b), nil
	})
}

type completerFunc func(ctx context.Context, sys, usr string, max int) (string, error)

func (f completerFunc) CompleteText(ctx context.Context, sys, usr string, max int) (string, error) {
	return f(ctx, sys, usr, max)
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	return func() time.Time { t = t.Add(30 * time.Second); return t }
}

func TestRun_RuleFailuresScoreZero_JudgeScoresTheRest(t *testing.T) {
	raw := map[string]string{"c1": "嗨，湯姆。", "c2": "Bye.", "c3": "軟件壞了。", "c4": "快跑！"}
	rep, err := Run(context.Background(), Options{
		Cues:      sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) { return raw, nil },
		Finalize:  func(s string) string { return strings.ReplaceAll(s, "軟件", "軟體") },
		Judge:     judgeAll(2),
		ModelID:   "m", PromptVersion: "p", LexiconVersion: "l",
		Spent: func() float64 { return 0.0123 },
		Now:   fixedNow(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// c2 echoed, c3 forbidden term → 0; c1, c4 judged 2.
	if rep.ZeroRate != 0.5 || rep.NaturalRate != 0.5 || rep.RuleFailRate != 0.5 {
		t.Fatalf("rates: %+v", rep)
	}
	if rep.RuleFailsByReason["echoed"] != 1 || rep.RuleFailsByReason[ReasonForbiddenTerm] != 1 {
		t.Fatalf("reasons: %v", rep.RuleFailsByReason)
	}
	if rep.Grade != GradeC || rep.CostUSD != 0.0123 || rep.JudgeModel != DefaultJudgeModel || rep.SampleVersion != SampleVersion {
		t.Fatalf("report: grade=%s cost=%v judge=%s sample=%s", rep.Grade, rep.CostUSD, rep.JudgeModel, rep.SampleVersion)
	}
	if rep.Results[2].Final != "軟體壞了。" || rep.Results[2].Raw != "軟件壞了。" {
		t.Fatalf("finalize must not touch the raw text the rules saw: %+v", rep.Results[2])
	}
	if rep.Minutes <= 0 || rep.JudgeNote == "" || rep.JudgeVersion != JudgeVersion {
		t.Fatalf("stamps: %+v", rep)
	}
}

func TestRun_BudgetStopIsIncompleteNotError(t *testing.T) {
	partial := map[string]string{"c1": "嗨，湯姆。"}
	rep, err := Run(context.Background(), Options{
		Cues: sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) {
			return partial, fmt.Errorf("batch 2: %w", ai.ErrBudgetExceeded)
		},
		Judge: judgeAll(2), Now: fixedNow(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Grade != GradeIncomplete || !strings.Contains(rep.Incomplete, "AI_BUDGET_EXCEEDED") {
		t.Fatalf("want incomplete, got %+v", rep)
	}
	// Judge was never asked once the translator reported a stop.
	for _, r := range rep.Results {
		if r.Judged {
			t.Fatalf("judge ran after budget stop: %+v", r)
		}
	}
}

func TestRun_TranslatorHardFailureIsAnError(t *testing.T) {
	_, err := Run(context.Background(), Options{
		Cues:      sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) { return nil, errors.New("401") },
	})
	if err == nil {
		t.Fatal("want error when nothing was translated")
	}
}

func TestRun_JudgeBudgetStopMidWay(t *testing.T) {
	calls := 0
	judge := completerFunc(func(context.Context, string, string, int) (string, error) {
		calls++
		return "", ai.ErrBudgetExceeded
	})
	rep, err := Run(context.Background(), Options{
		Cues: sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) {
			return map[string]string{"c1": "嗨，湯姆。", "c2": "再見。", "c3": "軟體壞了。", "c4": "快跑！"}, nil
		},
		Judge: judge, Now: fixedNow(),
	})
	if err != nil || rep.Grade != GradeIncomplete || calls != 1 {
		t.Fatalf("err=%v grade=%s calls=%d", err, rep.Grade, calls)
	}
}

func TestRun_MalformedJudgeCountsUnjudged(t *testing.T) {
	rep, err := Run(context.Background(), Options{
		Cues: sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) {
			return map[string]string{"c1": "嗨，湯姆。", "c2": "再見。", "c3": "軟體壞了。", "c4": "快跑！"}, nil
		},
		Judge: completerFunc(func(context.Context, string, string, int) (string, error) { return "nope", nil }),
		Now:   fixedNow(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Unjudged != 4 || rep.ZeroRate != 1 || rep.Grade != GradeC {
		t.Fatalf("unjudged cues must count as zeros, not vanish: %+v", rep)
	}
}

func TestRun_NoJudgeIsRulesOnly(t *testing.T) {
	rep, err := Run(context.Background(), Options{
		Cues: sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) {
			return map[string]string{"c1": "嗨，湯姆。", "c2": "再見。", "c3": "軟體壞了。", "c4": "快跑！"}, nil
		},
		Now: fixedNow(),
	})
	if err != nil || rep.NaturalRate != 0 || rep.ZeroRate != 0 || rep.JudgeModel != "" {
		t.Fatalf("rules-only run: err=%v %+v", err, rep)
	}
}
