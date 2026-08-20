package services

// Story 9R-10b AC #2 — UpdateLibraryRequest.AutoSubtitle is POINTER-optional.
//
// The failure this pins is quiet and expensive: if the field were a plain bool,
// every caller that does not know about the setting — the existing library edit
// form, a script, a future partial update — would send `false` implicitly and
// silently withdraw the user's opt-in on an unrelated save. Consent must only
// change when someone actually asks for it to change.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// stubLibraryRepo implements only what MediaLibraryService.UpdateLibrary uses;
// every other method returns a not-implemented error so an accidental call is
// loud rather than silently empty.
type stubLibraryRepo struct {
	lib     *models.MediaLibrary
	updated *models.MediaLibrary
	created *models.MediaLibrary
}

func (s *stubLibraryRepo) GetByID(_ context.Context, id string) (*models.MediaLibrary, error) {
	if s.lib == nil || s.lib.ID != id {
		return nil, repository.ErrLibraryNotFound
	}
	clone := *s.lib
	return &clone, nil
}

func (s *stubLibraryRepo) Update(_ context.Context, library *models.MediaLibrary) error {
	clone := *library
	s.updated = &clone
	s.lib = &clone
	return nil
}

var errStubNotImplemented = errors.New("stubLibraryRepo: method not used by these tests")

func (s *stubLibraryRepo) Create(_ context.Context, library *models.MediaLibrary) error {
	clone := *library
	s.created = &clone
	s.lib = &clone
	return nil
}
func (s *stubLibraryRepo) GetAll(context.Context) ([]models.MediaLibrary, error) {
	return nil, errStubNotImplemented
}
func (s *stubLibraryRepo) GetAllWithPathsAndCounts(context.Context) ([]models.MediaLibraryWithPaths, error) {
	return nil, errStubNotImplemented
}
func (s *stubLibraryRepo) Delete(context.Context, string) error { return errStubNotImplemented }
func (s *stubLibraryRepo) AddPath(context.Context, *models.MediaLibraryPath) error {
	return errStubNotImplemented
}
func (s *stubLibraryRepo) RemovePath(context.Context, string) error { return errStubNotImplemented }
func (s *stubLibraryRepo) GetPathsByLibraryID(context.Context, string) ([]models.MediaLibraryPath, error) {
	return nil, errStubNotImplemented
}
func (s *stubLibraryRepo) GetAllPaths(context.Context) ([]models.MediaLibraryPath, error) {
	return nil, errStubNotImplemented
}
func (s *stubLibraryRepo) UpdatePathStatus(context.Context, string, models.MediaLibraryPathStatus) error {
	return errStubNotImplemented
}

func newLibraryServiceWith(autoSubtitle bool) (*MediaLibraryService, *stubLibraryRepo) {
	repo := &stubLibraryRepo{lib: &models.MediaLibrary{
		ID:           "lib-1",
		Name:         "我的電影",
		ContentType:  models.ContentTypeMovie,
		AutoSubtitle: autoSubtitle,
	}}
	return NewMediaLibraryService(repo), repo
}

func TestUpdateLibrary_AutoSubtitleOmittedLeavesValueAlone(t *testing.T) {
	for _, existing := range []bool{true, false} {
		svc, repo := newLibraryServiceWith(existing)
		name := "改個名字"

		got, err := svc.UpdateLibrary(context.Background(), "lib-1", UpdateLibraryRequest{Name: &name})
		require.NoError(t, err)

		assert.Equal(t, existing, got.AutoSubtitle,
			"a request that never mentions auto_subtitle must not change it — renaming a library is not consent traffic")
		require.NotNil(t, repo.updated)
		assert.Equal(t, existing, repo.updated.AutoSubtitle, "and the persisted row must agree")
	}
}

func TestUpdateLibrary_AutoSubtitleSetsTrue(t *testing.T) {
	svc, repo := newLibraryServiceWith(false)
	on := true

	got, err := svc.UpdateLibrary(context.Background(), "lib-1", UpdateLibraryRequest{AutoSubtitle: &on})
	require.NoError(t, err)

	assert.True(t, got.AutoSubtitle)
	require.NotNil(t, repo.updated)
	assert.True(t, repo.updated.AutoSubtitle)
}

func TestUpdateLibrary_AutoSubtitleSetsFalse(t *testing.T) {
	svc, repo := newLibraryServiceWith(true)
	off := false

	got, err := svc.UpdateLibrary(context.Background(), "lib-1", UpdateLibraryRequest{AutoSubtitle: &off})
	require.NoError(t, err)

	assert.False(t, got.AutoSubtitle, "withdrawing the opt-in must be possible and must stick")
	require.NotNil(t, repo.updated)
	assert.False(t, repo.updated.AutoSubtitle)
}

// ─── CR M2: the opt-in must survive CREATE, not only UPDATE ───────────────

// TestCreateLibrary_CarriesAutoSubtitle pins the fix for a control that
// rendered, toggled, and then silently discarded its value: the modal's create
// branch dropped autoSubtitle, and neither CreateLibraryRequest carried it, so a
// user could tick the box, press 建立, and get a library that was off — with no
// error and nothing on screen to say so.
func TestCreateLibrary_CarriesAutoSubtitle(t *testing.T) {
	for _, want := range []bool{true, false} {
		repo := &stubLibraryRepo{}
		svc := NewMediaLibraryService(repo)

		got, err := svc.CreateLibrary(context.Background(), CreateLibraryRequest{
			Name:         "我的電影",
			ContentType:  "movie",
			AutoSubtitle: want,
		})
		require.NoError(t, err)

		assert.Equal(t, want, got.AutoSubtitle)
		require.NotNil(t, repo.created)
		assert.Equal(t, want, repo.created.AutoSubtitle, "and the persisted row must agree")
	}
}

func TestCreateLibrary_DefaultsOffWhenUnspecified(t *testing.T) {
	repo := &stubLibraryRepo{}
	svc := NewMediaLibraryService(repo)

	got, err := svc.CreateLibrary(context.Background(), CreateLibraryRequest{Name: "我的影集", ContentType: "series"})
	require.NoError(t, err)

	assert.False(t, got.AutoSubtitle, "a request that never mentions the opt-in must produce an OFF library")
}
