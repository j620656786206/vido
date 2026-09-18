package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/repository"
)

// ─── ASR chunk store (disc-2026-09-generation-resume-b-asr-chunk-store) ────
//
// A long film is transcribed in 600-second chunks, each a separate paid ASR
// call. Until this story, a run that died at chunk 9 of 16 — the batch money
// ceiling, a deadline, a restart — threw the eight transcripts it had already
// paid for away, and the next run paid for them again. Each chunk's TEXT is now
// stored the moment it comes back (≈6 KB, never the ≈19 MB WAV a small NAS
// cannot afford to keep), and the next run sends only the missing chunks.
//
// The rows live in the shared `cache_entries` table (no migration): type
// `asr_chunk`, TTL 30 days as a backstop; a run that wrote its .en.srt deletes
// its own rows. Every cache failure is Warn-only — a stale or unreachable cache
// can never fail a paid run (Rule 13).

const (
	asrChunkType         = "asr_chunk"
	asrChunkTTL          = 30 * 24 * time.Hour
	asrChunkKeyPrefix    = "asrchunk:v1:"
	asrManifestKeyPrefix = "asrchunk:v1:manifest:"
)

// ASRChunkIdentity is everything that decides what the engine would say for
// one chunk. Any field changing means a miss — on purpose: a re-encoded file,
// a different audio track, another language hint, a different chunk grid or
// another engine/model all produce different text.
type ASRChunkIdentity struct {
	MediaID      string
	FileSize     int64 // source media file (osFileIdentity), not the WAV
	FileMTime    int64 // unix seconds, same precision the route cache accepts
	TrackIndex   int
	Lang         string
	Start        int // chunk start offset in seconds
	ChunkSeconds int // the grid the chunk was cut on (0 = the whole file)
	Endpoint     string
}

// asrChunkKey hashes the identity: none of the raw fields leak into the key
// (the media id would otherwise make the cache table a list of what the user
// transcribed).
func asrChunkKey(id ASRChunkIdentity) string {
	material := fmt.Sprintf("%s|%d|%d|%d|%s|%d|%d|%s",
		id.MediaID, id.FileSize, id.FileMTime, id.TrackIndex, id.Lang, id.Start, id.ChunkSeconds, id.Endpoint)
	sum := sha256.Sum256([]byte(material))
	return asrChunkKeyPrefix + hex.EncodeToString(sum[:])
}

// asrManifestKey is per file and engine only: the manifest DESCRIBES the
// track, language and chunk grid the stored chunks were cut on.
func asrManifestKey(mediaID string, size, mtime int64, endpoint string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d|%s", mediaID, size, mtime, endpoint)))
	return asrManifestKeyPrefix + hex.EncodeToString(sum[:])
}

// asrChunkValue is one stored chunk: both the hallucination-filtered SRT and
// the unfiltered one, so a hit feeds guardAgainstEmptyTranscript exactly as a
// fresh transcribeOne would.
type asrChunkValue struct {
	Filtered   string `json:"filtered"`
	Unfiltered string `json:"unfiltered"`
}

// decodeASRChunkValue treats anything unreadable as a miss, never an error.
func decodeASRChunkValue(raw string) (asrChunkValue, bool) {
	var v asrChunkValue
	if raw == "" || json.Unmarshal([]byte(raw), &v) != nil {
		return asrChunkValue{}, false
	}
	return v, true
}

// asrManifest records which chunk starts are stored for a file, so the
// estimate (story C) learns "5 of 16 already paid for" from ONE read and the
// batch can tell a film with progress from one without.
type asrManifest struct {
	Track        int    `json:"track"`
	Lang         string `json:"lang"`
	ChunkSeconds int    `json:"chunk_seconds"`
	Done         []int  `json:"done"`
}

func decodeASRManifest(raw string) (asrManifest, bool) {
	var m asrManifest
	if raw == "" || json.Unmarshal([]byte(raw), &m) != nil {
		return asrManifest{}, false
	}
	return m, true
}

// ASRChunkStore is the narrow port the transcription service needs (Rule 11).
// A miss is expressed by absence from the GetMany result.
type ASRChunkStore interface {
	GetMany(ctx context.Context, keys []string) (map[string]string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// asrChunkStoreRepository adapts repository.CacheRepositoryInterface to the
// port, tagging every row `asr_chunk` (the routeCacheRepository precedent).
type asrChunkStoreRepository struct {
	cache repository.CacheRepositoryInterface
}

// NewASRChunkStore wires the chunk store onto the shared `cache_entries` table.
func NewASRChunkStore(cache repository.CacheRepositoryInterface) ASRChunkStore {
	return &asrChunkStoreRepository{cache: cache}
}

func (r *asrChunkStoreRepository) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
	entries, err := r.cache.GetMany(ctx, keys)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(entries))
	for key, entry := range entries {
		out[key] = entry.Value
	}
	return out, nil
}

func (r *asrChunkStoreRepository) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.cache.Set(ctx, key, value, asrChunkType, ttl)
}

func (r *asrChunkStoreRepository) Delete(ctx context.Context, key string) error {
	return r.cache.Delete(ctx, key)
}

var _ ASRChunkStore = (*asrChunkStoreRepository)(nil)

// asrEndpointFingerprinter is the optional seam an ASR provider implements to
// name the engine its transcripts came from (ASRProviderHolder does). A
// provider without it is taken to be the hosted default.
type asrEndpointFingerprinter interface {
	EndpointFingerprint() string
}

// asrEndpointOf returns the key material naming the engine — never the API
// key (the holder's private fingerprint carries it; this one must not).
func asrEndpointOf(asr ai.ASRProvider) string {
	if f, ok := asr.(asrEndpointFingerprinter); ok {
		return f.EndpointFingerprint()
	}
	return "|" + ai.WhisperModel
}
