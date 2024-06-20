package templates

import (
	"fmt"
	"strings"
	ttemplate "text/template"
)

// TxtFuncMap returns a 'text/template'.FuncMap
func TxtFuncMap() ttemplate.FuncMap {
	return ttemplate.FuncMap(map[string]any{
		"quote": quote,
	})
}

func quote(str ...any) string {
	out := make([]string, 0, len(str))
	for _, s := range str {
		if s != nil {
			out = append(out, fmt.Sprintf("%q", strval(s)))
		}
	}
	return strings.Join(out, " ")
}

func strval(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
