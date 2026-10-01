// Package eval grades a translation model on Vido's golden sample (story
// sub-7-8a): the same 200 English cues for every model, scored by a rule layer
// that reuses the pipeline's quality-gate vocabulary plus a fixed AI judge, and
// reduced to the A/B/C grade the model picker shows.
//
// Nothing here talks to the network. Translating and judging are injected as
// functions so the package is testable with fakes and so cmd/grade (the CLI)
// and the sub-7-8c preview endpoint can share one implementation.
package eval

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// SampleVersion is stamped on every report so a grade can be traced to the
// exact set of cues it was measured on. Changing the sample bumps this and
// invalidates every published rating (sub-7-8b re-grades).
const SampleVersion = "golden-v1"

//go:embed golden/golden-v1.jsonl
var goldenV1 []byte

// Cue is one golden-sample line: an English cue, the acceptable zh-TW
// renderings, and the per-cue checks the rule layer applies.
type Cue struct {
	ID     string `json:"id"`
	Source string `json:"source"` // tos | sita | trap
	Start  string `json:"start"`
	End    string `json:"end"`
	Text   string `json:"text"`
	// Refs are ≥2 acceptable translations. They are shown to the judge as
	// anchors for "what good looks like"; a different wording is still fine.
	Refs []string `json:"refs"`
	// Names maps a proper noun in Text to its accepted zh renderings. A
	// translation that drops the name or renders it another way fails
	// name_mismatch — consistency across cues is the whole point.
	Names map[string][]string `json:"names"`
	// Traps names the trap categories this cue was written for ("" for
	// borrowed cues). It is bookkeeping for coverage tests and the report.
	Traps []string `json:"traps"`
	// Anchors are tokens that must land on THIS cue (time_shift). When empty,
	// the first rendering of each name is used.
	Anchors []string `json:"anchors,omitempty"`
	// Forbid lists renderings a Taiwan audience would read as mainland or
	// Cantonese usage (軟件, 視頻…). Any hit fails forbidden_term.
	Forbid      []string `json:"forbid,omitempty"`
	Note        string   `json:"note,omitempty"`
	Attribution string   `json:"attribution"`
}

// ExpectedCounts pins the composition of golden-v1 (AC #1). The schema test
// and Validate both check it so a stray edit cannot silently shrink the sample.
var ExpectedCounts = map[string]int{"tos": 76, "sita": 84, "trap": 40}

var (
	goldenOnce sync.Once
	goldenCues []Cue
	goldenErr  error
)

// Golden returns the embedded golden-v1 sample, parsed and validated once.
func Golden() ([]Cue, error) {
	goldenOnce.Do(func() {
		goldenCues, goldenErr = ParseJSONL(goldenV1)
		if goldenErr == nil {
			goldenErr = Validate(goldenCues)
		}
	})
	return goldenCues, goldenErr
}

// ParseJSONL decodes one cue per non-empty line.
func ParseJSONL(data []byte) ([]Cue, error) {
	var cues []Cue
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := bytes.TrimSpace(sc.Bytes())
		if len(raw) == 0 {
			continue
		}
		var c Cue
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("golden line %d: %w", line, err)
		}
		cues = append(cues, c)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cues, nil
}

// Validate enforces the sample contract: unique ids, non-empty text, at least
// two references, known sources, and the pinned per-source counts.
func Validate(cues []Cue) error {
	seen := make(map[string]struct{}, len(cues))
	counts := map[string]int{}
	for i, c := range cues {
		if c.ID == "" {
			return fmt.Errorf("cue %d: empty id", i)
		}
		if _, dup := seen[c.ID]; dup {
			return fmt.Errorf("cue %s: duplicate id", c.ID)
		}
		seen[c.ID] = struct{}{}
		if _, ok := ExpectedCounts[c.Source]; !ok {
			return fmt.Errorf("cue %s: unknown source %q", c.ID, c.Source)
		}
		counts[c.Source]++
		if strings.TrimSpace(c.Text) == "" {
			return fmt.Errorf("cue %s: empty text", c.ID)
		}
		if len(c.Refs) < 2 {
			return fmt.Errorf("cue %s: needs at least 2 refs, has %d", c.ID, len(c.Refs))
		}
		for _, r := range c.Refs {
			if strings.TrimSpace(r) == "" {
				return fmt.Errorf("cue %s: empty ref", c.ID)
			}
		}
		if c.Source == "trap" && len(c.Traps) == 0 {
			return fmt.Errorf("cue %s: trap cue without a category", c.ID)
		}
	}
	for src, want := range ExpectedCounts {
		if counts[src] != want {
			return fmt.Errorf("source %s: want %d cues, have %d", src, want, counts[src])
		}
	}
	return nil
}
