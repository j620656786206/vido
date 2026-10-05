package zhtw

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConverter stands in for OpenCC. s2tw swaps characters only; s2twp also
// rewrites Taiwan phrases (软件→軟體), exactly what real OpenCC does.
type fakeConverter struct {
	err      error
	profiles []string
}

var s2twChars = strings.NewReplacer("这", "這", "个", "個", "软", "軟", "质", "質")

func (f *fakeConverter) ConvertS2TWP(b []byte) ([]byte, error) {
	f.profiles = append(f.profiles, "s2twp")
	if f.err != nil {
		return nil, f.err
	}
	return []byte(s2twChars.Replace(strings.ReplaceAll(string(b), "软件", "軟體"))), nil
}

func (f *fakeConverter) ConvertS2TW(b []byte) ([]byte, error) {
	f.profiles = append(f.profiles, "s2tw")
	if f.err != nil {
		return nil, f.err
	}
	return []byte(s2twChars.Replace(string(b))), nil
}

const simplified = "这个软件的质量很好"

func TestFinalize_ScriptThenVocabulary(t *testing.T) {
	conv := &fakeConverter{}
	out, err := Finalize(conv, simplified, []string{"US"})
	require.NoError(t, err)
	assert.Equal(t, []string{"s2twp"}, conv.profiles)
	assert.Equal(t, "這個軟體的品質很好", out)
}

// Alexyu 2026-10-05: mainland, Hong Kong and Macau titles keep their own
// wording — characters only (s2tw), no Taiwan phrases, no lexicon.
func TestFinalize_OwnWordingIsCharactersOnly(t *testing.T) {
	for _, countries := range [][]string{{"us", " cn "}, {"HK"}, {"mo"}} {
		conv := &fakeConverter{}
		out, err := Finalize(conv, simplified, countries)
		require.NoError(t, err)
		assert.Equal(t, []string{"s2tw"}, conv.profiles, "%v", countries)
		assert.Equal(t, "這個軟件的質量很好", out, "%v", countries)
	}
}

func TestFinalize_NilConverterOnlyRewritesVocabulary(t *testing.T) {
	out, err := Finalize(nil, "這個軟件的質量很好", nil)
	require.NoError(t, err)
	assert.Equal(t, "這個軟體的品質很好", out)
}

func TestFinalize_ConverterErrorReturnsInputThroughVocabulary(t *testing.T) {
	boom := errors.New("opencc down")
	out, err := Finalize(&fakeConverter{err: boom}, "這個軟件的質量很好", nil)
	require.ErrorIs(t, err, boom)
	assert.Equal(t, "這個軟體的品質很好", out)
}

func TestKeepsOwnWording(t *testing.T) {
	assert.True(t, KeepsOwnWording([]string{"US", "cn"}))
	assert.True(t, KeepsOwnWording([]string{" CN "}))
	assert.True(t, KeepsOwnWording([]string{"hk"}))
	assert.True(t, KeepsOwnWording([]string{"TW", "MO"}))
	assert.False(t, KeepsOwnWording([]string{"TW", "US", "JP"}))
	assert.False(t, KeepsOwnWording(nil))
}
