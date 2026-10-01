package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// sub-7-5b: AppendOnScanComplete keeps whatever was set and runs it first.
func TestScannerService_AppendOnScanComplete(t *testing.T) {
	s := &ScannerService{}
	var order []string
	s.AppendOnScanComplete(func() { order = append(order, "first") })
	s.AppendOnScanComplete(func() { order = append(order, "second") })
	s.AppendOnScanComplete(nil)
	s.onScanComplete()
	assert.Equal(t, []string{"first", "second"}, order)

	s2 := &ScannerService{}
	s2.AppendOnScanComplete(nil)
	assert.Nil(t, s2.onScanComplete, "appending nil to nothing leaves nothing")
}
