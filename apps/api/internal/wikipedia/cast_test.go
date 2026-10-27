package wikipedia

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shape of zh.wikipedia's 《百花殺》 演員陣容 table (2026-10-27), with the
// descriptions shortened.
const blossomsCastWikitext = `== 劇情簡介 ==
講述信王妃顧青梔以昭寧郡主沈汐和身份重生後……

==演員陣容==
{|class="wikitable" style="text-align:center"
|-
!width="20%"|演員
!width="20%"|角色
!介紹
|-
|[[何與]]
|蕭華雍 ||style="text-align: left;"|七皇子，冊封東宮太子。
長大後回到皇都，長年以病弱形象示人。
|-
|[[孟子義]]
|沈汐和 ||style="text-align: left;"|昭寧郡主。
|-
|[[曹衛宇]]
|祐寧帝(蕭覺崇) ||style="text-align: left;"|大興王朝皇帝。
|-
|陸思宇
|蕭覺崇 ||style="text-align: left;"|祐寧帝青年時期。
|-
|[[葉祖新|葉祖新]]<ref>{{cite news |title=x}}</ref>
|'''蕭長旻''' ||style="text-align: left;"|二皇子寧王。
|-
|rowspan="2"|[[某演員]]
|角色甲 ||說明
|-
|角色乙 ||說明
|}

== 播出時間 ==
{| class="wikitable"
|-
! 播放平台 || 所在地
|-
| [[腾讯视频]] || 中國大陸
|}
`

func TestParseCast_ReadsTheActorAndCharacterColumns(t *testing.T) {
	got := ParseCast(blossomsCastWikitext)
	assert.Equal(t, []CastEntry{
		{Actor: "何與", Character: "蕭華雍"},
		{Actor: "孟子義", Character: "沈汐和"},
		{Actor: "曹衛宇", Character: "祐寧帝"},
		{Actor: "陸思宇", Character: "蕭覺崇"},
		{Actor: "葉祖新", Character: "蕭長旻"},
		{Actor: "某演員", Character: "角色甲"},
		{Actor: "某演員", Character: "角色乙"},
	}, got, "a rowspan actor plays both roles; the 播出時間 table has no 演員 column")
}

// 《將夜》(2018) writes its header as a plain row of bold cells.
func TestParseCast_PlainHeaderRow(t *testing.T) {
	got := ParseCast(`=== 主要演员 ===
{|class="wikitable" width="75%"
|- style="background:green; color:snow" align=center
|width="12%"|'''演員''' || width="12%"|'''角色''' ||'''介紹'''
|-
|[[陈飞宇]]||宁缺||align=left|邊軍小卒寧缺
|-
|[[宋伊人]]||桑桑||align=left|寧缺的侍女
|}`)
	assert.Equal(t, []CastEntry{{Actor: "陈飞宇", Character: "宁缺"}, {Actor: "宋伊人", Character: "桑桑"}}, got)
}

func TestParseCast_ListForm(t *testing.T) {
	got := ParseCast("== 演員 ==\n* [[趙麗穎]] 飾 楚喬\n* 林更新 饰演 宇文玥：信陽侯世子\n* 某段落文字，不是演員表。\n")
	assert.Equal(t, []CastEntry{{Actor: "趙麗穎", Character: "楚喬"}, {Actor: "林更新", Character: "宇文玥"}}, got)
}

func TestCleanCastCell_VariantMarkup(t *testing.T) {
	assert.Equal(t, "余五娘子", cleanCastCell("余-{}-五娘子"))
	assert.Equal(t, "范閒", cleanCastCell("-{范閒}-"))
}

func TestParseCast_NoCastTable(t *testing.T) {
	assert.Empty(t, ParseCast("'''百花殺'''是一部電視劇。\n{| class=\"wikitable\"\n! 集數 !! 收視\n|-\n| 1 || 2.1\n|}"))
}

func TestGetWikitext_FollowsVariantTitlesAndSkipsMissingPages(t *testing.T) {
	var titles []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		titles = append(titles, q.Get("titles"))
		assert.Equal(t, "1", q.Get("redirects"))
		assert.Equal(t, "1", q.Get("converttitles"))
		w.Header().Set("Content-Type", "application/json")
		if q.Get("titles") == "百花杀" {
			_, _ = w.Write([]byte(`{"query":{"pages":[{"title":"百花殺","revisions":[{"slots":{"main":{"content":"==演員陣容=="}}}]}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"query":{"pages":[{"title":"不存在","missing":true}]}}`))
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{Enabled: true, RequestsPerSecond: 100}, nil)
	c.baseURL = srv.URL

	text, title, err := c.GetWikitext(context.Background(), "不存在", "百花杀")
	require.NoError(t, err)
	assert.Equal(t, "==演員陣容==", text)
	assert.Equal(t, "百花殺", title)
	assert.Equal(t, []string{"不存在", "百花杀"}, titles)

	_, _, err = c.GetWikitext(context.Background(), "不存在")
	assert.True(t, IsNotFound(err))
}
