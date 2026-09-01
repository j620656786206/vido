package handlers

import (
	"sync"
	"time"
)

// loginLimiter throttles password guessing per client IP. The single shared
// password is the only secret, so an online brute-force is the practical attack;
// this caps a client to maxFailures wrong tries, then locks that IP out for a
// window. State is in-memory (fine for a single-instance home NAS) and
// self-prunes, so it never grows unbounded.
type loginLimiter struct {
	mu          sync.Mutex
	byIP        map[string]*loginAttempts
	maxFailures int
	lockout     time.Duration
}

type loginAttempts struct {
	failures    int
	lockedUntil time.Time
	seen        time.Time
}

func newLoginLimiter(maxFailures int, lockout time.Duration) *loginLimiter {
	return &loginLimiter{
		byIP:        make(map[string]*loginAttempts),
		maxFailures: maxFailures,
		lockout:     lockout,
	}
}

// allow reports whether ip may attempt a login now. When locked out, retryAfter
// is the remaining wait.
func (l *loginLimiter) allow(ip string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)

	rec := l.byIP[ip]
	if rec == nil {
		return true, 0
	}
	if now.Before(rec.lockedUntil) {
		return false, rec.lockedUntil.Sub(now)
	}
	return true, 0
}

// recordFailure counts a wrong password; the maxFailures-th one locks the IP.
//
// It returns what the CALLER has to tell the user, because the UI cannot work
// either number out for itself: `remaining` is how many tries are left before
// the lock, and `lockout` is non-zero exactly on the attempt that just tripped
// it. Returning the lockout here (rather than leaving it to the next request's
// allow() check) is what makes the response the user sees match the documented
// rule — before this, the maxFailures-th wrong password still answered 401 and
// the 429 only appeared on the try AFTER the one that locked them out.
func (l *loginLimiter) recordFailure(ip string, now time.Time) (remaining int, lockout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec := l.byIP[ip]
	if rec == nil {
		rec = &loginAttempts{}
		l.byIP[ip] = rec
	}
	rec.seen = now
	rec.failures++
	if rec.failures >= l.maxFailures {
		rec.lockedUntil = now.Add(l.lockout)
		rec.failures = 0
		return 0, l.lockout
	}
	return l.maxFailures - rec.failures, 0
}

// recordSuccess clears an IP's failure history on a good login.
func (l *loginLimiter) recordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.byIP, ip)
}

// pruneLocked drops entries that are unlocked and idle. Caller holds mu.
func (l *loginLimiter) pruneLocked(now time.Time) {
	for ip, rec := range l.byIP {
		if now.After(rec.lockedUntil) && now.Sub(rec.seen) > time.Hour {
			delete(l.byIP, ip)
		}
	}
}
