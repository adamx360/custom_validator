package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type RuleResult struct {
	Code        string
	Description string
	Path        string
}

type RuleFunc func(fieldPath string, value reflect.Value, param string) []RuleResult

type registeredRule struct {
	name  string
	scope string
	fn    RuleFunc
}

func requiredRule(fieldPath string, v reflect.Value, _ string) []RuleResult {
	if !v.IsValid() {
		return []RuleResult{{
			Path:        fieldPath,
			Code:        CodeMissing,
			Description: DescriptionMissing,
		}}
	}

	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return []RuleResult{{
				Path:        fieldPath,
				Code:        CodeMissing,
				Description: DescriptionMissing,
			}}
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		if strings.TrimSpace(v.String()) == "" {
			return []RuleResult{{
				Path:        fieldPath,
				Code:        CodeMissing,
				Description: DescriptionMissing,
			}}
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		if v.Len() == 0 {
			return []RuleResult{{
				Path:        fieldPath,
				Code:        CodeMissing,
				Description: DescriptionMissing,
			}}
		}
	default:
		if v.IsZero() {
			return []RuleResult{{
				Path:        fieldPath,
				Code:        CodeMissing,
				Description: DescriptionMissing,
			}}
		}
	}

	return nil
}

func maxLengthRule(fieldPath string, v reflect.Value, param string) []RuleResult {
	limit, err := strconv.Atoi(param)
	if err != nil {
		return nil
	}

	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		if utf8.RuneCountInString(v.String()) > limit {
			return []RuleResult{{
				Path:        fieldPath,
				Code:        CodeTooLong,
				Description: DescriptionTooLong,
			}}
		}
	case reflect.Slice, reflect.Array:
		if v.Len() > limit {
			return []RuleResult{{
				Path:        fieldPath,
				Code:        CodeTooLong,
				Description: DescriptionTooLong,
			}}
		}
	}

	return nil
}

func requireContainsRule(fieldPath string, v reflect.Value, param string) []RuleResult {
	spec, ok := parseRequireContainsParam(param)
	if !ok {
		return nil
	}

	v, ok = unwrapRuleValue(v)
	if !ok {
		return nil
	}

	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}

	present := map[string]bool{}

	for i := 0; i < v.Len(); i++ {
		item, ok := unwrapRuleValue(v.Index(i))
		if !ok || item.Kind() != reflect.Struct {
			continue
		}

		fieldValue, ok := findFieldValueByJSONName(item, spec.Field)
		if !ok {
			continue
		}

		present[valueToString(fieldValue)] = true
	}

	results := make([]RuleResult, 0, len(spec.Values))
	for _, expected := range spec.Values {
		if present[expected] {
			continue
		}

		results = append(results, RuleResult{
			Path:        buildEqualityPath(fieldPath, spec.Field, expected),
			Code:        CodeMissing,
			Description: DescriptionMissing,
		})
	}

	return results
}

func unwrapRuleValue(v reflect.Value) (reflect.Value, bool) {
	if !v.IsValid() {
		return reflect.Value{}, false
	}

	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}

	return v, true
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
	unwrapped, ok := unwrapRuleValue(v)
	if !ok {
		return ""
	}

	if unwrapped.Kind() == reflect.String {
		return unwrapped.String()
	}

	return fmt.Sprint(unwrapped.Interface())
}
