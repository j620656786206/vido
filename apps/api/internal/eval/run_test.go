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

// TestRun_AlsoJudge scores cues whose failures are all in the set on their
// finalized text for the secondary rates, without moving the headline grade.
func TestRun_AlsoJudge(t *testing.T) {
	raw := map[string]string{"c1": "嗨，Thom。", "c2": "再见。", "c3": "軟件壞了。", "c4": "快跑！"}
	opts := Options{
		Cues:      sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) { return raw, nil },
		Finalize:  func(s string) string { return strings.ReplaceAll(s, "见", "見") },
		Judge:     judgeAll(2),
		ModelID:   "m", PromptVersion: "p", LexiconVersion: "l",
		Now: fixedNow(),
	}
	rep, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ZeroRateSecondary != nil || rep.Results[1].SecondaryJudged {
		t.Fatal("off by default")
	}
	opts.AlsoJudge = []string{"simplified_leak", ReasonNameMismatch}
	rep, err = Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{0, 1} { // c1 name_mismatch, c2 simplified_leak
		r := rep.Results[i]
		if r.Score != 0 || !r.SecondaryJudged || r.SecondaryScore != 2 {
			t.Fatalf("cue %s: headline 0, second pass 2; got %+v", r.ID, r)
		}
	}
	if rep.Results[2].SecondaryJudged {
		t.Fatal("forbidden_term is not in the set, so c3 stays gated")
	}
	// Headline zeros: c1, c2, c3 (3/4). Secondary: only c3 (1/4); naturals 1/4 → 3/4.
	if rep.ZeroRate != 0.75 || *rep.ZeroRateSecondary != 0.25 || *rep.NaturalRateSecondary != 0.75 {
		t.Fatalf("rates: zero=%v secondaryZero=%v secondaryNatural=%v", rep.ZeroRate, *rep.ZeroRateSecondary, *rep.NaturalRateSecondary)
	}
	if rep.Grade != GradeFor(rep.ZeroRate, rep.NaturalRate) {
		t.Fatal("the headline grade comes from the gated rates")
	}
}

func TestRun_AlsoJudge_FailedSecondBatchIsCounted(t *testing.T) {
	raw := map[string]string{"c1": "嗨，湯姆。", "c2": "再见。", "c3": "軟體壞了。", "c4": "快跑！"}
	good := judgeAll(2)
	judge := completerFunc(func(ctx context.Context, sys, usr string, max int) (string, error) {
		if strings.Contains(usr, "再見") { // only the second-pass batch carries c2's finalized text
			return "not json", nil
		}
		return good.CompleteText(ctx, sys, usr, max)
	})
	rep, err := Run(context.Background(), Options{
		Cues:      sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) { return raw, nil },
		Finalize:  func(s string) string { return strings.ReplaceAll(s, "见", "見") },
		Judge:     judge, AlsoJudge: []string{"simplified_leak"},
		ModelID: "m", PromptVersion: "p", LexiconVersion: "l", Now: fixedNow(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.SecondaryUnjudged != 1 || rep.Results[1].SecondaryJudged || !strings.HasPrefix(rep.Results[1].Why, "second_pass_judge_failed") {
		t.Fatalf("failed second pass must be counted and marked: %+v / %+v", rep, rep.Results[1])
	}
	if rep.Grade == GradeIncomplete {
		t.Fatal("a malformed second-pass reply is not a stop")
	}
}

func TestRejudgeSecondary_RepairsFailedBatches(t *testing.T) {
	raw := map[string]string{"c1": "嗨，Thom。", "c2": "再见。", "c3": "軟體壞了。", "c4": "快跑！"}
	broken := true
	good := judgeAll(2)
	judge := completerFunc(func(ctx context.Context, sys, usr string, max int) (string, error) {
		if broken && (strings.Contains(usr, "再見") || strings.Contains(usr, "Thom。")) {
			return "not json", nil
		}
		return good.CompleteText(ctx, sys, usr, max)
	})
	rep, err := Run(context.Background(), Options{
		Cues:      sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) { return raw, nil },
		Finalize:  func(s string) string { return strings.ReplaceAll(s, "见", "見") },
		Judge:     judge, AlsoJudge: []string{"simplified_leak", ReasonNameMismatch},
		ModelID: "m", PromptVersion: "p", LexiconVersion: "l", Now: fixedNow(),
	})
	if err != nil || rep.SecondaryUnjudged != 2 {
		t.Fatalf("setup: unjudged=%d err=%v", rep.SecondaryUnjudged, err)
	}
	headlineZero, headlineGrade := rep.ZeroRate, rep.Grade
	broken = false
	n, err := RejudgeSecondary(context.Background(), judge, &rep, sampleCues(), 1)
	if err != nil || n != 2 || rep.SecondaryUnjudged != 0 {
		t.Fatalf("repair: n=%d unjudged=%d err=%v", n, rep.SecondaryUnjudged, err)
	}
	if *rep.ZeroRateSecondary != 0 || *rep.NaturalRateSecondary != 1 {
		t.Fatalf("secondary after repair: %v / %v", *rep.ZeroRateSecondary, *rep.NaturalRateSecondary)
	}
	if rep.ZeroRate != headlineZero || rep.Grade != headlineGrade {
		t.Fatal("repair must not touch the headline")
	}
	if _, err := RejudgeSecondary(context.Background(), judge, &Report{}, sampleCues(), 1); err == nil {
		t.Fatal("a report without results cannot be repaired")
	}
}

// A budget stop in the second pass must not cost the headline grade.
func TestRun_AlsoJudge_BudgetStopKeepsHeadline(t *testing.T) {
	raw := map[string]string{"c1": "嗨，Thom。", "c2": "再见。", "c3": "軟體壞了。", "c4": "快跑！"}
	good := judgeAll(2)
	judge := completerFunc(func(ctx context.Context, sys, usr string, max int) (string, error) {
		if strings.Contains(usr, "再見") || strings.Contains(usr, "Thom。") {
			return "", ai.ErrBudgetExceeded
		}
		return good.CompleteText(ctx, sys, usr, max)
	})
	rep, err := Run(context.Background(), Options{
		Cues:      sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) { return raw, nil },
		Finalize:  func(s string) string { return strings.ReplaceAll(s, "见", "見") },
		Judge:     judge, AlsoJudge: []string{"simplified_leak", ReasonNameMismatch},
		ModelID: "m", PromptVersion: "p", LexiconVersion: "l", Now: fixedNow(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Grade == GradeIncomplete || rep.Incomplete != "" {
		t.Fatalf("headline must survive a second-pass budget stop: %s / %s", rep.Grade, rep.Incomplete)
	}
	if rep.SecondaryUnjudged != 2 || rep.SecondaryIncomplete == "" {
		t.Fatalf("remaining second-pass cues must be counted: unjudged=%d reason=%q", rep.SecondaryUnjudged, rep.SecondaryIncomplete)
	}
	if rep.TranslateMinutes <= 0 || rep.TranslateMinutes >= rep.Minutes {
		t.Fatalf("translate time is its own slice of the run: %v of %v", rep.TranslateMinutes, rep.Minutes)
	}
}

// A report judged by Gemini carries the Gemini disclosure, not Claude's.
func TestRun_JudgeNoteFollowsTheJudge(t *testing.T) {
	raw := map[string]string{"c1": "嗨，湯姆。", "c2": "再見。", "c3": "軟體壞了。", "c4": "快跑！"}
	opts := Options{
		Cues:      sampleCues(),
		Translate: func(context.Context, []Cue) (map[string]string, error) { return raw, nil },
		Judge:     judgeAll(2),
		ModelID:   "m",
		Now:       fixedNow(),
	}
	opts.JudgeModel = GeminiFallbackJudgeModel
	rep, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.JudgeModel != GeminiFallbackJudgeModel || !strings.HasPrefix(rep.JudgeNote, "裁判為 Gemini（gemini-3.7-flash）") {
		t.Fatalf("judge %q note %q", rep.JudgeModel, rep.JudgeNote)
	}

	opts.JudgeModel = ""
	opts.Now = fixedNow()
	rep, err = Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.JudgeModel != DefaultJudgeModel || rep.JudgeNote != JudgeNote {
		t.Fatalf("default judge %q note %q", rep.JudgeModel, rep.JudgeNote)
	}
}
