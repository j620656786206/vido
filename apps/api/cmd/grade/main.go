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
	// merge mode: fold a report into model_ratings.json instead of grading.
	mergePath   string
	ratingsPath string
}

// deps are the seams the tests replace: how a provider is built and how the
// finalizer (OpenCC) is obtained.
type deps struct {
	newCompleter func(apiKey, model string) ai.TextCompleter
	newFinalizer func(noOpenCC bool) eval.Finalizer
	apiKey       string
	now          func() time.Time
	stderr       io.Writer
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
		newFinalizer: newFinalizer,
		// Same variable the app reads (docs/development.md § Configuration), so
		// one key serves the NAS, local runs and the Model Grade workflow.
		apiKey: firstNonEmpty(os.Getenv("CLAUDE_API_KEY"), os.Getenv("ANTHROPIC_API_KEY")),
		now:    time.Now,
		stderr: os.Stderr,
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
	fs.StringVar(&cfg.samplePath, "sample", "", "golden sample JSONL (default: the embedded golden-v1)")
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
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.mergePath != "" {
		return cfg, nil
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
	if _, err := prompts.ParseLocalizationLevel(cfg.level); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func run(ctx context.Context, cfg config, d deps) (eval.Report, error) {
	if d.apiKey == "" {
		return eval.Report{}, errors.New("set CLAUDE_API_KEY (the same variable the app uses; ANTHROPIC_API_KEY also works)")
	}
	cues, err := loadSample(cfg.samplePath)
	if err != nil {
		return eval.Report{}, err
	}
	if cfg.limit > 0 && cfg.limit < len(cues) {
		cues = cues[:cfg.limit]
	}
	level, _ := prompts.ParseLocalizationLevel(cfg.level)

	budget := ai.NewBudget(cfg.budgetUSD)
	ctx = ai.WithBudget(ctx, budget)
	ctx = ai.WithModelID(ctx, cfg.model)

	translator := services.NewTranslationService(d.newCompleter(d.apiKey, cfg.model), nil)
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
			out, _, err := translator.TranslateWithGlossary(ctx, blocks, nil, nil, services.WithLocalizationLevel(level))
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

	var judge ai.TextCompleter
	if !cfg.noJudge {
		judge = d.newCompleter(d.apiKey, cfg.judgeModel)
	}
	opts := eval.Options{
		Cues: cues, Translate: translate, Finalize: d.newFinalizer(cfg.noOpenCC),
		Judge: judge, JudgeModel: cfg.judgeModel,
		ModelID: cfg.model, PromptVersion: prompts.PromptVersionFor(level), LexiconVersion: prompts.LexiconVersion(),
		GlossaryVersion: "", // the sample carries no show glossary
		Spent:           budget.SpentUSD, Now: d.now,
	}
	if cfg.noJudge {
		opts.JudgeModel = ""
	}
	rep, err := eval.Run(ctx, opts)
	if err != nil {
		return rep, err
	}
	if !cfg.withTrace {
		rep.Results = nil
	}
	return rep, nil
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

// newFinalizer mirrors delivery: OpenCC s2twp (when the helper is installed),
// then the zh-TW lexicon replacements — the order subtitle/process_item.go uses.
func newFinalizer(noOpenCC bool) eval.Finalizer {
	lexicon := prompts.ZhTWLexicon()
	var conv *subtitle.Converter
	if !noOpenCC {
		if c, err := subtitle.NewConverter(); err == nil && c.IsAvailable() {
			conv = c
		}
	}
	return func(raw string) string {
		s := raw
		if conv != nil {
			if out, err := conv.ConvertS2TWP([]byte(s)); err == nil {
				s = string(out)
			}
		}
		return lexicon.Apply(s)
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
