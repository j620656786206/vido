package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// sessionCookieName carries the signed session token.
const sessionCookieName = "vido_session"

// authPublicPrefix is the one /api/v1 subtree AuthGate lets through
// unauthenticated: the login/logout/status endpoints must be reachable before a
// session exists.
const authPublicPrefix = "/api/v1/auth/"

// defaultSessionTTL is how long a login stays valid. Deliberately long — this is
// a single shared password for a home NAS, not a multi-user portal, so forcing a
// weekly re-type buys nothing when the password is the only secret.
//
// ⚠️ Tokens are signed with the SESSION SECRET, which is independent of the
// password (VIDO_SESSION_SECRET, else ENCRYPTION_KEY, else a random secret
// persisted under the data dir). So changing VIDO_AUTH_PASSWORD does NOT log
// anyone out — every outstanding session stays valid for the rest of these 30
// days. Only rotating VIDO_SESSION_SECRET (or deleting the persisted
// .session_secret) invalidates them. If someone read the password over your
// shoulder, changing it alone is not enough; rotate the session secret too.
// Said plainly for the user in the login screen's help text.
const defaultSessionTTL = 30 * 24 * time.Hour

// Authenticator holds the single shared password and the key that signs session
// cookies. An empty password means auth is DISABLED and every request passes —
// the pre-VIDO_AUTH_PASSWORD behaviour, kept so a purely-LAN install is never
// forced to set one.
type Authenticator struct {
	password      string
	sessionSecret []byte
	ttl           time.Duration
	secureCookie  bool
}

// NewAuthenticator builds an Authenticator. password == "" disables auth.
func NewAuthenticator(password string, sessionSecret []byte) *Authenticator {
	return &Authenticator{
		password:      password,
		sessionSecret: sessionSecret,
		ttl:           defaultSessionTTL,
	}
}

// Enabled reports whether a password is configured (auth is enforced).
func (a *Authenticator) Enabled() bool { return a.password != "" }

// SetSecureCookie toggles the Secure flag on the session cookie. Leave off for a
// plain-HTTP LAN deployment (a Secure cookie would never be sent over HTTP, so
// login would break); turn on when Vido is fronted by HTTPS.
func (a *Authenticator) SetSecureCookie(secure bool) { a.secureCookie = secure }

// checkPassword compares a candidate against the configured password in constant
// time over a fixed-width hash, so neither the outcome timing nor the password
// length leaks.
func (a *Authenticator) checkPassword(candidate string) bool {
	want := sha256.Sum256([]byte(a.password))
	got := sha256.Sum256([]byte(candidate))
	return subtle.ConstantTimeCompare(want[:], got[:]) == 1
}

// issueToken returns "<expUnix>.<sig>" signed with the session secret.
func (a *Authenticator) issueToken(now time.Time) string {
	payload := strconv.FormatInt(now.Add(a.ttl).Unix(), 10)
	return payload + "." + a.sign(payload)
}

// verifyToken is true when token is well-formed, correctly signed, and unexpired.
func (a *Authenticator) verifyToken(token string, now time.Time) bool {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	if !hmac.Equal([]byte(sig), []byte(a.sign(payload))) {
		return false
	}
	exp, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return false
	}
	return now.Unix() < exp
}

func (a *Authenticator) sign(payload string) string {
	mac := hmac.New(sha256.New, a.sessionSecret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *Authenticator) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, token, int(a.ttl.Seconds()), "/", "", a.secureCookie, true)
}

func (a *Authenticator) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", a.secureCookie, true)
}

// authenticated reports whether the request carries a valid session cookie.
func (a *Authenticator) authenticated(c *gin.Context) bool {
	token, err := c.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return a.verifyToken(token, time.Now())
}

// AuthGate returns middleware that blocks every /api/v1 request lacking a valid
// session with a uniform 401, EXCEPT the /api/v1/auth/* endpoints (login/logout/
// status), which must stay reachable before a session exists. When auth is
// disabled it is a no-op. SameSite=Lax on the session cookie also means a
// cross-site request never carries it, so this gate doubles as CSRF protection.
func AuthGate(a *Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.Enabled() {
			c.Next()
			return
		}
		if strings.HasPrefix(c.FullPath(), authPublicPrefix) {
			c.Next()
			return
		}
		if a.authenticated(c) {
			c.Next()
			return
		}
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHENTICATED",
			"需要登入",
			"請先在登入頁輸入密碼。密碼是部署時設定的 VIDO_AUTH_PASSWORD。")
		c.Abort()
	}
}
