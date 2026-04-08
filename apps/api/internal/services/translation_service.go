package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/sse"
)

// SSE event type for translation progress (AC #6)
const (
	EventTranslationProgress sse.EventType = "translation_progress"
)

// Translation constants
const (
	// TranslationTimeout is the max time per batch request.
	TranslationTimeout = 60 * time.Second
	// TranslationMaxTokens is the max response tokens per batch (10 blocks × ~100 chars).
	TranslationMaxTokens = 4096
)

// TranslationBlock represents a subtitle block for the translation service.
// Mirrors subtitle.SubtitleBlock but is a separate type because subtitle.Engine
// imports services (for TerminologyCorrectionServiceInterface), creating a cycle
// if services also imported subtitle.
type TranslationBlock struct {
	Index int
	Start string
	End   string
	Text  string
}

// TranslationService uses Claude API to translate English subtitles to Traditional Chinese.
type TranslationService struct {
	provider ai.TextCompleter
	sseHub   *sse.Hub
}

// NewTranslationService creates a new translation service.
// Returns nil if provider is nil (graceful degradation per AC #4).
func NewTranslationService(provider ai.TextCompleter, sseHub *sse.Hub) *TranslationService {
	if provider == nil {
		slog.Info("Translation service not configured - no AI provider")
		return nil
	}

	slog.Info("Translation service initialized")
	return &TranslationService{
		provider: provider,
		sseHub:   sseHub,
	}
}

// IsConfigured returns true if Claude API key is available for translation.
func (s *TranslationService) IsConfigured() bool {
	return s != nil && s.provider != nil
}

// Translate translates subtitle blocks from English to Traditional Chinese.
// progressFn (optional) receives percentage updates per batch.
// On partial failure, translated blocks are returned with untranslated blocks
// retaining their original English text (AC #5).
func (s *TranslationService) Translate(ctx context.Context, blocks []TranslationBlock, progressFn func(float64)) ([]TranslationBlock, error) {
	if len(blocks) == 0 {
		return nil, nil
	}

	batchSize := prompts.SubtitleTranslatorBatchSize
	contextWindow := prompts.SubtitleTranslatorContextWindow

	// Copy blocks to result (will be modified in place)
	result := make([]TranslationBlock, len(blocks))
	copy(result, blocks)

	totalBlocks := len(blocks)
	processedBlocks := 0
	hasPartialFailure := false

	for batchStart := 0; batchStart < totalBlocks; batchStart += batchSize {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("translation cancelled: %w", err)
		}

		batchEnd := batchStart + batchSize
		if batchEnd > totalBlocks {
			batchEnd = totalBlocks
		}

		batch := blocks[batchStart:batchEnd]

		// Build context from previous translated blocks (AC #2)
		var contextBlocks []prompts.SubtitleTranslatorBlock
		if batchStart > 0 {
			contextStart := batchStart - contextWindow
			if contextStart < 0 {
				contextStart = 0
			}
			for i := contextStart; i < batchStart; i++ {
				contextBlocks = append(contextBlocks, prompts.SubtitleTranslatorBlock{
					Index: result[i].Index,
					Text:  result[i].Text, // Use translated text as context
				})
			}
		}

		// Build prompt blocks
		var promptBlocks []prompts.SubtitleTranslatorBlock
		for _, b := range batch {
			promptBlocks = append(promptBlocks, prompts.SubtitleTranslatorBlock{
				Index: b.Index,
				Text:  b.Text,
			})
		}

		// Call Claude API
		userPrompt := prompts.BuildSubtitleTranslatorPrompt(promptBlocks, contextBlocks)

		batchCtx, batchCancel := context.WithTimeout(ctx, TranslationTimeout)
		translated, err := s.provider.CompleteText(
			batchCtx,
			prompts.SubtitleTranslatorSystemPrompt,
			userPrompt,
			TranslationMaxTokens,
		)
		batchCancel()

		if err != nil {
			// AC #5: on error, keep English text for failed blocks
			slog.Warn("Translation batch failed — keeping English text for blocks",
				"batch_start", batchStart,
				"batch_end", batchEnd,
				"error", err,
			)

			// Check if this is a context cancellation (propagate it)
			if ctx.Err() != nil {
				return nil, fmt.Errorf("translation cancelled: %w", ctx.Err())
			}

			hasPartialFailure = true
			processedBlocks += len(batch)
			if progressFn != nil {
				progressFn(float64(processedBlocks) / float64(totalBlocks) * 100)
			}
			continue
		}

		// Parse response and apply translations
		indices := make([]int, len(batch))
		for i, b := range batch {
			indices[i] = b.Index
		}

		translations := parseTranslationResponse(translated, indices)

		for i, b := range batch {
			resultIdx := batchStart + i
			if text, ok := translations[b.Index]; ok {
				result[resultIdx].Text = text
			}
			// If translation not found for this block, original English text is kept
		}

		processedBlocks += len(batch)
		if progressFn != nil {
			progressFn(float64(processedBlocks) / float64(totalBlocks) * 100)
		}

		slog.Info("Translation batch completed",
			"batch", fmt.Sprintf("%d-%d", batch[0].Index, batch[len(batch)-1].Index),
			"translated", len(translations),
			"total", len(batch),
		)
	}

	if hasPartialFailure {
		slog.Warn("Translation completed with partial failures — some blocks retain English text",
			"total_blocks", totalBlocks,
		)
	}

	return result, nil
}

// responseLinePattern matches "[N] text" format from Claude's response.
var responseLinePattern = regexp.MustCompile(`^\[(\d+)\]\s*(.+)$`)

// parseTranslationResponse extracts translated text from Claude's response.
// Response format: "[1] 翻譯文字\n[2] 翻譯文字"
// Handles multi-line blocks: continuation lines (no [N] prefix) are appended
// to the most recent indexed block.
func parseTranslationResponse(response string, indices []int) map[int]string {
	result := make(map[int]string)

	// Build index lookup for validation
	validIndices := make(map[int]bool)
	for _, idx := range indices {
		validIndices[idx] = true
	}

	lines := strings.Split(strings.TrimSpace(response), "\n")
	var lastIdx int
	hasLast := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := responseLinePattern.FindStringSubmatch(line)
		if matches != nil {
			idx, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			if validIndices[idx] {
				result[idx] = strings.TrimSpace(matches[2])
				lastIdx = idx
				hasLast = true
			}
		} else if hasLast && validIndices[lastIdx] {
			// Continuation line for multi-line subtitle block
			result[lastIdx] += "\n" + line
		}
	}

	return result
}
