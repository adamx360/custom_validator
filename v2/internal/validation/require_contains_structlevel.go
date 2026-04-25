package validation

import (
	"reflect"

	playground "github.com/go-playground/validator/v10"
)

func (v *Validator) registerRequireContainsStructValidation(sample any) {
	if sample == nil {
		return
	}

	t := reflect.TypeOf(sample)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	v.engine.RegisterStructValidation(func(sl playground.StructLevel) {
		current := sl.Current()
		for current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return
			}
			current = current.Elem()
		}
		if !current.IsValid() || current.Kind() != reflect.Struct {
			return
		}

		validateRequireContainsTags(sl, current)
	}, sample)
}

func validateRequireContainsTags(sl playground.StructLevel, current reflect.Value) {
	t := current.Type()

	for i := 0; i < current.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		param := sf.Tag.Get("require_contains")
		if param == "" {
			continue
		}

		spec, ok := parseRequireContainsParam(param)
		if !ok {
			continue
		}

		fieldValue := unwrapPointers(current.Field(i))
		if !fieldValue.IsValid() {
			for _, expected := range spec.Values {
				sl.ReportError(expected, jsonFieldName(sf), jsonFieldName(sf), "require_contains", param)
			}
			continue
		}

		if fieldValue.Kind() != reflect.Slice && fieldValue.Kind() != reflect.Array {
			continue
		}

		present := map[string]bool{}

		for j := 0; j < fieldValue.Len(); j++ {
			item := unwrapPointers(fieldValue.Index(j))
			if !item.IsValid() || item.Kind() != reflect.Struct {
				continue
			}

			valueField, ok := findFieldValueByJSONName(item, spec.Field)
			if !ok {
				continue
			}

			present[valueToString(valueField)] = true
		}

		for _, expected := range spec.Values {
			if present[expected] {
				continue
			}

			sl.ReportError(expected, jsonFieldName(sf), jsonFieldName(sf), "require_contains", param)
		}
	}
}
