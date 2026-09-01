package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthHandler serves the login/logout/status endpoints that sit in front of the
// session gate (see AuthGate).
type AuthHandler struct {
	auth    *Authenticator
	limiter *loginLimiter
}

// NewAuthHandler creates an AuthHandler over the given Authenticator. Login is
// throttled to 5 wrong tries per client IP, then a 60s lockout.
func NewAuthHandler(auth *Authenticator) *AuthHandler {
	return &AuthHandler{
		auth:    auth,
		limiter: newLoginLimiter(5, time.Minute),
	}
}

// LoginRequest is the POST /api/v1/auth/login body.
type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

// GetStatus handles GET /api/v1/auth/status. Safe to call unauthenticated: the
// frontend uses it to choose between the login screen and the app.
func (h *AuthHandler) GetStatus(c *gin.Context) {
	if !h.auth.Enabled() {
		// No password configured → nothing to log into; treat everyone as in.
		SuccessResponse(c, gin.H{"authEnabled": false, "authenticated": true})
		return
	}
	SuccessResponse(c, gin.H{
		"authEnabled":   true,
		"authenticated": h.auth.authenticated(c),
	})
}

// Login handles POST /api/v1/auth/login. A correct password sets the signed
// session cookie.
func (h *AuthHandler) Login(c *gin.Context) {
	if !h.auth.Enabled() {
		SuccessResponse(c, gin.H{"authEnabled": false, "authenticated": true})
		return
	}
	ip := c.ClientIP()
	if ok, retryAfter := h.limiter.allow(ip, time.Now()); !ok {
		h.respondLocked(c, retryAfter)
		return
	}
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "密碼為必填")
		return
	}
	if !h.auth.checkPassword(req.Password) {
		remaining, lockout := h.limiter.recordFailure(ip, time.Now())
		if lockout > 0 {
			h.respondLocked(c, lockout)
			return
		}
		// The countdown to the lock is a number the server is genuinely
		// tracking, so it is safe to show — and showing it late (only once the
		// user is two tries away) keeps it a warning rather than an invitation
		// to guess.
		suggestion := "請確認密碼後再試一次。"
		if remaining <= 2 {
			suggestion = fmt.Sprintf("還可以再試 %d 次,之後會鎖定 %d 秒。",
				remaining, int(h.limiter.lockout.Seconds()))
		}
		ErrorResponse(c, http.StatusUnauthorized, "INVALID_CREDENTIALS",
			"密碼錯誤", suggestion)
		return
	}
	h.limiter.recordSuccess(ip)
	h.auth.setSessionCookie(c, h.auth.issueToken(time.Now()))
	SuccessResponse(c, gin.H{"authEnabled": true, "authenticated": true})
}

// respondLocked answers a throttled login. Retry-After carries the wait in
// SECONDS so the login screen can run a real countdown instead of asking the
// user to guess how long "稍後" is; the suggestion repeats it in words for
// anything that only reads the body.
func (h *AuthHandler) respondLocked(c *gin.Context, retryAfter time.Duration) {
	secs := int(retryAfter.Seconds())
	if retryAfter%time.Second != 0 {
		secs++ // never advertise a shorter wait than the server will enforce
	}
	c.Header("Retry-After", strconv.Itoa(secs))
	ErrorResponse(c, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS",
		"嘗試次數過多",
		fmt.Sprintf("密碼連續錯誤 %d 次,請等 %d 秒後再試。", h.limiter.maxFailures, secs))
}

// Logout handles POST /api/v1/auth/logout by clearing the session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	h.auth.clearSessionCookie(c)
	SuccessResponse(c, gin.H{"authenticated": false})
}

// RegisterRoutes registers the auth routes under /api/v1/auth, the one prefix
// AuthGate leaves reachable before a session exists.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.GET("/status", h.GetStatus)
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
	}
}
