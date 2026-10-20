package services

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// VidoTranscriptTag names the English speech-recognition SRT when the plain
// <base>.en.srt is already taken: <base>.vido.en.srt. Players read the last
// segment as the language, so it shows up as a second English subtitle; the
// subtitle inventory skips the tag when it reads the language.
const VidoTranscriptTag = "vido"

// saveEnglishTranscript writes the English SRT beside the media file and
// returns the path it went to (disc-2026-10-generate-keeps-user-en-srt-a).
//
// <base>.en.srt is written only when nothing is there yet. Anything already
// at that name — the user's own subtitle, or a file we cannot even stat — is
// left alone and the transcript goes to <base>.vido.en.srt instead: Tears of
// Steel's official 4,764-byte .en.srt was replaced by the transcript before
// this. The write is temp+rename, so a crash never leaves half a file; the
// temp name carries ".tmp." so the subtitle inventory and the miner skip it.
func saveEnglishTranscript(mediaDir, baseName, content string) (string, error) {
	path := filepath.Join(mediaDir, baseName+".en.srt")
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		path = filepath.Join(mediaDir, baseName+"."+VidoTranscriptTag+".en.srt")
	}

	tmp, err := os.CreateTemp(mediaDir, "."+filepath.Base(path)+".tmp.*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	_, werr := tmp.WriteString(content)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	// CreateTemp makes the file 0600; the SRT is read by the media server.
	if err := os.Chmod(tmpPath, 0644); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("rename %s: %w", filepath.Base(path), err)
	}
	return path, nil
}
