package main

// soloJobSource is what the Activity page and GET /ai/usage read for ad-hoc
// single-title generation: the in-flight row and the live spend. Both
// *services.TranscriptionService (legacy solo runs) and *subtitle.SoloRunner
// (pipeline-mode clicks, disc-2026-10-single-generate-ignores-embedded-english-a)
// satisfy it with the same primitive shapes (services.batchJobSource /
// handlers.ManualUsageReader).
type soloJobSource interface {
	ActivityProgress() (active bool, percentDone, current, total int, currentItem string)
	ActiveManualUsage() (spentUSD, budgetUSD float64, ok bool)
}

// composedSoloJobs sums two solo sources. It lives in cmd/api for the Rule 19
// reason every other bridge here does: services cannot see subtitle.
type composedSoloJobs struct {
	legacy, solo soloJobSource
}

func (c composedSoloJobs) ActivityProgress() (active bool, percentDone, current, total int, currentItem string) {
	_, _, n1, _, item1 := c.legacy.ActivityProgress()
	_, _, n2, _, item2 := c.solo.ActivityProgress()
	current = n1 + n2
	currentItem = item2
	if currentItem == "" {
		currentItem = item1
	}
	return current > 0, 0, current, 0, currentItem
}

func (c composedSoloJobs) ActiveManualUsage() (spentUSD, budgetUSD float64, ok bool) {
	s1, b1, ok1 := c.legacy.ActiveManualUsage()
	s2, b2, ok2 := c.solo.ActiveManualUsage()
	return s1 + s2, b1 + b2, ok1 || ok2
}
