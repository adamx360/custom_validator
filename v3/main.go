package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"custom_validator/v3/internal/validation"
)

type Address struct {
	Type   string `json:"type" validate:"required"`
	Street string `json:"street" validate:"required,max=10"`
	City   string `json:"city" validate:"required,max=20"`
}

type Person struct {
	Name      string    `json:"name" validate:"required,max=5"`
	Addresses []Address `json:"addresses" validate:"required,dive" require_contains:"type:REGISTERED&MAILING"`
}

func main() {
	person := Person{
		Name: "Adam",
		Addresses: []Address{
			{
				Type:   "REGISTERED",
				Street: "LongLongLongStreetName",
				City:   "Wroclaw",
			},
		},
	}
	scopes := []string{
		validation.ScopeMissing,
		validation.ScopeMaxLength,
	}

	errs := validation.Validate(person, "person", scopes)

	hasRegistered := false
	hasMailing := false

	for _, addr := range person.Addresses {
		switch strings.ToUpper(strings.TrimSpace(addr.Type)) {
		case "REGISTERED":
			hasRegistered = true
		case "MAILING":
			hasMailing = true
		}
	}

	if !hasRegistered {
		errs = append(errs, validation.InvalidField{
			Path: `person.addresses[?@.type=="REGISTERED"]`,
			Reasons: []validation.Reason{
				{
					Scopes:      []string{validation.ScopeMissing},
					Code:        validation.CodeMissing,
					Description: "Registered address is missing",
				},
			},
		})
	}

	if !hasMailing {
		errs = append(errs, validation.InvalidField{
			Path: `person.addresses[?@.type=="MAILING"]`,
			Reasons: []validation.Reason{
				{
					Scopes:      []string{validation.ScopeMissing},
					Code:        validation.CodeMissing,
					Description: "Mailing address is missing",
				},
			},
		})
	}

	b, _ := json.MarshalIndent(errs, "", "  ")
	fmt.Println(string(b))
}
