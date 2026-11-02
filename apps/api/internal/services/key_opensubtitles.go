package services

// OpenSubtitles needs three values (disc-2026-11-opensubtitles-keys): the API
// key of an app registered on opensubtitles.com for searching, and the user's
// account for logging in, which downloading requires. The storage names are
// the ones the provider has read since Story 8.x, so nothing moves.
const (
	KeyOpenSubtitles         KeyName = "opensubtitles"
	KeyOpenSubtitlesUsername KeyName = "opensubtitles_username"
	KeyOpenSubtitlesPassword KeyName = "opensubtitles_password"

	SecretNameOpenSubtitles         = "opensubtitles_api_key"
	SecretNameOpenSubtitlesUsername = "opensubtitles_username"
	SecretNameOpenSubtitlesPassword = "opensubtitles_password"
)

// isSecretWord marks values that are never shown, not even first6…last4.
func isSecretWord(name KeyName) bool {
	return name == KeyOpenSubtitlesPassword
}
