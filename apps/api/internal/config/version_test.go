package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_DefaultsToDevWhenUnset(t *testing.T) {
	t.Setenv("VIDO_VERSION", "")
	assert.Equal(t, "dev", Version())

	t.Setenv("VIDO_VERSION", "  \n")
	assert.Equal(t, "dev", Version(), "whitespace from a blank build arg is not a version")
}

func TestVersion_ReturnsBuildValue(t *testing.T) {
	t.Setenv("VIDO_VERSION", " 0.1.2 ")
	assert.Equal(t, "0.1.2", Version())

	t.Setenv("VIDO_VERSION", "main-81a4e5c")
	assert.Equal(t, "main-81a4e5c", Version())
}
