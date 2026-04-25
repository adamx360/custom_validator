package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type v1TestAddress struct {
	Type   string `json:"type" validate:"required"`
	Street string `json:"street" validate:"required,max=10"`
	City   string `json:"city" validate:"required,max=20"`
}

type v1TestPerson struct {
	Name      string          `json:"name" validate:"required,max=5"`
	Addresses []v1TestAddress `json:"addresses" validate:"required,require_contains=type:REGISTERED&MAILING"`
}

func v1PathsOf(items []InvalidField) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Path)
	}
	return out
}

func TestV1_UsesFilteredPathForSliceItems(t *testing.T) {
	v := New()
	v.RegisterSliceSelector("person.addresses", "type")

	person := v1TestPerson{
		Name: "Adam",
		Addresses: []v1TestAddress{
			{
				Type:   "MAILING",
				Street: "",
				City:   "Wroclaw",
			},
		},
	}

	got := v.ValidateByScopes(person, "person", []string{
		ScopeMissing,
	})

	require.ElementsMatch(t, []string{
		"person.addresses[?@.type==\"MAILING\"].street",
		"person.addresses[?@.type==\"REGISTERED\"]",
	}, v1PathsOf(got))
}
