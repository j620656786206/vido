package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

	// Two failures: still allowed, and each one reports how many tries are left
	// so the caller can warn BEFORE the lock rather than explain it after.
	if remaining, lockout := lim.recordFailure(ip, now); remaining != 2 || lockout != 0 {
		t.Fatalf("first failure: got remaining=%d lockout=%v, want 2/0", remaining, lockout)
	}
	if remaining, lockout := lim.recordFailure(ip, now); remaining != 1 || lockout != 0 {
		t.Fatalf("second failure: got remaining=%d lockout=%v, want 1/0", remaining, lockout)
	}
	if ok, _ := lim.allow(ip, now); !ok {
		t.Fatal("still allowed before hitting maxFailures")
	}
	// Third failure trips the lockout, and says so on the spot — the attempt
	// that locks you out is the one that has to tell you.
	if remaining, lockout := lim.recordFailure(ip, now); remaining != 0 || lockout != time.Minute {
		t.Fatalf("third failure: got remaining=%d lockout=%v, want 0/1m", remaining, lockout)
	}
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

	// The first four wrong attempts are plain 401s.
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrong))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}
	// The FIFTH — the one that actually trips the documented "5 wrong tries"
	// threshold — must answer 429 itself. It used to answer 401 identically to
	// the first four, so the lock only became visible on a sixth attempt: the
	// user was locked out by a response that gave no sign of it.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrong))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("fifth attempt: expected 429, got %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("expected Retry-After 60 on the locking attempt, got %q", got)
	}

	// Now locked out: even the CORRECT password gets 429, still with the header.
	good, _ := json.Marshal(LoginRequest{Password: "pw"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(good))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after lockout, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("a locked-out response must carry Retry-After — the UI counts down with it")
	}
}

// The login screen renders `suggestion`, so it is the only channel that can
// tell a user they are two tries from a lockout. Assert the words, not just the
// status code.
func TestAuthHandler_LoginSuggestionWarnsBeforeLockout(t *testing.T) {
	a := testAuthenticator("pw")
	r := newAuthTestRouter(a)
	wrong, _ := json.Marshal(LoginRequest{Password: "wrong"})

	suggestionAt := func(attempt int) string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrong))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var resp struct {
			Error struct {
				Suggestion string `json:"suggestion"`
			} `json:"error"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("attempt %d: decode: %v", attempt, err)
		}
		return resp.Error.Suggestion
	}

	// Early failures stay generic — a countdown from the first wrong keypress
	// would read as an invitation to guess.
	if got := suggestionAt(1); got != "請確認密碼後再試一次。" {
		t.Fatalf("attempt 1 suggestion = %q", got)
	}
	suggestionAt(2)
	// Two tries left: now the warning is worth spending.
	if got := suggestionAt(3); !strings.Contains(got, "還可以再試 2 次") {
		t.Fatalf("attempt 3 suggestion = %q, want a remaining-tries warning", got)
	}
	if got := suggestionAt(4); !strings.Contains(got, "還可以再試 1 次") {
		t.Fatalf("attempt 4 suggestion = %q, want a remaining-tries warning", got)
	}
	// The locking attempt says how long, not just that it happened.
	if got := suggestionAt(5); !strings.Contains(got, "60 秒") {
		t.Fatalf("attempt 5 suggestion = %q, want the wait in seconds", got)
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
