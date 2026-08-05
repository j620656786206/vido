package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/services"
)

// KeySettingsReader is the settings-state half the handler drives.
// *services.KeySettingsService satisfies it.
type KeySettingsReader interface {
	List(ctx context.Context) services.KeySettings
	Save(ctx context.Context, updates map[services.KeyName]string) error
}

// KeyTester validates a candidate key against the live provider.
// *services.ClaudeProviderHolder satisfies it.
type KeyTester interface {
	TestKey(ctx context.Context, candidate string) error
}

// KeySettingsHandler serves the FR25 provider-key settings triad (story
// sub-2-1a AC #3), mirroring the shipped qBittorrent/DVR settings shape.
type KeySettingsHandler struct {
	settings KeySettingsReader
	tester   KeyTester
}

// NewKeySettingsHandler creates the handler. tester may be nil (no Claude
// support compiled/wired) — the test endpoint then answers 409 rather than
// panicking.
func NewKeySettingsHandler(settings KeySettingsReader, tester KeyTester) *KeySettingsHandler {
	return &KeySettingsHandler{settings: settings, tester: tester}
}

// RegisterRoutes registers the key settings routes (Rule 10 — the group is
// already /api/v1).
func (h *KeySettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	keys := rg.Group("/settings/keys")
	{
		keys.GET("", h.GetKeys)
		keys.PUT("", h.SaveKeys)
		keys.POST("/test", h.TestKey)
	}
}

// SaveKeysRequest is the PUT body. Pointers distinguish the three cases that
// matter: absent = leave untouched, "" = delete the secret (revert to env),
// value = store. A plain string could not tell "omitted" from "cleared".
type SaveKeysRequest struct {
	Claude *string `json:"claude"`
	TMDb   *string `json:"tmdb"`
	OpenAI *string `json:"openai"`
}

// TestKeyRequest optionally carries the key to probe. Empty tests whatever
// currently resolves, so the page can verify an already-saved key.
type TestKeyRequest struct {
	Claude string `json:"claude"`
}

// GetKeys handles GET /api/v1/settings/keys.
// @Summary Read provider key configuration state
// @Description Per key: whether it is configured, WHERE it resolves from (secret > env), and a masked hint for secret-sourced keys only. The value is never returned, and an env-sourced key is never masked-echoed. `writable: false` with `reason: encryption_key_missing` means VIDO_ENCRYPTION_KEY is unset, so the page must render read-only.
// @Tags settings
// @Produce json
// @Success 200 {object} APIResponse "keys + writable/reason"
// @Router /api/v1/settings/keys [get]
func (h *KeySettingsHandler) GetKeys(c *gin.Context) {
	SuccessResponse(c, h.settings.List(c.Request.Context()))
}

// SaveKeys handles PUT /api/v1/settings/keys.
// @Summary Store or clear provider keys
// @Description Omitted fields are untouched; an explicit empty string DELETES the stored secret and reverts resolution to the environment variable (the mistyped-key recovery path). Requires VIDO_ENCRYPTION_KEY.
// @Tags settings
// @Accept json
// @Produce json
// @Param request body SaveKeysRequest true "any subset of claude/tmdb/openai"
// @Success 200 {object} APIResponse "updated key state (same shape as GET)"
// @Failure 400 {object} APIResponse "VALIDATION_INVALID_FORMAT — malformed body"
// @Failure 409 {object} APIResponse "AI_NOT_CONFIGURED — no encryption key, storage unavailable"
// @Failure 500 {object} APIResponse "DB_QUERY_FAILED — the secrets store rejected the write"
// @Router /api/v1/settings/keys [put]
func (h *KeySettingsHandler) SaveKeys(c *gin.Context) {
	var req SaveKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "請求格式錯誤：請提供 claude、tmdb 或 openai 其中至少一項")
		return
	}

	updates := map[services.KeyName]string{}
	for name, value := range map[services.KeyName]*string{
		services.KeyClaude: req.Claude,
		services.KeyTMDb:   req.TMDb,
		services.KeyOpenAI: req.OpenAI,
	} {
		if value != nil {
			updates[name] = *value
		}
	}
	if len(updates) == 0 {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "請至少提供一個要更新的金鑰欄位")
		return
	}

	if err := h.settings.Save(c.Request.Context(), updates); err != nil {
		if errors.Is(err, services.ErrKeysNotWritable) {
			// AC #4 — refuse honestly instead of accepting input that will fail
			// opaquely at the storage layer.
			ErrorResponse(c, http.StatusConflict, "AI_NOT_CONFIGURED",
				"未設定加密金鑰，無法安全儲存 API 金鑰",
				"請設定 VIDO_ENCRYPTION_KEY 環境變數後重啟伺服器。")
			return
		}
		slog.Error("Failed to save provider keys", "error", err)
		ErrorResponse(c, http.StatusInternalServerError, "DB_QUERY_FAILED",
			"無法儲存金鑰", "請稍後再試；若持續發生請檢查伺服器日誌。")
		return
	}

	// Return the fresh state so the page reflects the new `source` without a
	// second round trip — the flip from env to secret is the visible proof the
	// save took effect.
	SuccessResponse(c, h.settings.List(c.Request.Context()))
}

// TestKey handles POST /api/v1/settings/keys/test.
// @Summary Validate a Claude API key against the live API
// @Description Makes the smallest real call the API allows (max_tokens 1) so the check exercises the exact auth path. Tests the body's key when supplied — letting the page verify BEFORE saving — otherwise the currently-resolved one.
// @Tags settings
// @Accept json
// @Produce json
// @Param request body TestKeyRequest false "claude: key to test; omit to test the resolved key"
// @Success 200 {object} APIResponse "{valid: true}"
// @Failure 400 {object} APIResponse "VALIDATION_INVALID_FORMAT — malformed body"
// @Failure 409 {object} APIResponse "AI_NOT_CONFIGURED — nothing to test and no key supplied"
// @Failure 502 {object} APIResponse "AI_PROVIDER_ERROR / AI_QUOTA_EXCEEDED / AI_TIMEOUT"
// @Router /api/v1/settings/keys/test [post]
func (h *KeySettingsHandler) TestKey(c *gin.Context) {
	var req TestKeyRequest
	// An empty body is legitimate ("test what's configured"), so a decode
	// failure is only an error when there IS a body.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT", "請求格式錯誤")
			return
		}
	}

	if h.tester == nil {
		ErrorResponse(c, http.StatusConflict, "AI_NOT_CONFIGURED",
			"尚未設定翻譯服務金鑰", "請先儲存 CLAUDE_API_KEY 或在此提供要測試的金鑰。")
		return
	}

	err := h.tester.TestKey(c.Request.Context(), req.Claude)
	if err == nil {
		SuccessResponse(c, gin.H{"valid": true})
		return
	}

	code, status, message, suggestion := classifyKeyTestError(err)
	slog.Warn("Provider key test failed", "code", code, "error", err)
	ErrorResponse(c, status, code, message, suggestion)
}

// classifyKeyTestError turns a provider error into the Rule 3 envelope.
// Ordered most-specific first: an unauthorized error also satisfies
// ErrAIProviderError (they are wrapped together), so it must be checked first.
func classifyKeyTestError(err error) (code string, status int, message, suggestion string) {
	switch {
	case errors.Is(err, ai.ErrAINotConfigured):
		return "AI_NOT_CONFIGURED", http.StatusConflict,
			"尚未設定翻譯服務金鑰", "請提供要測試的金鑰，或先儲存一組金鑰。"
	case errors.Is(err, ai.ErrAIUnauthorized):
		return "AI_PROVIDER_ERROR", http.StatusBadGateway,
			"金鑰無效或已撤銷", "請確認金鑰是否正確，或至 Anthropic Console 重新產生。"
	case errors.Is(err, ai.ErrAIQuotaExceeded):
		// The key is VALID — it is the account that is rate-limited. Saying
		// "invalid key" here would send the user to regenerate a working key.
		return "AI_QUOTA_EXCEEDED", http.StatusBadGateway,
			"金鑰有效，但目前已達用量上限", "請稍後再試，或檢查 Anthropic 帳戶的配額。"
	case errors.Is(err, ai.ErrAITimeout):
		return "AI_TIMEOUT", http.StatusBadGateway,
			"連線逾時，無法驗證金鑰", "請檢查伺服器對外網路連線後再試。"
	default:
		return "AI_PROVIDER_ERROR", http.StatusBadGateway,
			"無法驗證金鑰", "請稍後再試；若持續發生請檢查伺服器日誌。"
	}
}
