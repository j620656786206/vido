package wikipedia

import (
	"regexp"
	"strconv"
	"strings"
)

// InfoboxParser parses Wikipedia Infobox templates from wikitext
type InfoboxParser struct {
	// supportedTemplates lists all supported Infobox template names
	supportedTemplates []string
}

// NewInfoboxParser creates a new Infobox parser
func NewInfoboxParser() *InfoboxParser {
	return &InfoboxParser{
		supportedTemplates: []string{
			// English templates
			"Infobox film",
			"Infobox Film",
			"Infobox movie",
			"Infobox television",
			"Infobox Television",
			"Infobox TV series",
			"Infobox animanga/Header",
			"Infobox animanga/Anime",
			"Infobox animanga",
			// Chinese templates (Traditional and Simplified)
			"電影資訊框",
			"電視節目資訊框",
			"電視劇資訊框",
			"動畫資訊框",
			"Infobox 電影",
			"Infobox 電視劇",
		},
	}
}

// Parse extracts InfoboxData from wikitext
func (p *InfoboxParser) Parse(wikitext string) (*InfoboxData, error) {
	// Find all Infobox templates in the wikitext
	// This handles animanga templates which have multiple blocks
	allFields := make(map[string]string)
	var firstTemplateType string
	found := false

	// Find all infobox positions
	positions := p.findAllInfoboxPositions(wikitext)

	for _, pos := range positions {
		content := p.extractBalancedBraces(wikitext[pos.start:])
		if content == "" {
			continue
		}

		found = true
		if firstTemplateType == "" {
			firstTemplateType = pos.templateType
		}

		// Extract fields from this Infobox
		fields := p.parseInfoboxFields(content)

		// Merge fields (first occurrence wins for duplicates)
		for k, v := range fields {
			if _, exists := allFields[k]; !exists {
				allFields[k] = v
			}
		}
	}

	if !found {
		return nil, &ParseError{
			Field:  "infobox",
			Reason: "no infobox found in wikitext",
		}
	}

	// Parse the Infobox content
	data := &InfoboxData{
		Type: firstTemplateType,
	}

	// Map fields to InfoboxData based on template type
	p.mapFieldsToData(allFields, data, firstTemplateType)

	return data, nil
}

// infoboxPosition holds the position and type of an infobox
type infoboxPosition struct {
	start        int
	templateType string
}

// findAllInfoboxPositions finds all infobox templates in the wikitext
func (p *InfoboxParser) findAllInfoboxPositions(wikitext string) []infoboxPosition {
	var positions []infoboxPosition

	for _, template := range p.supportedTemplates {
		// Build regex pattern for this template
		pattern := `(?i)\{\{\s*` + regexp.QuoteMeta(template)
		re := regexp.MustCompile(pattern)

		matches := re.FindAllStringIndex(wikitext, -1)
		for _, loc := range matches {
			positions = append(positions, infoboxPosition{
				start:        loc[0],
				templateType: p.normalizeTemplateType(template),
			})
		}
	}

	return positions
}

// extractInfobox finds and extracts the Infobox template content
func (p *InfoboxParser) extractInfobox(wikitext string) (content string, templateType string) {
	// Try each supported template
	for _, template := range p.supportedTemplates {
		// Build regex pattern for this template
		// Handle case-insensitive matching for the template name
		pattern := `(?i)\{\{\s*` + regexp.QuoteMeta(template) + `\s*\n`
		re := regexp.MustCompile(pattern)

		loc := re.FindStringIndex(wikitext)
		if loc == nil {
			continue
		}

		// Found the start of the Infobox, now find the matching closing }}
		startPos := loc[0]
		content := p.extractBalancedBraces(wikitext[startPos:])
		if content != "" {
			return content, p.normalizeTemplateType(template)
		}
	}

	return "", ""
}

// extractBalancedBraces extracts content with balanced {{ and }}
func (p *InfoboxParser) extractBalancedBraces(text string) string {
	if len(text) < 2 || text[0:2] != "{{" {
		return ""
	}

	depth := 0
	for i := 0; i < len(text); i++ {
		if i+1 < len(text) && text[i:i+2] == "{{" {
			depth++
			i++ // Skip the second {
		} else if i+1 < len(text) && text[i:i+2] == "}}" {
			depth--
			if depth == 0 {
				return text[2 : i] // Return content without outer {{ }}
			}
			i++ // Skip the second }
		}
	}

	return ""
}

// parseInfoboxFields parses the key-value pairs from Infobox content
func (p *InfoboxParser) parseInfoboxFields(content string) map[string]string {
	fields := make(map[string]string)

	// Split by | but handle nested templates
	lines := p.splitInfoboxLines(content)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Find the = separator
		eqPos := strings.Index(line, "=")
		if eqPos <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:eqPos])
		value := strings.TrimSpace(line[eqPos+1:])

		// Clean the value
		value = p.cleanValue(value)

		if key != "" && value != "" {
			fields[strings.ToLower(key)] = value
		}
	}

	return fields
}

// splitInfoboxLines splits Infobox content by | while respecting nested templates
func (p *InfoboxParser) splitInfoboxLines(content string) []string {
	var lines []string
	var current strings.Builder
	depth := 0

	for i := 0; i < len(content); i++ {
		ch := content[i]

		// Track template depth
		if i+1 < len(content) {
			if content[i:i+2] == "{{" {
				depth++
				current.WriteString("{{")
				i++
				continue
			} else if content[i:i+2] == "}}" {
				depth--
				current.WriteString("}}")
				i++
				continue
			} else if content[i:i+2] == "[[" {
				depth++
				current.WriteString("[[")
				i++
				continue
			} else if content[i:i+2] == "]]" {
				depth--
				current.WriteString("]]")
				i++
				continue
			}
		}

		// Split on | only at top level
		if ch == '|' && depth == 0 {
			if current.Len() > 0 {
				lines = append(lines, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	// Add the last line
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}

	return lines
}

// cleanValue cleans wiki markup from a value
func (p *InfoboxParser) cleanValue(value string) string {
	// Remove wiki links: [[Link|Text]] → Text or [[Link]] → Link
	linkRe := regexp.MustCompile(`\[\[(?:[^|\]]*\|)?([^\]]+)\]\]`)
	value = linkRe.ReplaceAllString(value, "$1")

	// Remove simple templates like {{KOR}}, {{Film date|...}}
	// But preserve the main text
	filmDateRe := regexp.MustCompile(`\{\{Film date\|([^}|]+)(?:\|[^}]*)?\}\}`)
	value = filmDateRe.ReplaceAllString(value, "$1")

	startDateRe := regexp.MustCompile(`\{\{Start date\|(\d{4})\|(\d+)\|(\d+)\}\}`)
	value = startDateRe.ReplaceAllString(value, "$1-$2-$3")

	// Remove country templates like {{KOR}}, {{USA}}, {{JPN}}
	countryRe := regexp.MustCompile(`\{\{[A-Z]{2,3}\}\}`)
	value = countryRe.ReplaceAllString(value, "")

	// Remove other simple templates but keep content
	simpleTemplateRe := regexp.MustCompile(`\{\{([^{}|]+)\}\}`)
	value = simpleTemplateRe.ReplaceAllString(value, "$1")

	// Remove HTML tags
	htmlRe := regexp.MustCompile(`<[^>]+>`)
	value = htmlRe.ReplaceAllString(value, "")

	// Remove HTML entities
	value = strings.ReplaceAll(value, "&nbsp;", " ")
	value = strings.ReplaceAll(value, "&ndash;", "-")
	value = strings.ReplaceAll(value, "&mdash;", "—")

	// Clean up whitespace
	value = strings.TrimSpace(value)
	spaceRe := regexp.MustCompile(`\s+`)
	value = spaceRe.ReplaceAllString(value, " ")

	return value
}

// mapFieldsToData maps parsed fields to InfoboxData based on template type
func (p *InfoboxParser) mapFieldsToData(fields map[string]string, data *InfoboxData, templateType string) {
	// Name/Title mapping
	for _, key := range []string{"name", "show_name", "名稱", "片名", "劇名"} {
		if v, ok := fields[key]; ok && data.Name == "" {
			data.Name = v
		}
	}

	// Original name
	for _, key := range []string{"original_name", "原名", "原片名"} {
		if v, ok := fields[key]; ok && data.OriginalName == "" {
			data.OriginalName = v
		}
	}

	// Image
	for _, key := range []string{"image", "圖片", "封面"} {
		if v, ok := fields[key]; ok && data.Image == "" {
			data.Image = p.cleanImageName(v)
		}
	}

	// Director
	for _, key := range []string{"director", "導演", "导演"} {
		if v, ok := fields[key]; ok && data.Director == "" {
			data.Director = v
		}
	}

	// Creator (for TV shows)
	for _, key := range []string{"creator", "創作者", "创作者"} {
		if v, ok := fields[key]; ok && data.Creator == "" {
			data.Creator = v
		}
	}

	// Starring/Cast
	for _, key := range []string{"starring", "主演", "演員", "cast"} {
		if v, ok := fields[key]; ok && len(data.Starring) == 0 {
			data.Starring = p.parseList(v)
		}
	}

	// Producer
	for _, key := range []string{"producer", "製片", "制片人"} {
		if v, ok := fields[key]; ok && data.Producer == "" {
			data.Producer = v
		}
	}

	// Writer
	for _, key := range []string{"writer", "screenplay", "編劇", "编剧"} {
		if v, ok := fields[key]; ok && data.Writer == "" {
			data.Writer = v
		}
	}

	// Music
	for _, key := range []string{"music", "音樂", "音乐"} {
		if v, ok := fields[key]; ok && data.Music == "" {
			data.Music = v
		}
	}

	// Country
	for _, key := range []string{"country", "國家", "国家", "地區", "地区"} {
		if v, ok := fields[key]; ok && data.Country == "" {
			data.Country = v
		}
	}

	// Language
	for _, key := range []string{"language", "語言", "语言"} {
		if v, ok := fields[key]; ok && data.Language == "" {
			data.Language = v
		}
	}

	// Genre
	for _, key := range []string{"genre", "類型", "类型"} {
		if v, ok := fields[key]; ok && len(data.Genre) == 0 {
			data.Genre = p.parseList(v)
		}
	}

	// Released/First aired
	for _, key := range []string{"released", "release_date", "上映日期", "首播"} {
		if v, ok := fields[key]; ok && data.Released == "" {
			data.Released = v
			// Try to extract year
			if year := p.extractYear(v); year > 0 {
				data.Year = year
			}
		}
	}

	for _, key := range []string{"first_aired", "first", "首播日期"} {
		if v, ok := fields[key]; ok && data.FirstAired == "" {
			data.FirstAired = v
			// Try to extract year
			if year := p.extractYear(v); year > 0 && data.Year == 0 {
				data.Year = year
			}
		}
	}

	// Runtime
	for _, key := range []string{"runtime", "片長", "片长", "時長", "时长"} {
		if v, ok := fields[key]; ok && data.Runtime == "" {
			data.Runtime = v
		}
	}

	// Number of seasons/episodes (for TV)
	for _, key := range []string{"num_seasons", "季數", "季数"} {
		if v, ok := fields[key]; ok && data.NumSeasons == 0 {
			if num, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				data.NumSeasons = num
			}
		}
	}

	for _, key := range []string{"num_episodes", "集數", "集数"} {
		if v, ok := fields[key]; ok && data.NumEpisodes == 0 {
			if num, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				data.NumEpisodes = num
			}
		}
	}

	// Studio (for anime)
	for _, key := range []string{"studio", "動畫製作", "动画制作"} {
		if v, ok := fields[key]; ok && data.Studio == "" {
			data.Studio = v
		}
	}
}

// normalizeTemplateType normalizes template name to a standard type
func (p *InfoboxParser) normalizeTemplateType(template string) string {
	lower := strings.ToLower(template)

	if strings.Contains(lower, "film") || strings.Contains(lower, "movie") || strings.Contains(lower, "電影") {
		return "film"
	}
	if strings.Contains(lower, "television") || strings.Contains(lower, "tv") || strings.Contains(lower, "電視") {
		return "television"
	}
	if strings.Contains(lower, "animanga") || strings.Contains(lower, "anime") || strings.Contains(lower, "動畫") {
		return "anime"
	}

	return "unknown"
}

// cleanImageName extracts the clean image filename
func (p *InfoboxParser) cleanImageName(value string) string {
	// Remove File: or Image: prefix if present
	value = strings.TrimPrefix(value, "File:")
	value = strings.TrimPrefix(value, "Image:")
	value = strings.TrimPrefix(value, "file:")
	value = strings.TrimPrefix(value, "image:")

	// Remove any remaining wiki syntax
	if idx := strings.Index(value, "|"); idx > 0 {
		value = value[:idx]
	}

	return strings.TrimSpace(value)
}

// parseList parses a comma or newline separated list
func (p *InfoboxParser) parseList(value string) []string {
	var result []string

	// Split by common separators
	separators := []string{"\n", "、", "，", ",", "<br>", "<br/>", "<br />"}
	items := []string{value}

	for _, sep := range separators {
		var newItems []string
		for _, item := range items {
			newItems = append(newItems, strings.Split(item, sep)...)
		}
		items = newItems
	}

	// Clean and filter items
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" && item != "*" && item != "-" {
			result = append(result, item)
		}
	}

	return result
}

// extractYear extracts a 4-digit year from a string
func (p *InfoboxParser) extractYear(value string) int {
	yearRe := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	match := yearRe.FindString(value)
	if match != "" {
		if year, err := strconv.Atoi(match); err == nil {
			return year
		}
	}
	return 0
}

// DetectMediaType tries to detect the media type from Infobox data
func (p *InfoboxParser) DetectMediaType(data *InfoboxData) MediaType {
	switch data.Type {
	case "film":
		return MediaTypeMovie
	case "television":
		return MediaTypeTV
	case "anime":
		return MediaTypeAnime
	}

	// Try to detect from content
	if data.NumSeasons > 0 || data.NumEpisodes > 0 {
		return MediaTypeTV
	}
	if data.Studio != "" {
		return MediaTypeAnime
	}
	if data.Runtime != "" {
		return MediaTypeMovie
	}

	return MediaTypeMovie // Default to movie
}
