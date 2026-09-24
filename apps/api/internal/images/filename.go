package images

import "regexp"

// posterFileName is the ONLY shape an uploaded poster file may have: what
// ImageProcessor writes — <mediaID>.jpg and <mediaID>-thumb.jpg, where the
// media ID is a UUID ("-thumb" is covered by the character class). Anything
// else (a "..", a sub-directory, another extension, an encoded slash) is not a
// poster. Shared by the file handler (what may be served) and the orphan sweep
// (what may be deleted), so the two can never disagree.
var posterFileName = regexp.MustCompile(`^[A-Za-z0-9_-]+\.jpg$`)

// IsPosterFileName reports whether name is shaped like an uploaded poster file.
func IsPosterFileName(name string) bool {
	return posterFileName.MatchString(name)
}
