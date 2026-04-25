package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type v2TestAddress struct {
	Type   string `json:"type" validate:"required"`
	Street string `json:"street" validate:"required,max=10"`
	City   string `json:"city" validate:"required,max=20"`
}

type v2TestPerson struct {
	Name      string          `json:"name" validate:"required,max=5"`
	Addresses []v2TestAddress `json:"addresses" validate:"required,dive" require_contains:"type:REGISTERED&MAILING"`
}

func v2PathsOf(items []InvalidField) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Path)
	}
	return out
}

func TestV2_RewritesIndexedPathToFilteredPath(t *testing.T) {
	v := New(v2TestPerson{})
	v.RegisterSliceSelector("person.addresses", "type")

	person := v2TestPerson{
		Name: "Adam",
		Addresses: []v2TestAddress{
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
	}, v2PathsOf(got))
}
