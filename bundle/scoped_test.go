package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppliesTo(t *testing.T) {
	makeTestTypes := func(applyTo []string) []Scoped {
		return []Scoped{
			&Credential{ApplyTo: applyTo},
			Output{ApplyTo: applyTo},
			&Parameter{ApplyTo: applyTo},
		}
	}

	t.Run("empty", func(t *testing.T) {
		testTypes := makeTestTypes(nil)

		for _, tt := range testTypes {
			assert.True(t, AppliesTo(tt, "install"), "%T.AppliesTo returned an incorrect result", tt)
		}
	})

	t.Run("hit", func(t *testing.T) {
		testTypes := makeTestTypes([]string{"install", "upgrade", "custom"})

		for _, tt := range testTypes {
			assert.True(t, AppliesTo(tt, "custom"), "%T.AppliesTo returned an incorrect result", tt)
		}
	})

	t.Run("miss", func(t *testing.T) {
		testTypes := makeTestTypes([]string{"install", "upgrade", "uninstall"})

		for _, tt := range testTypes {
			assert.False(t, AppliesTo(tt, "custom"), "%T.AppliesTo returned an incorrect result", tt)
		}
	})
}
