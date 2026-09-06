package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// backlog-glossary-seed-existing-library-and-parse-queue: the seed happens on
// the FIRST Resolve of a shared scope, whoever triggers it — so a show that
// was already in the library, or arrived through the download parse queue,
// is seeded before its first translation, not only when a scan matches it.

func seedOnResolveFixture(t *testing.T, repo glossarySeedRepo) (*GlossarySeeder, *fakeCreditsClient, *time.Time) {
	t.Helper()
	client := &fakeCreditsClient{
		movie: map[string]*tmdb.MovieCredits{
			"en-US": {Cast: []tmdb.CreditCast{{ID: 1, CreditID: "c1", Name: "Bryan Cranston", Character: "Walter White"}}},
			"zh-TW": {Cast: []tmdb.CreditCast{{ID: 1, CreditID: "c1", Name: "布萊恩·克蘭斯頓", Character: "華特·懷特"}}},
		},
		tv: map[string]*tmdb.TVAggregateCredits{
			"en-US": {Cast: []tmdb.AggregateCast{{ID: 1, Name: "Bryan Cranston", Roles: []tmdb.AggregateRole{{CreditID: "w", Character: "Walter White"}}}}},
			"zh-TW": {Cast: []tmdb.AggregateCast{{ID: 1, Name: "布萊恩·克蘭斯頓", Roles: []tmdb.AggregateRole{{CreditID: "w", Character: "華特·懷特"}}}}},
		},
	}
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	seeder := NewGlossarySeeder(client, repo, &fakeSeedOpenCC{available: true}, slog.Default())
	seeder.now = func() time.Time { return now }
	return seeder, client, &now
}

func TestEnsureSeeded_SeedsOncePerScope(t *testing.T) {
	repo := &fakeSeedInserter{}
	seeder, client, _ := seedOnResolveFixture(t, repo)
	ctx := context.Background()

	seeder.EnsureSeeded(ctx, "tmdb:tv:1396", "series-1")
	require.Len(t, repo.terms, 2, "actor + character")
	assert.Equal(t, "series-1", repo.terms[0].MediaID)
	assert.Equal(t, models.GlossarySourceMetadata, repo.terms[0].Source)
	assert.Len(t, client.calls, 2)
	assert.Equal(t, 1, repo.probes)
	assert.Equal(t, map[string]int{"tmdb:tv:1396": 2}, repo.marks, "durably marked with the seeded count")

	// Second resolve of the same scope: no probe, no fetch, no inserts.
	seeder.EnsureSeeded(ctx, "tmdb:tv:1396", "series-1")
	seeder.EnsureSeeded(ctx, "tmdb:tv:1396", "episode-of-series-1")
	assert.Len(t, repo.terms, 2)
	assert.Len(t, client.calls, 2)
	assert.Equal(t, 1, repo.probes, "settled scopes are a map hit, not a query")
}

func TestEnsureSeeded_LocalScopeAndMalformedScopeAreNoops(t *testing.T) {
	repo := &fakeSeedInserter{}
	seeder, client, _ := seedOnResolveFixture(t, repo)
	for _, scope := range []string{"local:abc", "tmdb:tv:", "tmdb:movie:x", "", "tmdb:tv:0"} {
		seeder.EnsureSeeded(context.Background(), scope, "m")
	}
	assert.Empty(t, client.calls)
	assert.Empty(t, repo.terms)
	assert.Equal(t, 0, repo.probes)
}

func TestEnsureSeeded_AlreadySeededDrawerIsNotFetchedAgain(t *testing.T) {
	// A drawer with a seed mark (previous process) is settled on the first
	// probe: one lookup, then never again — even when the user has since
	// deleted every seeded term (the mark, not the terms, is the memory).
	repo := &fakeSeedInserter{marks: map[string]int{"tmdb:movie:550": 12}}
	seeder, client, _ := seedOnResolveFixture(t, repo)
	seeder.EnsureSeeded(context.Background(), "tmdb:movie:550", "m1")
	seeder.EnsureSeeded(context.Background(), "tmdb:movie:550", "m1")
	assert.Empty(t, client.calls)
	assert.Equal(t, 1, repo.probes)
}

func TestEnsureSeeded_FetchFailureRetriesAfterBackoff(t *testing.T) {
	repo := &fakeSeedInserter{}
	seeder, client, now := seedOnResolveFixture(t, repo)
	client.err = map[string]error{"zh-TW": errors.New("429")}
	ctx := context.Background()

	seeder.EnsureSeeded(ctx, "tmdb:movie:550", "m1")
	assert.Len(t, client.calls, 1)
	assert.Empty(t, repo.terms)

	// Within the backoff: nothing.
	*now = now.Add(glossarySeedRetryAfter - time.Minute)
	seeder.EnsureSeeded(ctx, "tmdb:movie:550", "m1")
	assert.Len(t, client.calls, 1, "no hammering TMDb while it is failing")

	// After the backoff, TMDb is back: the drawer gets seeded.
	*now = now.Add(2 * time.Minute)
	client.err = nil
	seeder.EnsureSeeded(ctx, "tmdb:movie:550", "m1")
	assert.Len(t, client.calls, 3)
	assert.Len(t, repo.terms, 2)
}

func TestEnsureSeeded_NothingToSeedIsSettledForTheProcess(t *testing.T) {
	// A Chinese-origin show: every name is CJK on both sides → zero seeds.
	// Must not be re-fetched on every Resolve for the rest of the process.
	repo := &fakeSeedInserter{}
	seeder, client, now := seedOnResolveFixture(t, repo)
	client.movie = map[string]*tmdb.MovieCredits{
		"en-US": {Cast: []tmdb.CreditCast{{ID: 1, CreditID: "c1", Name: "周潤發", Character: "小馬哥"}}},
		"zh-TW": {Cast: []tmdb.CreditCast{{ID: 1, CreditID: "c1", Name: "周潤發", Character: "小馬哥"}}},
	}
	ctx := context.Background()
	seeder.EnsureSeeded(ctx, "tmdb:movie:11902", "m1")
	assert.Empty(t, repo.terms)
	*now = now.Add(48 * time.Hour)
	seeder.EnsureSeeded(ctx, "tmdb:movie:11902", "m1")
	assert.Len(t, client.calls, 2, "fetched exactly once")
	assert.Equal(t, map[string]int{"tmdb:movie:11902": 0}, repo.marks, "marked with 0 so a restart does not re-fetch either")
}

func TestEnsureSeeded_CancelledCallerIsNotCountedAsAnAttempt(t *testing.T) {
	// The user opened the glossary panel and navigated away mid-fetch (Gin
	// cancels the request context). That must not lock the show out of
	// seeding for an hour — its first subtitle run may be seconds away.
	repo := &fakeSeedInserter{}
	seeder, client, _ := seedOnResolveFixture(t, repo)
	ctx, cancel := context.WithCancel(context.Background())
	client.err = map[string]error{"zh-TW": context.Canceled}
	cancel()
	seeder.EnsureSeeded(ctx, "tmdb:tv:1396", "series-1")
	assert.Len(t, client.calls, 1)
	assert.Empty(t, repo.marks)

	// The very next resolve (a live caller) seeds.
	client.err = nil
	seeder.EnsureSeeded(context.Background(), "tmdb:tv:1396", "series-1")
	assert.Len(t, client.calls, 3)
	assert.Len(t, repo.terms, 2)
}

func TestEnsureSeeded_FailedInsertsAreNotMarkedAndRetryLater(t *testing.T) {
	// SQLite busy under a scan burst: the fetch worked, the inserts did not.
	// Marking now would freeze the drawer half-empty for good.
	repo := &fakeSeedInserter{}
	seeder, client, now := seedOnResolveFixture(t, repo)
	repo.err = errors.New("database is locked")
	// IsScopeSeeded also fails with repo.err → that is the probe path; make
	// the probe succeed but inserts fail by toggling around the calls.
	repo.err = nil
	insertFail := &insertFailingRepo{fakeSeedInserter: repo, fail: true}
	seeder.repo = insertFail
	ctx := context.Background()

	seeder.EnsureSeeded(ctx, "tmdb:tv:1396", "series-1")
	assert.Len(t, client.calls, 2)
	assert.Empty(t, repo.marks, "not marked: inserts failed")

	// After the back-off the pass re-runs and lands.
	insertFail.fail = false
	*now = now.Add(glossarySeedRetryAfter + time.Minute)
	seeder.EnsureSeeded(ctx, "tmdb:tv:1396", "series-1")
	assert.Len(t, repo.terms, 2)
	assert.Equal(t, map[string]int{"tmdb:tv:1396": 2}, repo.marks)
}

// insertFailingRepo fails InsertIfAbsent while fail is true, everything else
// delegates to the plain fake.
type insertFailingRepo struct {
	*fakeSeedInserter
	fail bool
}

func (r *insertFailingRepo) InsertIfAbsent(ctx context.Context, term *models.GlossaryTerm) (bool, error) {
	if r.fail {
		return false, errors.New("database is locked")
	}
	return r.fakeSeedInserter.InsertIfAbsent(ctx, term)
}

func TestEnsureSeeded_ProbeErrorBacksOffToo(t *testing.T) {
	repo := &fakeSeedInserter{err: errors.New("db locked")}
	seeder, client, _ := seedOnResolveFixture(t, repo)
	seeder.EnsureSeeded(context.Background(), "tmdb:movie:550", "m1")
	seeder.EnsureSeeded(context.Background(), "tmdb:movie:550", "m1")
	assert.Empty(t, client.calls)
	assert.Equal(t, 1, repo.probes)
}

func TestEnsureSeeded_ConcurrentResolvesSeedOnce(t *testing.T) {
	repo := &lockedSeedRepo{}
	seeder, client, _ := seedOnResolveFixture(t, repo)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seeder.EnsureSeeded(context.Background(), "tmdb:tv:1396", "series-1")
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, repo.probeCount(), "one goroutine claims the scope; the rest skip")
	assert.Len(t, client.callsSnapshot(), 2)
	assert.Len(t, repo.snapshot(), 2)
}

// lockedSeedRepo is the fake inserter with a mutex, for the concurrency test.
type lockedSeedRepo struct {
	mu    sync.Mutex
	inner fakeSeedInserter
}

func (l *lockedSeedRepo) InsertIfAbsent(ctx context.Context, term *models.GlossaryTerm) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inner.InsertIfAbsent(ctx, term)
}
func (l *lockedSeedRepo) IsScopeSeeded(ctx context.Context, scope string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inner.IsScopeSeeded(ctx, scope)
}
func (l *lockedSeedRepo) MarkScopeSeeded(ctx context.Context, scope string, seeded int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inner.MarkScopeSeeded(ctx, scope, seeded)
}
func (l *lockedSeedRepo) probeCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inner.probes
}
func (l *lockedSeedRepo) snapshot() []*models.GlossaryTerm {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]*models.GlossaryTerm(nil), l.inner.terms...)
}

// ---- resolver hook -------------------------------------------------------

type recordingScopeSeeder struct {
	calls  [][2]string
	events *[]string
}

func (r *recordingScopeSeeder) EnsureSeeded(_ context.Context, scope, mediaID string) {
	r.calls = append(r.calls, [2]string{scope, mediaID})
	if r.events != nil {
		*r.events = append(*r.events, "seed:"+scope)
	}
}

type eventMover struct{ events *[]string }

func (m *eventMover) MigrateScope(_ context.Context, from, to string) (int64, int64, error) {
	*m.events = append(*m.events, "move:"+from+"→"+to)
	return 0, 0, nil
}

func TestGlossaryScopeResolver_SeedsSharedScopesOnlyAndAfterTheMove(t *testing.T) {
	var events []string
	seeder := &recordingScopeSeeder{events: &events}
	r := newResolverFixture(&eventMover{events: &events})
	r.SetSeeder(seeder)
	ctx := context.Background()

	for _, id := range []string{"s-matched", "m-matched", "e-of-matched", "s-unmatched", "m-unmatched", "never-heard-of"} {
		_, err := r.Resolve(ctx, id)
		require.NoError(t, err)
	}
	assert.Equal(t, [][2]string{
		{"tmdb:tv:66732", "s-matched"},
		{"tmdb:movie:27205", "m-matched"},
		{"tmdb:tv:66732", "s-matched"}, // an episode seeds under its SERIES id
	}, seeder.calls, "local: drawers are never seeded")

	// Order for the first resolve: local→tmdb move first, seed second — so a
	// term the user confirmed while unmatched is in the drawer before TMDb's.
	assert.Equal(t, []string{"move:local:s-matched→tmdb:tv:66732", "seed:tmdb:tv:66732"}, events[:2])
}

func TestGlossaryScopeResolver_NoSeederIsTheOldBehaviour(t *testing.T) {
	r := newResolverFixture(nil)
	got, err := r.Resolve(context.Background(), "s-matched")
	require.NoError(t, err)
	assert.Equal(t, "tmdb:tv:66732", got)
}

// ---- end to end over the real repository ---------------------------------

// A show transcribed while UNMATCHED has a user-confirmed term in `local:`.
// The match lands; the first Resolve must (1) move that term into the shared
// drawer and (2) seed TMDb's names around it — and the user's rendering must
// win over TMDb's for the same term.
func TestSeedOnFirstResolve_UserTermMigratesFirstAndWins(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGlossaryRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{
		MediaID: "s-matched", Scope: models.GlossaryScopeLocal("s-matched"),
		TermSrc: "Walter White", TermZh: "老白", Source: models.GlossarySourceSubtitle, Confirmed: true,
	}))

	seeder, client, _ := seedOnResolveFixture(t, repo)
	r := newResolverFixture(repo)
	r.SetSeeder(seeder)

	scope, err := r.Resolve(ctx, "s-matched")
	require.NoError(t, err)
	assert.Equal(t, "tmdb:tv:66732", scope)
	assert.Equal(t, []string{"tv/66732?zh-TW", "tv/66732?en-US"}, client.calls)

	terms, err := repo.ListByScope(ctx, scope)
	require.NoError(t, err)
	byTerm := map[string]models.GlossaryTerm{}
	for _, tm := range terms {
		byTerm[tm.TermSrc] = tm
	}
	require.Len(t, byTerm, 2)
	assert.Equal(t, "老白", byTerm["Walter White"].TermZh, "the user's confirmed rendering, moved in first, wins")
	assert.True(t, byTerm["Walter White"].Confirmed)
	assert.Equal(t, "布萊恩·克蘭斯頓", byTerm["Bryan Cranston"].TermZh)
	assert.Equal(t, models.GlossarySourceMetadata, byTerm["Bryan Cranston"].Source)

	// Existing-library case: a second resolve (the next subtitle run) is a
	// single EXISTS and no TMDb traffic.
	_, err = r.Resolve(ctx, "e-of-matched")
	require.NoError(t, err)
	assert.Len(t, client.calls, 2)
	marked, err := repo.IsScopeSeeded(ctx, scope)
	require.NoError(t, err)
	assert.True(t, marked)
}

// The user deletes every seeded term as junk; a restart (fresh seeder, empty
// attempted map) must NOT plant them again — the mark outlives the terms.
func TestSeedOnFirstResolve_DeletedSeedsStayDeletedAcrossRestart(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGlossaryRepository(db)
	ctx := context.Background()

	first, client, _ := seedOnResolveFixture(t, repo)
	r := newResolverFixture(repo)
	r.SetSeeder(first)
	_, err := r.Resolve(ctx, "m-matched")
	require.NoError(t, err)
	terms, err := repo.ListByScope(ctx, "tmdb:movie:27205")
	require.NoError(t, err)
	require.Len(t, terms, 2)
	for _, tm := range terms {
		require.NoError(t, repo.Delete(ctx, tm.ID))
	}

	// "Restart": a new seeder with no memory, same database.
	second := NewGlossarySeeder(client, repo, &fakeSeedOpenCC{available: true}, slog.Default())
	r2 := newResolverFixture(repo)
	r2.SetSeeder(second)
	_, err = r2.Resolve(ctx, "m-matched")
	require.NoError(t, err)
	terms, err = repo.ListByScope(ctx, "tmdb:movie:27205")
	require.NoError(t, err)
	assert.Empty(t, terms, "deleted seeds are not re-planted")
	assert.Len(t, client.calls, 2, "and TMDb is not asked again")
}
