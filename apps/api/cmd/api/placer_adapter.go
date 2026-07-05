package main

import (
	"github.com/vido/api/internal/subtitle"
)

// subtitlePlacerAdapter bridges *subtitle.Placer to the services.SubtitlePlacer
// interface (Story 9R-10) so the transcription service can place subtitles
// without importing the subtitle package (Rule 19).
type subtitlePlacerAdapter struct {
	placer *subtitle.Placer
}

func (a subtitlePlacerAdapter) PlaceSubtitle(mediaFilePath string, subtitleData []byte, language, format string) (string, error) {
	res, err := a.placer.Place(subtitle.PlaceRequest{
		MediaFilePath: mediaFilePath,
		SubtitleData:  subtitleData,
		Language:      language,
		Format:        format,
	})
	if err != nil {
		return "", err
	}
	return res.SubtitlePath, nil
}
