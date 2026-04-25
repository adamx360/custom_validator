package validation

import (
	"reflect"
	"strings"
)

type tagRule struct {
	Name  string
	Param string
}

func parseRules(tag string) []tagRule {
	if tag == "" || tag == "-" {
		return nil
	}

	parts := strings.Split(tag, ",")
	out := make([]tagRule, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		chunks := strings.SplitN(p, "=", 2)
		rule := tagRule{Name: strings.TrimSpace(chunks[0])}
		if len(chunks) == 2 {
			rule.Param = strings.TrimSpace(chunks[1])
		}

		out = append(out, rule)
	}

	return out
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return lowerFirst(f.Name)
	}

	name := strings.Split(tag, ",")[0]
	if name == "" {
		return lowerFirst(f.Name)
	}

	return name
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}
