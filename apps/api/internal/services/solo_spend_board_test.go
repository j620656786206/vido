package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

func TestSoloSpendBoard_TrackLookupUntrack(t *testing.T) {
	board := NewSoloSpendBoard()
	b := ai.NewBudget(5)

	assert.Nil(t, board.Lookup("m1"), "nothing tracked yet")
	board.Track("m1", b)
	assert.Same(t, b, board.Lookup("m1"))
	assert.Nil(t, board.Lookup("m2"))

	board.Untrack("m1", b)
	assert.Nil(t, board.Lookup("m1"))
}

func TestSoloSpendBoard_UntrackKeepsANewerJob(t *testing.T) {
	board := NewSoloSpendBoard()
	old, newer := ai.NewBudget(5), ai.NewBudget(5)
	board.Track("m1", old)
	board.Track("m1", newer) // the next click on the same item

	board.Untrack("m1", old) // the first job ends late
	assert.Same(t, newer, board.Lookup("m1"), "a late Untrack must not erase the newer job")
}

func TestSoloSpendBoard_NilSafe(t *testing.T) {
	var board *SoloSpendBoard
	board.Track("m1", ai.NewBudget(5))
	board.Untrack("m1", nil)
	assert.Nil(t, board.Lookup("m1"))

	empty := NewSoloSpendBoard()
	empty.Track("", ai.NewBudget(5))
	empty.Track("m1", nil)
	assert.Nil(t, empty.Lookup(""))
	assert.Nil(t, empty.Lookup("m1"))
}

func TestAddRunCost(t *testing.T) {
	t.Run("spent always, ceiling only when set", func(t *testing.T) {
		b := ai.NewBudget(5)
		b.RecordASR(60) // one hosted minute
		data := map[string]interface{}{}
		AddRunCost(data, b)
		require.Contains(t, data, "spent_usd")
		assert.Equal(t, b.SpentUSD(), data["spent_usd"])
		assert.Greater(t, data["spent_usd"].(float64), 0.0)
		assert.Equal(t, 5.0, data["budget_usd"])
	})
	t.Run("nothing spent yet is 0, not absent", func(t *testing.T) {
		data := map[string]interface{}{}
		AddRunCost(data, ai.NewBudget(5))
		assert.Equal(t, 0.0, data["spent_usd"])
	})
	t.Run("no ceiling: no budget_usd", func(t *testing.T) {
		data := map[string]interface{}{}
		AddRunCost(data, ai.NewBudget(0))
		assert.Contains(t, data, "spent_usd")
		assert.NotContains(t, data, "budget_usd")
	})
	t.Run("nil budget writes nothing", func(t *testing.T) {
		data := map[string]interface{}{}
		AddRunCost(data, nil)
		assert.Empty(t, data)
		AddRunCost(nil, ai.NewBudget(5)) // must not panic
	})
}
