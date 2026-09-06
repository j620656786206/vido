package services

import (
	"context"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// GlossarySeeder plants a show's characters and actors into `show_glossary`
// the moment enrichment matches it to TMDb (sub-7-3, eval-1 加速器①). Before
// this, every one of the 261 production glossary rows had come out of a
// subtitle run — the first episode was always translated with an EMPTY
// glossary, and whether "Walter White" stayed one name across a season
// depended on the model getting lucky in episode one.
//
// Seeds are `source=metadata`, `confirmed=0`, inserted only where the term
// is absent: a user's edit (`manual`) or confirmation is never touched, and a
// re-scan is a no-op (AC #4). The same interface is meant to be reused by
// sub-7-5 (official-subtitle mining), which differs only in where the pairs
// come from.
type GlossarySeeder struct {
	credits GlossaryCreditsClient
	repo    glossarySeedRepo
	opencc  OpenCCConverter // the official C++ OpenCC the subtitle pipeline uses; nil/unavailable = store as returned
	logger  *slog.Logger
	now     func() time.Time

	// attempted remembers, per shared scope, until WHEN EnsureSeeded may skip
	// the work. It is process-local on purpose: the durable answer is the
	// seed mark row (one primary-key lookup); this map only turns the steady
	// state into a map hit and holds the back-off after a failure.
	mu        sync.Mutex
	attempted map[string]time.Time
}

// GlossaryCreditsClient is the narrow TMDb surface the seeder needs — the
// raw client satisfies it (see tmdb/credits.go for why it is not on
// ClientInterface).
type GlossaryCreditsClient interface {
	GetMovieCreditsWithLanguage(ctx context.Context, movieID int, language string) (*tmdb.MovieCredits, error)
	GetTVAggregateCreditsWithLanguage(ctx context.Context, tvID int, language string) (*tmdb.TVAggregateCredits, error)
}

// glossarySeedRepo is the repository surface the seeder needs: the term
// write and the durable per-scope seed mark.
type glossarySeedRepo interface {
	InsertIfAbsent(ctx context.Context, term *models.GlossaryTerm) (bool, error)
	IsScopeSeeded(ctx context.Context, scope string) (bool, error)
	MarkScopeSeeded(ctx context.Context, scope string, seeded int) error
}

// CastPair is one seed candidate: the same credit in the source language and
// in zh-TW. Kind is "character" or "actor" — kept for logging/tests only, the
// glossary itself does not distinguish.
type CastPair struct {
	Kind string
	Src  string
	Zh   string
}

// SeedResult is what one seeding pass did (AC #5: logged per scope).
type SeedResult struct {
	Seeded  int
	Skipped int // filtered as noise, untranslated, or already present
	Failed  int // repository errors (logged individually)
}

const (
	glossarySeedSourceLanguage = "en-US"
	glossarySeedTargetLanguage = "zh-TW"

	// enrichmentCreditsCastLimit caps how many cast rows are STORED on the
	// media row. TMDb returns 50–200 for a long series; the UI shows five,
	// the NFO writes them all, and the translator context takes the first
	// MetadataCastLimit. Thirty keeps credits JSON small without cutting
	// anything a consumer reads.
	enrichmentCreditsCastLimit = 30

	// glossarySeedMaxTermRunes rejects anything that is not a name (AC #3).
	glossarySeedMaxTermRunes = 40

	// glossarySeedRetryAfter is how long EnsureSeeded waits before re-trying
	// a scope whose TMDb fetch FAILED (outage, 429). A scope that was fetched
	// fine but yielded nothing is not retried for the life of the process.
	glossarySeedRetryAfter = time.Hour
	// glossarySeedInFlight guards the window while one goroutine is seeding a
	// scope: a concurrent Resolve of the same scope skips instead of racing.
	glossarySeedInFlight = 5 * time.Minute
	// glossarySeedTimeout bounds one seeding pass. Resolve is on the read
	// path of every glossary consumer (the HTTP panel included); a degraded
	// TMDb must cost seconds, not the client's two 30 s timeouts back to back.
	glossarySeedTimeout = 20 * time.Second
)

// NewGlossarySeeder wires the seeder. opencc is the same converter the
// subtitle safety net uses (one dictionary set for the whole product — a
// glossary converted by a different OpenCC build than the subtitles would
// disagree with them on phrase choices); nil is tolerated. A nil logger falls
// back to slog.Default.
func NewGlossarySeeder(credits GlossaryCreditsClient, repo glossarySeedRepo, opencc OpenCCConverter, logger *slog.Logger) *GlossarySeeder {
	if logger == nil {
		logger = slog.Default()
	}
	return &GlossarySeeder{
		credits:   credits,
		repo:      repo,
		opencc:    opencc,
		logger:    logger.With("service", "glossary_seeder"),
		now:       time.Now,
		attempted: make(map[string]time.Time),
	}
}

var _ GlossaryScopeSeeder = (*GlossarySeeder)(nil)

// EnsureSeeded implements GlossaryScopeSeeder: the seed-on-first-resolve seam
// (backlog-glossary-seed-existing-library-and-parse-queue). It runs inside
// GlossaryScopeResolver.Resolve, i.e. right before the first translation of
// ANY title — a show the user already owned, one that arrived through the
// download parse queue, one matched by a scan — and plants the TMDb cast
// exactly once per shared drawer. Cheap on the steady state: one primary-key
// lookup of the seed mark (then a map hit for the rest of the process).
//
// Outcomes: a completed pass (even with 0 seeds) is marked durably; a pass
// with failed inserts or a failed fetch is retried after a back-off; a pass
// the CALLER abandoned (context cancelled — the user closed the panel, the
// scan was stopped) is not counted as an attempt at all.
func (s *GlossarySeeder) EnsureSeeded(ctx context.Context, scope, mediaID string) {
	kind, tmdbID, ok := models.ParseSharedGlossaryScope(scope)
	if !ok || s.repo == nil || s.credits == nil {
		return
	}
	if !s.claim(scope) {
		return
	}
	seedCtx, cancel := context.WithTimeout(ctx, glossarySeedTimeout)
	defer cancel()

	seeded, err := s.repo.IsScopeSeeded(seedCtx, scope)
	if err != nil {
		s.settleAfterError(ctx, scope, "glossary seed probe failed", err)
		return
	}
	if seeded {
		s.release(scope, glossarySeedNever)
		return
	}
	_, pairs, err := s.FetchCredits(seedCtx, kind, tmdbID)
	if err != nil {
		s.settleAfterError(ctx, scope, "glossary seed skipped: TMDb credits fetch failed", err)
		return
	}
	res := s.SeedFromCredits(seedCtx, scope, mediaID, pairs)
	if res.Failed > 0 {
		// Some inserts did not land (SQLite busy under a scan burst). Do NOT
		// mark: the next resolve after the back-off re-runs the pass and
		// InsertIfAbsent fills only the gaps.
		s.logger.Warn("glossary seed incomplete; will retry later",
			"scope", scope, "failed", res.Failed, "seeded", res.Seeded, "retry_after", glossarySeedRetryAfter)
		s.release(scope, s.now().Add(glossarySeedRetryAfter))
		return
	}
	if err := s.repo.MarkScopeSeeded(seedCtx, scope, res.Seeded); err != nil {
		s.settleAfterError(ctx, scope, "glossary seed mark failed", err)
		return
	}
	s.release(scope, glossarySeedNever)
}

// settleAfterError decides how long to stay away from a scope after a failed
// pass. A cancelled CALLER is not the scope's fault: forget the attempt so
// the very next resolve (the subtitle run that is about to start) seeds.
func (s *GlossarySeeder) settleAfterError(ctx context.Context, scope, msg string, err error) {
	if ctx.Err() != nil {
		s.logger.Debug(msg+" (caller went away; not counted)", "scope", scope, "error", err)
		s.forget(scope)
		return
	}
	s.logger.Warn(msg+"; will retry later", "scope", scope, "retry_after", glossarySeedRetryAfter, "error", err)
	s.release(scope, s.now().Add(glossarySeedRetryAfter))
}

// glossarySeedNever marks a scope as settled for the life of the process.
var glossarySeedNever = time.Unix(1<<62, 0)

// claim reserves a scope for seeding; false when another attempt is in flight
// or the scope was settled / recently failed.
func (s *GlossarySeeder) claim(scope string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if until, ok := s.attempted[scope]; ok && s.now().Before(until) {
		return false
	}
	s.attempted[scope] = s.now().Add(glossarySeedInFlight)
	return true
}

func (s *GlossarySeeder) release(scope string, until time.Time) {
	s.mu.Lock()
	s.attempted[scope] = until
	s.mu.Unlock()
}

func (s *GlossarySeeder) forget(scope string) {
	s.mu.Lock()
	delete(s.attempted, scope)
	s.mu.Unlock()
}

// FetchCredits implements GlossarySeederInterface.
func (s *GlossarySeeder) FetchCredits(ctx context.Context, mediaType string, tmdbID int64) (*models.Credits, []CastPair, error) {
	if s.credits == nil || tmdbID <= 0 {
		return nil, nil, nil
	}
	id := int(tmdbID)
	switch mediaType {
	case "tv":
		zh, err := s.credits.GetTVAggregateCreditsWithLanguage(ctx, id, glossarySeedTargetLanguage)
		if err != nil {
			return nil, nil, err
		}
		credits := tvCreditsToModel(zh)
		en, err := s.credits.GetTVAggregateCreditsWithLanguage(ctx, id, glossarySeedSourceLanguage)
		if err != nil {
			return credits, nil, err
		}
		return credits, pairTVCredits(en, zh), nil
	default:
		zh, err := s.credits.GetMovieCreditsWithLanguage(ctx, id, glossarySeedTargetLanguage)
		if err != nil {
			return nil, nil, err
		}
		credits := movieCreditsToModel(zh)
		en, err := s.credits.GetMovieCreditsWithLanguage(ctx, id, glossarySeedSourceLanguage)
		if err != nil {
			return credits, nil, err
		}
		return credits, pairMovieCredits(en, zh), nil
	}
}

// SeedFromCredits implements GlossarySeederInterface (AC #1–#5).
func (s *GlossarySeeder) SeedFromCredits(ctx context.Context, scope, mediaID string, pairs []CastPair) SeedResult {
	var res SeedResult
	if s.repo == nil || scope == "" || len(pairs) == 0 {
		return res
	}

	// Pass 1: filter and dedupe (by source term, case-insensitive — the
	// unique index is NOCASE too).
	type candidate struct{ src, zh string }
	seen := make(map[string]struct{}, len(pairs))
	var cands []candidate
	for _, p := range pairs {
		src, zh, ok := normalizeSeedPair(p)
		if !ok {
			res.Skipped++
			continue
		}
		key := strings.ToLower(src)
		if _, dup := seen[key]; dup {
			res.Skipped++
			continue
		}
		seen[key] = struct{}{}
		cands = append(cands, candidate{src: src, zh: zh})
	}

	// Pass 2: one OpenCC call for the whole title (AC #2 簡→繁). TMDb's
	// zh-TW slot is sometimes filled with Simplified by a mainland editor.
	zhs := make([]string, len(cands))
	for i, c := range cands {
		zhs[i] = c.zh
	}
	zhs = s.toTaiwan(scope, zhs)

	// Pass 3: insert-if-absent.
	for i, c := range cands {
		zh := zhs[i]
		if strings.EqualFold(c.src, zh) {
			res.Skipped++
			continue
		}
		inserted, err := s.repo.InsertIfAbsent(ctx, &models.GlossaryTerm{
			MediaID: mediaID,
			Scope:   scope,
			TermSrc: c.src,
			TermZh:  zh,
			Source:  models.GlossarySourceMetadata,
			// Confirmed stays false: a TMDb translation is a good default, not
			// the user's word — it enters the F6 review flow like every other
			// machine source.
		})
		if err != nil {
			res.Failed++
			s.logger.Warn("glossary seed insert failed", "scope", scope, "term", c.src, "error", err)
			continue
		}
		if inserted {
			res.Seeded++
		} else {
			res.Skipped++ // already there — a manual/confirmed row, or a previous pass
		}
	}
	s.logger.Info("glossary seeded from TMDb credits",
		"scope", scope,
		"media_id", mediaID,
		"seeded", res.Seeded,
		"skipped", res.Skipped,
		"failed", res.Failed,
	)
	return res
}

// toTaiwan runs every string through OpenCC s2twp in ONE subprocess call
// (the converter shells out to the official CLI; per-name calls would be
// twenty spawns per title). Lines are the batch delimiter, so any string
// that contains one is passed through untouched. If the converter is
// missing or the round trip loses lines, the input is returned as-is —
// a Simplified name in the review list beats a dropped one.
func (s *GlossarySeeder) toTaiwan(scope string, in []string) []string {
	if len(in) == 0 {
		return in
	}
	if s.opencc == nil || !s.opencc.IsAvailable() {
		s.logger.Warn("OpenCC unavailable; glossary seeds stored as TMDb returned them", "scope", scope)
		return in
	}
	for _, v := range in {
		if strings.ContainsAny(v, "\r\n") {
			return in
		}
	}
	out, err := s.opencc.ConvertS2TWP([]byte(strings.Join(in, "\n")))
	if err != nil {
		s.logger.Warn("OpenCC conversion failed; glossary seeds stored as TMDb returned them", "scope", scope, "error", err)
		return in
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != len(in) {
		s.logger.Warn("OpenCC changed the line count; glossary seeds stored as TMDb returned them", "scope", scope, "in", len(in), "out", len(lines))
		return in
	}
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
		if lines[i] == "" {
			lines[i] = in[i]
		}
	}
	return lines
}

// ---- pairing -------------------------------------------------------------

// pairMovieCredits joins the two language responses by credit id (the same
// person can appear twice under different credits) and expands each cast
// row into an actor pair and a character pair. Only the first
// prompts.MetadataCastLimit rows by TMDb order are considered (AC #2).
func pairMovieCredits(en, zh *tmdb.MovieCredits) []CastPair {
	if en == nil || zh == nil {
		return nil
	}
	zhByCredit := make(map[string]tmdb.CreditCast, len(zh.Cast))
	for _, c := range zh.Cast {
		zhByCredit[c.CreditID] = c
	}
	cast := append([]tmdb.CreditCast(nil), en.Cast...)
	sort.SliceStable(cast, func(i, j int) bool { return cast[i].Order < cast[j].Order })
	if len(cast) > prompts.MetadataCastLimit {
		cast = cast[:prompts.MetadataCastLimit]
	}
	var pairs []CastPair
	for _, c := range cast {
		z, ok := zhByCredit[c.CreditID]
		if !ok {
			continue
		}
		pairs = append(pairs,
			CastPair{Kind: "actor", Src: c.Name, Zh: z.Name},
			CastPair{Kind: "character", Src: c.Character, Zh: z.Character},
		)
	}
	return pairs
}

// pairTVCredits is the aggregate-credits twin: rows are ranked by how many
// episodes the actor is in (a recurring guest with 40 episodes matters more
// than a season-one regular with 8), and every role is paired by credit id.
func pairTVCredits(en, zh *tmdb.TVAggregateCredits) []CastPair {
	if en == nil || zh == nil {
		return nil
	}
	zhName := make(map[int]string, len(zh.Cast))
	zhRole := make(map[string]string)
	for _, c := range zh.Cast {
		zhName[c.ID] = c.Name
		for _, r := range c.Roles {
			zhRole[r.CreditID] = r.Character
		}
	}
	cast := append([]tmdb.AggregateCast(nil), en.Cast...)
	sort.SliceStable(cast, func(i, j int) bool {
		if cast[i].TotalEpisodeCount != cast[j].TotalEpisodeCount {
			return cast[i].TotalEpisodeCount > cast[j].TotalEpisodeCount
		}
		return cast[i].Order < cast[j].Order
	})
	if len(cast) > prompts.MetadataCastLimit {
		cast = cast[:prompts.MetadataCastLimit]
	}
	var pairs []CastPair
	for _, c := range cast {
		if name, ok := zhName[c.ID]; ok {
			pairs = append(pairs, CastPair{Kind: "actor", Src: c.Name, Zh: name})
		}
		for _, r := range c.Roles {
			if character, ok := zhRole[r.CreditID]; ok {
				pairs = append(pairs, CastPair{Kind: "character", Src: r.Character, Zh: character})
			}
		}
	}
	return pairs
}

// ---- storage shape -------------------------------------------------------

func movieCreditsToModel(c *tmdb.MovieCredits) *models.Credits {
	if c == nil {
		return nil
	}
	out := &models.Credits{}
	cast := append([]tmdb.CreditCast(nil), c.Cast...)
	sort.SliceStable(cast, func(i, j int) bool { return cast[i].Order < cast[j].Order })
	for i, m := range cast {
		if i >= enrichmentCreditsCastLimit {
			break
		}
		out.Cast = append(out.Cast, models.CastMember{
			ID: m.ID, Name: m.Name, Character: m.Character, Order: m.Order, ProfilePath: derefString(m.ProfilePath),
		})
	}
	for _, m := range c.Crew {
		if !isKeyCrewJob(m.Job) {
			continue
		}
		out.Crew = append(out.Crew, models.CrewMember{
			ID: m.ID, Name: m.Name, Job: m.Job, Department: m.Department, ProfilePath: derefString(m.ProfilePath),
		})
	}
	return out
}

func tvCreditsToModel(c *tmdb.TVAggregateCredits) *models.Credits {
	if c == nil {
		return nil
	}
	out := &models.Credits{}
	cast := append([]tmdb.AggregateCast(nil), c.Cast...)
	sort.SliceStable(cast, func(i, j int) bool {
		if cast[i].TotalEpisodeCount != cast[j].TotalEpisodeCount {
			return cast[i].TotalEpisodeCount > cast[j].TotalEpisodeCount
		}
		return cast[i].Order < cast[j].Order
	})
	for i, m := range cast {
		if i >= enrichmentCreditsCastLimit {
			break
		}
		var roles []string
		for _, r := range m.Roles {
			if strings.TrimSpace(r.Character) != "" {
				roles = append(roles, r.Character)
			}
		}
		out.Cast = append(out.Cast, models.CastMember{
			ID: m.ID, Name: m.Name, Character: strings.Join(roles, " / "), Order: i, ProfilePath: derefString(m.ProfilePath),
		})
	}
	for _, m := range c.Crew {
		for _, j := range m.Jobs {
			if !isKeyCrewJob(j.Job) {
				continue
			}
			out.Crew = append(out.Crew, models.CrewMember{
				ID: m.ID, Name: m.Name, Job: j.Job, Department: m.Department, ProfilePath: derefString(m.ProfilePath),
			})
			break
		}
	}
	return out
}

// isKeyCrewJob keeps the crew rows the NFO generator and detail page read
// (director / writer / creator); TMDb's full crew list runs to hundreds.
func isKeyCrewJob(job string) bool {
	switch strings.ToLower(strings.TrimSpace(job)) {
	case "director", "writer", "screenplay", "creator", "executive producer", "producer", "story", "novel", "original story":
		return true
	}
	return false
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ---- noise filters (AC #3) ----------------------------------------------

// glossarySeedGenericRoles is the closed list of "characters" that are not
// names — TMDb credits are full of Self / Narrator / Man at Bar. Matched
// case-insensitively after parenthetical qualifiers are stripped. Table-
// driven so a new entry is one line and the test enumerates it.
var glossarySeedGenericRoles = map[string]struct{}{
	"self": {}, "himself": {}, "herself": {}, "themselves": {},
	"narrator": {}, "host": {}, "presenter": {}, "announcer": {}, "voice": {}, "voices": {},
	"additional voices": {}, "various": {}, "unknown": {}, "uncredited": {}, "cameo": {}, "extra": {},
	"guest": {}, "guest star": {}, "special guest": {}, "interviewee": {}, "contestant": {}, "panelist": {},
	"man": {}, "woman": {}, "boy": {}, "girl": {}, "kid": {}, "child": {}, "baby": {},
	"mother": {}, "father": {}, "mom": {}, "dad": {}, "wife": {}, "husband": {}, "son": {}, "daughter": {},
	"friend": {}, "neighbor": {}, "neighbour": {}, "stranger": {}, "customer": {}, "passenger": {},
	"cop": {}, "police officer": {}, "officer": {}, "detective": {}, "soldier": {}, "guard": {}, "security guard": {},
	"doctor": {}, "nurse": {}, "patient": {}, "teacher": {}, "student": {}, "judge": {}, "lawyer": {}, "reporter": {},
	"waiter": {}, "waitress": {}, "bartender": {}, "driver": {}, "taxi driver": {}, "pilot": {}, "receptionist": {},
}

var (
	// glossarySeedParenthetical strips TMDb's role qualifiers: "(voice)",
	// "(uncredited)", "(archive footage)", "(as Bob)" — ASCII or full-width
	// parentheses, since zh-TW editors type either.
	// Matches the INNERMOST group (no parenthesis inside) so stripQualifiers
	// can peel nested ones from the inside out.
	glossarySeedParenthetical = regexp.MustCompile(`\s*[(（][^()（）]*[)）]`)
	// glossarySeedFileShape rejects anything that looks like it came off a
	// filesystem rather than a cast list: extensions, path separators, SxxEyy
	// tags, release-group tokens.
	glossarySeedFileShape = regexp.MustCompile(`(?i)(\.(mkv|mp4|avi|mov|wmv|flv|ts|srt|ass|ssa|sub|nfo|jpg|jpeg|png)\b|[\\/]|\bS\d{1,2}E\d{1,3}\b|\b(1080p|2160p|720p|480p|x264|x265|h\.?264|h\.?265|hevc|bluray|web-?dl|webrip|hdtv)\b)`)
	// glossarySeedNumberedRole rejects "Guard #2", "Thug 3" — an unnamed
	// extra whose "name" is a counter. Anchored on a generic noun so that a
	// real digit-suffixed name (Agent 47, Android 18, Johnny 5) survives.
	glossarySeedNumberedRole = regexp.MustCompile(`(?i)(#\s*\d+$|^(?:guard|thug|goon|henchman|man|woman|boy|girl|kid|child|cop|police officer|officer|soldier|nurse|doctor|waiter|waitress|student|extra|reporter|customer|passenger|prisoner|inmate|villager|citizen|worker|employee|agent smith)\s*#?\s*\d+$)`)
	// glossarySeedMultiRoleSep splits "Walter White / Heisenberg" and the
	// unspaced "Bruce Wayne/Batman" alike — both spellings are common on TMDb.
	glossarySeedMultiRoleSep = regexp.MustCompile(`\s*/\s*`)
)

// normalizeSeedPair applies the noise filters to one pair and returns the
// (not yet script-converted) term to insert. ok=false means "skip, not an
// error". Multi-role strings
// ("Walter White / Heisenberg") are handled by the caller splitting; here a
// slash-joined src whose zh side has a different number of parts is rejected
// rather than guessed.
func normalizeSeedPair(p CastPair) (src, zh string, ok bool) {
	src, ok = stripQualifiers(p.Src)
	if !ok {
		return "", "", false
	}
	zh, ok = stripQualifiers(p.Zh)
	if !ok {
		return "", "", false
	}
	if src == "" || zh == "" {
		return "", "", false
	}
	// A multi-role credit: only seed when both sides agree on the count,
	// and then seed the first role (the others are usually aliases the
	// glossary would confuse with the primary name).
	if strings.Contains(src, "/") || strings.Contains(zh, "/") {
		sp := glossarySeedMultiRoleSep.Split(src, -1)
		zp := glossarySeedMultiRoleSep.Split(zh, -1)
		if len(sp) != len(zp) {
			return "", "", false
		}
		src, zh = strings.TrimSpace(sp[0]), strings.TrimSpace(zp[0])
		if src == "" || zh == "" {
			return "", "", false
		}
	}
	if utf8.RuneCountInString(src) > glossarySeedMaxTermRunes || utf8.RuneCountInString(zh) > glossarySeedMaxTermRunes {
		return "", "", false
	}
	if _, generic := glossarySeedGenericRoles[strings.ToLower(src)]; generic {
		return "", "", false
	}
	if strings.HasPrefix(strings.ToLower(src), "self ") || strings.HasPrefix(strings.ToLower(src), "self-") {
		return "", "", false
	}
	if glossarySeedFileShape.MatchString(src) || glossarySeedFileShape.MatchString(zh) || glossarySeedNumberedRole.MatchString(src) {
		return "", "", false
	}
	// The source side must be a Latin-script term: that is what an English
	// subtitle line will contain and what the glossary is keyed by.
	if !hasASCIILetter(src) || containsHan(src) {
		return "", "", false
	}
	// The zh side must actually be Chinese — TMDb returns the English name
	// when no zh-TW translation exists, and we never fabricate one (AC #2).
	// Script conversion happens later, in one batch (SeedFromCredits).
	if !containsHan(zh) {
		return "", "", false
	}
	return src, zh, true
}

// containsHan reports whether s has at least one CJK ideograph — the cheap
// "is this actually Chinese" test that decides whether TMDb gave us a
// translation or just echoed the English name.
func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// stripQualifiers removes every parenthetical, including nested ones, and
// refuses a term that still carries an unbalanced parenthesis — "Bob)" or
// "Bob (as Robert" would otherwise become a permanent junk glossary row that
// no subtitle line ever matches and that the NOCASE index will not dedupe
// against a later correct "Bob".
func stripQualifiers(s string) (string, bool) {
	for {
		next := glossarySeedParenthetical.ReplaceAllString(s, "")
		if next == s {
			break
		}
		s = next
	}
	if strings.ContainsAny(s, "()（）") {
		return "", false
	}
	return strings.TrimSpace(s), true
}

func hasASCIILetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}
