package validation

import (
	"errors"
	"reflect"
	"sort"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

var tagToScope = map[string]string{
	"required": ScopeMissing,
	"max":      ScopeMaxLength,
}

func Validate(x any, root string, scopes []string) []InvalidField {
	validate := playground.New()

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return jsonFieldName(fld)
	})

	err := validate.Struct(x)
	if err == nil {
		return nil
	}

	var validationErrs playground.ValidationErrors
	ok := errors.As(err, &validationErrs)
	if !ok {
		return nil
	}

	scopeSet := buildScopeSet(scopes)
	fields := map[string]map[string]*Reason{}

	for _, fe := range validationErrs {
		scope, ok := tagToScope[strings.ToLower(fe.Tag())]
		if !ok {
			continue
		}

		if !scopeSet[scope] {
			continue
		}

		path := fieldErrorPath(root, fe)
		code, description := mapTagToReason(fe.Tag())
		if code == "" {
			continue
		}

		addReason(fields, path, code, description, scope)
	}

	return flatten(fields)
}

func buildScopeSet(scopes []string) map[string]bool {
	out := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		out[s] = true
	}
	return out
}

func fieldErrorPath(root string, fe playground.FieldError) string {
	ns := fe.Namespace()
	if ns == "" {
		return root
	}

	parts := strings.Split(ns, ".")
	if len(parts) > 0 {
		parts = parts[1:]
	}

	path := strings.Join(parts, ".")
	if root == "" {
		return path
	}
	if path == "" {
		return root
	}
	return root + "." + path
}

func mapTagToReason(tag string) (code, description string) {
	switch strings.ToLower(tag) {
	case "required":
		return CodeMissing, DescriptionMissing
	case "max":
		return CodeTooLong, DescriptionTooLong
	default:
		return strings.ToUpper(tag), "Validation failed"
	}
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
