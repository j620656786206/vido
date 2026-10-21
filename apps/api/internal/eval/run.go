package eval

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
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

	ModelID string
	// SampleVersion stamps the report: SampleVersion when the translator was
	// handed the sample's name glossary, SampleVersionNoGlossary when it was
	// not. "" = SampleVersion (the sub-7-8a default); every caller that sends
	// no glossary — cmd/grade --no-glossary, the app's 20-cue preview — must
	// say so explicitly.
	SampleVersion   string
	PromptVersion   string
	LexiconVersion  string
	GlossaryVersion string

	// AlsoJudge names rule-failure reasons whose cues the judge scores
	// anyway, on their finalized text, in a second pass. A cue qualifies when
	// EVERY one of its failures is in the set. Their Score stays 0 — the
	// headline grade is still the gate's verdict — and the report carries the
	// "secondary" rates as if those reasons were not gating
	// (spike-local-translation-golden-eval: local models leak 简体 that OpenCC
	// converts on delivery; Claude leaves names in English, which the
	// name_mismatch rule fails when the run carries no glossary).
	AlsoJudge []string

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
	// SecondaryScore is the second-pass judge score for a cue whose failures
	// are all in Options.AlsoJudge; SecondaryJudged says it was scored.
	SecondaryScore  int  `json:"secondary_score,omitempty"`
	SecondaryJudged bool `json:"secondary_judged,omitempty"`
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

	// Local-model runs (spike-local-translation-golden-eval) stamp how the
	// model was driven and what a full episode would cost in time. cmd/grade
	// fills these; Run leaves them zero, and --merge never reads them.
	PromptStyle       string  `json:"prompt_style,omitempty"`
	BatchSize         int     `json:"batch_size,omitempty"`
	Endpoint          string  `json:"endpoint,omitempty"`
	EpisodeMinutesEst float64 `json:"episode_minutes_est,omitempty"`
	// Set only with Options.AlsoJudge: the rates if those reasons did not
	// gate (their cues scored by the judge instead).
	SecondaryReasons     []string `json:"secondary_reasons,omitempty"`
	ZeroRateSecondary    *float64 `json:"zero_rate_secondary,omitempty"`
	NaturalRateSecondary *float64 `json:"natural_rate_secondary,omitempty"`
	// SecondaryUnjudged counts second-pass cues whose judge batch failed
	// (malformed reply after the retry); they stay 0 in the secondary rates.
	SecondaryUnjudged int `json:"secondary_unjudged,omitempty"`
	// SecondaryIncomplete says why the second pass stopped early (budget,
	// cancel); the headline grade is unaffected. --rejudge can finish it.
	SecondaryIncomplete string `json:"secondary_incomplete,omitempty"`
	// TranslateMinutes is the wall time of the translation step alone
	// (Minutes also includes the judge); EpisodeMinutesEst scales this one.
	TranslateMinutes float64 `json:"translate_minutes,omitempty"`

	Results []CueResult `json:"results,omitempty"`
}

// EpisodeCues is a one-hour episode's cue count for time estimates: eval-1
// measured 10,304 cues over 740 minutes = 13.92 cues/min.
const EpisodeCues = 835

// EpisodeMinutes scales a run's wall time to one EpisodeCues-cue episode.
func EpisodeMinutes(minutes float64, cues int) float64 {
	if cues <= 0 {
		return 0
	}
	return minutes * EpisodeCues / float64(cues)
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
		ModelID: opts.ModelID, SampleVersion: sampleVersion(opts.SampleVersion),
		PromptVersion: opts.PromptVersion, LexiconVersion: opts.LexiconVersion, GlossaryVersion: opts.GlossaryVersion,
		JudgeModel: opts.JudgeModel, JudgeVersion: JudgeVersion,
		Cues: len(opts.Cues), RuleFailsByReason: map[string]int{},
	}
	if rep.JudgeModel == "" && opts.Judge != nil {
		rep.JudgeModel = DefaultJudgeModel
	}
	rep.JudgeNote = JudgeNoteFor(rep.JudgeModel)

	translateStarted := now()
	raw, terr := opts.Translate(ctx, opts.Cues)
	rep.TranslateMinutes = now().Sub(translateStarted).Minutes()
	if terr != nil {
		if len(raw) == 0 {
			return Report{}, fmt.Errorf("eval: translate: %w", terr)
		}
		rep.Incomplete = "translate: " + terr.Error()
	}

	results := make([]CueResult, len(opts.Cues))
	var toJudge, second []JudgeItem
	idx := map[string]int{}
	alsoJudge := map[string]bool{}
	for _, r := range opts.AlsoJudge {
		alsoJudge[r] = true
	}
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
			if opts.Judge != nil && len(alsoJudge) > 0 && strings.TrimSpace(r.Raw) != "" && allIn(r.RuleFails, alsoJudge) {
				second = append(second, JudgeItem{ID: c.ID, Text: c.Text, Refs: c.Refs, Candidate: r.Final})
			}
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
	// The second pass never moves the headline numbers; a failed batch leaves
	// those cues at 0 in the secondary rates too, and is counted.
	// A budget/cancel stop here is NOT a run stop: the headline is complete,
	// so the remaining cues are only counted as unjudged (for --rejudge).
	for start := 0; start < len(second) && rep.Incomplete == ""; start += JudgeBatchSize {
		end := min(start+JudgeBatchSize, len(second))
		scores, err := JudgeBatch(ctx, opts.Judge, second[start:end])
		if err != nil {
			stop := errors.Is(err, ai.ErrBudgetExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
			if stop {
				end = len(second)
				rep.SecondaryIncomplete = err.Error()
			}
			for _, it := range second[start:end] {
				results[idx[it.ID]].Why = "second_pass_judge_failed: " + err.Error()
			}
			rep.SecondaryUnjudged += end - start
			if stop {
				break
			}
			continue
		}
		for _, it := range second[start:end] {
			r := &results[idx[it.ID]]
			r.SecondaryScore, r.SecondaryJudged, r.Why = scores[it.ID].Score, true, scores[it.ID].Why
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
	if len(alsoJudge) > 0 && opts.Judge != nil && rep.Incomplete == "" {
		rep.SecondaryReasons = append([]string(nil), opts.AlsoJudge...)
		rep.ZeroRateSecondary, rep.NaturalRateSecondary = secondaryRates(results)
	}
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

// allIn reports whether every reason is in set (and there is at least one).
func allIn(reasons []string, set map[string]bool) bool {
	if len(reasons) == 0 {
		return false
	}
	for _, r := range reasons {
		if !set[r] {
			return false
		}
	}
	return true
}

// secondaryRates are the zero and natural rates with every second-pass score
// standing in for its cue's gated 0.
func secondaryRates(results []CueResult) (*float64, *float64) {
	var zeros, naturals int
	for _, r := range results {
		score := r.Score
		if r.SecondaryJudged {
			score = r.SecondaryScore
		}
		if score == 0 {
			zeros++
		}
		if score == 2 {
			naturals++
		}
	}
	n := float64(len(results))
	zr, nr := float64(zeros)/n, float64(naturals)/n
	return &zr, &nr
}

// RejudgeSecondary retries the second pass for a traced report's cues whose
// judge batch failed (SecondaryUnjudged), in batches of batchSize, and
// recomputes the secondary rates. It never touches the headline numbers.
// It returns how many cues were newly scored; cues that fail again stay
// counted in SecondaryUnjudged.
func RejudgeSecondary(ctx context.Context, judge ai.TextCompleter, rep *Report, cues []Cue, batchSize int) (int, error) {
	if len(rep.Results) == 0 {
		return 0, errors.New("eval: report has no per-cue results (re-run cmd/grade with --trace)")
	}
	if len(rep.SecondaryReasons) == 0 {
		return 0, errors.New("eval: report has no secondary pass to repair")
	}
	if batchSize <= 0 {
		batchSize = JudgeBatchSize
	}
	set := map[string]bool{}
	for _, r := range rep.SecondaryReasons {
		set[r] = true
	}
	byID := make(map[string]Cue, len(cues))
	for _, c := range cues {
		byID[c.ID] = c
	}
	idx := map[string]int{}
	var todo []JudgeItem
	for i, r := range rep.Results {
		if r.SecondaryJudged || strings.TrimSpace(r.Raw) == "" || !allIn(r.RuleFails, set) {
			continue
		}
		c, ok := byID[r.ID]
		if !ok {
			return 0, fmt.Errorf("eval: cue %s is not in the sample", r.ID)
		}
		idx[r.ID] = i
		todo = append(todo, JudgeItem{ID: c.ID, Text: c.Text, Refs: c.Refs, Candidate: r.Final})
	}
	scored := 0
	var lastErr error
	defer func() {
		// Every exit — including a budget stop — leaves the counters and
		// rates consistent with the scores written so far.
		rep.SecondaryUnjudged = len(todo) - scored
		rep.ZeroRateSecondary, rep.NaturalRateSecondary = secondaryRates(rep.Results)
		if rep.SecondaryUnjudged == 0 {
			rep.SecondaryIncomplete = ""
		}
	}()
	for start := 0; start < len(todo); start += batchSize {
		end := min(start+batchSize, len(todo))
		scores, err := JudgeBatch(ctx, judge, todo[start:end])
		if err != nil {
			if errors.Is(err, ai.ErrBudgetExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return scored, err
			}
			lastErr = err
			continue
		}
		for _, it := range todo[start:end] {
			r := &rep.Results[idx[it.ID]]
			r.SecondaryScore, r.SecondaryJudged, r.Why = scores[it.ID].Score, true, scores[it.ID].Why
			scored++
		}
	}
	return scored, lastErr
}

func sampleVersion(v string) string {
	if v == "" {
		return SampleVersion
	}
	return v
}
