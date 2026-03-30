package qbittorrent

import (
	"log/slog"
	"time"
)

// TorrentStatus represents the normalized status of a torrent.
type TorrentStatus string

const (
	StatusDownloading TorrentStatus = "downloading"
	StatusPaused      TorrentStatus = "paused"
	StatusSeeding     TorrentStatus = "seeding"
	StatusCompleted   TorrentStatus = "completed"
	StatusStalled     TorrentStatus = "stalled"
	StatusError       TorrentStatus = "error"
	StatusQueued      TorrentStatus = "queued"
	StatusChecking    TorrentStatus = "checking"
)

// MapQBState maps qBittorrent internal state strings to our normalized TorrentStatus enum.
// Covers both qBittorrent 4.x (pausedDL/pausedUP) and 5.0+ (stoppedDL/stoppedUP) states.
// Mapping follows the Sonarr/Radarr industry standard.
//
// Reference: https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-5.0)
func MapQBState(state string) TorrentStatus {
	switch state {
	// Downloading: actively transferring or fetching metadata
	case "downloading", "forcedDL", "metaDL", "allocating":
		return StatusDownloading
	// Paused: download not complete, user stopped the torrent
	case "pausedDL",  // qBT 4.x
		"stoppedDL": // qBT 5.0+
		return StatusPaused
	// Completed: download finished, no longer actively seeding
	case "pausedUP",  // qBT 4.x — paused after completing download
		"stoppedUP", // qBT 5.0+ — stopped after completing download
		"stalledUP": // seeding but no peer connections → effectively complete
		return StatusCompleted
	// Seeding: actively uploading to peers
	case "uploading", "forcedUP":
		return StatusSeeding
	// Stalled: download in progress but no peer connections
	case "stalledDL":
		return StatusStalled
	// Queued: waiting in queue
	case "queuedDL", "queuedUP":
		return StatusQueued
	// Checking: verifying data integrity
	case "checkingDL", "checkingUP", "checkingResumeData":
		return StatusChecking
	// Error: torrent has errors or missing files
	case "error", "missingFiles":
		return StatusError
	// Moving: torrent data being relocated (qBT 5.0+)
	case "moving":
		return StatusChecking
	default:
		slog.Warn("Unknown qBittorrent state, defaulting to downloading", "state", state)
		return StatusDownloading
	}
}

// TorrentsFilter defines available filter values for torrent listing.
type TorrentsFilter string

const (
	FilterAll         TorrentsFilter = "all"
	FilterDownloading TorrentsFilter = "downloading"
	FilterPaused      TorrentsFilter = "paused"
	FilterCompleted   TorrentsFilter = "completed"
	FilterSeeding     TorrentsFilter = "seeding"
	FilterErrored     TorrentsFilter = "errored"
)

// TorrentsSort defines available sort fields for torrent listing.
type TorrentsSort string

const (
	SortAddedOn  TorrentsSort = "added_on"
	SortName     TorrentsSort = "name"
	SortProgress TorrentsSort = "progress"
	SortSize     TorrentsSort = "size"
	SortStatus   TorrentsSort = "status" // Sorted server-side (not supported by qBittorrent API)
)

// ListTorrentsOptions configures the torrent list request.
type ListTorrentsOptions struct {
	Filter  TorrentsFilter
	Sort    TorrentsSort
	Reverse bool
}

// Torrent represents a torrent with its current status and progress.
type Torrent struct {
	Hash          string        `json:"hash"`
	Name          string        `json:"name"`
	Size          int64         `json:"size"`
	Progress      float64       `json:"progress"`
	DownloadSpeed int64         `json:"download_speed"`
	UploadSpeed   int64         `json:"upload_speed"`
	ETA           int64         `json:"eta"`
	Status        TorrentStatus `json:"status"`
	AddedOn       time.Time     `json:"added_on"`
	CompletedOn   *time.Time    `json:"completed_on,omitempty"`
	Seeds         int           `json:"seeds"`
	Peers         int           `json:"peers"`
	Downloaded    int64         `json:"downloaded"`
	Uploaded      int64         `json:"uploaded"`
	Ratio         float64       `json:"ratio"`
	SavePath      string        `json:"save_path"`
}

// TorrentDetails extends Torrent with additional properties.
type TorrentDetails struct {
	Torrent
	PieceSize    int64     `json:"piece_size"`
	Comment      string    `json:"comment,omitempty"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreationDate time.Time `json:"creation_date"`
	TotalWasted  int64     `json:"total_wasted"`
	TimeElapsed  int64     `json:"time_elapsed"`
	SeedingTime  int64     `json:"seeding_time"`
	AvgDownSpeed int64     `json:"avg_down_speed"`
	AvgUpSpeed   int64     `json:"avg_up_speed"`
}

// DownloadCounts holds the count of torrents grouped by status.
type DownloadCounts struct {
	All         int `json:"all"`
	Downloading int `json:"downloading"`
	Paused      int `json:"paused"`
	Completed   int `json:"completed"`
	Seeding     int `json:"seeding"`
	Error       int `json:"error"`
}

// Error code for torrent not found.
const ErrCodeTorrentNotFound = "QB_TORRENT_NOT_FOUND"

// qbTorrentInfo is the internal representation of a torrent from the qBittorrent API.
// Field names match the qBittorrent API JSON response (snake_case).
type qbTorrentInfo struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	Progress     float64 `json:"progress"`
	DLSpeed      int64   `json:"dlspeed"`
	UPSpeed      int64   `json:"upspeed"`
	ETA          int64   `json:"eta"`
	State        string  `json:"state"`
	AddedOn      int64   `json:"added_on"`
	CompletionOn int64   `json:"completion_on"`
	NumSeeds     int     `json:"num_seeds"`
	NumLeechs    int     `json:"num_leechs"`
	SavePath     string  `json:"save_path"`
	Downloaded   int64   `json:"downloaded"`
	Uploaded     int64   `json:"uploaded"`
	Ratio        float64 `json:"ratio"`
}

// qbTorrentProperties is the internal representation from /torrents/properties.
type qbTorrentProperties struct {
	SavePath        string `json:"save_path"`
	CreationDate    int64  `json:"creation_date"`
	PieceSize       int64  `json:"piece_size"`
	Comment         string `json:"comment"`
	TotalWasted     int64  `json:"total_wasted"`
	TotalUploaded   int64  `json:"total_uploaded"`
	TotalDownloaded int64  `json:"total_downloaded"`
	Peers           int    `json:"peers"`
	Seeds           int    `json:"seeds"`
	AdditionDate    int64  `json:"addition_date"`
	CompletionDate  int64  `json:"completion_date"`
	CreatedBy       string `json:"created_by"`
	DLSpeedAvg      int64  `json:"dl_speed_avg"`
	UPSpeedAvg      int64  `json:"up_speed_avg"`
	TimeElapsed     int64  `json:"time_elapsed"`
	SeedingTime     int64  `json:"seeding_time"`
}

// mapQBTorrentInfo converts an internal qBittorrent torrent info to our Torrent type.
func mapQBTorrentInfo(qbt qbTorrentInfo) Torrent {
	t := Torrent{
		Hash:          qbt.Hash,
		Name:          qbt.Name,
		Size:          qbt.Size,
		Progress:      qbt.Progress,
		DownloadSpeed: qbt.DLSpeed,
		UploadSpeed:   qbt.UPSpeed,
		ETA:           qbt.ETA,
		Status:        MapQBState(qbt.State),
		AddedOn:       time.Unix(qbt.AddedOn, 0).UTC(),
		Seeds:         qbt.NumSeeds,
		Peers:         qbt.NumLeechs,
		Downloaded:    qbt.Downloaded,
		Uploaded:      qbt.Uploaded,
		Ratio:         qbt.Ratio,
		SavePath:      qbt.SavePath,
	}

	if qbt.CompletionOn > 0 {
		completedOn := time.Unix(qbt.CompletionOn, 0).UTC()
		t.CompletedOn = &completedOn
	}

	return t
}

// mapTorrentDetails combines Torrent and properties into TorrentDetails.
func mapTorrentDetails(torrent *Torrent, props qbTorrentProperties) *TorrentDetails {
	return &TorrentDetails{
		Torrent:      *torrent,
		PieceSize:    props.PieceSize,
		Comment:      props.Comment,
		CreatedBy:    props.CreatedBy,
		CreationDate: time.Unix(props.CreationDate, 0).UTC(),
		TotalWasted:  props.TotalWasted,
		TimeElapsed:  props.TimeElapsed,
		SeedingTime:  props.SeedingTime,
		AvgDownSpeed: props.DLSpeedAvg,
		AvgUpSpeed:   props.UPSpeedAvg,
	}
}
