// Command grade scores one translation model on Vido's golden sample
// (story sub-7-8a) and prints the report sub-7-8b merges into the model
// ratings table.
//
// It runs the REAL translation stack — services.TranslationService with the
// production prompt, localization level and zh-TW lexicon — in-process, the
// way cmd/route-c-poc does, so the grade reflects what the pipeline ships and
// the version triple (prompt / lexicon / model) is stamped automatically. No
// server, no library item, no TMDb match is needed: CI can run it with one key.
//
//	CLAUDE_API_KEY=… go run ./cmd/grade --model claude-haiku-4-5 --out haiku.json
//	go run ./cmd/grade --merge haiku.json            # fold it into internal/ai/model_ratings.json
//
// A model on an OpenAI-compatible server (Ollama, llama-server) is graded with
// --openai-base-url + --openai-model (spike-local-translation-golden-eval); the
// judge is still Claude, and the report is stamped "local:<model>" so --merge
// refuses it — local models never enter the built-in ratings table:
//
//	CLAUDE_API_KEY=… go run ./cmd/grade --openai-base-url http://nas:8080/v1 \
//	    --openai-model hy-mt2-7b --prompt-style native --trace --out hy.json
//
// One ai.Budget covers both the model under test and the judge; hitting it
// stops the run and the report says "incomplete" instead of inventing a grade.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/eval"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/zhtw"
)

// translateChunk is how many cues go to the translator per call (2 batches of 10).
const translateChunk = 20

// config is everything the flags decide.
type config struct {
	model      string
	samplePath string
	limit      int
	budgetUSD  float64
	judgeModel string
	out        string
	level      string
	noJudge    bool
	noOpenCC   bool
	withTrace  bool
	// local-model mode (spike-local-translation-golden-eval): set together.
	openaiBaseURL  string
	openaiModel    string
	callTimeout    time.Duration
	pace           time.Duration
	promptStyle    string
	batchSize      int
	nativeTemplate string
	alsoJudge      string
	noGlossary     bool
	// rejudge mode: repair a traced report's failed second-pass judge batches.
	rejudgePath string
	// merge mode: fold a report into model_ratings.json instead of grading.
	mergePath   string
	ratingsPath string
}

// deps are the seams the tests replace: how a provider is built and how the
// finalizer (OpenCC) is obtained.
type deps struct {
	newCompleter func(apiKey, model string) ai.TextCompleter
	// newLocalCompleter builds the model under test for --openai-base-url.
	newLocalCompleter func(baseURL, model string, timeout, pace time.Duration) ai.TextCompleter
	newFinalizer      func(noOpenCC bool) eval.Finalizer
	apiKey            string
	now               func() time.Time
	stderr            io.Writer
}

func main() {
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if cfg.mergePath != "" {
		row, err := mergeReport(cfg.mergePath, cfg.ratingsPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "grade --merge:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "merged %s → %s into %s\n", row.ModelID, row.Grade, cfg.ratingsPath)
		return
	}
	d := deps{
		newCompleter: func(key, model string) ai.TextCompleter {
			return ai.NewClaudeProvider(key, ai.WithClaudeModel(model))
		},
		newLocalCompleter: func(baseURL, model string, timeout, pace time.Duration) ai.TextCompleter {
			return ai.NewOpenAICompatProvider(baseURL, model,
				ai.WithOpenAICompatAPIKey(os.Getenv("OPENAI_COMPAT_API_KEY")),
				ai.WithOpenAICompatTimeout(timeout),
				ai.WithOpenAICompatMinInterval(pace))
		},
		newFinalizer: newFinalizer,
		// Same variable the app reads (docs/development.md § Configuration), so
		// one key serves the NAS, local runs and the Model Grade workflow.
		apiKey: firstNonEmpty(os.Getenv("CLAUDE_API_KEY"), os.Getenv("ANTHROPIC_API_KEY")),
		now:    time.Now,
		stderr: os.Stderr,
	}
	if cfg.rejudgePath != "" {
		n, err := rejudge(context.Background(), cfg, d)
		if err != nil {
			fmt.Fprintln(os.Stderr, "grade --rejudge:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "rejudged %d cues in %s\n", n, cfg.rejudgePath)
		return
	}
	rep, err := run(context.Background(), cfg, d)
	if err != nil {
		fmt.Fprintln(os.Stderr, "grade:", err)
		os.Exit(1)
	}
	if err := writeReport(cfg.out, rep); err != nil {
		fmt.Fprintln(os.Stderr, "grade:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%s → %s (0 分率 %.1f%%, 2 分率 %.1f%%, $%.4f, %.1f min)\n",
		rep.ModelID, rep.Grade, rep.ZeroRate*100, rep.NaturalRate*100, rep.CostUSD, rep.Minutes)
	if rep.Grade == eval.GradeIncomplete {
		os.Exit(3)
	}
}

func parseFlags(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("grade", flag.ContinueOnError)
	fs.StringVar(&cfg.model, "model", "", "model id to grade (one of ai.Catalog(), e.g. claude-haiku-4-5)")
	fs.StringVar(&cfg.samplePath, "sample", "", "golden sample JSONL (default: the embedded sample)")
	fs.BoolVar(&cfg.noGlossary, "no-glossary", false, "do not hand the translator the sample's name glossary (the golden-v1 exam; never mergeable)")
	fs.IntVar(&cfg.limit, "limit", 0, "grade only the first N cues (0 = all)")
	fs.Float64Var(&cfg.budgetUSD, "budget-usd", 0.50, "hard USD ceiling for translation + judge together")
	fs.StringVar(&cfg.judgeModel, "judge-model", eval.DefaultJudgeModel, "fixed judge model")
	fs.StringVar(&cfg.out, "out", "", "write the report JSON here (default: stdout)")
	fs.StringVar(&cfg.level, "level", string(prompts.DefaultLocalizationLevel), "localization level: literal | standard | ott")
	fs.BoolVar(&cfg.noJudge, "no-judge", false, "rules only, skip the AI judge (every clean cue scores 1)")
	fs.BoolVar(&cfg.noOpenCC, "no-opencc", false, "do not run OpenCC s2twp before judging")
	fs.BoolVar(&cfg.withTrace, "trace", false, "include per-cue raw/final/score in the report")
	fs.StringVar(&cfg.mergePath, "merge", "", "merge this report JSON into the ratings table and exit (no grading)")
	fs.StringVar(&cfg.ratingsPath, "ratings", defaultRatingsPath, "ratings table to update with --merge")
	fs.StringVar(&cfg.rejudgePath, "rejudge", "", "re-run the failed second-pass judge batches of this traced report (5 cues per call) and rewrite it in place; the headline grade is untouched")
	fs.StringVar(&cfg.openaiBaseURL, "openai-base-url", "", "grade a model behind an OpenAI-compatible endpoint (Ollama, llama-server), e.g. http://nas:11434/v1")
	fs.StringVar(&cfg.openaiModel, "openai-model", "", "model id that endpoint serves (with --openai-base-url)")
	fs.DurationVar(&cfg.callTimeout, "call-timeout", ai.DefaultOpenAICompatTimeout, "per-request deadline for --openai-base-url (CPU inference is slow)")
	fs.DurationVar(&cfg.pace, "pace", 0, "minimum gap between calls to the model under test (a free tier's requests-per-minute limit; e.g. 13s for 5 RPM)")
	fs.StringVar(&cfg.promptStyle, "prompt-style", promptStyleProduction, "production (Vido's translator prompt) | native (the model's own template, one cue per call; needs --openai-base-url)")
	fs.IntVar(&cfg.batchSize, "batch-size", 0, "cues per request in production style (0 = the app's default)")
	fs.StringVar(&cfg.nativeTemplate, "native-template", "", "translategemma | hymt (default: guessed from --openai-model)")
	fs.StringVar(&cfg.alsoJudge, "also-judge", "", "comma-separated rule reasons (e.g. simplified_leak,name_mismatch) whose cues the judge scores anyway for secondary rates; the headline grade is unchanged")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.mergePath != "" || cfg.rejudgePath != "" {
		return cfg, nil
	}
	if cfg.batchSize < 0 {
		return cfg, errors.New("--batch-size must be 0 (default) or positive")
	}
	switch cfg.promptStyle {
	case promptStyleProduction, promptStyleNative:
	default:
		return cfg, fmt.Errorf("--prompt-style %q: want %s or %s", cfg.promptStyle, promptStyleProduction, promptStyleNative)
	}
	if _, err := prompts.ParseLocalizationLevel(cfg.level); err != nil {
		return cfg, err
	}
	for _, r := range splitList(cfg.alsoJudge) {
		if !secondaryReasons[r] {
			return cfg, fmt.Errorf("--also-judge %q: a judged second pass only makes sense for text that exists — want simplified_leak, name_mismatch, forbidden_term, misaligned or time_shift", r)
		}
	}
	if cfg.alsoJudge != "" && cfg.noJudge {
		return cfg, errors.New("--also-judge needs the judge (drop --no-judge)")
	}
	if cfg.openaiBaseURL != "" || cfg.openaiModel != "" {
		return parseLocalFlags(cfg)
	}
	if cfg.promptStyle == promptStyleNative {
		return cfg, errors.New("--prompt-style native needs --openai-base-url (Claude is only graded through the production prompt)")
	}
	if cfg.batchSize != 0 {
		// A Claude grade can be merged into the shipped ratings table, so it
		// must be measured exactly the way the app translates.
		return cfg, errors.New("--batch-size is only for --openai-base-url runs; Claude is graded at the app's batch size")
	}
	if cfg.model == "" {
		return cfg, errors.New("--model is required")
	}
	if !ai.IsSelectableModel(cfg.model) {
		return cfg, fmt.Errorf("--model %q is not a priced, selectable catalog model", cfg.model)
	}
	if ai.ProviderOf(cfg.model) != ai.ProviderNameClaude {
		// The translation path can only dispatch to Claude today
		// (services/model_catalog.go CR H2; backlog-gemini-translation-dispatch).
		return cfg, fmt.Errorf("--model %q: only Claude models can be graded until translation dispatches per provider", cfg.model)
	}
	return cfg, nil
}

// localModelPrefix stamps a local model's report so it can never collide with
// a catalog id, and so --merge can refuse it.
const localModelPrefix = "local:"

// parseLocalFlags validates the --openai-base-url mode and derives the report
// model id.
func parseLocalFlags(cfg config) (config, error) {
	if cfg.openaiBaseURL == "" || cfg.openaiModel == "" {
		return cfg, errors.New("--openai-base-url and --openai-model go together")
	}
	if cfg.model != "" {
		return cfg, errors.New("--model is for Claude catalog models; with --openai-base-url name the model with --openai-model")
	}
	if cfg.promptStyle == promptStyleNative {
		if cfg.batchSize != 0 {
			return cfg, errors.New("--batch-size only applies to --prompt-style production (native sends one cue per call)")
		}
		if cfg.nativeTemplate == "" {
			cfg.nativeTemplate = guessNativeTemplate(cfg.openaiModel)
		}
		if _, err := nativePrompt(cfg.nativeTemplate, "", nil, nil); err != nil {
			return cfg, fmt.Errorf("--native-template: %w", err)
		}
	}
	cfg.model = localModelPrefix + cfg.openaiModel
	return cfg, nil
}

func run(ctx context.Context, cfg config, d deps) (eval.Report, error) {
	local := cfg.openaiBaseURL != ""
	// The key pays for the judge, and for the model under test unless it is local.
	if d.apiKey == "" && (!local || !cfg.noJudge) {
		return eval.Report{}, errors.New("set CLAUDE_API_KEY (the same variable the app uses; ANTHROPIC_API_KEY also works)")
	}
	cues, err := loadSample(cfg.samplePath)
	if err != nil {
		return eval.Report{}, err
	}
	// The glossary is built from the WHOLE sample, like a show's glossary
	// covers every episode, before --limit trims what gets translated.
	var glossaries map[string][]eval.NameTerm
	if !cfg.noGlossary {
		if glossaries, err = eval.NameGlossaries(cues); err != nil {
			return eval.Report{}, err
		}
	}
	if cfg.limit > 0 && cfg.limit < len(cues) {
		cues = cues[:cfg.limit]
	}
	level, _ := prompts.ParseLocalizationLevel(cfg.level)
	// TranslateGemma's published template has no slot for a glossary, so a
	// native run of it is the no-glossary exam whatever the flag says.
	// A custom --sample with no `names` builds an empty glossary: nothing is
	// sent, so it is the no-glossary exam too.
	noSlot := cfg.promptStyle == promptStyleNative && cfg.nativeTemplate == nativeTranslateGemma
	sendsGlossary := len(eval.AllTerms(glossaries)) > 0 && !noSlot

	budget := ai.NewBudget(cfg.budgetUSD)
	ctx = ai.WithBudget(ctx, budget)

	var underTest ai.TextCompleter
	if local {
		underTest = d.newLocalCompleter(cfg.openaiBaseURL, cfg.openaiModel, cfg.callTimeout, cfg.pace)
	} else {
		ctx = ai.WithModelID(ctx, cfg.model)
		underTest = d.newCompleter(d.apiKey, cfg.model)
	}

	translator := services.NewTranslationService(underTest, nil)
	batchOpt := services.WithBatchSize(cfg.batchSize)
	// Chunked on purpose: TranslateWithGlossary returns NOTHING when the
	// budget sentinel fires mid-track (the segment cache is its resume seam,
	// and this run has none), so a stop in cue 180 would throw away 179 paid
	// translations. Chunks of translateChunk cues keep what was already bought;
	// the only cost is the 5-cue context window resetting at chunk edges.
	translate := func(ctx context.Context, cues []eval.Cue) (map[string]string, error) {
		got := make(map[string]string, len(cues))
		for start := 0; start < len(cues); start += translateChunk {
			end := start + translateChunk
			if end > len(cues) {
				end = len(cues)
			}
			chunk := cues[start:end]
			blocks := make([]services.TranslationBlock, len(chunk))
			for i, c := range chunk {
				blocks[i] = services.TranslationBlock{Index: i + 1, Start: c.Start, End: c.End, Text: c.Text}
			}
			var pairs []services.GlossaryPair
			if sendsGlossary {
				pairs = glossaryPairs(eval.GlossaryFor(glossaries, chunk))
			}
			out, _, err := translator.TranslateWithGlossary(ctx, blocks, pairs, nil, services.WithLocalizationLevel(level), batchOpt)
			if err != nil {
				fmt.Fprintln(d.stderr)
				return got, err
			}
			// A batch the service gave up on keeps its English text; passing it
			// through lets the rule layer call it what the gate would: echoed.
			for _, b := range out {
				if b.Index >= 1 && b.Index <= len(chunk) {
					got[chunk[b.Index-1].ID] = b.Text
				}
			}
			fmt.Fprintf(d.stderr, "\r  translating… %d/%d", end, len(cues))
		}
		fmt.Fprintln(d.stderr)
		return got, nil
	}
	if cfg.promptStyle == promptStyleNative {
		var native map[string][]eval.NameTerm
		if sendsGlossary {
			native = glossaries
		}
		translate = nativeTranslator(underTest, cfg.nativeTemplate, native, d.stderr)
	}

	var judge ai.TextCompleter
	if !cfg.noJudge {
		judge = d.newCompleter(d.apiKey, cfg.judgeModel)
	}
	opts := eval.Options{
		Cues: cues, Translate: translate, Finalize: d.newFinalizer(cfg.noOpenCC),
		Judge: judge, JudgeModel: cfg.judgeModel,
		ModelID: cfg.model, PromptVersion: prompts.PromptVersionFor(level), LexiconVersion: prompts.LexiconVersion(),
		SampleVersion: eval.SampleVersionNoGlossary,
		Spent:         budget.SpentUSD, Now: d.now,
		AlsoJudge: splitList(cfg.alsoJudge),
	}
	if sendsGlossary {
		opts.SampleVersion = eval.SampleVersion
		opts.GlossaryVersion = subtitle.GlossaryVersionHash(promptGlossary(eval.AllTerms(glossaries)))
	}
	if cfg.noJudge {
		opts.JudgeModel = ""
	}
	if cfg.promptStyle == promptStyleNative {
		// The production prompt version does not describe what was sent.
		opts.PromptVersion = "native-" + cfg.nativeTemplate
	}
	rep, err := eval.Run(ctx, opts)
	if err != nil {
		return rep, err
	}
	if local {
		rep.PromptStyle = cfg.promptStyle
		rep.Endpoint = cfg.openaiBaseURL
		if cfg.promptStyle == promptStyleProduction {
			rep.BatchSize = cfg.batchSize
			if rep.BatchSize == 0 {
				rep.BatchSize = prompts.SubtitleTranslatorBatchSize
			}
		}
		rep.EpisodeMinutesEst = eval.EpisodeMinutes(rep.TranslateMinutes, rep.Cues)
		if rep.JudgeModel != "" {
			rep.JudgeNote += "受考模型是本機模型、不是 Claude，沒有自評偏誤，但分數仍是 AI 評分。"
		}
		if noSlot && glossaries != nil {
			rep.JudgeNote += "TranslateGemma 的官方格式沒有放對照表的位置，所以這份一律以 golden-v1（不帶對照表）計。"
		}
	}
	if !cfg.withTrace {
		rep.Results = nil
	}
	return rep, nil
}

// rejudgeBatch is smaller than eval.JudgeBatchSize: the batches being
// repaired already failed to parse once at 20.
const rejudgeBatch = 5

// rejudge repairs a report's failed second-pass batches with the judge the
// report names (or --judge-model) under --budget-usd, and rewrites the file.
// A batch that fails again is left counted in secondary_unjudged; the report
// is still written.
func rejudge(ctx context.Context, cfg config, d deps) (int, error) {
	if d.apiKey == "" {
		return 0, errors.New("set CLAUDE_API_KEY for the judge")
	}
	data, err := os.ReadFile(cfg.rejudgePath)
	if err != nil {
		return 0, err
	}
	var rep eval.Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return 0, fmt.Errorf("%s: %w", cfg.rejudgePath, err)
	}
	if cfg.samplePath == "" && rep.SampleVersion != eval.SampleVersion && rep.SampleVersion != eval.SampleVersionNoGlossary {
		return 0, fmt.Errorf("report measured on %q but this build embeds %q — pass the same --sample", rep.SampleVersion, eval.SampleVersion)
	}
	cues, err := loadSample(cfg.samplePath)
	if err != nil {
		return 0, err
	}
	judgeModel := rep.JudgeModel
	if judgeModel == "" {
		judgeModel = cfg.judgeModel
	}
	budget := ai.NewBudget(cfg.budgetUSD)
	ctx = ai.WithBudget(ctx, budget)
	n, rerr := eval.RejudgeSecondary(ctx, d.newCompleter(d.apiKey, judgeModel), &rep, cues, rejudgeBatch)
	rep.CostUSD += budget.SpentUSD()
	if werr := writeReport(cfg.rejudgePath, rep); werr != nil {
		return n, werr
	}
	if rerr != nil {
		if errors.Is(rerr, ai.ErrBudgetExceeded) || errors.Is(rerr, context.Canceled) || errors.Is(rerr, context.DeadlineExceeded) {
			// The partial repair is saved; say so loudly so it gets re-run.
			return n, fmt.Errorf("stopped after %d cues, %d still unjudged (report saved; run --rejudge again): %w", n, rep.SecondaryUnjudged, rerr)
		}
		fmt.Fprintln(d.stderr, "some batches failed again:", rerr)
	}
	return n, nil
}

func loadSample(path string) ([]eval.Cue, error) {
	if path == "" {
		return eval.Golden()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cues, err := eval.ParseJSONL(data)
	if err != nil {
		return nil, err
	}
	if len(cues) == 0 {
		return nil, fmt.Errorf("%s: no cues", path)
	}
	return cues, nil
}

// newFinalizer is delivery's finishing step: zhtw.Finalize — OpenCC s2twp
// (when the helper is installed), then the zh-TW lexicon replacements.
func newFinalizer(noOpenCC bool) eval.Finalizer {
	var conv zhtw.Converter
	if !noOpenCC {
		if c, err := subtitle.NewConverter(); err == nil && c.IsAvailable() {
			conv = c
		}
	}
	return func(raw string) string {
		out, _ := zhtw.Finalize(conv, raw, nil)
		return out
	}
}

func writeReport(path string, rep eval.Report) error {
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if path == "" {
		_, err = os.Stdout.Write(b)
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// secondaryReasons are the rule failures a second judge pass can score: the
// cue has text, just text the gate rejects. missing/empty/echoed have nothing
// worth judging.
var secondaryReasons = map[string]bool{
	subtitle.GateReasonSimplifiedLeak: true,
	subtitle.GateReasonMisaligned:     true,
	eval.ReasonNameMismatch:           true,
	eval.ReasonForbiddenTerm:          true,
	eval.ReasonTimeShift:              true,
}

// glossaryPairs converts exam terms to the translation service's shape.
func glossaryPairs(terms []eval.NameTerm) []services.GlossaryPair {
	out := make([]services.GlossaryPair, len(terms))
	for i, t := range terms {
		out[i] = services.GlossaryPair{Source: t.Source, Target: t.Target}
	}
	return out
}

// promptGlossary converts exam terms to the shape the version hash takes.
func promptGlossary(terms []eval.NameTerm) []prompts.GlossaryEntry {
	out := make([]prompts.GlossaryEntry, len(terms))
	for i, t := range terms {
		out[i] = prompts.GlossaryEntry{Source: t.Source, Target: t.Target}
	}
	return out
}
