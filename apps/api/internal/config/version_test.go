package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_DefaultsToDevWhenNotInjected(t *testing.T) {
	orig := buildVersion
	t.Cleanup(func() { buildVersion = orig })

	buildVersion = ""
	assert.Equal(t, "dev", Version())

	buildVersion = "  \n"
	assert.Equal(t, "dev", Version(), "whitespace from a blank build arg is not a version")
}

func TestVersion_ReturnsInjectedValue(t *testing.T) {
	orig := buildVersion
	t.Cleanup(func() { buildVersion = orig })

	buildVersion = " 0.1.2 "
	assert.Equal(t, "0.1.2", Version())
}
