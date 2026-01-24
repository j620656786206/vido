package wikipedia

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sample wikitext for testing
const filmInfoboxWikitext = `
{{Infobox film
| name           = 寄生上流
| original_name  = 기생충
| image          = Parasite (2019 film).png
| director       = [[奉俊昊]]
| producer       = 奉俊昊、郭信愛
| writer         = 奉俊昊、韓進元
| starring       = [[宋康昊]]、[[李善均]]、[[崔宇植]]
| music          = 鄭在日
| country        = {{KOR}}
| language       = 韓語
| released       = {{Film date|2019|5|21|坎城影展}}
| runtime        = 132分鐘
}}
這是一部2019年韓國電影...
`

const tvInfoboxWikitext = `
{{Infobox television
| show_name      = 魷魚遊戲
| image          = Squid Game.jpg
| genre          = 驚悚、生存
| creator        = [[黃東赫]]
| starring       = [[李政宰]]、[[朴海秀]]、[[鄭浩妍]]
| country        = {{KOR}}
| language       = 韓語
| num_seasons    = 1
| num_episodes   = 9
| first_aired    = {{Start date|2021|9|17}}
}}
魷魚遊戲是一部Netflix原創韓劇...
`

const animeInfoboxWikitext = `
{{Infobox animanga/Header
| name           = 鬼滅之刃
| image          = Kimetsu no Yaiba.jpg
| genre          = 動作、黑暗奇幻
}}
{{Infobox animanga/Anime
| director       = 外崎春雄
| studio         = [[ufotable]]
| first          = 2019年4月6日
| last           = 2019年9月28日
}}
鬼滅之刃是一部日本動漫...
`

const chineseFilmInfoboxWikitext = `
{{電影資訊框
| 片名           = 臥虎藏龍
| 原片名         = Crouching Tiger, Hidden Dragon
| 圖片           = Crouching Tiger.jpg
| 導演           = [[李安]]
| 主演           = [[周潤發]]、[[楊紫瓊]]、[[章子怡]]
| 類型           = 武俠、劇情
| 上映日期       = 2000年7月7日
| 片長           = 120分鐘
}}
臥虎藏龍是一部武俠電影...
`

func TestInfoboxParser_Parse(t *testing.T) {
	parser := NewInfoboxParser()

	t.Run("parses film infobox", func(t *testing.T) {
		data, err := parser.Parse(filmInfoboxWikitext)
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, "film", data.Type)
		assert.Equal(t, "寄生上流", data.Name)
		assert.Equal(t, "기생충", data.OriginalName)
		assert.Equal(t, "Parasite (2019 film).png", data.Image)
		assert.Equal(t, "奉俊昊", data.Director)
		assert.Equal(t, "奉俊昊、郭信愛", data.Producer)
		assert.Equal(t, "奉俊昊、韓進元", data.Writer)
		assert.Equal(t, "鄭在日", data.Music)
		assert.Equal(t, "韓語", data.Language)
		assert.Equal(t, "132分鐘", data.Runtime)
		assert.Equal(t, 2019, data.Year)

		// Starring should be parsed as list
		assert.Contains(t, data.Starring, "宋康昊")
		assert.Contains(t, data.Starring, "李善均")
	})

	t.Run("parses television infobox", func(t *testing.T) {
		data, err := parser.Parse(tvInfoboxWikitext)
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, "television", data.Type)
		assert.Equal(t, "魷魚遊戲", data.Name)
		assert.Equal(t, "黃東赫", data.Creator)
		assert.Equal(t, 1, data.NumSeasons)
		assert.Equal(t, 9, data.NumEpisodes)
		assert.Equal(t, 2021, data.Year)

		// Genre should be parsed as list
		assert.Contains(t, data.Genre, "驚悚")
		assert.Contains(t, data.Genre, "生存")
	})

	t.Run("parses anime infobox", func(t *testing.T) {
		data, err := parser.Parse(animeInfoboxWikitext)
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, "anime", data.Type)
		assert.Equal(t, "鬼滅之刃", data.Name)
		assert.Equal(t, "ufotable", data.Studio)
		assert.Equal(t, 2019, data.Year)
	})

	t.Run("parses chinese film infobox", func(t *testing.T) {
		data, err := parser.Parse(chineseFilmInfoboxWikitext)
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, "film", data.Type)
		assert.Equal(t, "臥虎藏龍", data.Name)
		assert.Equal(t, "Crouching Tiger, Hidden Dragon", data.OriginalName)
		assert.Equal(t, "李安", data.Director)
		assert.Equal(t, 2000, data.Year)
		assert.Equal(t, "120分鐘", data.Runtime)
	})

	t.Run("returns error for no infobox", func(t *testing.T) {
		wikitext := "This is just regular text without any infobox."
		_, err := parser.Parse(wikitext)

		assert.Error(t, err)
		assert.IsType(t, &ParseError{}, err)
	})
}

func TestInfoboxParser_cleanValue(t *testing.T) {
	parser := NewInfoboxParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes wiki links with display text",
			input:    "[[奉俊昊|Bong Joon-ho]]",
			expected: "Bong Joon-ho",
		},
		{
			name:     "removes wiki links without display text",
			input:    "[[奉俊昊]]",
			expected: "奉俊昊",
		},
		{
			name:     "removes film date template",
			input:    "{{Film date|2019|5|21|坎城影展}}",
			expected: "2019",
		},
		{
			name:     "removes start date template",
			input:    "{{Start date|2021|9|17}}",
			expected: "2021-9-17",
		},
		{
			name:     "removes country templates",
			input:    "{{KOR}}",
			expected: "",
		},
		{
			name:     "removes HTML tags",
			input:    "<small>small text</small>",
			expected: "small text",
		},
		{
			name:     "handles multiple wiki links",
			input:    "[[宋康昊]]、[[李善均]]",
			expected: "宋康昊、李善均",
		},
		{
			name:     "cleans whitespace",
			input:    "  multiple   spaces  ",
			expected: "multiple spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.cleanValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInfoboxParser_parseList(t *testing.T) {
	parser := NewInfoboxParser()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "comma separated",
			input:    "動作, 冒險, 劇情",
			expected: []string{"動作", "冒險", "劇情"},
		},
		{
			name:     "chinese comma separated",
			input:    "動作、冒險、劇情",
			expected: []string{"動作", "冒險", "劇情"},
		},
		{
			name:     "newline separated",
			input:    "動作\n冒險\n劇情",
			expected: []string{"動作", "冒險", "劇情"},
		},
		{
			name:     "html br separated",
			input:    "動作<br>冒險<br/>劇情",
			expected: []string{"動作", "冒險", "劇情"},
		},
		{
			name:     "single item",
			input:    "動作",
			expected: []string{"動作"},
		},
		{
			name:     "empty items filtered",
			input:    "動作, , 劇情",
			expected: []string{"動作", "劇情"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInfoboxParser_extractYear(t *testing.T) {
	parser := NewInfoboxParser()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "simple year",
			input:    "2019",
			expected: 2019,
		},
		{
			name:     "year in date",
			input:    "2019年5月21日",
			expected: 2019,
		},
		{
			name:     "year in film date template result",
			input:    "2019-5-21",
			expected: 2019,
		},
		{
			name:     "year in text",
			input:    "Released in 2019",
			expected: 2019,
		},
		{
			name:     "no year",
			input:    "unknown date",
			expected: 0,
		},
		{
			name:     "invalid year",
			input:    "1899", // Too old
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractYear(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInfoboxParser_cleanImageName(t *testing.T) {
	parser := NewInfoboxParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "Parasite.jpg",
			expected: "Parasite.jpg",
		},
		{
			name:     "with File prefix",
			input:    "File:Parasite.jpg",
			expected: "Parasite.jpg",
		},
		{
			name:     "with Image prefix",
			input:    "Image:Parasite.jpg",
			expected: "Parasite.jpg",
		},
		{
			name:     "with pipe parameters",
			input:    "Parasite.jpg|300px|thumb",
			expected: "Parasite.jpg",
		},
		{
			name:     "full wiki syntax",
			input:    "File:Parasite.jpg|300px|thumb|caption",
			expected: "Parasite.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.cleanImageName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInfoboxParser_normalizeTemplateType(t *testing.T) {
	parser := NewInfoboxParser()

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "english film",
			template: "Infobox film",
			expected: "film",
		},
		{
			name:     "english movie",
			template: "Infobox movie",
			expected: "film",
		},
		{
			name:     "chinese film",
			template: "電影資訊框",
			expected: "film",
		},
		{
			name:     "english tv",
			template: "Infobox television",
			expected: "television",
		},
		{
			name:     "english tv series",
			template: "Infobox TV series",
			expected: "television",
		},
		{
			name:     "chinese tv",
			template: "電視節目資訊框",
			expected: "television",
		},
		{
			name:     "anime header",
			template: "Infobox animanga/Header",
			expected: "anime",
		},
		{
			name:     "anime",
			template: "Infobox animanga/Anime",
			expected: "anime",
		},
		{
			name:     "chinese anime",
			template: "動畫資訊框",
			expected: "anime",
		},
		{
			name:     "unknown",
			template: "Infobox person",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.normalizeTemplateType(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInfoboxParser_DetectMediaType(t *testing.T) {
	parser := NewInfoboxParser()

	tests := []struct {
		name     string
		data     *InfoboxData
		expected MediaType
	}{
		{
			name:     "film type",
			data:     &InfoboxData{Type: "film"},
			expected: MediaTypeMovie,
		},
		{
			name:     "television type",
			data:     &InfoboxData{Type: "television"},
			expected: MediaTypeTV,
		},
		{
			name:     "anime type",
			data:     &InfoboxData{Type: "anime"},
			expected: MediaTypeAnime,
		},
		{
			name:     "detect TV from seasons",
			data:     &InfoboxData{Type: "unknown", NumSeasons: 3},
			expected: MediaTypeTV,
		},
		{
			name:     "detect TV from episodes",
			data:     &InfoboxData{Type: "unknown", NumEpisodes: 24},
			expected: MediaTypeTV,
		},
		{
			name:     "detect anime from studio",
			data:     &InfoboxData{Type: "unknown", Studio: "ufotable"},
			expected: MediaTypeAnime,
		},
		{
			name:     "detect movie from runtime",
			data:     &InfoboxData{Type: "unknown", Runtime: "120分鐘"},
			expected: MediaTypeMovie,
		},
		{
			name:     "default to movie",
			data:     &InfoboxData{Type: "unknown"},
			expected: MediaTypeMovie,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.DetectMediaType(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInfoboxParser_extractBalancedBraces(t *testing.T) {
	parser := NewInfoboxParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple template",
			input:    "{{template}}",
			expected: "template",
		},
		{
			name:     "nested template",
			input:    "{{outer|{{inner}}}}",
			expected: "outer|{{inner}}",
		},
		{
			name:     "no closing",
			input:    "{{template",
			expected: "",
		},
		{
			name:     "not a template",
			input:    "regular text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractBalancedBraces(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
