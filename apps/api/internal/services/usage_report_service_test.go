package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/repository"
)

// infra-optin-usage-report-a2 — the opt-in anonymous weekly usage report.
// Settings run on a REAL migrated DB through the real repository (Rule 28):
// the "never sent" / "off by default" answers come from absent keys.

func usageReportDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open(database.DriverName, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

type fakeAutoCounter struct {
	counts  repository.AutoProducedCounts
	gotFrom time.Time
	gotTo   time.Time
	calls   int
	err     error
}

func (f *fakeAutoCounter) AutoProducedBetween(_ context.Context, from, to time.Time) (repository.AutoProducedCounts, error) {
	f.calls++
	f.gotFrom, f.gotTo = from, to
	return f.counts, f.err
}

type fakeReportSender struct {
	bodies [][]byte
	err    error
}

func (f *fakeReportSender) Send(_ context.Context, body []byte) error {
	f.bodies = append(f.bodies, append([]byte(nil), body...))
	return f.err
}

type usageHarness struct {
	svc     *UsageReportService
	db      *sql.DB
	counter *fakeAutoCounter
	sender  *fakeReportSender
	now     time.Time
}

func newUsageHarness(t *testing.T, available bool) *usageHarness {
	t.Helper()
	db := usageReportDB(t)
	h := &usageHarness{
		db:      db,
		counter: &fakeAutoCounter{counts: repository.AutoProducedCounts{Embedded: 4, Online: 2, ASR: 1}},
		sender:  &fakeReportSender{},
		now:     time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
	}
	cfg := UsageReportConfig{Version: "0.1.2"}
	if available {
		cfg.WebsiteID = "11111111-2222-3333-4444-555555555555"
	}
	var sender ReportSender
	if available {
		sender = h.sender
	}
	h.svc = NewUsageReportService(repository.NewSettingsRepository(db), h.counter, sender, cfg, nil)
	h.svc.now = func() time.Time { return h.now }
	return h
}

// NFR-T3 / P1-040-1 — a fresh install is off, sends nothing, has no id.
func TestUsageReport_FreshInstallIsOffAndSendsNothing(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()

	st, err := h.svc.Status(ctx)
	require.NoError(t, err)
	assert.True(t, st.Available)
	assert.False(t, st.Enabled)
	assert.Nil(t, st.LastSentAt)
	assert.Nil(t, st.LastPayload)

	h.svc.Tick(ctx)
	assert.Empty(t, h.sender.bodies)
	assert.Equal(t, 0, h.counter.calls)

	_, err = repository.NewSettingsRepository(h.db).GetString(ctx, usageReportKeyInstallID)
	assert.Error(t, err, "no install id exists until the user turns the report on")
}

// AC #6 — no receiver configured (local build, fork) → unavailable, never sends.
func TestUsageReport_UnavailableWithoutReceiverNeverSends(t *testing.T) {
	h := newUsageHarness(t, false)
	ctx := context.Background()

	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)
	st, err := h.svc.Status(ctx)
	require.NoError(t, err)
	assert.False(t, st.Available)

	h.svc.Tick(ctx)
	assert.Equal(t, 0, h.counter.calls)
}

// P1-040-4a / AC #2 — a random UUID, created on first enable, kept across off/on.
func TestUsageReport_InstallIDIsCreatedOnFirstEnableAndSurvivesOffOn(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	repo := repository.NewSettingsRepository(h.db)

	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)
	id, err := repo.GetString(ctx, usageReportKeyInstallID)
	require.NoError(t, err)
	_, perr := uuid.Parse(id)
	require.NoError(t, perr, "a dashed UUID — a 32-hex id would be masked in the DB log")
	assert.Contains(t, id, "-")

	_, err = h.svc.SetEnabled(ctx, false)
	require.NoError(t, err)
	_, err = h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)
	again, err := repo.GetString(ctx, usageReportKeyInstallID)
	require.NoError(t, err)
	assert.Equal(t, id, again, "the same machine keeps the same id")
}

// NFR-T1 — the user-data fields are exactly the allow-list; everything else is
// a fixed protocol constant.
func TestUsageReport_BodyCarriesOnlyAllowListedFields(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	require.Len(t, h.sender.bodies, 1)

	var sent struct {
		Type    string                     `json:"type"`
		Payload map[string]json.RawMessage `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(h.sender.bodies[0], &sent))
	assert.Equal(t, "event", sent.Type)
	assert.Equal(t, []string{"data", "hostname", "id", "ip", "name", "url", "website"}, usageReportKeys(sent.Payload))
	assert.JSONEq(t, `"11111111-2222-3333-4444-555555555555"`, string(sent.Payload["website"]))
	assert.JSONEq(t, `"vido"`, string(sent.Payload["hostname"]))
	assert.JSONEq(t, `"/usage-report"`, string(sent.Payload["url"]))
	assert.JSONEq(t, `"weekly_usage"`, string(sent.Payload["name"]))
	assert.JSONEq(t, `"127.0.0.1"`, string(sent.Payload["ip"]), "a loopback ip makes Umami skip the country lookup")

	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(sent.Payload["data"], &data))
	assert.Equal(t, []string{"subtitles_asr_7d", "subtitles_auto_7d", "subtitles_embedded_7d", "subtitles_online_7d", "version"}, usageReportKeys(data))
	assert.JSONEq(t, `"0.1.2"`, string(data["version"]))
	assert.JSONEq(t, `7`, string(data["subtitles_auto_7d"]))
	assert.JSONEq(t, `4`, string(data["subtitles_embedded_7d"]))
	assert.JSONEq(t, `2`, string(data["subtitles_online_7d"]))
	assert.JSONEq(t, `1`, string(data["subtitles_asr_7d"]))

	assert.Equal(t, h.now.Add(-7*24*time.Hour), h.counter.gotFrom)
	assert.Equal(t, h.now, h.counter.gotTo)
}

// NFR-T2 — on a DB full of media, keys and paths, none of it reaches the body.
func TestUsageReport_BodyNeverContainsLibraryData(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	secret := []string{"如懿傳", "Breaking.Bad.S01E01.mkv", "/volume1/media/tv", "sk-ant-api03-secret", "claude-sonnet-5"}
	_, err := h.db.Exec(`INSERT INTO movies (id, title, release_date, file_path) VALUES ('m1', ?, '2024-01-01', ?)`,
		secret[0], secret[2]+"/"+secret[1])
	require.NoError(t, err)
	repo := repository.NewSettingsRepository(h.db)
	require.NoError(t, repo.SetString(ctx, "claude.api_key", secret[3]))
	require.NoError(t, repo.SetString(ctx, "subtitle.model", secret[4]))

	_, err = h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)
	h.svc.Tick(ctx)
	require.Len(t, h.sender.bodies, 1)
	for _, s := range secret {
		assert.NotContains(t, string(h.sender.bodies[0]), s)
	}
}

// NFR-T4 — at most one attempt per 7 days, success or failure; the first send
// happens on the first check after turning it on.
func TestUsageReport_AtMostOneAttemptPerSevenDays(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	require.Len(t, h.sender.bodies, 1, "first check after enabling sends")

	h.now = h.now.Add(7*24*time.Hour - time.Minute)
	h.svc.Tick(ctx)
	assert.Len(t, h.sender.bodies, 1, "inside the 7-day window nothing is sent")

	h.now = h.now.Add(time.Minute)
	h.svc.Tick(ctx)
	assert.Len(t, h.sender.bodies, 2, "7 days after the last attempt the next one goes")
}

func TestUsageReport_AFailedAttemptAlsoWaitsSevenDays(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	h.sender.err = errors.New("receiver unreachable")
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	h.now = h.now.Add(time.Hour)
	h.svc.Tick(ctx)

	assert.Len(t, h.sender.bodies, 1, "no retry storm — a failure is an attempt")
}

// NFR-T5 / P1-040-6 — a failed send leaves the last-sent view untouched and
// surfaces nothing.
func TestUsageReport_FailureKeepsTheLastSuccessfulPayload(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	first, err := h.svc.Status(ctx)
	require.NoError(t, err)
	require.NotNil(t, first.LastPayload)
	require.NotNil(t, first.LastSentAt)
	assert.Equal(t, string(h.sender.bodies[0]), *first.LastPayload, "the page shows the exact bytes sent")
	assert.True(t, first.LastSentAt.Equal(h.now))

	h.sender.err = errors.New("HTTP 400 Website not found.")
	h.counter.counts = repository.AutoProducedCounts{Online: 99}
	h.now = h.now.Add(8 * 24 * time.Hour)
	assert.NotPanics(t, func() { h.svc.Tick(ctx) })

	after, err := h.svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, *first.LastPayload, *after.LastPayload)
	assert.True(t, after.LastSentAt.Equal(first.LastSentAt.UTC()))
}

func TestUsageReport_CountFailureSkipsTheSendButCountsAsAttempt(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	h.counter.err = errors.New("db locked")
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	assert.Empty(t, h.sender.bodies, "an unknown count is not reported as 0")
	h.counter.err = nil
	h.now = h.now.Add(time.Hour)
	h.svc.Tick(ctx)
	assert.Empty(t, h.sender.bodies, "still inside the window")
}

func TestUsageReport_TurningOffStopsTheNextSend(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)
	_, err = h.svc.SetEnabled(ctx, false)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	assert.Empty(t, h.sender.bodies)
}

func usageReportKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// P1-040-7 — the example in the user docs is the real wire format, not a
// paraphrase: build the same report and compare byte for byte (EN + zh-TW).
func TestUsageReport_DocsExampleMatchesTheRealBody(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	repo := repository.NewSettingsRepository(h.db)
	require.NoError(t, repo.SetString(ctx, usageReportKeyInstallID, "3f2a6c1e-8f0b-4d5e-9a7c-1b2c3d4e5f60"))
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	require.Len(t, h.sender.bodies, 1)

	for _, doc := range []string{"../../../../docs/usage-report.md", "../../../../docs/usage-report.zh-TW.md"} {
		raw, err := os.ReadFile(doc)
		require.NoError(t, err)
		var example string
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, `{"type":"event"`) {
				example = line
			}
		}
		require.NotEmpty(t, example, "%s must show a complete report", doc)
		assert.Equal(t, example, string(h.sender.bodies[0]), doc)
	}
}

// Review LOW-4 — the generic settings API can write any key. Values this
// service did not write must fail safe, not break the page or block forever.

func TestUsageReport_WrongTypeSwitchReadsAsOff(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	require.NoError(t, repository.NewSettingsRepository(h.db).SetString(ctx, usageReportKeyEnabled, "true"))

	st, err := h.svc.Status(ctx)
	require.NoError(t, err, "the settings page must still load")
	assert.False(t, st.Enabled)
	h.svc.Tick(ctx)
	assert.Empty(t, h.sender.bodies, "when in doubt, do not send")
}

func TestUsageReport_UnparseableLastAttemptDoesNotBlockForever(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)
	require.NoError(t, repository.NewSettingsRepository(h.db).SetString(ctx, usageReportKeyLastAttemptAt, "yesterday"))

	h.svc.Tick(ctx)
	assert.Len(t, h.sender.bodies, 1)
}

func TestUsageReport_DeletedInstallIDIsRecreatedNotStuck(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	_, err := h.svc.SetEnabled(ctx, true)
	require.NoError(t, err)
	require.NoError(t, repository.NewSettingsRepository(h.db).Delete(ctx, usageReportKeyInstallID))

	h.svc.Tick(ctx)
	require.Len(t, h.sender.bodies, 1)
	assert.Contains(t, string(h.sender.bodies[0]), `"id":"`)
}

func TestUsageReport_ForeignLastSentRecordShowsAsNeverSent(t *testing.T) {
	h := newUsageHarness(t, true)
	ctx := context.Background()
	require.NoError(t, repository.NewSettingsRepository(h.db).SetString(ctx, usageReportKeyLastSent, "not json"))

	st, err := h.svc.Status(ctx)
	require.NoError(t, err)
	assert.Nil(t, st.LastPayload)
	assert.Nil(t, st.LastSentAt)
}
