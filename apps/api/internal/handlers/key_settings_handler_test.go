package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/services"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

type fakeKeySettings struct {
	state    services.KeySettings
	saveErr  error
	saved    []map[services.KeyName]string
	listCall int
}

func (f *fakeKeySettings) List(context.Context) services.KeySettings {
	f.listCall++
	return f.state
}

func (f *fakeKeySettings) Save(_ context.Context, updates map[services.KeyName]string) error {
	f.saved = append(f.saved, updates)
	return f.saveErr
}

type fakeKeyTester struct {
	err    error
	called int
	last   string
}

func (f *fakeKeyTester) TestKey(_ context.Context, candidate string) error {
	f.called++
	f.last = candidate
	return f.err
}

func configuredState() services.KeySettings {
	return services.KeySettings{
		Writable: true,
		Keys: []services.KeyState{
			{Name: services.KeyClaude, Configured: true, Source: services.KeySourceSecret, Masked: "sk-ant…7f3a"},
			{Name: services.KeyTMDb, Configured: true, Source: services.KeySourceEnv},
			{Name: services.KeyOpenAI, Configured: false, Source: services.KeySourceNone},
		},
	}
}

func newKeyRouter(h *KeySettingsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func doKeyReq(t *testing.T, h *KeySettingsHandler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	req.Header.Set("Content-Type", "application/json")
	newKeyRouter(h).ServeHTTP(w, req)
	return w
}

func envelopeOf(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var resp APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

// ─── AC #3: GET ────────────────────────────────────────────────────────────

func TestGetKeys_ReturnsStateAndNeverAValue(t *testing.T) {
	h := NewKeySettingsHandler(&fakeKeySettings{state: configuredState()}, &fakeKeyTester{})

	w := doKeyReq(t, h, http.MethodGet, "/api/v1/settings/keys", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, envelopeOf(t, w).Success)
	body := w.Body.String()
	assert.Contains(t, body, `"source":"secret"`)
	assert.Contains(t, body, `"masked":"sk-ant…7f3a"`)
	// The env-sourced key carries no mask at all.
	assert.NotContains(t, body, `"name":"tmdb","configured":true,"source":"env","masked"`)
}

func TestGetKeys_SurfacesReadOnlyReason(t *testing.T) {
	state := configuredState()
	state.Writable = false
	state.Reason = services.ReasonEncryptionKeyMissing
	h := NewKeySettingsHandler(&fakeKeySettings{state: state}, &fakeKeyTester{})

	w := doKeyReq(t, h, http.MethodGet, "/api/v1/settings/keys", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"writable":false`)
	assert.Contains(t, w.Body.String(), services.ReasonEncryptionKeyMissing)
}

// ─── AC #3: PUT ────────────────────────────────────────────────────────────

func TestSaveKeys_StoresOnlySuppliedFields(t *testing.T) {
	settings := &fakeKeySettings{state: configuredState()}
	h := NewKeySettingsHandler(settings, &fakeKeyTester{})

	w := doKeyReq(t, h, http.MethodPut, "/api/v1/settings/keys", `{"claude":"sk-new"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, settings.saved, 1)
	assert.Equal(t, map[services.KeyName]string{services.KeyClaude: "sk-new"}, settings.saved[0],
		"an omitted field must not reach Save at all — a nil pointer means untouched")
}

// TestSaveKeys_ExplicitEmptyStringReachesSaveAsADelete — the distinction a plain
// string body could not express: absent (leave alone) vs "" (clear it).
func TestSaveKeys_ExplicitEmptyStringReachesSaveAsADelete(t *testing.T) {
	settings := &fakeKeySettings{state: configuredState()}
	h := NewKeySettingsHandler(settings, &fakeKeyTester{})

	w := doKeyReq(t, h, http.MethodPut, "/api/v1/settings/keys", `{"claude":""}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, settings.saved, 1)
	value, present := settings.saved[0][services.KeyClaude]
	assert.True(t, present, "an explicit \"\" must be forwarded, not dropped as empty")
	assert.Equal(t, "", value)
}

func TestSaveKeys_ReturnsFreshStateSoTheSourceFlipIsVisible(t *testing.T) {
	settings := &fakeKeySettings{state: configuredState()}
	h := NewKeySettingsHandler(settings, &fakeKeyTester{})

	doKeyReq(t, h, http.MethodPut, "/api/v1/settings/keys", `{"claude":"sk-new"}`)

	assert.Equal(t, 1, settings.listCall, "PUT re-reads state so the page sees source flip to secret")
}

func TestSaveKeys_EmptyBodyIsRejected(t *testing.T) {
	settings := &fakeKeySettings{state: configuredState()}
	h := NewKeySettingsHandler(settings, &fakeKeyTester{})

	w := doKeyReq(t, h, http.MethodPut, "/api/v1/settings/keys", `{}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_REQUIRED_FIELD", envelopeOf(t, w).Error.Code)
	assert.Empty(t, settings.saved)
}

// ─── AC #4: encryption pre-flight ──────────────────────────────────────────

func TestSaveKeys_409WhenStorageIsNotWritable(t *testing.T) {
	settings := &fakeKeySettings{state: configuredState(), saveErr: services.ErrKeysNotWritable}
	h := NewKeySettingsHandler(settings, &fakeKeyTester{})

	w := doKeyReq(t, h, http.MethodPut, "/api/v1/settings/keys", `{"claude":"sk-new"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	body := envelopeOf(t, w)
	assert.Equal(t, "AI_NOT_CONFIGURED", body.Error.Code, "reuse — this story adds ZERO Rule 7 codes")
	assert.Equal(t, "未設定加密金鑰，無法安全儲存 API 金鑰", body.Error.Message)
	assert.Contains(t, body.Error.Suggestion, "ENCRYPTION_KEY")
	// The server reads ENCRYPTION_KEY (config.go:138, docker-compose.yml) — the
	// suggestion once said VIDO_ENCRYPTION_KEY, which does not exist, so an
	// operator following it verbatim changed nothing (the silent-no-op class
	// this story removes). Pin the exact var name.
	assert.NotContains(t, body.Error.Suggestion, "VIDO_ENCRYPTION_KEY")
}

func TestSaveKeys_500OnStorageFailure(t *testing.T) {
	settings := &fakeKeySettings{state: configuredState(), saveErr: errors.New("database is locked")}
	h := NewKeySettingsHandler(settings, &fakeKeyTester{})

	w := doKeyReq(t, h, http.MethodPut, "/api/v1/settings/keys", `{"claude":"sk-new"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "DB_QUERY_FAILED", envelopeOf(t, w).Error.Code)
}

// ─── AC #3: POST /test ─────────────────────────────────────────────────────

func TestTestKey_ValidKeyReturns200(t *testing.T) {
	tester := &fakeKeyTester{}
	h := NewKeySettingsHandler(&fakeKeySettings{state: configuredState()}, tester)

	w := doKeyReq(t, h, http.MethodPost, "/api/v1/settings/keys/test", `{"claude":"sk-candidate"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":true`)
	assert.Equal(t, "sk-candidate", tester.last, "the body's key must be what gets probed, pre-save")
}

func TestTestKey_EmptyBodyTestsTheResolvedKey(t *testing.T) {
	tester := &fakeKeyTester{}
	h := NewKeySettingsHandler(&fakeKeySettings{state: configuredState()}, tester)

	w := doKeyReq(t, h, http.MethodPost, "/api/v1/settings/keys/test", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, tester.called)
	assert.Empty(t, tester.last, "no candidate means test whatever currently resolves")
}

func TestTestKey_ErrorClassification(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode string
		wantHTTP int
		wantMsg  string
	}{
		{
			name: "401 is an invalid key",
			// Wrapped exactly as claude.go wraps it: BOTH sentinels.
			err:      fmt.Errorf("%w: %w: status 401", ai.ErrAIProviderError, ai.ErrAIUnauthorized),
			wantCode: "AI_PROVIDER_ERROR", wantHTTP: http.StatusBadGateway, wantMsg: "金鑰無效或已撤銷",
		},
		{
			name:     "429 means the key WORKS but is throttled",
			err:      ai.ErrAIQuotaExceeded,
			wantCode: "AI_QUOTA_EXCEEDED", wantHTTP: http.StatusBadGateway, wantMsg: "金鑰有效，但目前已達用量上限",
		},
		{
			name:     "timeout is a network problem, not a key problem",
			err:      ai.ErrAITimeout,
			wantCode: "AI_TIMEOUT", wantHTTP: http.StatusBadGateway, wantMsg: "連線逾時，無法驗證金鑰",
		},
		{
			name:     "nothing configured and nothing supplied",
			err:      ai.ErrAINotConfigured,
			wantCode: "AI_NOT_CONFIGURED", wantHTTP: http.StatusConflict, wantMsg: "尚未設定翻譯服務金鑰",
		},
		{
			name: "404 means the MODEL is wrong, not the key",
			// Wrapped exactly as claude.go wraps it: BOTH sentinels (AC #3's
			// sub-1-1 model diagnostic).
			err:      fmt.Errorf("%w: %w: status 404: model %q not found", ai.ErrAIProviderError, ai.ErrAIModelNotFound, "bogus-model"),
			wantCode: "AI_PROVIDER_ERROR", wantHTTP: http.StatusBadGateway, wantMsg: "金鑰可用，但設定的模型識別碼無效或已棄用",
		},
		{
			name:     "anything else stays generic",
			err:      fmt.Errorf("%w: status 500", ai.ErrAIProviderError),
			wantCode: "AI_PROVIDER_ERROR", wantHTTP: http.StatusBadGateway, wantMsg: "無法驗證金鑰",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewKeySettingsHandler(&fakeKeySettings{state: configuredState()}, &fakeKeyTester{err: tc.err})

			w := doKeyReq(t, h, http.MethodPost, "/api/v1/settings/keys/test", `{"claude":"sk"}`)

			require.Equal(t, tc.wantHTTP, w.Code)
			body := envelopeOf(t, w)
			assert.Equal(t, tc.wantCode, body.Error.Code)
			assert.Equal(t, tc.wantMsg, body.Error.Message)
		})
	}
}

// TestTestKey_UnauthorizedBeatsGenericProviderError pins the ordering in
// classifyKeyTestError: a 401 satisfies BOTH sentinels, so checking
// ErrAIProviderError first would collapse "your key is wrong" into "something
// went wrong" — sending the user to debug their network instead of their key.
func TestTestKey_UnauthorizedBeatsGenericProviderError(t *testing.T) {
	err := fmt.Errorf("%w: %w: status 403", ai.ErrAIProviderError, ai.ErrAIUnauthorized)
	require.ErrorIs(t, err, ai.ErrAIProviderError, "precondition: it really is both")
	require.ErrorIs(t, err, ai.ErrAIUnauthorized)

	code, _, message, _ := classifyKeyTestError(err)

	assert.Equal(t, "AI_PROVIDER_ERROR", code)
	assert.Equal(t, "金鑰無效或已撤銷", message)
}

func TestTestKey_409WhenNoTesterWired(t *testing.T) {
	h := NewKeySettingsHandler(&fakeKeySettings{state: configuredState()}, nil)

	w := doKeyReq(t, h, http.MethodPost, "/api/v1/settings/keys/test", `{"claude":"sk"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "AI_NOT_CONFIGURED", envelopeOf(t, w).Error.Code)
}

// ─── sub-6-6 AC #3: the 200 says which model was verified ───────────────────

type fakeModelKeyTester struct{ fakeKeyTester }

func (f *fakeModelKeyTester) EffectiveModel() string { return "claude-sonnet-5" }

func TestTestKey_ValidKeyReportsModel(t *testing.T) {
	h := NewKeySettingsHandler(&fakeKeySettings{state: configuredState()}, &fakeModelKeyTester{})
	w := doKeyReq(t, h, http.MethodPost, "/api/v1/settings/keys/test", `{"claude":"sk-candidate"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"model":"claude-sonnet-5"`)
}
