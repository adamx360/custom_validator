package validation

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type Validator struct {
	rules          map[string]registeredRule
	sliceSelectors map[string]SliceSelector
}

func New() *Validator {
	v := &Validator{
		rules:          map[string]registeredRule{},
		sliceSelectors: map[string]SliceSelector{},
	}

	v.Register("required", ScopeMissing, requiredRule)
	v.Register("max", ScopeMaxLength, maxLengthRule)
	v.Register("require_contains", ScopeMissing, requireContainsRule)

	return v
}

func (v *Validator) Register(name, scope string, fn RuleFunc) {
	v.rules[name] = registeredRule{
		name:  name,
		scope: scope,
		fn:    fn,
	}
}

func (v *Validator) ValidateByScopes(x any, root string, scopes []string) []InvalidField {
	scopeSet := buildScopeSet(scopes)
	fields := map[string]map[string]*Reason{}
	visited := map[uintptr]bool{}

	v.walkValue(reflect.ValueOf(x), normalizePath(root), scopeSet, fields, visited)

	return flatten(fields)
}

func buildScopeSet(scopes []string) map[string]bool {
	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[strings.ToUpper(strings.TrimSpace(s))] = true
	}
	return scopeSet
}

func (v *Validator) walkValue(
	val reflect.Value,
	path string,
	scopeSet map[string]bool,
	fields map[string]map[string]*Reason,
	visited map[uintptr]bool,
) {
	unwrapped, ok := unwrapValue(val, visited)
	if !ok {
		return
	}

	switch unwrapped.Kind() {
	case reflect.Struct:
		v.walkStruct(unwrapped, path, scopeSet, fields, visited)
	case reflect.Slice, reflect.Array:
		v.walkSlice(unwrapped, path, scopeSet, fields, visited)
	}
}

func unwrapValue(val reflect.Value, visited map[uintptr]bool) (reflect.Value, bool) {
	if !val.IsValid() {
		return reflect.Value{}, false
	}

	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return reflect.Value{}, false
		}

		if val.Kind() == reflect.Ptr {
			ptr := val.Pointer()
			if ptr != 0 {
				if visited[ptr] {
					return reflect.Value{}, false
				}
				visited[ptr] = true
			}
		}

		val = val.Elem()
	}

	return val, true
}

func (v *Validator) walkStruct(
	val reflect.Value,
	path string,
	scopeSet map[string]bool,
	fields map[string]map[string]*Reason,
	visited map[uintptr]bool,
) {
	t := val.Type()

	for i := 0; i < val.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		jsonName := jsonFieldName(sf)
		if jsonName == "-" {
			continue
		}

		fieldPath := joinPath(path, jsonName)
		fv := val.Field(i)

		v.applyFieldRules(sf, fv, fieldPath, scopeSet, fields)
		v.walkChild(fv, fieldPath, scopeSet, fields, visited)
	}
}

func (v *Validator) applyFieldRules(
	sf reflect.StructField,
	fv reflect.Value,
	fieldPath string,
	scopeSet map[string]bool,
	fields map[string]map[string]*Reason,
) {
	rules := parseRules(sf.Tag.Get("validate"))

	for _, rule := range rules {
		reg, ok := v.rules[rule.Name]
		if !ok {
			continue
		}
		if !scopeSet[strings.ToUpper(reg.scope)] {
			continue
		}

		results := reg.fn(fieldPath, fv, rule.Param)
		for _, result := range results {
			path := result.Path
			if path == "" {
				path = fieldPath
			}
			addReason(fields, path, result.Code, result.Description, reg.scope)
		}
	}
}

func (v *Validator) walkChild(
	fv reflect.Value,
	fieldPath string,
	scopeSet map[string]bool,
	fields map[string]map[string]*Reason,
	visited map[uintptr]bool,
) {
	switch indirectKind(fv) {
	case reflect.Struct, reflect.Slice, reflect.Array:
		v.walkValue(fv, fieldPath, scopeSet, fields, visited)
	}
}

func (v *Validator) walkSlice(
	val reflect.Value,
	path string,
	scopeSet map[string]bool,
	fields map[string]map[string]*Reason,
	visited map[uintptr]bool,
) {
	val = unwrapPointers(val)
	if !val.IsValid() {
		return
	}

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		itemPath := v.buildSliceItemPath(path, item, i)

		switch indirectKind(item) {
		case reflect.Struct, reflect.Slice, reflect.Array:
			v.walkValue(item, itemPath, scopeSet, fields, visited)
		}
	}
}

func (v *Validator) buildSliceItemPath(collectionPath string, item reflect.Value, index int) string {
	collectionPath = normalizePath(collectionPath)

	selector, ok := v.sliceSelectors[collectionPath]
	if !ok {
		return fmt.Sprintf("%s[%d]", collectionPath, index)
	}

	item = unwrapPointers(item)
	if !item.IsValid() || item.Kind() != reflect.Struct {
		return fmt.Sprintf("%s[%d]", collectionPath, index)
	}

	keyValue, ok := findFieldValueByJSONName(item, selector.KeyField)
	if !ok {
		return fmt.Sprintf("%s[%d]", collectionPath, index)
	}

	key := valueToString(keyValue)
	if strings.TrimSpace(key) == "" {
		return fmt.Sprintf("%s[%d]", collectionPath, index)
	}

	return buildEqualityPath(collectionPath, selector.KeyField, key)
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

func indirectKind(v reflect.Value) reflect.Kind {
	for v.IsValid() {
		switch v.Kind() {
		case reflect.Ptr, reflect.Interface:
			if v.IsNil() {
				return v.Kind()
			}
			v = v.Elem()
		default:
			return v.Kind()
		}
	}
	return reflect.Invalid
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
