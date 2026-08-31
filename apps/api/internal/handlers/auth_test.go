package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func testAuthenticator(password string) *Authenticator {
	return NewAuthenticator(password, []byte("test-session-secret-value-abc123"))
}

func newAuthTestRouter(a *Authenticator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	grp := r.Group("/api/v1", AuthGate(a))
	NewAuthHandler(a).RegisterRoutes(grp)
	grp.GET("/protected", func(c *gin.Context) { SuccessResponse(c, gin.H{"ok": true}) })
	return r
}

func TestAuthenticator_TokenRoundTrip(t *testing.T) {
	a := testAuthenticator("hunter2")
	now := time.Now()
	tok := a.issueToken(now)

	if !a.verifyToken(tok, now) {
		t.Fatal("freshly issued token should verify")
	}
	if a.verifyToken(tok, now.Add(defaultSessionTTL+time.Minute)) {
		t.Fatal("expired token must not verify")
	}
	if a.verifyToken(tok+"x", now) {
		t.Fatal("tampered token must not verify")
	}
	if a.verifyToken("not-a-token", now) {
		t.Fatal("malformed token must not verify")
	}
	other := NewAuthenticator("hunter2", []byte("a-completely-different-secret-xyz"))
	if other.verifyToken(tok, now) {
		t.Fatal("token signed with a different secret must not verify")
	}
}

func TestAuthenticator_CheckPassword(t *testing.T) {
	a := testAuthenticator("s3cr3t")
	if !a.checkPassword("s3cr3t") {
		t.Fatal("correct password should pass")
	}
	if a.checkPassword("wrong") {
		t.Fatal("wrong password should fail")
	}
	if a.checkPassword("") {
		t.Fatal("empty candidate should fail")
	}
}

func TestAuthenticator_Enabled(t *testing.T) {
	if testAuthenticator("").Enabled() {
		t.Fatal("empty password means auth disabled")
	}
	if !testAuthenticator("x").Enabled() {
		t.Fatal("non-empty password means auth enabled")
	}
}

func TestAuthGate_DisabledPassesThrough(t *testing.T) {
	r := newAuthTestRouter(testAuthenticator(""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("disabled auth should allow through, got %d", w.Code)
	}
}

func TestAuthGate_BlocksWithoutCookie(t *testing.T) {
	r := newAuthTestRouter(testAuthenticator("pw"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing cookie should 401, got %d", w.Code)
	}
}

func TestAuthGate_AllowsValidCookie(t *testing.T) {
	a := testAuthenticator("pw")
	r := newAuthTestRouter(a)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: a.issueToken(time.Now())})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("valid cookie should pass, got %d", w.Code)
	}
}

func TestAuthGate_RejectsForgedCookie(t *testing.T) {
	r := newAuthTestRouter(testAuthenticator("pw"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	// A far-future expiry with a bogus signature must not get in.
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "99999999999.forged"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("forged cookie should 401, got %d", w.Code)
	}
}

func TestAuthGate_LoginPathIsPublic(t *testing.T) {
	a := testAuthenticator("pw")
	r := newAuthTestRouter(a)
	body, _ := json.Marshal(LoginRequest{Password: "pw"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login must be reachable without a session, got %d", w.Code)
	}
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("successful login should set a session cookie")
	}
}

func TestAuthHandler_LoginWrongPassword(t *testing.T) {
	r := newAuthTestRouter(testAuthenticator("pw"))
	body, _ := json.Marshal(LoginRequest{Password: "nope"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password should 401, got %d", w.Code)
	}
	if len(w.Result().Cookies()) != 0 {
		t.Fatal("failed login must not set a session cookie")
	}
}

func TestAuthHandler_StatusReportsUnauthenticated(t *testing.T) {
	r := newAuthTestRouter(testAuthenticator("pw"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status should be reachable, got %d", w.Code)
	}
	var resp struct {
		Data struct {
			AuthEnabled   bool `json:"authEnabled"`
			Authenticated bool `json:"authenticated"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !resp.Data.AuthEnabled || resp.Data.Authenticated {
		t.Fatalf("expected authEnabled=true authenticated=false, got %+v", resp.Data)
	}
}

func TestLoginLimiter(t *testing.T) {
	lim := newLoginLimiter(3, time.Minute)
	now := time.Now()
	ip := "10.0.0.5"

	// Two failures: still allowed.
	lim.recordFailure(ip, now)
	lim.recordFailure(ip, now)
	if ok, _ := lim.allow(ip, now); !ok {
		t.Fatal("still allowed before hitting maxFailures")
	}
	// Third failure trips the lockout.
	lim.recordFailure(ip, now)
	ok, retry := lim.allow(ip, now)
	if ok {
		t.Fatal("should be locked after maxFailures")
	}
	if retry <= 0 || retry > time.Minute {
		t.Fatalf("unexpected retryAfter %v", retry)
	}
	// Lockout expires.
	if ok, _ := lim.allow(ip, now.Add(time.Minute+time.Second)); !ok {
		t.Fatal("allowed again after lockout window")
	}
	// A good login clears the IP's history.
	lim.recordFailure(ip, now)
	lim.recordSuccess(ip)
	if ok, _ := lim.allow(ip, now); !ok {
		t.Fatal("recordSuccess should reset the IP")
	}
}

func TestAuthHandler_LoginLockout(t *testing.T) {
	a := testAuthenticator("pw")
	r := newAuthTestRouter(a)
	wrong, _ := json.Marshal(LoginRequest{Password: "wrong"})

	// Five wrong attempts (the default threshold) → all 401.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrong))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}
	// Now locked out: even the CORRECT password gets 429.
	good, _ := json.Marshal(LoginRequest{Password: "pw"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(good))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after lockout, got %d", w.Code)
	}
}

func TestAuthHandler_StatusWhenDisabled(t *testing.T) {
	r := newAuthTestRouter(testAuthenticator(""))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp struct {
		Data struct {
			AuthEnabled   bool `json:"authEnabled"`
			Authenticated bool `json:"authenticated"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.AuthEnabled || !resp.Data.Authenticated {
		t.Fatalf("disabled auth should report authEnabled=false authenticated=true, got %+v", resp.Data)
	}
}
