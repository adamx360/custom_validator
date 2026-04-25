package main

import (
	"encoding/json"
	"fmt"

	"custom_validator/v1/internal/validation"
)

type Address struct {
	Type   string `json:"type" validate:"required"`
	Street string `json:"street" validate:"required,max=10"`
	City   string `json:"city" validate:"required,max=20"`
}

type Person struct {
	Name      string    `json:"name" validate:"required,max=5"`
	Addresses []Address `json:"addresses" validate:"required,require_contains=type:REGISTERED&MAILING"`
}

func main() {
	v := validation.New()
	v.RegisterSliceSelector("person.addresses", "type")

	person := Person{
		Name: "Adam",
		Addresses: []Address{
			{
				Type:   "MAILING",
				Street: "",
				City:   "Wroclaw",
			},
		},
	}

	errs := v.ValidateByScopes(person, "person", []string{
		validation.ScopeMissing,
		validation.ScopeMaxLength,
	})

	b, _ := json.MarshalIndent(errs, "", "  ")
	fmt.Println(string(b))
}
