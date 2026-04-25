package validation

import (
	"reflect"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

type Validator struct {
	engine         *playground.Validate
	sliceSelectors map[string]SliceSelector
}

func New(structTypesForRequireContains ...any) *Validator {
	engine := playground.New()

	engine.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return jsonFieldName(fld)
	})

	v := &Validator{
		engine:         engine,
		sliceSelectors: map[string]SliceSelector{},
	}

	for _, sample := range structTypesForRequireContains {
		v.registerRequireContainsStructValidation(sample)
	}

	return v
}

func (v *Validator) Engine() *playground.Validate {
	return v.engine
}

func (v *Validator) ValidateByScopes(x any, root string, scopes []string) []InvalidField {
	err := v.engine.Struct(x)
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(playground.ValidationErrors)
	if !ok {
		return nil
	}

	scopeSet := buildScopeSet(scopes)
	fields := map[string]map[string]*Reason{}
	root = normalizePath(root)

	for _, fe := range validationErrs {
		scope, code, description, path, include := mapFieldError(root, fe, scopeSet)
		if !include {
			continue
		}

		path = v.rewriteIndexedPathToFilteredPath(x, path)
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

func mapFieldError(root string, fe playground.FieldError, scopeSet map[string]bool) (scope, code, description, path string, include bool) {
	switch fe.Tag() {
	case "required":
		return ScopeMissing, CodeMissing, DescriptionMissing, fieldErrorPath(root, fe), scopeSet[ScopeMissing]

	case "max":
		return ScopeMaxLength, CodeTooLong, DescriptionTooLong, fieldErrorPath(root, fe), scopeSet[ScopeMaxLength]

	case "require_contains":
		return ScopeMissing, CodeMissing, DescriptionMissing, buildRequireContainsPath(root, fe), scopeSet[ScopeMissing]

	default:
		return "", "", "", "", false
	}
}

func fieldErrorPath(root string, fe playground.FieldError) string {
	ns := fe.Namespace()
	if ns == "" {
		return normalizePath(root)
	}

	parts := strings.Split(ns, ".")
	if len(parts) > 0 {
		parts = parts[1:]
	}

	jsonPath := strings.Join(parts, ".")
	if root != "" {
		if jsonPath == "" {
			jsonPath = root
		} else {
			jsonPath = root + "." + jsonPath
		}
	}

	return normalizePath(jsonPath)
}

func buildRequireContainsPath(root string, fe playground.FieldError) string {
	base := fieldErrorPath(root, fe)

	spec, ok := parseRequireContainsParam(fe.Param())
	if !ok || len(spec.Values) == 0 {
		return base
	}

	expected := spec.Values[0]
	if s, ok := fe.Value().(string); ok && s != "" {
		expected = s
	}

	return buildEqualityPath(base, spec.Field, expected)
}
