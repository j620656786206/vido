package main

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// disc-2026-10-activity-solo-jobs-collapsed: composedSoloJobs lists every job
// from both sources — one Activity row each — instead of one summed count.

type fakeSolo []services.SoloJobSnapshot

func (f fakeSolo) ActiveSoloJobs() []services.SoloJobSnapshot {
	return append([]services.SoloJobSnapshot(nil), f...)
}
func (f fakeSolo) ActiveManualUsage() (float64, float64, bool) { return 0, 0, false }

func TestComposedSoloJobs_ListsBothSources(t *testing.T) {
	cases := []struct {
		name         string
		legacy, solo fakeSolo
		want         []string
	}{
		{"idle", nil, nil, nil},
		{"pipeline clicks", nil, fakeSolo{{MediaID: "ep-1"}, {MediaID: "ep-2"}}, []string{"ep-1", "ep-2"}},
		{"legacy click", fakeSolo{{MediaID: "mv-1"}}, nil, []string{"mv-1"}},
		{"both sides", fakeSolo{{MediaID: "mv-1"}}, fakeSolo{{MediaID: "ep-1", Queued: true}}, []string{"ep-1", "mv-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, s := range (composedSoloJobs{tc.legacy, tc.solo}).ActiveSoloJobs() {
				got = append(got, s.MediaID)
			}
			sort.Strings(got)
			if len(got) != len(tc.want) {
				t.Fatalf("ActiveSoloJobs() = %v; want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("ActiveSoloJobs() = %v; want %v", got, tc.want)
				}
			}
		})
	}
}

// The audio extractor's wait on the shared gate reaches the job's observer.
func TestExtractSlotAdapter_ReportsTheWait(t *testing.T) {
	gate := subtitle.NewExtractGate()
	hold, _, err := gate.Acquire(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	seen := make(chan bool, 4)
	ctx := services.WithExtractWaitObserver(context.Background(), func(w bool) { seen <- w })
	acquired := make(chan func(), 1)
	go func() {
		release, err := extractSlotAdapter{gate: gate}.Acquire(ctx)
		if err != nil {
			t.Error(err)
		}
		acquired <- release
	}()

	select {
	case w := <-seen:
		if !w {
			t.Fatal("first report should be waiting=true")
		}
	case <-time.After(time.Second):
		t.Fatal("a busy gate must report the wait")
	}
	hold()
	release := <-acquired
	defer release()
	if w := <-seen; w {
		t.Fatal("taking the slot should report waiting=false")
	}
}
