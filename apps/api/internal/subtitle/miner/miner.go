// Package miner wires 加速器② (story sub-7-5b) into the app: which shows,
// which episodes, which files, which glossary scope, and the insert-only
// write. The algorithm lives in internal/subtitle/mine; this package is the
// only place that touches repositories. It sits beside subtitle rather than
// inside it because mine imports subtitle (ParseSRT), and subtitle importing
// mine back would be a cycle.
package miner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/mine"
)

// fansubSignature marks a sidecar as a fan-group release rather than an
// official subtitle (sub-7-5a finding: Scorpion S01's `.zh-TW.hi.srt` files
// were 人人影視 / ZiMuZu, with inconsistent names and a drifted timeline).
// Such a file teaches the wrong thing, so it is skipped and reported.
var fansubSignature = regexp.MustCompile(`字幕組|字幕组|人人影視|人人影视|ZiMuZu|YYeTs|SubHD|射手網|射手网|翻譯[:：]|翻译[:：]|校對[:：]|校对[:：]|時間軸[:：]|时间轴[:：]`)

// fansubSniffBytes is how much of a sidecar is read to look for a signature
// — credits sit at the very top (or, for one group, around minute ten, which
// the 4 KB head still covers because the preceding cues are short).
const fansubSniffBytes = 4096

// mineVideoExts is the set of container extensions the miner treats as an
// episode file when it has to find sidecars next to it.
var mineVideoExts = map[string]bool{".mkv": true, ".mp4": true, ".avi": true, ".mov": true, ".m4v": true, ".ts": true, ".wmv": true}

// MineEpisodeReport is one episode's outcome inside a MineResult.
type MineEpisodeReport struct {
	EpisodeID string `json:"episode_id"`
	File      string `json:"file"`
	ZhSource  string `json:"zh_source,omitempty"`
	EnSource  string `json:"en_source,omitempty"`
	Segments  int    `json:"segments,omitempty"`
	Skipped   string `json:"skipped,omitempty"`
}

// MineResult is what one show's run produced.
//
// [@contract-v1] — served by GET /subtitles/glossary/mine (last run) and
// POST /subtitles/glossary/mine (synchronous single-show run).
type MineResult struct {
	SeriesID      string              `json:"series_id"`
	Title         string              `json:"title"`
	Scope         string              `json:"scope"`
	EpisodesTotal int                 `json:"episodes_total"`
	EpisodesUsed  int                 `json:"episodes_used"`
	FansubSkipped int                 `json:"fansub_skipped"`
	TermsFound    int                 `json:"terms_found"`
	TermsInserted int                 `json:"terms_inserted"`
	Terms         []mine.Term         `json:"terms,omitempty"`
	Episodes      []MineEpisodeReport `json:"episodes,omitempty"`
	StartedAt     time.Time           `json:"started_at"`
	FinishedAt    time.Time           `json:"finished_at"`
	Error         string              `json:"error,omitempty"`
}

// MineStatus is the miner's observable state: whether a run is in flight and
// what the last run (manual or post-scan) produced per show.
//
// [@contract-v1].
type MineStatus struct {
	Running    bool         `json:"running"`
	RunningFor string       `json:"running_for,omitempty"` // series id, or "partial" for the library sweep
	LastRunAt  *time.Time   `json:"last_run_at,omitempty"`
	Results    []MineResult `json:"results"`
}

// ErrMinerBusy is returned when a run is already in flight.
var ErrMinerBusy = errors.New("official-subtitle miner: a run is already in progress")

// mineEpisodeLister / mineSeriesReader / mineGlossaryRepo are the narrow
// repository surfaces the miner needs (Rule 11).
type mineEpisodeLister interface {
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error)
}

type mineSeriesReader interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
	List(ctx context.Context, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error)
}

type mineGlossaryRepo interface {
	ListByScope(ctx context.Context, scope string) ([]models.GlossaryTerm, error)
	InsertIfAbsent(ctx context.Context, term *models.GlossaryTerm) (bool, error)
}

// OfficialSubtitleMiner runs 加速器② over the library.
type OfficialSubtitleMiner struct {
	episodes  mineEpisodeLister
	series    mineSeriesReader
	scopes    subtitle.GlossaryScopeResolver
	glossary  mineGlossaryRepo
	prober    subtitle.TechProber     // nil = sidecars only
	extractor subtitle.TrackExtractor // nil = sidecars only
	logger    *slog.Logger
	now       func() time.Time
	tmpDir    string

	mu         sync.Mutex
	running    bool
	runningFor string
	lastRunAt  *time.Time
	results    []MineResult
}

// NewOfficialSubtitleMiner wires the miner. prober and extractor may be nil
// (the miner then reads sidecar files only — embedded tracks need ffmpeg).
func NewOfficialSubtitleMiner(episodes mineEpisodeLister, series mineSeriesReader, scopes subtitle.GlossaryScopeResolver, glossary mineGlossaryRepo, prober subtitle.TechProber, extractor subtitle.TrackExtractor, logger *slog.Logger) *OfficialSubtitleMiner {
	if logger == nil {
		logger = slog.Default()
	}
	return &OfficialSubtitleMiner{
		episodes: episodes, series: series, scopes: scopes, glossary: glossary,
		prober: prober, extractor: extractor,
		logger: logger.With("component", "official_subtitle_miner"),
		now:    time.Now,
	}
}

// WithClock replaces the clock (Rule 23).
func (m *OfficialSubtitleMiner) WithClock(now func() time.Time) *OfficialSubtitleMiner {
	m.now = now
	return m
}

// Status reports the in-flight flag and the last results.
func (m *OfficialSubtitleMiner) Status() MineStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := MineStatus{Running: m.running, RunningFor: m.runningFor, LastRunAt: m.lastRunAt, Results: append([]MineResult(nil), m.results...)}
	if out.Results == nil {
		out.Results = []MineResult{}
	}
	return out
}

func (m *OfficialSubtitleMiner) begin(what string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return false
	}
	m.running, m.runningFor = true, what
	return true
}

func (m *OfficialSubtitleMiner) finish(results []MineResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	m.running, m.runningFor = false, ""
	m.lastRunAt = &now
	m.results = results
}

// MineSeries runs one show synchronously and records the result as the last
// run. ErrMinerBusy when another run is in flight.
func (m *OfficialSubtitleMiner) MineSeries(ctx context.Context, seriesID string) (MineResult, error) {
	if !m.begin(seriesID) {
		return MineResult{}, ErrMinerBusy
	}
	res := m.mineOne(ctx, seriesID)
	m.finish([]MineResult{res})
	if res.Error != "" {
		return res, errors.New(res.Error)
	}
	return res, nil
}

// ScanCallback is the post-scan hook: a background sweep over every show
// that looks partial (some episodes carry an official zh sidecar, some do
// not). Pure local work, $0; failures are logged, never surfaced.
func (m *OfficialSubtitleMiner) ScanCallback() func() {
	return func() {
		if err := m.StartPartial(context.Background()); err != nil && !errors.Is(err, ErrMinerBusy) {
			m.logger.Warn("post-scan official-subtitle mining failed to start", "error", err)
		}
	}
}

// StartPartial claims the miner SYNCHRONOUSLY and runs the library sweep in
// the background. Claiming before returning is the point: a client that
// POSTs and immediately GETs the status must see running=true, not a stale
// idle state that makes the button flash back to clickable.
func (m *OfficialSubtitleMiner) StartPartial(ctx context.Context) error {
	if !m.begin("partial") {
		return ErrMinerBusy
	}
	go func() {
		results, err := m.runPartial(ctx)
		if err != nil {
			m.logger.Warn("official-subtitle sweep stopped", "error", err)
		}
		m.finish(results)
	}()
	return nil
}

// MinePartial sweeps the library for partial shows and mines each,
// synchronously. The partial test is CHEAP on purpose — sidecar file names
// only, no ffprobe — so a scan of a thousand episodes does not probe a
// thousand files; a show whose only Chinese is an embedded track is reached
// through the single-show endpoint instead.
func (m *OfficialSubtitleMiner) MinePartial(ctx context.Context) ([]MineResult, error) {
	if !m.begin("partial") {
		return nil, ErrMinerBusy
	}
	results, err := m.runPartial(ctx)
	m.finish(results)
	return results, err
}

func (m *OfficialSubtitleMiner) runPartial(ctx context.Context) ([]MineResult, error) {
	ids, err := m.partialSeries(ctx)
	if err != nil {
		return nil, err
	}
	var results []MineResult
	for _, id := range ids {
		if ctx.Err() != nil {
			return results, ctx.Err()
		}
		results = append(results, m.mineOne(ctx, id))
	}
	return results, nil
}

// partialSeries lists series ids where at least one episode has an official
// zh sidecar and at least one has none.
func (m *OfficialSubtitleMiner) partialSeries(ctx context.Context) ([]string, error) {
	var ids []string
	for page := 1; ; page++ {
		series, pg, err := m.series.List(ctx, repository.ListParams{Page: page, PageSize: 200, SortBy: "title", SortOrder: "asc"})
		if err != nil {
			return nil, fmt.Errorf("list series: %w", err)
		}
		for _, s := range series {
			eps, err := m.episodes.FindBySeriesID(ctx, s.ID)
			if err != nil {
				m.logger.Warn("partial check: episodes unreadable", "series_id", s.ID, "error", err)
				continue
			}
			withZh, without := 0, 0
			for _, e := range eps {
				if !e.FilePath.Valid || e.FilePath.String == "" {
					continue
				}
				if len(sidecarsOf(e.FilePath.String, mine.IsOfficialZhSidecar)) > 0 {
					withZh++
				} else {
					without++
				}
			}
			if withZh > 0 && without > 0 {
				ids = append(ids, s.ID)
			}
		}
		if pg == nil || page >= pg.TotalPages || len(series) == 0 {
			break
		}
	}
	return ids, nil
}

// mineOne does the whole job for one show and never panics the caller: every
// failure lands in MineResult.Error.
func (m *OfficialSubtitleMiner) mineOne(ctx context.Context, seriesID string) MineResult {
	res := MineResult{SeriesID: seriesID, StartedAt: m.now()}
	defer func() { res.FinishedAt = m.now() }()

	s, err := m.series.FindByID(ctx, seriesID)
	if err != nil {
		res.Error = fmt.Sprintf("series %s: %v", seriesID, err)
		return res
	}
	res.Title = s.Title
	scope, err := m.scopes.Resolve(ctx, seriesID)
	if err != nil {
		res.Error = fmt.Sprintf("resolve scope: %v", err)
		return res
	}
	res.Scope = scope

	eps, err := m.episodes.FindBySeriesID(ctx, seriesID)
	if err != nil {
		res.Error = fmt.Sprintf("list episodes: %v", err)
		return res
	}
	sort.SliceStable(eps, func(i, j int) bool {
		if eps[i].SeasonNumber != eps[j].SeasonNumber {
			return eps[i].SeasonNumber < eps[j].SeasonNumber
		}
		return eps[i].EpisodeNumber < eps[j].EpisodeNumber
	})

	tmp, err := os.MkdirTemp(m.tmpDir, "vido-mine-")
	if err != nil {
		res.Error = fmt.Sprintf("temp dir: %v", err)
		return res
	}
	defer os.RemoveAll(tmp)

	var all []mine.Segment
	for _, e := range eps {
		if !e.FilePath.Valid || e.FilePath.String == "" {
			continue
		}
		res.EpisodesTotal++
		rep := MineEpisodeReport{EpisodeID: e.ID, File: filepath.Base(e.FilePath.String)}
		segs, fansubs := m.mineEpisode(ctx, e.FilePath.String, tmp, &rep)
		res.FansubSkipped += fansubs
		if rep.Skipped == "" {
			res.EpisodesUsed++
			all = append(all, segs...)
		}
		res.Episodes = append(res.Episodes, rep)
	}

	known, err := m.knownRenderings(ctx, scope)
	if err != nil {
		m.logger.Warn("existing glossary unreadable — mining without it", "scope", scope, "error", err)
	}
	terms := mine.Mine(all, mine.Options{Known: known})
	res.Terms = terms
	res.TermsFound = len(terms)
	for _, t := range terms {
		if t.How == "known" {
			continue // already in the glossary; verified, nothing to write
		}
		inserted, err := m.glossary.InsertIfAbsent(ctx, &models.GlossaryTerm{
			MediaID: seriesID,
			Scope:   scope,
			TermSrc: t.Src,
			TermZh:  t.Zh,
			Source:  models.GlossarySourceOfficialSubtitle,
			// Confirmed stays false: the professionals' rendering is a strong
			// default, and the user still gets to approve it in the panel
			// (Scorpion's "official" files turned out to be a fan group's).
		})
		if err != nil {
			m.logger.Warn("glossary insert failed", "scope", scope, "term", t.Src, "error", err)
			continue
		}
		if inserted {
			res.TermsInserted++
		}
	}
	m.logger.Info("official-subtitle mining finished",
		"series_id", seriesID, "title", s.Title, "scope", scope,
		"episodes_used", res.EpisodesUsed, "episodes_total", res.EpisodesTotal,
		"fansub_skipped", res.FansubSkipped, "terms_found", res.TermsFound, "terms_inserted", res.TermsInserted)
	return res
}

// knownRenderings is the show's existing glossary (TMDb-seeded, manual,
// harvested), handed to the miner as the trusted set to verify first.
func (m *OfficialSubtitleMiner) knownRenderings(ctx context.Context, scope string) (map[string]string, error) {
	rows, err := m.glossary.ListByScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	known := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Source == models.GlossarySourceOfficialSubtitle && !r.Confirmed {
			continue // our own earlier guess is not evidence
		}
		known[r.TermSrc] = r.TermZh
	}
	return known, nil
}

// mineEpisode loads one episode's two sides and aligns them. It returns the
// segments and how many zh sidecars were skipped as fan-group files; rep is
// filled with what happened.
func (m *OfficialSubtitleMiner) mineEpisode(ctx context.Context, media, tmp string, rep *MineEpisodeReport) ([]mine.Segment, int) {
	zhSidecars := sidecarsOf(media, mine.IsOfficialZhSidecar)
	enSidecars := sidecarsOf(media, mine.IsEnglishSidecar)

	var official []string
	fansubs := 0
	for _, sc := range zhSidecars {
		if looksLikeFansub(filepath.Join(filepath.Dir(media), sc)) {
			fansubs++
			continue
		}
		official = append(official, sc)
	}

	var tracks []services.SubtitleTrack
	needProbe := (len(official) == 0 && fansubs == 0) || len(enSidecars) == 0
	if needProbe && m.prober != nil {
		if info, err := m.prober.Probe(ctx, media); err == nil && info != nil {
			tracks = info.SubtitleTracks
		}
	}
	src := mine.Classify(media, append(official, enSidecars...), tracks)
	if !src.Usable() {
		switch {
		case fansubs > 0 && !src.HasZh():
			rep.Skipped = "only fan-group zh subtitles"
		case src.HasZh():
			rep.Skipped = "has zh, no English text track"
		default:
			rep.Skipped = "no official zh subtitle"
		}
		return nil, fansubs
	}
	zh, zhFrom, err := m.loadSide(ctx, media, tmp, src.ZhSidecars, src.ZhStreams)
	if err != nil {
		rep.Skipped = "zh: " + err.Error()
		return nil, fansubs
	}
	en, enFrom, err := m.loadSide(ctx, media, tmp, src.EnSidecars, src.EnStreams)
	if err != nil {
		rep.Skipped = "en: " + err.Error()
		return nil, fansubs
	}
	segs := mine.Align(en, zh)
	rep.ZhSource, rep.EnSource, rep.Segments = zhFrom, enFrom, len(segs)
	return segs, fansubs
}

func (m *OfficialSubtitleMiner) loadSide(ctx context.Context, media, tmp string, sidecars []string, streams []int) ([]mine.Cue, string, error) {
	dir := filepath.Dir(media)
	for _, sc := range sidecars {
		cues, err := mine.LoadCues(filepath.Join(dir, sc))
		if err == nil && len(cues) > 0 {
			return cues, sc, nil
		}
	}
	if len(streams) == 0 {
		return nil, "", errors.New("no readable sidecar and no embedded track")
	}
	if m.extractor == nil {
		return nil, "", errors.New("embedded track needs ffmpeg")
	}
	out, err := m.extractor.Extract(ctx, media, tmp, streams[:1])
	if err != nil {
		return nil, "", err
	}
	path, ok := out[streams[0]]
	if !ok {
		return nil, "", fmt.Errorf("stream %d not extracted", streams[0])
	}
	cues, err := mine.LoadCues(path)
	if err != nil {
		return nil, "", err
	}
	return cues, fmt.Sprintf("embedded stream %d", streams[0]), nil
}

// sidecarsOf lists the subtitle files beside media whose name passes keep.
func sidecarsOf(media string, keep func(string) bool) []string {
	dir := filepath.Dir(media)
	stem := strings.TrimSuffix(filepath.Base(media), filepath.Ext(media))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == filepath.Base(media) || !strings.HasPrefix(name, stem+".") {
			continue
		}
		if mineVideoExts[strings.ToLower(filepath.Ext(name))] {
			continue
		}
		if keep(name) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// looksLikeFansub sniffs the head of a subtitle file for a fan-group credit.
func looksLikeFansub(path string) bool {
	f, err := os.Open(path) //nolint:gosec // library path
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, fansubSniffBytes)
	n, _ := f.Read(buf)
	return fansubSignature.Match(buf[:n])
}
