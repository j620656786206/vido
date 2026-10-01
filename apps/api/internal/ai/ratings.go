package ai

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// Rating sources. A golden row was produced by cmd/grade on the shared
// sample; an eval-1 row is the 2026-09 human blind test that predates it.
// When both exist for one model the golden row wins (sub-7-8b AC #1): it is
// reproducible and re-runs the week a prompt or model changes.
const (
	RatingSourceGolden = "golden"
	RatingSourceEval1  = "eval-1"
)

// ModelRating is one measured grade — never a vendor claim. Every field a
// reader needs to decide whether to trust the letter rides with it.
//
// This is the shape cmd/grade --merge writes and model_ratings.json stores;
// the picker only ever sees the derived QualityGrade / QualityNote.
type ModelRating struct {
	ModelID        string  `json:"model_id"`
	Grade          string  `json:"grade"`
	ZeroRate       float64 `json:"zero_rate"`
	NaturalRate    float64 `json:"natural_rate"`
	CostUSD        float64 `json:"cost_usd"`
	Cues           int     `json:"cues"`
	PromptVersion  string  `json:"prompt_version"`
	LexiconVersion string  `json:"lexicon_version"`
	JudgeModel     string  `json:"judge_model"`
	JudgeVersion   string  `json:"judge_version"`
	GradedAt       string  `json:"graded_at"` // RFC 3339
	Source         string  `json:"source"`    // golden | eval-1
	// SampleVersion is the eval.SampleVersion the row was measured on
	// ("golden-v1"); eval-1 rows leave it empty.
	SampleVersion string `json:"sample_version,omitempty"`
	// Note is the provenance line the UI shows. Golden rows leave it empty
	// and get the standard line from RatingNote; eval-1 rows carry theirs.
	Note string `json:"note,omitempty"`
}

// RatingsFile is model_ratings.json.
type RatingsFile struct {
	Schema        int           `json:"schema"`
	SampleVersion string        `json:"sample_version"`
	Ratings       []ModelRating `json:"ratings"`
}

//go:embed model_ratings.json
var embeddedRatings []byte

var (
	ratingsOnce sync.Once
	ratingsByID map[string]ModelRating
)

// Ratings returns the embedded ratings keyed by model id, golden rows
// overriding eval-1 rows for the same model. The file is parsed once; a
// malformed file is a build defect and panics on first use (the catalog is
// read at startup, so a bad merge cannot ship quietly).
func Ratings() map[string]ModelRating {
	ratingsOnce.Do(func() {
		file, err := ParseRatings(embeddedRatings)
		if err != nil {
			panic(fmt.Sprintf("ai: embedded model_ratings.json is invalid: %v", err))
		}
		ratingsByID = indexRatings(file)
		for id := range ratingsByID {
			if _, described := modelMetadata[id]; !described {
				// Rated but not sellable: the row stays (history), the
				// picker never sees it.
				slog.Warn("model_ratings.json rates a model the catalog does not describe — ignored", "model_id", id)
			}
		}
	})
	return ratingsByID
}

// ParseRatings decodes and validates a ratings file.
func ParseRatings(data []byte) (RatingsFile, error) {
	var file RatingsFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return file, err
	}
	if file.Schema != 1 {
		return file, fmt.Errorf("unsupported schema %d", file.Schema)
	}
	seen := map[string]struct{}{}
	for i, r := range file.Ratings {
		if r.ModelID == "" {
			return file, fmt.Errorf("rating %d: empty model_id", i)
		}
		key := r.ModelID + "|" + r.Source
		if _, dup := seen[key]; dup {
			return file, fmt.Errorf("rating %s (%s): duplicate", r.ModelID, r.Source)
		}
		seen[key] = struct{}{}
		if r.Grade != "A" && r.Grade != "B" && r.Grade != "C" {
			return file, fmt.Errorf("rating %s: grade %q is not A/B/C — an incomplete run must not be merged", r.ModelID, r.Grade)
		}
		if r.Source != RatingSourceGolden && r.Source != RatingSourceEval1 {
			return file, fmt.Errorf("rating %s: unknown source %q", r.ModelID, r.Source)
		}
		if _, err := time.Parse(time.RFC3339, r.GradedAt); err != nil {
			return file, fmt.Errorf("rating %s: graded_at: %w", r.ModelID, err)
		}
		if r.ZeroRate < 0 || r.ZeroRate > 1 || r.NaturalRate < 0 || r.NaturalRate > 1 {
			return file, fmt.Errorf("rating %s: rates must be within [0,1]", r.ModelID)
		}
		if r.Source == RatingSourceEval1 && r.Note == "" {
			return file, fmt.Errorf("rating %s: an eval-1 row must carry its own note", r.ModelID)
		}
		if r.Source == RatingSourceGolden && (r.SampleVersion == "" || r.JudgeModel == "" || r.JudgeVersion == "" || r.PromptVersion == "") {
			return file, fmt.Errorf("rating %s: a golden row needs sample_version, prompt_version, judge_model and judge_version", r.ModelID)
		}
	}
	return file, nil
}

// indexRatings keys rows by model, letting golden beat eval-1.
func indexRatings(file RatingsFile) map[string]ModelRating {
	out := make(map[string]ModelRating, len(file.Ratings))
	for _, r := range file.Ratings {
		if cur, ok := out[r.ModelID]; ok && cur.Source == RatingSourceGolden && r.Source != RatingSourceGolden {
			continue
		}
		out[r.ModelID] = r
	}
	return out
}

// RatingNote is the provenance line shown next to a grade: which run, which
// sample, and — because the judge is itself a Claude model — who judged.
func RatingNote(r ModelRating) string {
	if r.Note != "" {
		return r.Note
	}
	month := r.GradedAt
	if t, err := time.Parse(time.RFC3339, r.GradedAt); err == nil {
		month = t.Format("2006-01")
	}
	sample := "黃金樣本"
	if v := strings.TrimPrefix(r.SampleVersion, "golden-"); v != "" && v != r.SampleVersion {
		sample += " " + v
	}
	return fmt.Sprintf("Vido 實測 %s，%s；裁判 %s", month, sample, r.JudgeModel)
}

// MergeRating returns the ratings file with one golden row added or replaced
// (same model_id + source), rows sorted by model id, as pretty JSON. It is
// what cmd/grade --merge and the model-grade workflow write back.
func MergeRating(file []byte, r ModelRating) ([]byte, error) {
	parsed, err := ParseRatings(file)
	if err != nil {
		return nil, fmt.Errorf("existing ratings: %w", err)
	}
	if r.Source == "" {
		r.Source = RatingSourceGolden
	}
	replaced := false
	for i := range parsed.Ratings {
		if parsed.Ratings[i].ModelID == r.ModelID && parsed.Ratings[i].Source == r.Source {
			parsed.Ratings[i] = r
			replaced = true
		}
	}
	if !replaced {
		parsed.Ratings = append(parsed.Ratings, r)
	}
	sort.SliceStable(parsed.Ratings, func(i, j int) bool {
		if parsed.Ratings[i].ModelID != parsed.Ratings[j].ModelID {
			return parsed.Ratings[i].ModelID < parsed.Ratings[j].ModelID
		}
		return parsed.Ratings[i].Source < parsed.Ratings[j].Source
	})
	// Validate the result the same way the embed will on next start.
	out, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')
	if _, err := ParseRatings(out); err != nil {
		return nil, fmt.Errorf("merged ratings would not load: %w", err)
	}
	return out, nil
}
