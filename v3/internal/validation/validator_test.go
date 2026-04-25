package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type v3TestAddress struct {
	Type   string `json:"type" validate:"required"`
	Street string `json:"street" validate:"required,max=10"`
	City   string `json:"city" validate:"required,max=20"`
}

type v3TestPerson struct {
	Name      string          `json:"name" validate:"required,max=5"`
	Addresses []v3TestAddress `json:"addresses" validate:"required,dive"`
}

func v3PathsOf(items []InvalidField) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Path)
	}
	return out
}

func TestV3_MapsValidatorErrorsToInvalidField(t *testing.T) {
	v := New()
	v.RegisterTagScope("required", ScopeMissing)
	v.RegisterTagScope("max", ScopeMaxLength)

	person := v3TestPerson{
		Name: "Alexander",
		Addresses: []v3TestAddress{
			{
				Type:   "",
				Street: "",
				City:   "VeryVeryVeryVeryVeryVeryLongCityName",
			},
		},
	}

	got := v.Validate(person, "person")

	require.Contains(t, v3PathsOf(got), "person.name")
	require.Contains(t, v3PathsOf(got), "person.addresses[0].type")
	require.Contains(t, v3PathsOf(got), "person.addresses[0].street")
	require.Contains(t, v3PathsOf(got), "person.addresses[0].city")
}
