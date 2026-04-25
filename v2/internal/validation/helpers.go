package validation

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

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

func unwrapPointers(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func findFieldValueByJSONName(item reflect.Value, jsonName string) (reflect.Value, bool) {
	t := item.Type()

	for i := 0; i < item.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if jsonFieldName(sf) != jsonName {
			continue
		}
		return item.Field(i), true
	}

	return reflect.Value{}, false
}

func valueToString(v reflect.Value) string {
	v = unwrapPointers(v)
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	return fmt.Sprint(v.Interface())
}

func addReason(fields map[string]map[string]*Reason, path, code, description, scope string) {
	if _, ok := fields[path]; !ok {
		fields[path] = map[string]*Reason{}
	}

	if existing, ok := fields[path][code]; ok {
		existing.Scopes = appendUniqueScope(existing.Scopes, scope)
		sort.Strings(existing.Scopes)
		return
	}

	fields[path][code] = &Reason{
		Scopes:      []string{scope},
		Code:        code,
		Description: description,
	}
}

func appendUniqueScope(scopes []string, scope string) []string {
	for _, s := range scopes {
		if s == scope {
			return scopes
		}
	}
	return append(scopes, scope)
}

func flatten(fields map[string]map[string]*Reason) []InvalidField {
	paths := make([]string, 0, len(fields))
	for path := range fields {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	out := make([]InvalidField, 0, len(paths))
	for _, path := range paths {
		reasonMap := fields[path]
		codes := make([]string, 0, len(reasonMap))
		for code := range reasonMap {
			codes = append(codes, code)
		}
		sort.Strings(codes)

		reasons := make([]Reason, 0, len(codes))
		for _, code := range codes {
			reasons = append(reasons, *reasonMap[code])
		}

		out = append(out, InvalidField{
			Path:    path,
			Reasons: reasons,
		})
	}

	return out
}
