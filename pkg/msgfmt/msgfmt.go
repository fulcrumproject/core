// Package msgfmt renders {key} message templates against a data map. It is a
// leaf package so both domain and schema can share one renderer without an
// import cycle.
package msgfmt

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var placeholder = regexp.MustCompile(`\{(\w+)\}`)

// RenderMessage substitutes {key} placeholders in template with data values.
// Unknown keys are left untouched so malformed templates stay visible. Slice
// values render as a comma-separated list rather than Go's bracketed syntax.
func RenderMessage(template string, data map[string]any) string {
	if len(data) == 0 {
		return template
	}
	return placeholder.ReplaceAllStringFunc(template, func(match string) string {
		key := match[1 : len(match)-1]
		v, ok := data[key]
		if !ok {
			return match
		}
		return format(v)
	})
}

func format(v any) string {
	// Types that know how to print themselves (e.g. uuid.UUID, a [16]byte array)
	// must not be split into their elements.
	switch v.(type) {
	case fmt.Stringer, error:
		return fmt.Sprint(v)
	}
	if rv := reflect.ValueOf(v); rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = fmt.Sprint(rv.Index(i).Interface())
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprint(v)
}
