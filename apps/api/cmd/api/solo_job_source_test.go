package main

import "testing"

// disc-2026-10-activity-generation-progress-detail: composedSoloJobs names the
// one running item across both sources, and nothing when it is not exactly one.

type fakeSolo struct {
	running int
	id      string
}

func (f fakeSolo) ActivityProgress() (bool, int, int, int, string) {
	return f.running > 0, 0, f.running, 0, ""
}
func (f fakeSolo) ActiveManualUsage() (float64, float64, bool) { return 0, 0, false }
func (f fakeSolo) ActiveMediaID() (string, bool)               { return f.id, f.running == 1 }

func TestComposedSoloJobs_ActiveMediaID(t *testing.T) {
	cases := []struct {
		name         string
		legacy, solo fakeSolo
		want         string
		ok           bool
	}{
		{"idle", fakeSolo{}, fakeSolo{}, "", false},
		{"one pipeline click", fakeSolo{}, fakeSolo{running: 1, id: "ep-1"}, "ep-1", true},
		{"one legacy click", fakeSolo{running: 1, id: "mv-1"}, fakeSolo{}, "mv-1", true},
		{"one on each side", fakeSolo{running: 1, id: "mv-1"}, fakeSolo{running: 1, id: "ep-1"}, "", false},
		{"two on one side", fakeSolo{}, fakeSolo{running: 2}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := composedSoloJobs{tc.legacy, tc.solo}.ActiveMediaID()
			if got != tc.want || ok != tc.ok {
				t.Errorf("ActiveMediaID() = %q, %v; want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}
