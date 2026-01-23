package douban

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChineseConverter(t *testing.T) {
	converter := NewChineseConverter(nil)
	assert.NotNil(t, converter)
}

func TestChineseConverter_ToTraditional(t *testing.T) {
	converter := NewChineseConverter(nil)

	tests := []struct {
		name       string
		simplified string
		want       string
	}{
		{
			name:       "basic conversion",
			simplified: "国家",
			want:       "國家",
		},
		{
			name:       "movie title - Parasite",
			simplified: "寄生虫",
			want:       "寄生蟲",
		},
		{
			name:       "longer text",
			simplified: "基泽一家四口全是无业游民",
			want:       "基澤一家四口全是無業遊民", // 游 -> 遊 (Taiwan phrase conversion)
		},
		{
			name:       "empty string",
			simplified: "",
			want:       "",
		},
		{
			name:       "already traditional - unchanged",
			simplified: "台灣",
			want:       "臺灣", // s2twp converts to Taiwan standard
		},
		{
			name:       "mixed content",
			simplified: "Hello 世界",
			want:       "Hello 世界",
		},
		{
			name:       "numbers unchanged",
			simplified: "2019年",
			want:       "2019年",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.ToTraditional(tt.simplified)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChineseConverter_IsTraditional(t *testing.T) {
	converter := NewChineseConverter(nil)

	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "traditional text",
			text: "這是繁體中文，國家發展很好",
			want: true,
		},
		{
			name: "simplified text",
			text: "这是简体中文，国家发展很好",
			want: false,
		},
		{
			name: "empty string",
			text: "",
			want: false,
		},
		{
			name: "english only",
			text: "Hello World",
			want: false,
		},
		{
			name: "numbers only",
			text: "12345",
			want: false,
		},
		{
			name: "mixed with traditional",
			text: "電影很好看",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := converter.IsTraditional(tt.text)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChineseConverter_ConvertIfSimplified(t *testing.T) {
	converter := NewChineseConverter(nil)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simplified gets converted",
			input: "国家发展",
			want:  "國家發展",
		},
		{
			name:  "traditional stays unchanged",
			input: "國家發展",
			want:  "國家發展",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.ConvertIfSimplified(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGlobalConverter(t *testing.T) {
	// Test the global convenience functions
	t.Run("ToTraditional", func(t *testing.T) {
		result, err := ToTraditional("国家")
		require.NoError(t, err)
		assert.Equal(t, "國家", result)
	})

	t.Run("ConvertIfSimplified", func(t *testing.T) {
		result, err := ConvertIfSimplified("国家")
		require.NoError(t, err)
		assert.Equal(t, "國家", result)
	})
}

func TestChineseConverter_MovieTitles(t *testing.T) {
	converter := NewChineseConverter(nil)

	// Test with actual movie titles that might appear on Douban
	tests := []struct {
		name       string
		simplified string
		want       string
	}{
		{
			name:       "Parasite",
			simplified: "寄生虫",
			want:       "寄生蟲",
		},
		{
			name:       "Farewell My Concubine",
			simplified: "霸王别姬",
			want:       "霸王別姬",
		},
		{
			name:       "In the Mood for Love",
			simplified: "花样年华",
			want:       "花樣年華",
		},
		{
			name:       "Crouching Tiger Hidden Dragon",
			simplified: "卧虎藏龙",
			want:       "臥虎藏龍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.ToTraditional(tt.simplified)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChineseConverter_Concurrent(t *testing.T) {
	converter := NewChineseConverter(nil)

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := converter.ToTraditional("国家发展")
			assert.NoError(t, err)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
