package eval

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/vido/api/internal/ai"
)

// Translator renders the cues with the model under test and returns the RAW
// output (pre-OpenCC) keyed by cue id. A cue the model did not answer is
// simply absent. cmd/grade adapts services.TranslationService to this.
type Translator func(ctx context.Context, cues []Cue) (map[string]string, error)

// Finalizer turns raw output into what the viewer would get (OpenCC s2twp,
// then the zh-TW lexicon — the pipeline's delivery order). The judge reads the
// finalized text; the rule layer reads the raw text, like the gate does.
type Finalizer func(raw string) string

// Options wires one grading run.
type Options struct {
	Cues       []Cue
	Translate  Translator
	Finalize   Finalizer        // nil = identity
	Judge      ai.TextCompleter // nil = rules only (every unfailed cue scores 1)
	JudgeModel string

	ModelID         string
	PromptVersion   string
	LexiconVersion  string
	GlossaryVersion string

	// Spent reports the money the run has consumed so far (ai.Budget.SpentUSD);
	// nil = 0. Now is the clock (Rule 23); nil = time.Now.
	Spent func() float64
	Now   func() time.Time
}

// CueResult is the per-cue trace a report can carry for inspection.
type CueResult struct {
	ID        string   `json:"id"`
	Source    string   `json:"source"`
	Raw       string   `json:"raw"`
	Final     string   `json:"final"`
	RuleFails []string `json:"rule_fails,omitempty"`
	Score     int      `json:"score"`
	Judged    bool     `json:"judged"`
	Why       string   `json:"why,omitempty"`
}

// Report is cmd/grade's output and sub-7-8b's input.
type Report struct {
	ModelID         string `json:"model_id"`
	SampleVersion   string `json:"sample_version"`
	PromptVersion   string `json:"prompt_version"`
	LexiconVersion  string `json:"lexicon_version"`
	GlossaryVersion string `json:"glossary_version"`
	JudgeModel      string `json:"judge_model"`
	JudgeVersion    string `json:"judge_version"`

	Cues              int            `json:"cues"`
	ZeroRate          float64        `json:"zero_rate"`
	NaturalRate       float64        `json:"natural_rate"`
	RuleFailRate      float64        `json:"rule_fail_rate"`
	RuleFailsByReason map[string]int `json:"rule_fails_by_reason"`
	Unjudged          int            `json:"unjudged,omitempty"`

	CostUSD   float64 `json:"cost_usd"`
	Minutes   float64 `json:"minutes"`
	Grade     string  `json:"grade"`
	JudgeNote string  `json:"judge_note"`
	// Incomplete explains a grade of "incomplete" (budget hit, translator
	// gave up); empty otherwise.
	Incomplete string    `json:"incomplete,omitempty"`
	GradedAt   time.Time `json:"graded_at"`

	Results []CueResult `json:"results,omitempty"`
}

// Run translates, rule-checks, judges and grades. Only a hard failure before
// any cue was translated is returned as an error; a budget stop mid-way comes
// back as a report with Grade "incomplete" so the money already spent is
// still accounted for.
func Run(ctx context.Context, opts Options) (Report, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	finalize := opts.Finalize
	if finalize == nil {
		finalize = func(s string) string { return s }
	}
	if len(opts.Cues) == 0 {
		return Report{}, errors.New("eval: no cues")
	}
	if opts.Translate == nil {
		return Report{}, errors.New("eval: no translator")
	}
	started := now()
	rep := Report{
		ModelID: opts.ModelID, SampleVersion: SampleVersion,
		PromptVersion: opts.PromptVersion, LexiconVersion: opts.LexiconVersion, GlossaryVersion: opts.GlossaryVersion,
		JudgeModel: opts.JudgeModel, JudgeVersion: JudgeVersion, JudgeNote: JudgeNote,
		Cues: len(opts.Cues), RuleFailsByReason: map[string]int{},
	}
	if rep.JudgeModel == "" && opts.Judge != nil {
		rep.JudgeModel = DefaultJudgeModel
	}

	raw, terr := opts.Translate(ctx, opts.Cues)
	if terr != nil {
		if len(raw) == 0 {
			return Report{}, fmt.Errorf("eval: translate: %w", terr)
		}
		rep.Incomplete = "translate: " + terr.Error()
	}

	results := make([]CueResult, len(opts.Cues))
	var toJudge []JudgeItem
	idx := map[string]int{}
	for i := range opts.Cues {
		c := &opts.Cues[i]
		var prev, next *Cue
		if i > 0 {
			prev = &opts.Cues[i-1]
		}
		if i+1 < len(opts.Cues) {
			next = &opts.Cues[i+1]
		}
		r := CueResult{ID: c.ID, Source: c.Source, Raw: raw[c.ID]}
		r.Final = finalize(r.Raw)
		r.RuleFails = RuleCheck(prev, c, next, r.Raw)
		for _, reason := range r.RuleFails {
			rep.RuleFailsByReason[reason]++
		}
		if len(r.RuleFails) > 0 {
			r.Score = 0
		} else if opts.Judge == nil {
			r.Score = 1
		} else {
			toJudge = append(toJudge, JudgeItem{ID: c.ID, Text: c.Text, Refs: c.Refs, Candidate: r.Final})
		}
		idx[c.ID] = i
		results[i] = r
	}

	for start := 0; start < len(toJudge) && rep.Incomplete == ""; start += JudgeBatchSize {
		end := start + JudgeBatchSize
		if end > len(toJudge) {
			end = len(toJudge)
		}
		batch := toJudge[start:end]
		scores, err := JudgeBatch(ctx, opts.Judge, batch)
		if err != nil {
			if errors.Is(err, ai.ErrBudgetExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				rep.Incomplete = "judge: " + err.Error()
				break
			}
			// A malformed reply after the retry: those cues stay unjudged at
			// 0 and are counted, never silently skipped.
			for _, it := range batch {
				results[idx[it.ID]].Why = "judge_failed: " + err.Error()
			}
			rep.Unjudged += len(batch)
			continue
		}
		for _, it := range batch {
			s := scores[it.ID]
			r := &results[idx[it.ID]]
			r.Score, r.Judged, r.Why = s.Score, true, s.Why
		}
	}
	if rep.Incomplete == "" {
		// Cues the translator never answered, after a successful call, are
		// legitimate zeros (missing). Cues left unjudged by a judge stop are
		// what Incomplete already covers.
		for i := range results {
			if !results[i].Judged && len(results[i].RuleFails) == 0 && opts.Judge != nil && results[i].Why == "" {
				rep.Unjudged++
			}
		}
	}

	var zeros, naturals, ruleFails int
	for _, r := range results {
		if r.Score == 0 {
			zeros++
		}
		if r.Score == 2 {
			naturals++
		}
		if len(r.RuleFails) > 0 {
			ruleFails++
		}
	}
	n := float64(len(results))
	rep.ZeroRate = float64(zeros) / n
	rep.NaturalRate = float64(naturals) / n
	rep.RuleFailRate = float64(ruleFails) / n
	if rep.Incomplete != "" {
		rep.Grade = GradeIncomplete
	} else {
		rep.Grade = GradeFor(rep.ZeroRate, rep.NaturalRate)
	}
	if opts.Spent != nil {
		rep.CostUSD = opts.Spent()
	}
	rep.GradedAt = now()
	rep.Minutes = rep.GradedAt.Sub(started).Minutes()
	sort.SliceStable(results, func(i, j int) bool { return idx[results[i].ID] < idx[results[j].ID] })
	rep.Results = results
	return rep, nil
}
