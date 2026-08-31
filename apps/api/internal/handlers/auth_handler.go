package handlers

import (
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
		c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		ErrorResponse(c, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS",
			"嘗試次數過多",
			"密碼錯誤太多次,請稍後再試。")
		return
	}
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "密碼為必填")
		return
	}
	if !h.auth.checkPassword(req.Password) {
		h.limiter.recordFailure(ip, time.Now())
		ErrorResponse(c, http.StatusUnauthorized, "INVALID_CREDENTIALS",
			"密碼錯誤",
			"請確認密碼後再試一次。")
		return
	}
	h.limiter.recordSuccess(ip)
	h.auth.setSessionCookie(c, h.auth.issueToken(time.Now()))
	SuccessResponse(c, gin.H{"authEnabled": true, "authenticated": true})
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
