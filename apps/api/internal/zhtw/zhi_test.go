package zhtw

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buggyOpenCC reproduces what OpenCC 1.4.2 does to 只 (issue #939): after 是
// or 多 it becomes the measure word 隻, on Simplified and Traditional input
// alike. It also converts a few characters, so the test sees a real script
// step happen around the placeholder.
type buggyOpenCC struct{}

var buggyZhi = strings.NewReplacer("是只", "是隻", "多只", "多隻", "别只", "別隻", "别", "別", "这", "這", "还", "還", "两", "兩", "辆", "輛", "会", "會", "说", "說", "鸟", "鳥")

func (buggyOpenCC) ConvertS2TWP(b []byte) ([]byte, error) {
	return []byte(buggyZhi.Replace(string(b))), nil
}
func (buggyOpenCC) ConvertS2TW(b []byte) ([]byte, error) {
	return []byte(buggyZhi.Replace(string(b))), nil
}

func TestFinalize_KeepsAdverbZhi(t *testing.T) {
	cases := map[string]string{
		// issue #939, Lioness S01E01 01:51 — Simplified and Traditional input
		"是大家都知道还是只有她知道": "是大家都知道還是只有她知道",
		"是大家都知道還是只有她知道": "是大家都知道還是只有她知道",
		// seen in production subtitles, 2026-10
		"不是只有我们在研究这件事": "不是只有我们在研究這件事",
		"我最多只能派两辆":     "我最多只能派兩輛",
		"还是只想著回家":      "還是只想著回家",
		"别只顾着自己":       "別只顾着自己",
	}
	for in, want := range cases {
		for _, countries := range [][]string{nil, {"CN"}} {
			out, err := Finalize(buggyOpenCC{}, in, countries)
			require.NoError(t, err)
			assert.Equal(t, want, out, "input %q countries %v", in, countries)
		}
	}
}

func TestFinalize_LeavesMeasureWordZhi(t *testing.T) {
	cases := map[string]string{
		// a 隻 the input already has is never rewritten
		"那是隻會說話的鳥": "那是隻會說話的鳥",
		// 只 before a noun is the measure word, so OpenCC's 隻 stands
		"这是只狗":    "這是隻狗",
		"抓到最多只的人": "抓到最多隻的人",
	}
	for in, want := range cases {
		out, err := Finalize(buggyOpenCC{}, in, nil)
		require.NoError(t, err)
		assert.Equal(t, want, out, "input %q", in)
	}
}

func TestProtectAdverbZhi(t *testing.T) {
	assert.Equal(t, "還是有", protectAdverbZhi("還是只有"))
	assert.Equal(t, "最多能，頂多是", protectAdverbZhi("最多只能，頂多只是"))
	// no trigger, no follower, or at the end of the text: untouched
	assert.Equal(t, "也只有", protectAdverbZhi("也只有"))
	assert.Equal(t, "是只狗", protectAdverbZhi("是只狗"))
	assert.Equal(t, "就是只", protectAdverbZhi("就是只"))
	// text that already holds the placeholder is left alone, so restoreZhi
	// cannot invent a 只
	assert.Equal(t, "還是只有", protectAdverbZhi("還是只有"))
}

// TestFinalize_RealOpenCC runs the real binary when it is installed (the
// Docker image always has it; CI and most dev machines do not).
func TestFinalize_RealOpenCC(t *testing.T) {
	bin := os.Getenv("VIDO_OPENCC_BIN")
	if bin == "" {
		bin = "opencc"
	}
	if _, err := exec.LookPath(bin); err != nil {
		t.Skip("opencc not installed")
	}
	conv := realOpenCC{bin: bin}
	for in, want := range map[string]string{
		"是大家都知道还是只有她知道": "是大家都知道還是只有她知道",
		"是大家都知道還是只有她知道": "是大家都知道還是只有她知道",
		"我最多只能派两辆":      "我最多只能派兩輛",
		"不是只有你一个母亲":     "不是只有你一個母親",
		"那是隻會說話的鳥":      "那是隻會說話的鳥",
		"一只有翅膀的鸟":       "一隻有翅膀的鳥",
	} {
		out, err := Finalize(conv, in, nil)
		require.NoError(t, err)
		assert.Equal(t, want, out, "input %q", in)
	}
}

type realOpenCC struct{ bin string }

func (r realOpenCC) run(profile string, b []byte) ([]byte, error) {
	cmd := exec.Command(r.bin, "-c", profile)
	cmd.Stdin = strings.NewReader(string(b))
	return cmd.Output()
}

func (r realOpenCC) ConvertS2TWP(b []byte) ([]byte, error) { return r.run("s2twp", b) }
func (r realOpenCC) ConvertS2TW(b []byte) ([]byte, error)  { return r.run("s2tw", b) }
