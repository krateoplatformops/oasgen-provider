package generator

import (
	"testing"

	"github.com/gobuffalo/flect"
)

func TestPluralize(t *testing.T) {
	t.Logf(flect.Pluralize("Dummy-Chart"))
	t.Logf(flect.Pascalize("Dummy-Chart"))
}
