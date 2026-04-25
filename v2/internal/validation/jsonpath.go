package validation

import "strings"

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	return path
}

func escapePathString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func buildEqualityPath(fieldPath, jsonField, expected string) string {
	base := normalizePath(fieldPath)
	if base == "" {
		return `[?@.` + jsonField + `=="` + escapePathString(expected) + `"]`
	}
	return base + `[?@.` + jsonField + `=="` + escapePathString(expected) + `"]`
}
