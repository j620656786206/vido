package services

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
)

// Telling one show from another with the same name
// (disc-2026-10-series-wrong-version-a). Searching TMDb for 「將夜」 lists the
// 2018 drama first and the 2026 animation second, and the series path used to
// take the first hit — while the show's folder already held a tvshow.nfo with
// the right TMDb id and every file name said 2026.

// seriesYearScanLimit bounds how many video file names the year hint reads.
const seriesYearScanLimit = 20

// SetLibraryRoots wires the library root folders. A series whose folder IS a
// root (a flat layout) shares it with every other show, so nothing in it —
// a tvshow.nfo, the folder name, the files — says which show this row is.
func (s *EnrichmentService) SetLibraryRoots(fn func(ctx context.Context) ([]string, error)) {
	s.libraryRoots = fn
}

// seriesOwnDir is the series folder when it belongs to this show alone, or ""
// (no path, a library root, or roots that could not be read).
func (s *EnrichmentService) seriesOwnDir(ctx context.Context, series *models.Series) string {
	if !series.FilePath.Valid || strings.TrimSpace(series.FilePath.String) == "" {
		return ""
	}
	dir := filepath.Clean(series.FilePath.String)
	if s.libraryRoots == nil {
		return dir
	}
	roots, err := s.libraryRoots(ctx)
	if err != nil {
		s.logger.Warn("library roots unreadable; series folder not used for matching",
			"id", series.ID, "dir", dir, "error", err)
		return ""
	}
	for _, r := range roots {
		if r != "" && filepath.Clean(r) == dir {
			return ""
		}
	}
	return dir
}

// readSeriesNFO parses <dir>/tvshow.nfo, or returns nil when there is none or
// it cannot be read (logged — matching falls back to the search).
func (s *EnrichmentService) readSeriesNFO(seriesID, dir string) *NFOData {
	if s.nfoReader == nil || dir == "" {
		return nil
	}
	path := filepath.Join(dir, "tvshow.nfo")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	data, err := s.nfoReader.Parse(path)
	if err != nil {
		s.logger.Warn("tvshow.nfo unreadable; matching by search", "id", seriesID, "nfo", path, "error", err)
		return nil
	}
	if data.MediaType != "" && data.MediaType != "tvshow" {
		s.logger.Warn("tvshow.nfo is not a <tvshow>; ignored", "id", seriesID, "nfo", path, "root", data.MediaType)
		return nil
	}
	return data
}

// enrichSeriesFromNFO matches the series by the TMDb id its tvshow.nfo names
// (or the IMDb id, through TMDb's find). handled is false when the NFO gives
// no usable id or TMDb cannot resolve it: the caller then searches.
func (s *EnrichmentService) enrichSeriesFromNFO(ctx context.Context, series *models.Series, nfo *NFOData) (handled bool, err error) {
	if nfo == nil || s.tmdbService == nil {
		return false, nil
	}
	previous := currentMetadataSource(series.MetadataSource)
	if !models.ShouldOverwrite(previous, models.MetadataSourceNFO) {
		return false, nil
	}

	tmdbID := 0
	if nfo.TMDbID != "" {
		if id, convErr := strconv.Atoi(strings.TrimSpace(nfo.TMDbID)); convErr == nil && id > 0 {
			tmdbID = id
		} else {
			s.logger.Warn("tvshow.nfo has an invalid TMDb id", "id", series.ID, "tmdb_id", nfo.TMDbID)
		}
	}
	if tmdbID == 0 && nfo.IMDbID != "" {
		found, findErr := s.tmdbService.FindByExternalID(ctx, nfo.IMDbID, "imdb_id")
		switch {
		case findErr != nil:
			s.logger.Warn("TMDb find by tvshow.nfo IMDb id failed", "id", series.ID, "imdb_id", nfo.IMDbID, "error", findErr)
		case found == nil || len(found.TVResults) == 0:
			s.logger.Warn("no TMDb show for the tvshow.nfo IMDb id", "id", series.ID, "imdb_id", nfo.IMDbID)
		default:
			tmdbID = found.TVResults[0].ID
		}
	}
	if tmdbID == 0 {
		return false, nil
	}

	details, err := s.tmdbService.GetTVShowDetails(ctx, tmdbID)
	if err != nil || details == nil {
		s.logger.Warn("TMDb details for the tvshow.nfo id failed; matching by search",
			"id", series.ID, "tmdb_id", tmdbID, "error", err)
		return false, nil
	}

	oldID := series.TMDbID
	applyTMDbSeriesDetails(series, details)
	series.MetadataSource = models.NewNullString(string(models.MetadataSourceNFO))
	series.ParseStatus = models.ParseStatusSuccess
	series.UpdatedAt = time.Now()
	credits := s.matchedCredits(ctx, series.ID, "tv", series.TMDbID, previous, models.MetadataSourceNFO)
	if credits == nil && oldID.Valid && oldID.Int64 != series.TMDbID.Int64 {
		// A different show now: never keep the wrong one's cast (the same
		// rule ApplyTMDbMatch follows).
		credits = &models.Credits{}
	}

	if s.userTookOverSeries(ctx, series.ID) {
		return true, nil
	}
	if err := s.persistSeriesMatch(ctx, series, oldID, credits); err != nil {
		return true, err
	}
	s.logger.Info("series matched from tvshow.nfo", "id", series.ID, "tmdb_id", tmdbID)
	return true, nil
}

// seriesYearHint finds the year that tells this show from others with the
// same name, in order: the tvshow.nfo <year>, the folder name, the series
// title (already split by the caller into titleYear), the first episode file
// name. 0 = none.
func (s *EnrichmentService) seriesYearHint(dir string, nfo *NFOData, titleYear int) int {
	if nfo != nil {
		if y, err := strconv.Atoi(strings.TrimSpace(nfo.Year)); err == nil && validYear(y) {
			return y
		}
	}
	if dir != "" {
		if _, y := parser.SplitTrailingYear(filepath.Base(dir)); y > 0 {
			return y
		}
	}
	if titleYear > 0 {
		return titleYear
	}
	if dir != "" {
		return yearFromEpisodeFiles(dir)
	}
	return 0
}

func validYear(y int) bool { return y >= 1900 && y <= 2099 }

// yearFromEpisodeFiles reads the year off the first episode file name that has
// one right before its SxxExx ("Ever.Night.2026.S01E01…"), looking in the
// series folder and then its season folders. Regex only — no AI parse.
func yearFromEpisodeFiles(dir string) int {
	checked := 0
	var seasons []string
	scan := func(d string, collectSeasons bool) int {
		entries, err := os.ReadDir(d)
		if err != nil {
			return 0
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		tv := parser.NewTVParser()
		for _, e := range entries {
			if e.IsDir() {
				if collectSeasons && seasonDirPattern.MatchString(e.Name()) {
					seasons = append(seasons, filepath.Join(d, e.Name()))
				}
				continue
			}
			if !isVideoFile(e.Name()) {
				continue
			}
			if checked >= seriesYearScanLimit {
				return 0
			}
			checked++
			if r := tv.Parse(e.Name()); r != nil && r.Episode > 0 {
				if _, y := parser.SplitTrailingYear(r.Title); y > 0 {
					return y
				}
			}
		}
		return 0
	}
	if y := scan(dir, true); y > 0 {
		return y
	}
	for _, sd := range seasons {
		if y := scan(sd, false); y > 0 {
			return y
		}
	}
	return 0
}

// pickSeriesMatch is the first hit that shares a name with the top hit and
// first aired in the hint year — another version of the same-named show —
// else the top hit, the pre-fix choice. The name guard keeps a year that is
// not the premiere year (a later season's release year in a file name) from
// trading the right show for an unrelated one that happened to air then.
func pickSeriesMatch(items []metadata.MetadataItem, year int) metadata.MetadataItem {
	top := items[0]
	if year <= 0 || top.Year == year {
		return top
	}
	names := seriesNames(top)
	for _, it := range items[1:] {
		if it.Year != year {
			continue
		}
		for n := range seriesNames(it) {
			if names[n] {
				return it
			}
		}
	}
	return top
}

func seriesNames(it metadata.MetadataItem) map[string]bool {
	out := map[string]bool{}
	for _, n := range []string{it.Title, it.OriginalTitle, it.TitleZhTW} {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			out[n] = true
		}
	}
	return out
}
