package main

import "github.com/vido/api/internal/services"

// soloJobSource is what the Activity page and GET /ai/usage read for ad-hoc
// single-title generation: one snapshot per running job and the live spend.
// Both *services.TranscriptionService (legacy solo runs) and
// *subtitle.SoloRunner (pipeline-mode clicks,
// disc-2026-10-single-generate-ignores-embedded-english-a) satisfy it.
type soloJobSource interface {
	ActiveSoloJobs() []services.SoloJobSnapshot
	ActiveManualUsage() (spentUSD, budgetUSD float64, ok bool)
}

// composedSoloJobs joins two solo sources. It lives in cmd/api for the Rule 19
// reason every other bridge here does: services cannot see subtitle.
type composedSoloJobs struct {
	legacy, solo soloJobSource
}

// ActiveSoloJobs lists both sources' jobs — one Activity row each
// (disc-2026-10-activity-solo-jobs-collapsed). Nothing is listed twice: the
// ASR leg of a pipeline run registers in the legacy service as Solo=false.
func (c composedSoloJobs) ActiveSoloJobs() []services.SoloJobSnapshot {
	return append(c.legacy.ActiveSoloJobs(), c.solo.ActiveSoloJobs()...)
}

func (c composedSoloJobs) ActiveManualUsage() (spentUSD, budgetUSD float64, ok bool) {
	s1, b1, ok1 := c.legacy.ActiveManualUsage()
	s2, b2, ok2 := c.solo.ActiveManualUsage()
	return s1 + s2, b1 + b2, ok1 || ok2
}
