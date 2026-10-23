package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// trailingYearPattern is a title that ends in a year: "Ever Night 2026",
// "將夜 (2026)", "將夜.2026". Something must come before the year, so a show
// called "1923" keeps its name.
var trailingYearPattern = regexp.MustCompile(`^(.*\S)[\s._]+\(?((?:19|20)\d{2})\)?$`)

// SplitTrailingYear splits a year off the end of a title. It returns the
// title unchanged and 0 when there is none (disc-2026-10-series-wrong-version-a:
// the year is what tells a 2026 remake from the 2018 original).
func SplitTrailingYear(title string) (string, int) {
	title = strings.TrimSpace(title)
	m := trailingYearPattern.FindStringSubmatch(title)
	if m == nil {
		return title, 0
	}
	year, err := strconv.Atoi(m[2])
	if err != nil {
		return title, 0
	}
	return strings.TrimSpace(m[1]), year
}
