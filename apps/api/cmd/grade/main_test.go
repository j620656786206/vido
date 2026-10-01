package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/eval"
)

// fakeModel answers the production translator prompt by echoing a canned
// rendering per cue index, and answers the judge prompt with a fixed score.
// It also records spend on the ctx budget like the real provider does, so the
// budget stop can be exercised without the network.
type fakeModel struct {
	model      string
	judgeScore int
	usdPerCall float64
	calls      *int
}

var idxLine = regexp.MustCompile(`(?m)^\[(\d+)\] `)

func (f fakeModel) CompleteText(ctx context.Context, sys, usr string, _ int) (string, error) {
	*f.calls++
	if b := ai.BudgetFromContext(ctx); b != nil {
		if b.Exceeded() {
			return "", ai.ErrBudgetExceeded
		}
		// 1M output tokens at the model's rate = price; fake a call that costs usdPerCall.
		b.RecordLLM(f.model, 0, int64(f.usdPerCall/ai.PricingFor(f.model).OutputPer1M.InexactFloat64()*1_000_000))
	}
	if strings.Contains(sys, "資深審稿人") { // judge rubric
		start := strings.Index(usr, "[")
		var items []eval.JudgeItem
		if err := json.Unmarshal([]byte(usr[start:]), &items); err != nil {
			return "", err
		}
		var out []eval.JudgeScore
		for _, it := range items {
			out = append(out, eval.JudgeScore{ID: it.ID, Score: f.judgeScore})
		}
		b, _ := json.Marshal(out)
		return string(b), nil
	}
	// Translator: answer only the "[N] text" lines under the translate header,
	// never the context section, in the production "[N] 譯文" shape.
	body := usr
	if i := strings.Index(usr, "## Translate the following blocks:"); i >= 0 {
		body = usr[i:]
	}
	var sb strings.Builder
	for _, m := range idxLine.FindAllStringSubmatch(body, -1) {
		sb.WriteString("[" + m[1] + "] 這是第 " + m[1] + " 句的中文\n")
	}
	if sb.Len() == 0 {
		return "", io.ErrUnexpectedEOF
	}
	return sb.String(), nil
}

func testDeps(t *testing.T, model fakeModel) deps {
	t.Helper()
	return deps{
		newCompleter: func(_, m string) ai.TextCompleter { model.model = m; return model },
		newFinalizer: func(bool) eval.Finalizer { return func(s string) string { return s } },
		apiKey:       "test-key",
		now:          func() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) },
		stderr:       io.Discard,
	}
}

func TestParseFlags(t *testing.T) {
	if _, err := parseFlags(nil); err == nil {
		t.Fatal("--model is required")
	}
	if _, err := parseFlags([]string{"--model", "gpt-9"}); err == nil {
		t.Fatal("unknown model must be refused at the flag")
	}
	if _, err := parseFlags([]string{"--model", "gemini-2.5-flash"}); err == nil || !strings.Contains(err.Error(), "Claude") {
		t.Fatalf("gemini must be refused with the dispatch reason, got %v", err)
	}
	if _, err := parseFlags([]string{"--model", "claude-haiku-4-5", "--level", "spicy"}); err == nil {
		t.Fatal("bad level must be refused")
	}
	cfg, err := parseFlags([]string{"--model", "claude-haiku-4-5", "--limit", "20"})
	if err != nil || cfg.budgetUSD != 0.50 || cfg.judgeModel != eval.DefaultJudgeModel || cfg.limit != 20 {
		t.Fatalf("defaults: %+v err=%v", cfg, err)
	}
}

func TestRun_EndToEndWithFakeProvider(t *testing.T) {
	calls := 0
	d := testDeps(t, fakeModel{judgeScore: 2, usdPerCall: 0.001, calls: &calls})
	cfg, _ := parseFlags([]string{"--model", "claude-haiku-4-5", "--limit", "25", "--trace"})
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Cues != 25 || rep.ModelID != "claude-haiku-4-5" || rep.SampleVersion != eval.SampleVersion {
		t.Fatalf("report header: %+v", rep)
	}
	if !strings.HasPrefix(rep.PromptVersion, "m1-") || !strings.Contains(rep.PromptVersion, rep.LexiconVersion) || !strings.HasSuffix(rep.PromptVersion, "+standard") {
		t.Fatalf("version triple not stamped from the real prompt stack: %q / %q", rep.PromptVersion, rep.LexiconVersion)
	}
	if rep.CostUSD <= 0 {
		t.Fatalf("budget spend must surface as cost_usd, got %v", rep.CostUSD)
	}
	// Our canned rendering drops every name → the Thom/Celia cues fail name_mismatch; the rest are judged 2.
	if rep.RuleFailsByReason[eval.ReasonNameMismatch] == 0 || rep.NaturalRate == 0 {
		t.Fatalf("rules and judge both have to run: %+v", rep.RuleFailsByReason)
	}
	if len(rep.Results) != 25 {
		t.Fatalf("--trace keeps per-cue results, got %d", len(rep.Results))
	}
}

func TestRun_BudgetCeilingStopsAndReportsIncomplete(t *testing.T) {
	calls := 0
	d := testDeps(t, fakeModel{judgeScore: 2, usdPerCall: 0.30, calls: &calls})
	cfg, _ := parseFlags([]string{"--model", "claude-sonnet-5", "--limit", "40", "--budget-usd", "0.50"})
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Grade != eval.GradeIncomplete || rep.Incomplete == "" {
		t.Fatalf("want incomplete after the ceiling, got grade=%s inc=%q", rep.Grade, rep.Incomplete)
	}
	if rep.CostUSD < 0.50 {
		t.Fatalf("spend should have reached the ceiling, got %v", rep.CostUSD)
	}
}

func TestRun_NoJudgeAndCustomSample(t *testing.T) {
	calls := 0
	d := testDeps(t, fakeModel{judgeScore: 2, usdPerCall: 0.001, calls: &calls})
	tmp := t.TempDir() + "/s.jsonl"
	sample := `{"id":"a","source":"trap","text":"Hello there.","refs":["你好。","哈囉。"],"names":{},"traps":["slang"],"attribution":"t"}` + "\n"
	if err := writeFile(tmp, sample); err != nil {
		t.Fatal(err)
	}
	cfg, _ := parseFlags([]string{"--model", "claude-haiku-4-5", "--sample", tmp, "--no-judge"})
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Cues != 1 || rep.JudgeModel != "" || calls != 1 {
		t.Fatalf("rules-only run on a custom sample: %+v calls=%d", rep, calls)
	}
	if _, err := run(context.Background(), cfg, deps{apiKey: ""}); err == nil {
		t.Fatal("missing key must fail before any call")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
