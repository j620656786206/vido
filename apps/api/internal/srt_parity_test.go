// Mirror-Types drift detector for project-context.md Rule 19.
//
// services.ParseSRTToTranslationBlocks is a deliberate inline copy of
// subtitle.ParseSRT (the services package cannot import subtitle without
// creating an import cycle — see Rule 19). Step 4 of the Mirror-Types
// workaround says "Keep the two implementations in sync via code review",
// but humans miss things. This test runs both parsers against a shared
// fixture set on every CI run and fails on any output divergence.
//
// Lives in package internal because that is the only place that can import
// both subtitle and services without violating Rule 19. The test file is
// outside internal/services/ so boundaries_test.go won't flag it as a
// services ↛ subtitle violation.
package internal

import (
	"testing"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

func TestParseSRT_ParityWithSubtitle(t *testing.T) {
	fixtures := map[string]string{
		"basic two blocks": "1\n00:00:01,000 --> 00:00:04,000\nHello world\n\n2\n00:00:05,000 --> 00:00:08,000\nGoodbye\n",
		"empty input":      "",
		"single block":     "1\n00:00:01,000 --> 00:00:04,000\nOnly one\n",
		"multi-line text":  "1\n00:00:01,000 --> 00:00:04,000\nLine one\nLine two\nLine three\n",
		"utf-8 BOM":        "\xEF\xBB\xBF1\n00:00:01,000 --> 00:00:04,000\nWith BOM\n",
		"windows CRLF":     "1\r\n00:00:01,000 --> 00:00:04,000\r\nWith CRLF\r\n",
		"old-mac CR":       "1\r00:00:01,000 --> 00:00:04,000\rWith CR\r",
		"extra blank lines": "1\n00:00:01,000 --> 00:00:04,000\nHello\n\n\n\n2\n00:00:05,000 --> 00:00:08,000\nWorld\n",
		"unicode text":     "1\n00:00:01,000 --> 00:00:04,000\n你好世界\n",
		"malformed timestamp gets dropped": "1\n00:00:01,000 -> 00:00:04,000\nBad\n\n2\n00:00:05,000 --> 00:00:08,000\nGood\n",
	}

	for name, input := range fixtures {
		t.Run(name, func(t *testing.T) {
			subBlocks, errSub := subtitle.ParseSRT(input)
			svcBlocks, errSvc := services.ParseSRTToTranslationBlocks(input)

			if (errSub == nil) != (errSvc == nil) {
				t.Fatalf("error parity broken: subtitle err=%v, services err=%v", errSub, errSvc)
			}

			if len(subBlocks) != len(svcBlocks) {
				t.Fatalf(
					"block count parity broken (Mirror-Types drift): subtitle=%d, services=%d. "+
						"Sync the two implementations — see project-context.md Rule 19 Step 4.",
					len(subBlocks), len(svcBlocks),
				)
			}

			for i := range subBlocks {
				sb, sv := subBlocks[i], svcBlocks[i]
				if sb.Index != sv.Index || sb.Start != sv.Start || sb.End != sv.End || sb.Text != sv.Text {
					t.Errorf(
						"block %d parity broken (Mirror-Types drift):\n"+
							"  subtitle: {Index:%d Start:%q End:%q Text:%q}\n"+
							"  services: {Index:%d Start:%q End:%q Text:%q}\n"+
							"Sync the two implementations — see project-context.md Rule 19 Step 4.",
						i,
						sb.Index, sb.Start, sb.End, sb.Text,
						sv.Index, sv.Start, sv.End, sv.Text,
					)
				}
			}
		})
	}
}
