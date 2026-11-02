package providers

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/nwaples/rardecode/v2"
)

// assrtMaxEntryBytes caps one subtitle read out of a pack. A season of .ass
// files runs a few hundred KB each; anything bigger is not a subtitle.
const assrtMaxEntryBytes = 8 << 20

// archiveEntry is one subtitle file read out of a pack.
type archiveEntry struct {
	name string
	data []byte
}

// extractSubtitle returns the subtitle named want from a .zip or .rar pack.
// Names inside older packs are often GBK-encoded and no longer match the
// detail listing, so when the exact name is absent the same choice rule as
// pickAssrtFile runs over the pack's own subtitle files.
func extractSubtitle(archive []byte, want, mediaFilePath string) ([]byte, error) {
	entries, err := readArchiveSubtitles(archive)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("pack holds no subtitle file")
	}
	for _, e := range entries {
		if e.name == want {
			return e.data, nil
		}
	}

	listing := assrtDetailItem{}
	for i, e := range entries {
		listing.Filelist = append(listing.Filelist, assrtDetailFile{Filename: e.name, URL: strconv.Itoa(i)})
	}
	chosen, err := pickAssrtFile(listing, mediaFilePath)
	if err != nil {
		return nil, err
	}
	i, _ := strconv.Atoi(chosen.URL)
	return entries[i].data, nil
}

func readArchiveSubtitles(archive []byte) ([]archiveEntry, error) {
	switch {
	case bytes.HasPrefix(archive, []byte("PK\x03\x04")):
		return readZipSubtitles(archive)
	case bytes.HasPrefix(archive, []byte("Rar!\x1a\x07")):
		return readRarSubtitles(archive)
	default:
		return nil, errors.New("pack is neither .zip nor .rar")
	}
}

// entryName normalises a member path to its base name (RAR uses backslashes).
func entryName(name string) string {
	return path.Base(strings.ReplaceAll(name, "\\", "/"))
}

func readZipSubtitles(archive []byte) ([]archiveEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open zip pack: %w", err)
	}
	var out []archiveEntry
	for _, f := range zr.File {
		name := entryName(f.Name)
		if f.FileInfo().IsDir() || !hasExt(name, assrtSubtitleExts) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s in zip pack: %w", name, err)
		}
		data, err := io.ReadAll(io.LimitReader(rc, assrtMaxEntryBytes))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s in zip pack: %w", name, err)
		}
		out = append(out, archiveEntry{name: name, data: data})
	}
	return out, nil
}

func readRarSubtitles(archive []byte) ([]archiveEntry, error) {
	rr, err := rardecode.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open rar pack: %w", err)
	}
	var out []archiveEntry
	for {
		hdr, err := rr.Next()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read rar pack: %w", err)
		}
		name := entryName(hdr.Name)
		if hdr.IsDir || !hasExt(name, assrtSubtitleExts) {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(rr, assrtMaxEntryBytes))
		if err != nil {
			return nil, fmt.Errorf("read %s in rar pack: %w", name, err)
		}
		out = append(out, archiveEntry{name: name, data: data})
	}
}
