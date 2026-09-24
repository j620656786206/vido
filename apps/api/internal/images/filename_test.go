package images

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPosterFileName(t *testing.T) {
	for _, ok := range []string{"0fe13b88-a374-4039-9e8a-f80556eb7c78.jpg", "m1-thumb.jpg", "A_b-9.jpg"} {
		assert.True(t, IsPosterFileName(ok), ok)
	}
	for _, bad := range []string{"", ".jpg", "a.b.jpg", "../x.jpg", "a/b.jpg", "x.png", "x.jpg.bak", "notes.txt", "x.JPG"} {
		assert.False(t, IsPosterFileName(bad), bad)
	}
}
