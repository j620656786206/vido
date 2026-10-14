package main

import (
	"context"
	"encoding/json"
	"errors"
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

// fakeLocal stands in for a model on an OpenAI-compatible server: it answers
// the production "[N]" prompt and the native one-cue prompts, never touches the
// budget (local inference is free), and records what it was sent.
type fakeLocal struct {
	prompts *[]string
	failIDs map[string]bool // native: source texts to fail on
}

func (f fakeLocal) CompleteText(_ context.Context, _, usr string, _ int) (string, error) {
	*f.prompts = append(*f.prompts, usr)
	if strings.Contains(usr, "## Translate the following blocks:") {
		body := usr[strings.Index(usr, "## Translate the following blocks:"):]
		var sb strings.Builder
		for _, m := range idxLine.FindAllStringSubmatch(body, -1) {
			sb.WriteString("[" + m[1] + "] 本機第 " + m[1] + " 句\n")
		}
		return sb.String(), nil
	}
	for text := range f.failIDs {
		if strings.HasSuffix(usr, text) {
			return "", ai.ErrAIInvalidResponse
		}
	}
	return "  本機譯文  ", nil
}

func localDeps(t *testing.T, local fakeLocal, judgeCalls *int) deps {
	t.Helper()
	d := testDeps(t, fakeModel{judgeScore: 2, usdPerCall: 0.001, calls: judgeCalls})
	d.newLocalCompleter = func(_, _ string, _, _ time.Duration) ai.TextCompleter { return local }
	return d
}

func TestParseFlags_LocalMode(t *testing.T) {
	base := []string{"--openai-base-url", "http://nas:8080/v1", "--openai-model", "hy-mt2-7b"}
	cfg, err := parseFlags(base)
	if err != nil || cfg.model != "local:hy-mt2-7b" || cfg.promptStyle != promptStyleProduction || cfg.callTimeout != ai.DefaultOpenAICompatTimeout {
		t.Fatalf("local defaults: %+v err=%v", cfg, err)
	}
	if cfg, err := parseFlags(append(base, "--also-judge", " simplified_leak , name_mismatch ")); err != nil || len(splitList(cfg.alsoJudge)) != 2 {
		t.Fatalf("--also-judge list: %+v %v", cfg, err)
	}
	cfg, err = parseFlags(append(base, "--prompt-style", "native"))
	if err != nil || cfg.nativeTemplate != nativeHyMT {
		t.Fatalf("native template must be guessed from the model id: %+v err=%v", cfg, err)
	}
	for name, args := range map[string][]string{
		"url without model":     {"--openai-base-url", "http://nas/v1"},
		"model without url":     {"--openai-model", "x"},
		"also --model":          append([]string{"--model", "claude-haiku-4-5"}, base...),
		"native for claude":     {"--model", "claude-haiku-4-5", "--prompt-style", "native"},
		"unknown style":         append([]string{"--prompt-style", "fancy"}, base...),
		"unguessable template":  {"--openai-base-url", "http://nas/v1", "--openai-model", "qwen3:4b", "--prompt-style", "native"},
		"batch size in native":  append([]string{"--prompt-style", "native", "--batch-size", "5"}, base...),
		"negative batch size":   append([]string{"--batch-size", "-1"}, base...),
		"batch size for claude": {"--model", "claude-haiku-4-5", "--batch-size", "5"},
		"bad explicit template": append([]string{"--prompt-style", "native", "--native-template", "gpt"}, base...),
		"also-judge empty cues": append([]string{"--also-judge", "simplified_leak,missing"}, base...),
		"also-judge no judge":   append([]string{"--also-judge", "simplified_leak", "--no-judge"}, base...),
	} {
		if _, err := parseFlags(args); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestRun_LocalProductionStyle(t *testing.T) {
	var sent []string
	judgeCalls := 0
	d := localDeps(t, fakeLocal{prompts: &sent}, &judgeCalls)
	cfg, err := parseFlags([]string{"--openai-base-url", "http://nas:8080/v1", "--openai-model", "translategemma:12b", "--limit", "25", "--batch-size", "5"})
	if err != nil {
		t.Fatal(err)
	}
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ModelID != "local:translategemma:12b" || rep.PromptStyle != promptStyleProduction || rep.BatchSize != 5 || rep.Endpoint != "http://nas:8080/v1" {
		t.Fatalf("local stamps: %+v", rep)
	}
	if len(sent) != 5 { // 25 cues in chunks of 20 → batches of 5,5,5,5 and 5
		t.Fatalf("--batch-size 5 over 25 cues should make 5 requests, got %d", len(sent))
	}
	if !strings.HasPrefix(rep.PromptVersion, "m1-") {
		t.Fatalf("production style keeps the production prompt version, got %q", rep.PromptVersion)
	}
	if judgeCalls == 0 || rep.JudgeModel != eval.DefaultJudgeModel || !strings.Contains(rep.JudgeNote, "本機模型") {
		t.Fatalf("Claude still judges, and the note says so: calls=%d %+v", judgeCalls, rep)
	}
	if rep.EpisodeMinutesEst != eval.EpisodeMinutes(rep.Minutes, rep.Cues) {
		t.Fatalf("episode estimate: %v", rep.EpisodeMinutesEst)
	}
}

func TestRun_LocalNativeStyleSkipsFailedCuesAndNeedsNoKeyWithoutJudge(t *testing.T) {
	cues, err := eval.Golden()
	if err != nil {
		t.Fatal(err)
	}
	var sent []string
	judgeCalls := 0
	d := localDeps(t, fakeLocal{prompts: &sent, failIDs: map[string]bool{cues[2].Text: true}}, &judgeCalls)
	d.apiKey = "" // rules only + local model: no Claude key needed
	cfg, err := parseFlags([]string{"--openai-base-url", "http://nas:8080/v1", "--openai-model", "hy-mt2-7b",
		"--prompt-style", "native", "--limit", "4", "--no-judge", "--trace", "--no-glossary"})
	if err != nil {
		t.Fatal(err)
	}
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(sent) != 4 || judgeCalls != 0 {
		t.Fatalf("one call per cue and no judge: sent=%d judge=%d", len(sent), judgeCalls)
	}
	if !strings.HasPrefix(sent[0], "Translate the following text into Traditional Chinese.") ||
		!strings.Contains(sent[1], "[Background Information]\n"+cues[0].Text) {
		t.Fatalf("HY-MT templates: first=%q second=%q", sent[0], sent[1])
	}
	if rep.PromptVersion != "native-hymt" || rep.Results[0].Raw != "本機譯文" {
		t.Fatalf("native stamps / trimmed output: %q %q", rep.PromptVersion, rep.Results[0].Raw)
	}
	if rep.Results[2].Raw != "" || rep.Results[2].Score != 0 {
		t.Fatalf("a failed cue is left missing and scores 0: %+v", rep.Results[2])
	}
	if rep.Grade == eval.GradeIncomplete {
		t.Fatalf("one failed cue must not void the run: %s", rep.Incomplete)
	}
}

func TestNativeTranslator_StopsAfterConsecutiveFailures(t *testing.T) {
	cues, _ := eval.Golden()
	fail := map[string]bool{}
	for _, c := range cues[:nativeMaxConsecutiveFailures] {
		fail[c.Text] = true
	}
	var sent []string
	got, err := nativeTranslator(fakeLocal{prompts: &sent, failIDs: fail}, nativeTranslateGemma, nil, io.Discard)(context.Background(), cues[:10])
	if err == nil || len(sent) != nativeMaxConsecutiveFailures || len(got) != 0 {
		t.Fatalf("want a stop after %d failures, got err=%v sent=%d got=%d", nativeMaxConsecutiveFailures, err, len(sent), len(got))
	}
}

func TestNativePrompt_TranslateGemmaWording(t *testing.T) {
	p, err := nativePrompt(nativeTranslateGemma, "Hello.", []string{"ignored"}, nil)
	if err != nil || !strings.HasPrefix(p, "You are a professional English (en) to Traditional Chinese (zh-TW) translator.") ||
		!strings.HasSuffix(p, "into Traditional Chinese:\n\n\nHello.") || strings.Contains(p, "ignored") {
		t.Fatalf("translategemma prompt: %q %v", p, err)
	}
}

func TestMerge_RefusesLocalModels(t *testing.T) {
	_, err := ratingFromReport(eval.Report{ModelID: "local:hy-mt2-7b", Grade: eval.GradeA, SampleVersion: eval.SampleVersion})
	if err == nil || !strings.Contains(err.Error(), "local model") {
		t.Fatalf("local reports must not reach the ratings table: %v", err)
	}
}

func TestRejudge_RewritesReportAndKeepsHeadline(t *testing.T) {
	calls := 0
	d := testDeps(t, fakeModel{judgeScore: 2, usdPerCall: 0.001, calls: &calls})
	tmp := t.TempDir() + "/r.json"
	cues, _ := eval.Golden()
	rep := eval.Report{
		ModelID: "local:x", Grade: eval.GradeC, ZeroRate: 1, JudgeModel: eval.DefaultJudgeModel, SampleVersion: eval.SampleVersion,
		SecondaryReasons: []string{eval.ReasonNameMismatch}, SecondaryUnjudged: 1,
		Results: []eval.CueResult{{ID: cues[0].ID, Raw: "托姆", Final: "托姆", RuleFails: []string{eval.ReasonNameMismatch}}},
	}
	if err := writeReport(tmp, rep); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseFlags([]string{"--rejudge", tmp})
	if err != nil {
		t.Fatal(err)
	}
	n, err := rejudge(context.Background(), cfg, d)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	data, _ := os.ReadFile(tmp)
	var got eval.Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.SecondaryUnjudged != 0 || !got.Results[0].SecondaryJudged || got.ZeroRate != 1 || got.Grade != eval.GradeC || got.CostUSD <= 0 {
		t.Fatalf("rewritten report: %+v", got)
	}
}

func TestRejudge_RefusesAnotherSampleVersion(t *testing.T) {
	calls := 0
	d := testDeps(t, fakeModel{judgeScore: 2, usdPerCall: 0.001, calls: &calls})
	tmp := t.TempDir() + "/r.json"
	if err := writeReport(tmp, eval.Report{SampleVersion: "golden-v0", SecondaryReasons: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := parseFlags([]string{"--rejudge", tmp})
	if _, err := rejudge(context.Background(), cfg, d); err == nil || !strings.Contains(err.Error(), "golden-v0") {
		t.Fatalf("want a sample-version refusal, got %v", err)
	}
}

func TestRejudge_BudgetStopSavesAndFails(t *testing.T) {
	calls := 0
	d := testDeps(t, fakeModel{judgeScore: 2, usdPerCall: 0.30, calls: &calls})
	tmp := t.TempDir() + "/r.json"
	cues, _ := eval.Golden()
	rep := eval.Report{ModelID: "local:x", Grade: eval.GradeC, JudgeModel: eval.DefaultJudgeModel, SampleVersion: eval.SampleVersion,
		SecondaryReasons: []string{eval.ReasonNameMismatch}}
	for _, c := range cues[:12] {
		rep.Results = append(rep.Results, eval.CueResult{ID: c.ID, Raw: "托姆", Final: "托姆", RuleFails: []string{eval.ReasonNameMismatch}})
	}
	if err := writeReport(tmp, rep); err != nil {
		t.Fatal(err)
	}
	cfg, _ := parseFlags([]string{"--rejudge", tmp, "--budget-usd", "0.50"})
	n, err := rejudge(context.Background(), cfg, d)
	if err == nil || !errors.Is(err, ai.ErrBudgetExceeded) {
		t.Fatalf("a budget stop must fail the command, got n=%d err=%v", n, err)
	}
	data, _ := os.ReadFile(tmp)
	var got eval.Report
	_ = json.Unmarshal(data, &got)
	judged := 0
	for _, r := range got.Results {
		if r.SecondaryJudged {
			judged++
		}
	}
	if judged != n || got.SecondaryUnjudged != 12-n || got.ZeroRateSecondary == nil {
		t.Fatalf("saved report must be consistent: judged=%d n=%d unjudged=%d", judged, n, got.SecondaryUnjudged)
	}
}

// recordingModel is the production-prompt fake that also keeps every
// translator prompt (system + user) it was sent, so a test can see what the
// translator was told.
type recordingModel struct {
	fakeModel
	systems *[]string
}

func (r recordingModel) CompleteText(ctx context.Context, sys, usr string, max int) (string, error) {
	if !strings.Contains(sys, "資深審稿人") {
		*r.systems = append(*r.systems, sys+"\n"+usr)
	}
	return r.fakeModel.CompleteText(ctx, sys, usr, max)
}

func TestRun_GoldenV2SendsTheNameGlossary(t *testing.T) {
	var systems []string
	calls := 0
	d := testDeps(t, fakeModel{})
	m := recordingModel{fakeModel: fakeModel{judgeScore: 2, usdPerCall: 0.001, calls: &calls}, systems: &systems}
	d.newCompleter = func(_, model string) ai.TextCompleter { m.model = model; return m }

	cfg, _ := parseFlags([]string{"--model", "claude-haiku-4-5", "--limit", "20", "--no-judge"})
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.SampleVersion != "golden-v2" || rep.GlossaryVersion == "" {
		t.Fatalf("v2 stamps: sample=%q glossary=%q", rep.SampleVersion, rep.GlossaryVersion)
	}
	joined := strings.Join(systems, "\n")
	for _, want := range []string{"## Glossary — MANDATORY fixed renderings:", "- Thom → 湯姆", "- Celia → 希莉亞"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("translator prompt is missing %q", want)
		}
	}
	if strings.Contains(joined, "Sita →") {
		t.Fatal("the first 20 cues are all tos: a batch only carries its own source's names")
	}

	systems = nil
	cfg, _ = parseFlags([]string{"--model", "claude-haiku-4-5", "--limit", "20", "--no-judge", "--no-glossary"})
	rep, err = run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.SampleVersion != eval.SampleVersionNoGlossary || rep.GlossaryVersion != "" {
		t.Fatalf("--no-glossary stamps: %q / %q", rep.SampleVersion, rep.GlossaryVersion)
	}
	if strings.Contains(strings.Join(systems, "\n"), "## Glossary — MANDATORY fixed renderings:") {
		t.Fatal("--no-glossary must send the v1 prompt, with no glossary section")
	}
}

func TestRun_NativeHyMTUsesTheTerminologyTemplate(t *testing.T) {
	var sent []string
	judgeCalls := 0
	d := localDeps(t, fakeLocal{prompts: &sent}, &judgeCalls)
	d.apiKey = ""
	cfg, err := parseFlags([]string{"--openai-base-url", "http://nas/v1", "--openai-model", "hy-mt2-7b",
		"--prompt-style", "native", "--limit", "3", "--no-judge"})
	if err != nil {
		t.Fatal(err)
	}
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	// tos-01 "You're a jerk, Thom." / tos-02 "Look Celia, …" / tos-03 has no name.
	if !strings.HasPrefix(sent[0], "Reference the following translations:\nThom translates to 湯姆\n\n") || strings.Contains(sent[0], "Celia") {
		t.Fatalf("cue 1 lists only its own names: %q", sent[0])
	}
	if !strings.Contains(sent[1], "Celia translates to 希莉亞") || strings.Contains(sent[1], "Thom translates") {
		t.Fatalf("cue 2 lists only its own name, Celia (Thom is only in its context): %q", sent[1])
	}
	if !strings.HasPrefix(sent[2], "[Background Information]\n") || strings.Contains(sent[2], "translates to") {
		t.Fatalf("a nameless cue keeps its context instead of a glossary: %q", sent[2])
	}
	if rep.SampleVersion != eval.SampleVersion || rep.GlossaryVersion == "" {
		t.Fatalf("hymt native sends the glossary, so it is the v2 exam: %+v", rep)
	}
}

func TestRun_NativeTranslateGemmaIsTheNoGlossaryExam(t *testing.T) {
	var sent []string
	judgeCalls := 0
	d := localDeps(t, fakeLocal{prompts: &sent}, &judgeCalls)
	cfg, _ := parseFlags([]string{"--openai-base-url", "http://nas/v1", "--openai-model", "translategemma:12b",
		"--prompt-style", "native", "--limit", "2"})
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sent[0], "translates to") || rep.SampleVersion != eval.SampleVersionNoGlossary || rep.GlossaryVersion != "" {
		t.Fatalf("TranslateGemma's template has no glossary slot, so it must be stamped v1: %+v", rep)
	}
	if !strings.Contains(rep.JudgeNote, "TranslateGemma 的官方格式沒有放對照表的位置") {
		t.Fatalf("the report must say the template could not carry the glossary: %q", rep.JudgeNote)
	}
}

func TestMerge_RefusesTheNoGlossaryExam(t *testing.T) {
	_, err := ratingFromReport(eval.Report{ModelID: "claude-haiku-4-5", Grade: eval.GradeA, SampleVersion: eval.SampleVersionNoGlossary})
	if err == nil || !strings.Contains(err.Error(), eval.SampleVersionNoGlossary) {
		t.Fatalf("a --no-glossary report must not reach the ratings table: %v", err)
	}
}

func TestRun_CustomSampleWithoutNamesIsTheNoGlossaryExam(t *testing.T) {
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
	if rep.SampleVersion != eval.SampleVersionNoGlossary || rep.GlossaryVersion != "" {
		t.Fatalf("an empty glossary sends nothing, so it is not the v2 exam: %q / %q", rep.SampleVersion, rep.GlossaryVersion)
	}
}
