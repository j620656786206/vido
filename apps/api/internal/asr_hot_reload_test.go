package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file guards the ASR-KEY-HOT-RELOAD invariant introduced by story sub-5-2.
//
// WHY THIS TEST EXISTS — the bug it prevents shipped twice. sub-2-1a found that
// the Claude provider was built ONCE at boot inside `if cfg.HasClaudeKey()`, so
// a key saved in /settings/keys reached a client that did not exist. The ASR leg
// kept the identical shape (`if cfg.HasOpenAIKey()` at main.go) for two more
// months: the settings page said 「已設定」 while every transcribe answered 503
// until the container was restarted, and the product copy had to tell users to
// go restart their NAS.
//
// The second, quieter half was worse. The else-branch of that guard constructed
// the transcription service WITHOUT its four pipeline setters (run budget,
// per-show glossary, OpenCC safety net, atomic placer), so a keyless boot could
// never recover into a correct pipeline — only into a silently degraded one.
//
// sub-5-2 removes the gate entirely: construction is unconditional and
// "is ASR configured" is a per-call question asked of services.KeyResolver
// through ASRProviderHolder. These tests make that structural, so re-introducing
// a boot-time key gate is a red test rather than another two months of a
// settings page that quietly does nothing.

// keyGateCall is the config predicate that must never gate construction again.
// Its DEFINITION (`func (c *Config) HasOpenAIKey(`) does not contain the leading
// dot, so scanning for the call form finds callers only.
const keyGateCall = ".HasOpenAIKey("

func TestASRMustNotBeGatedOnTheBootTimeEnvKey(t *testing.T) {
	callers, scanned, err := findKeyGateCallers("..")
	if err != nil {
		t.Fatalf("scanning apps/api for %q failed: %v", keyGateCall, err)
	}
	if scanned == 0 {
		t.Fatal("no production .go files scanned — wrong working directory?")
	}

	for _, c := range callers {
		t.Errorf("%s calls %s — speech-recognition availability must come from "+
			"services.KeyResolver (via ASRProviderHolder.IsConfigured), not from the "+
			"boot-time environment snapshot (story sub-5-2 AC #2). A key saved in "+
			"/settings/keys never reaches this predicate, and a self-hosted endpoint "+
			"is configured with no key at all.", c, keyGateCall)
	}
}

// TestTranscriptionServiceIsConstructedExactlyOnce pins the other half: the old
// shape was an if/else pair that built the service TWO ways, and only one of
// them wired the four pipeline setters. One construction site means one wiring.
func TestTranscriptionServiceIsConstructedExactlyOnce(t *testing.T) {
	src, err := os.ReadFile("../cmd/api/main.go") //nolint:gosec // our own source tree
	if err != nil {
		t.Fatalf("reading main.go failed: %v", err)
	}
	main := string(src)

	if got := strings.Count(main, "services.NewTranscriptionService("); got != 1 {
		t.Errorf("main.go constructs TranscriptionService %d times, want exactly 1 — "+
			"a second construction site is how the keyless branch lost its four "+
			"pipeline setters (story sub-5-2 AC #2)", got)
	}

	// Each setter must be wired, and wired once, on the single service.
	for _, setter := range []string{
		"transcriptionService.SetRunBudgetUSD(",
		"transcriptionService.SetGlossaryRepository(",
		"transcriptionService.SetOpenCCConverter(",
		"transcriptionService.SetPlacer(",
	} {
		if got := strings.Count(main, setter); got != 1 {
			t.Errorf("main.go wires %s %d times, want exactly 1 — the run budget, "+
				"per-show glossary, OpenCC safety net and atomic placer must reach "+
				"EVERY boot, not only a key-configured one (story sub-5-2 AC #2)", setter, got)
		}
	}
}

// findKeyGateCallers walks production Go files under root and returns every file
// calling the boot-time key predicate. Test files are excluded: config's own
// tests still cover the accessor, which is exactly how it stays tested while
// unused as a gate. The config package itself is excluded — that is where the
// accessor is defined and documented.
func findKeyGateCallers(root string) (callers []string, scanned int, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if strings.Contains(filepath.ToSlash(path), "/config/") {
			return nil
		}
		src, rerr := os.ReadFile(path) //nolint:gosec // walking our own source tree
		if rerr != nil {
			return rerr
		}
		scanned++
		// Comments legitimately name the removed gate (they explain why it is
		// gone); only real call expressions are violations.
		for _, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(line, keyGateCall) {
				callers = append(callers, filepath.ToSlash(path))
				break
			}
		}
		return nil
	})
	return callers, scanned, err
}
