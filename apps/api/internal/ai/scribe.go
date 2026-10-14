package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// ElevenLabs Scribe v2 speech-to-text (asr-scribe-a-provider).
//
// Why a second cloud ASR: on the 2026-10 exam (eval/asr-exam-2026-10, the
// official English track of See S01E02 as the answer key) Scribe v2 passed
// three runs out of three — ≤1.2% invented cues, 0.11 s median onset error,
// 6% WER — while whisper-1 passed none (10–18% missed cues, a bear fight
// heard as "Joe! How are you?"). It is also cheaper: $0.22/h against $0.36/h.
//
// What is different from Whisper, and what this file does about it:
//   - Scribe returns a flat WORD list, never segments. Cues are built here
//     (scribeCues): a cue ends at terminal punctuation, at a pause of
//     scribeCueGapSeconds, at a non-speech event, or when it would run past
//     scribeCueMaxSeconds / scribeCueMaxChars. Scribe's own `additional_formats`
//     SRT was tried and rejected: it stretches sound tags over 20 s and breaks
//     mid-phrase ("Hey, face / me!").
//   - Scribe has no no_speech_prob / avg_logprob / compression_ratio. Those
//     three Whisper rules in filterHallucinationsWith simply never fire (the
//     fields are zero); the text rules — music_only, symbol_only, credit_line,
//     prompt_echo, repeat runs — still do. Scribe's own `audio_event` tokens
//     ([bear roaring], [dramatic music]) are the replacement signal: they are
//     dropped from the text and used as cue boundaries.
//   - Billing is per audio second at scribePerMinuteUSD, through the same
//     RecordASRWithRate seam Whisper uses, so the estimate (asr-scribe-b) and
//     the invoice are one number.
//
// NOT sent on purpose: `keyterms` (the name prompt — a 20% surcharge, and the
// Whisper name prompt never fixed a name in six NAS trials), `diarize`,
// `additional_formats`.
const (
	// ScribeAPIURL is the ElevenLabs speech-to-text endpoint.
	ScribeAPIURL = "https://api.elevenlabs.io/v1/speech-to-text"
	// ScribeModel is the batch model the exam passed with.
	ScribeModel = "scribe_v2"
	// ScribeProvider is the lower-case provider name used in errors and the
	// chunk-cache fingerprint.
	ScribeProvider = "elevenlabs"
	// ElevenLabsBillingURL is where the user tops the account up.
	ElevenLabsBillingURL = "https://elevenlabs.io/app/subscription"

	// scribeCueGapSeconds: a silence this long between two words ends a cue.
	// 0.8 s matched the official subtitle's cueing on the exam (0.11 s median).
	scribeCueGapSeconds = 0.8
	// scribeCueMaxSeconds caps one cue's on-screen time; past it the cue is
	// split at the widest pause inside it.
	scribeCueMaxSeconds = 7.0
	// scribeCueMaxChars caps one cue's length (two 42-character subtitle lines).
	scribeCueMaxChars = 84
	// scribeMinCueSeconds is the shortest cue rendered — a one-word "No!" still
	// has to stay on screen long enough to read. Extended only into free time,
	// never into the next cue. Same number as minTightenedCueSeconds.
	scribeMinCueSeconds = 0.6
)

// Scribe API errors. Separate sentinels from Whisper's so a log line or an
// envelope names the provider that actually failed.
var (
	ErrScribeNotConfigured = errors.New("scribe: ElevenLabs API key not configured")
	ErrScribeAPIError      = errors.New("scribe: API error")
	ErrScribeTimeout       = errors.New("scribe: request timed out")
)

// scribeCreditCodes are the status strings ElevenLabs uses for an account
// that cannot pay. ⚠️ Not yet confirmed against a real exhausted-account
// response (the exam never ran the free tier dry); HTTP 402 is matched as
// well so an unforeseen code still stops the run instead of retrying it.
var scribeCreditCodes = []string{"quota_exceeded", "insufficient_credits", "payment_required"}

// ScribeClient transcribes audio with ElevenLabs Scribe v2.
type ScribeClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
	governor   *Governor
	logger     *slog.Logger
	language   string
}

// Compile-time proof the Scribe client serves both seams.
var (
	_ ASRProvider         = (*ScribeClient)(nil)
	_ DetailedTranscriber = (*ScribeClient)(nil)
)

// ScribeOption configures a ScribeClient.
type ScribeOption func(*ScribeClient)

// WithScribeBaseURL points the client at another endpoint (tests, regional
// hosts such as api.eu.residency.elevenlabs.io).
func WithScribeBaseURL(url string) ScribeOption {
	return func(c *ScribeClient) { c.baseURL = url }
}

// WithScribeModel overrides the model id.
func WithScribeModel(model string) ScribeOption {
	return func(c *ScribeClient) {
		if strings.TrimSpace(model) != "" {
			c.model = model
		}
	}
}

// WithScribeHTTPClient sets a custom HTTP client.
func WithScribeHTTPClient(client *http.Client) ScribeOption {
	return func(c *ScribeClient) { c.httpClient = client }
}

// WithScribeTimeout sets the per-attempt timeout.
func WithScribeTimeout(d time.Duration) ScribeOption {
	return func(c *ScribeClient) { c.timeout = d }
}

// WithScribeGovernor injects the shared AI throttle.
func WithScribeGovernor(g *Governor) ScribeOption {
	return func(c *ScribeClient) { c.governor = g }
}

// WithScribeLanguage pins the source language (ISO-639-1) for Transcribe.
func WithScribeLanguage(lang string) ScribeOption {
	return func(c *ScribeClient) { c.language = lang }
}

// NewScribeClient builds a client for the hosted ElevenLabs API.
func NewScribeClient(apiKey string, opts ...ScribeOption) *ScribeClient {
	c := &ScribeClient{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: ScribeAPIURL,
		model:   ScribeModel,
		timeout: 5 * time.Minute,
		logger:  slog.Default().With("service", "scribe"),
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}
	return c
}

// EndpointFingerprint identifies this provider+model for the ASR chunk cache,
// so a chunk Whisper transcribed is never served back as Scribe's.
func (c *ScribeClient) EndpointFingerprint() string {
	return ScribeProvider + "|" + c.model
}

// Governor returns the shared throttle (nil when unthrottled).
func (c *ScribeClient) Governor() *Governor { return c.governor }

// Transcribe uses the client-level language hint.
func (c *ScribeClient) Transcribe(ctx context.Context, audioPath string) (string, error) {
	return c.TranscribeWithLanguage(ctx, audioPath, c.language)
}

// TranscribeWithLanguage transcribes with an explicit ISO-639-1 hint ("" =
// let Scribe detect the language).
func (c *ScribeClient) TranscribeWithLanguage(ctx context.Context, audioPath, lang string) (string, error) {
	detail, err := c.TranscribeDetailed(ctx, audioPath, lang)
	if err != nil {
		return "", err
	}
	return detail.SRT, nil
}

// TranscribeDetailed transcribes one audio file and reports what the
// hallucination filter removed, the same seam the chunking loop reads from
// Whisper.
func (c *ScribeClient) TranscribeDetailed(ctx context.Context, audioPath, lang string) (TranscriptionDetail, error) {
	if c.apiKey == "" {
		return TranscriptionDetail{}, ErrScribeNotConfigured
	}
	audio, err := readScribeAudio(audioPath)
	if err != nil {
		return TranscriptionDetail{}, err
	}
	filename := filepath.Base(audioPath)

	body, err := c.post(ctx, audio, filename, lang)
	if err != nil {
		return TranscriptionDetail{}, err
	}
	resp, err := parseScribeResponse(body)
	if err != nil {
		return TranscriptionDetail{}, err
	}

	// Metered ONCE per file, after the answer is in hand (the Whisper rule).
	// Scribe reports the billed duration itself; the WAV header is the
	// fallback when it does not.
	if b := BudgetFromContext(ctx); b != nil {
		seconds := resp.AudioDurationSecs
		if seconds <= 0 {
			if dur, _, derr := parseWAVInfo(audioPath); derr == nil {
				seconds = dur
			}
		}
		if seconds > 0 {
			b.RecordASRWithRate(seconds, ScribePerMinuteUSD())
		}
	}

	segments := scribeCues(resp.Words)
	if len(segments) == 0 {
		c.logger.Info("transcription returned no speech", "file", filename, "words", len(resp.Words))
		return TranscriptionDetail{Filtered: true}, nil
	}

	kept, dropped := filterHallucinationsWith(segments, !IsMidFileChunk(ctx), ASRPromptFromContext(ctx))
	detail := TranscriptionDetail{
		SRT:          segmentsToSRT(kept),
		Unfiltered:   segmentsToSRT(segments),
		SegmentsIn:   len(segments),
		SegmentsKept: len(kept),
		Filtered:     true,
	}
	for _, d := range dropped {
		c.logger.Debug("hallucination filter dropped a segment",
			"file", filename, "reason", d.Reason,
			"start", d.Segment.Start, "end", d.Segment.End,
			"text", strings.TrimSpace(d.Segment.Text))
	}
	c.logger.Info("Scribe transcription complete",
		"file", filename,
		"language", resp.LanguageCode,
		"words", len(resp.Words),
		"segments_in", detail.SegmentsIn,
		"segments_kept", detail.SegmentsKept,
		"dropped_by_reason", dropReasonCounts(dropped),
		"srt_bytes", len(detail.SRT),
	)
	return detail, nil
}

// readScribeAudio loads the audio with the same size guard as Whisper: the
// chunking layer above sizes uploads for Whisper's 25 MB limit, and an
// oversized file here means it misbehaved — refusing is better than silently
// sending a truncated film.
func readScribeAudio(audioPath string) ([]byte, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("scribe: open audio file: %w", err)
	}
	defer file.Close()
	if info, err := file.Stat(); err == nil && info.Size() > WhisperMaxFileSize {
		return nil, fmt.Errorf("scribe: audio file %q is %d bytes, exceeds the %d-byte upload budget — chunking failed upstream", filepath.Base(audioPath), info.Size(), int64(WhisperMaxFileSize))
	}
	audio, err := io.ReadAll(io.LimitReader(file, WhisperMaxFileSize))
	if err != nil {
		return nil, fmt.Errorf("scribe: read audio data: %w", err)
	}
	return audio, nil
}

// ─── HTTP ───────────────────────────────────────────────────────────────────

func (c *ScribeClient) buildBody(audio []byte, filename, lang string) ([]byte, string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("scribe: create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return nil, "", fmt.Errorf("scribe: copy audio data: %w", err)
	}
	fields := [][2]string{
		{"model_id", c.model},
		{"timestamps_granularity", "word"},
		{"tag_audio_events", "true"},
	}
	if lang != "" {
		fields = append(fields, [2]string{"language_code", lang})
	}
	for _, f := range fields {
		if err := w.WriteField(f[0], f[1]); err != nil {
			return nil, "", fmt.Errorf("scribe: write %s field: %w", f[0], err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("scribe: close writer: %w", err)
	}
	return body.Bytes(), w.FormDataContentType(), nil
}

// post runs one transcription under the shared throttle + per-run budget with
// bounded transient retry. 401/403/4xx fail fast quoting the server; an
// exhausted account stops the run as CreditExhaustedError; 429/5xx and
// per-attempt timeouts retry.
func (c *ScribeClient) post(ctx context.Context, audio []byte, filename, lang string) ([]byte, error) {
	bodyBytes, contentType, err := c.buildBody(audio, filename, lang)
	if err != nil {
		return nil, err
	}
	return governed(ctx, c.governor, "scribe.transcribe", func() ([]byte, error) {
		return retryTransient(ctx, "scribe.transcribe", func() ([]byte, bool, error) {
			attemptCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, c.baseURL, bytes.NewReader(bodyBytes))
			if err != nil {
				return nil, false, fmt.Errorf("scribe: create request: %w", err)
			}
			req.Header.Set("xi-api-key", c.apiKey)
			req.Header.Set("Content-Type", contentType)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return nil, false, fmt.Errorf("%w: %w", ErrScribeTimeout, ctxErr)
				}
				if attemptCtx.Err() == context.DeadlineExceeded {
					return nil, true, ErrScribeTimeout
				}
				return nil, true, fmt.Errorf("%w: %v", ErrScribeAPIError, err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(io.LimitReader(resp.Body, WhisperMaxResponseSize))
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return nil, false, fmt.Errorf("%w: %w", ErrScribeTimeout, ctxErr)
				}
				return nil, true, fmt.Errorf("scribe: read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				c.logger.Warn("Scribe API error", "status_code", resp.StatusCode, "body", string(respBody))
				// Only a 4xx can mean "the account cannot pay"; a proxy's 502
				// page that happens to say "payment required" must retry.
				if resp.StatusCode >= 400 && resp.StatusCode < 500 {
					if credit := scribeCreditExhausted(resp.StatusCode, string(respBody)); credit != nil {
						return nil, false, fmt.Errorf("%w: %w", ErrScribeAPIError, credit)
					}
				}
				return nil, isTransientStatus(resp.StatusCode),
					fmt.Errorf("%w: status %d — %s", ErrScribeAPIError, resp.StatusCode, scribeErrorText(respBody))
			}
			return respBody, false, nil
		})
	})
}

// scribeCreditExhausted recognises an out-of-credit answer (see
// scribeCreditCodes for what is and is not verified).
func scribeCreditExhausted(status int, body string) *CreditExhaustedError {
	lower := strings.ToLower(body)
	for _, code := range scribeCreditCodes {
		if strings.Contains(lower, code) {
			return &CreditExhaustedError{Provider: ScribeProvider, Code: code, BillingURL: ElevenLabsBillingURL}
		}
	}
	if status == http.StatusPaymentRequired {
		return &CreditExhaustedError{Provider: ScribeProvider, Code: "payment_required", BillingURL: ElevenLabsBillingURL}
	}
	return nil
}

// scribeErrorText pulls the human sentence out of an ElevenLabs error body.
// 400 is {"detail":{"status","message"}}; 422 is {"detail":[{"loc","msg"}]}.
// Anything else is returned as-is, trimmed.
func scribeErrorText(body []byte) string {
	var one struct {
		Detail struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(body, &one); err == nil && one.Detail.Message != "" {
		if one.Detail.Status != "" {
			return one.Detail.Status + ": " + one.Detail.Message
		}
		return one.Detail.Message
	}
	var many struct {
		Detail []struct {
			Msg string `json:"msg"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(body, &many); err == nil && len(many.Detail) > 0 {
		msgs := make([]string, 0, len(many.Detail))
		for _, d := range many.Detail {
			if d.Msg != "" {
				msgs = append(msgs, d.Msg)
			}
		}
		if len(msgs) > 0 {
			return strings.Join(msgs, "; ")
		}
	}
	return strings.TrimSpace(string(body))
}

// ─── Response ───────────────────────────────────────────────────────────────

// scribeWord is one token of Scribe's flat word list. Type is "word",
// "spacing" (the whitespace between words, carrying no text of its own worth
// keeping) or "audio_event" ("[bear roaring]").
type scribeWord struct {
	Text  string  `json:"text"`
	Type  string  `json:"type"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type scribeResponse struct {
	LanguageCode        string       `json:"language_code"`
	LanguageProbability float64      `json:"language_probability"`
	Text                string       `json:"text"`
	Words               []scribeWord `json:"words"`
	AudioDurationSecs   float64      `json:"audio_duration_secs"`
}

func parseScribeResponse(body []byte) (*scribeResponse, error) {
	var r scribeResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("%w: scribe: %v", ErrAIInvalidResponse, err)
	}
	// A transcript with no word list at all is a shape we cannot cue from, and
	// a body that names no language and no duration either is not Scribe's
	// answer (an empty `{}` from a proxy must not be cached as silence). An
	// EMPTY word list on a real reply is silence, which is a valid answer.
	if r.Words == nil && strings.TrimSpace(r.Text) != "" {
		return nil, fmt.Errorf("%w: scribe: transcript without words[]", ErrAIInvalidResponse)
	}
	if r.Words == nil && r.LanguageCode == "" && r.AudioDurationSecs == 0 {
		return nil, fmt.Errorf("%w: scribe: empty body", ErrAIInvalidResponse)
	}
	return &r, nil
}

// ─── Cueing ─────────────────────────────────────────────────────────────────

// scribeCues groups Scribe's word list into subtitle cues (whisperSegment is
// the package's cue type; the three Whisper score fields stay zero).
//
// A cue ends when the word ends a sentence (. ? ! and their CJK forms), when
// the next word starts more than scribeCueGapSeconds after this one ended,
// when a non-speech event sits between them, or when adding the next word
// would push the cue past scribeCueMaxSeconds or scribeCueMaxChars. Cues
// shorter than scribeMinCueSeconds are extended toward the next cue's start
// but never into it.
func scribeCues(words []scribeWord) []whisperSegment {
	type cue struct {
		start, end float64
		text       strings.Builder
		width      int
	}
	var cues []*cue
	var cur *cue
	// Scribe says where whitespace belongs with `spacing` tokens: English
	// words are separated by one, CJK characters are not (「我們走吧。」 must
	// not become 「我 們 走 吧 。」). So a space is written only when a spacing
	// token sat between the previous word and this one.
	pendingSpace := false
	flush := func() {
		if cur != nil && cur.text.Len() > 0 {
			cues = append(cues, cur)
		}
		cur = nil
		pendingSpace = false
	}
	for _, w := range words {
		switch w.Type {
		case "audio_event":
			flush()
			continue
		case "spacing":
			pendingSpace = true
			continue
		case "word":
		default:
			continue
		}
		text := strings.TrimSpace(w.Text)
		if text == "" {
			continue
		}
		width := displayWidth(text)
		if cur != nil {
			gap := w.Start - cur.end
			tooLong := w.End-cur.start > scribeCueMaxSeconds
			tooWide := cur.width+1+width > scribeCueMaxChars
			if gap > scribeCueGapSeconds || tooLong || tooWide {
				flush()
			}
		}
		if cur == nil {
			cur = &cue{start: w.Start, end: w.End}
			pendingSpace = false
		}
		if pendingSpace {
			cur.text.WriteByte(' ')
			cur.width++
			pendingSpace = false
		}
		cur.text.WriteString(text)
		cur.width += width
		cur.end = w.End
		if endsSentence(text) {
			flush()
		}
	}
	flush()

	segs := make([]whisperSegment, 0, len(cues))
	for i, c := range cues {
		start, end := c.start, c.end
		// One token Scribe stretched over many seconds (a drawn-out cry, a
		// sound it took for a word) is cut to the cap rather than left to sit
		// on screen through the next scene.
		if end-start > scribeCueMaxSeconds {
			end = start + scribeCueMaxSeconds
		}
		if end-start < scribeMinCueSeconds {
			held := start + scribeMinCueSeconds
			if i+1 < len(cues) && held > cues[i+1].start {
				held = cues[i+1].start
			}
			if held > end {
				end = held
			}
		}
		segs = append(segs, whisperSegment{ID: i, Start: start, End: end, Text: c.text.String()})
	}
	return segs
}

// displayWidth counts a word against scribeCueMaxChars: a CJK character
// takes the room of two Latin letters, so 42 漢字 fill the same two lines
// as 84 letters do.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// scribeAbbreviations end in a period without ending a sentence.
var scribeAbbreviations = map[string]bool{
	"mr.": true, "mrs.": true, "ms.": true, "dr.": true, "st.": true, "jr.": true, "sr.": true,
	"vs.": true, "mt.": true, "lt.": true, "sgt.": true, "capt.": true, "prof.": true, "gen.": true,
	"a.m.": true, "p.m.": true, "u.s.": true, "u.k.": true, "e.g.": true, "i.e.": true,
}

// endsSentence reports whether a word closes a sentence — terminal
// punctuation, possibly followed by a closing quote or bracket.
func endsSentence(word string) bool {
	trimmed := strings.TrimRightFunc(word, func(r rune) bool {
		return r == '"' || r == '\'' || r == ')' || r == ']' || r == '”' || r == '’' || r == '」' || r == '』'
	})
	if trimmed == "" {
		return false
	}
	runes := []rune(trimmed)
	switch runes[len(runes)-1] {
	case '?', '!', '。', '？', '！', '…', '‽':
		return true
	case '.':
		lower := strings.ToLower(trimmed)
		if scribeAbbreviations[lower] {
			return false
		}
		// A single initial ("J." in "J. Smith") does not end a sentence.
		if len(runes) == 2 && unicode.IsLetter(runes[0]) {
			return false
		}
		return true
	}
	return false
}

// ScribePerMinuteUSD is the hosted Scribe v2 price per audio minute
// ($0.22/h, ElevenLabs API pricing 2026-10-14).
func ScribePerMinuteUSD() float64 { return scribePerMinuteUSD.InexactFloat64() }
