package validation

import "strings"

type requireContainsSpec struct {
	Field  string
	Values []string
}

func parseRequireContainsParam(param string) (requireContainsSpec, bool) {
	param = strings.TrimSpace(param)
	if param == "" {
		return requireContainsSpec{}, false
	}

	parts := strings.SplitN(param, ":", 2)
	if len(parts) != 2 {
		return requireContainsSpec{}, false
	}

	field := strings.TrimSpace(parts[0])
	rawValues := strings.TrimSpace(parts[1])

	if field == "" || rawValues == "" {
		return requireContainsSpec{}, false
	}

	if strings.Contains(rawValues, "|") {
		return requireContainsSpec{}, false
	}

	values := splitAndTrim(rawValues, "&")
	if len(values) == 0 {
		return requireContainsSpec{}, false
	}

	return requireContainsSpec{
		Field:  field,
		Values: values,
	}, true
}

func splitAndTrim(s, sep string) []string {
	raw := strings.Split(s, sep)
	out := make([]string, 0, len(raw))

	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}

	return out
}
