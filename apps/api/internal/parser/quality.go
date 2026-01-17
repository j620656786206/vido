package parser

// DetectQuality extracts and normalizes the video quality from a filename.
// Returns empty string if no quality indicator is found.
func DetectQuality(filename string) string {
	match := qualityPattern.FindStringSubmatch(filename)
	if match == nil {
		return ""
	}
	return normalizeQuality(match[1])
}

// DetectSource extracts and normalizes the release source from a filename.
// Returns empty string if no source indicator is found.
func DetectSource(filename string) string {
	match := sourcePattern.FindStringSubmatch(filename)
	if match == nil {
		return ""
	}
	return normalizeSource(match[1])
}

// DetectVideoCodec extracts and normalizes the video codec from a filename.
// Returns empty string if no codec indicator is found.
func DetectVideoCodec(filename string) string {
	match := videoCodecPattern.FindStringSubmatch(filename)
	if match == nil {
		return ""
	}
	return normalizeVideoCodec(match[1])
}

// DetectAudioCodec extracts and normalizes the audio codec from a filename.
// Returns empty string if no audio codec indicator is found.
func DetectAudioCodec(filename string) string {
	match := audioCodecPattern.FindStringSubmatch(filename)
	if match == nil {
		return ""
	}
	return normalizeAudioCodec(match[1])
}

// DetectReleaseGroup extracts the release group from a filename.
// Returns empty string if no release group is found.
func DetectReleaseGroup(filename string) string {
	match := releaseGroupPattern.FindStringSubmatch(filename)
	if match == nil {
		return ""
	}
	return match[1]
}
