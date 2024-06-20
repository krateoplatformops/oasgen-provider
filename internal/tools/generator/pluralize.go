package generator

import (
	"strings"

	"github.com/gobuffalo/flect"
)

func Plural(kind string) string {
	return strings.ToLower(flect.Pluralize(kind))
}
