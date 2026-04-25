package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var indexedSegmentRegexp = regexp.MustCompile(`^([a-zA-Z0-9_-]+)\[(\d+)\]$`)

func (v *Validator) rewriteIndexedPathToFilteredPath(rootValue any, path string) string {
	path = normalizePath(path)
	if path == "" {
		return path
	}

	segments := strings.Split(path, ".")
	if len(segments) == 0 {
		return path
	}

	current := unwrapPointers(reflect.ValueOf(rootValue))
	if !current.IsValid() {
		return path
	}

	var resolved []string

	for i, seg := range segments {
		name, index, hasIndex := parseIndexedSegment(seg)

		if i == 0 {
			resolved = append(resolved, seg)
			continue
		}

		if !hasIndex {
			if current.IsValid() {
				current = descendByJSONName(current, name)
			}
			resolved = append(resolved, seg)
			continue
		}

		collectionPath := strings.Join(append(resolved, name), ".")
		selector, ok := v.sliceSelectors[normalizePath(collectionPath)]
		if !ok {
			resolved = append(resolved, seg)
			if current.IsValid() {
				current = descendByJSONName(current, name)
				current = sliceIndex(current, index)
			}
			continue
		}

		if current.IsValid() {
			sliceValue := descendByJSONName(current, name)
			item := sliceIndex(sliceValue, index)
			filteredSeg := buildFilteredSegment(name, item, selector.KeyField, index)
			resolved = append(resolved, filteredSeg)
			current = item
			continue
		}

		resolved = append(resolved, seg)
	}

	return strings.Join(resolved, ".")
}

func parseIndexedSegment(seg string) (name string, index int, ok bool) {
	m := indexedSegmentRegexp.FindStringSubmatch(seg)
	if len(m) != 3 {
		return "", 0, false
	}

	i, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, false
	}

	return m[1], i, true
}

func descendByJSONName(v reflect.Value, jsonName string) reflect.Value {
	v = unwrapPointers(v)
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if jsonFieldName(sf) != jsonName {
			continue
		}
		return v.Field(i)
	}

	return reflect.Value{}
}

func sliceIndex(v reflect.Value, index int) reflect.Value {
	v = unwrapPointers(v)
	if !v.IsValid() {
		return reflect.Value{}
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return reflect.Value{}
	}
	if index < 0 || index >= v.Len() {
		return reflect.Value{}
	}
	return v.Index(index)
}

func buildFilteredSegment(name string, item reflect.Value, keyField string, index int) string {
	item = unwrapPointers(item)
	if !item.IsValid() || item.Kind() != reflect.Struct {
		return fmt.Sprintf("%s[%d]", name, index)
	}

	keyValue, ok := findFieldValueByJSONName(item, keyField)
	if !ok {
		return fmt.Sprintf("%s[%d]", name, index)
	}

	key := valueToString(keyValue)
	if strings.TrimSpace(key) == "" {
		return fmt.Sprintf("%s[%d]", name, index)
	}

	return name + `[?@.` + keyField + `=="` + escapePathString(key) + `"]`
}
