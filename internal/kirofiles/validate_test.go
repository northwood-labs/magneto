// Copyright 2026, Northwood Labs, LLC <license@northwood-labs.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kirofiles_test

import (
	"errors"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/kirofiles"
)

const minimumServerNameValidationChecks = 100

func TestValidateServerName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{name: "empty", input: "", valid: false},
		{name: "starts with digit", input: "1magneto", valid: false},
		{name: "starts uppercase", input: "Magneto", valid: false},
		{name: "contains hyphen", input: "my-magneto", valid: false},
		{name: "contains underscore", input: "my_magneto", valid: false},
		{name: "contains whitespace", input: "my magneto", valid: false},
		{name: "contains special character", input: "magneto!", valid: false},
		{name: "contains non ASCII character", input: "magnetö", valid: false},
		{name: "single lowercase letter", input: "m", valid: true},
		{name: "camel case", input: "myMagneto", valid: true},
		{name: "contains uppercase letters and digits", input: "magnetoMCP2", valid: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErr := kirofiles.ValidateServerName(test.input)

			if test.valid {
				require.NoError(t, validationErr)
				return
			}

			require.Error(t, validationErr)
			assert.True(
				t,
				errors.Is(validationErr, kirofiles.ErrMCPServerNameInvalid),
				"expected error for %q to wrap ErrMCPServerNameInvalid",
				test.input,
			)
		})
	}
}

// TestProperty_ServerNameValidationAcceptsOnlyValidNames verifies Property 5:
// server name validation accepts only valid names.
//
// For every generated string, validation must agree with the specified rule:
// the value is non-empty, begins with a lowercase ASCII letter, and otherwise
// contains only ASCII letters and digits. Invalid values must wrap the public
// invalid-name sentinel.
//
// **Validates: Requirements 9.1, 9.2, 10.3, 10.4**.
func TestProperty_ServerNameValidationAcceptsOnlyValidNames(t *testing.T) {
	checks := 0

	rapid.Check(t, func(rt *rapid.T) {
		checks++

		name := rapid.String().Draw(rt, "server_name")
		valid := isValidServerNameBySpecification(name)
		validationErr := kirofiles.ValidateServerName(name)

		if valid {
			if validationErr != nil {
				rt.Fatalf("valid server name %q returned error: %v", name, validationErr)
			}

			return
		}

		if validationErr == nil {
			rt.Fatalf("invalid server name %q returned no error", name)
		}

		if !errors.Is(validationErr, kirofiles.ErrMCPServerNameInvalid) {
			rt.Fatalf("invalid server name %q did not wrap ErrMCPServerNameInvalid: %v", name, validationErr)
		}
	})

	if checks < minimumServerNameValidationChecks {
		t.Fatalf("expected at least %d Rapid checks, ran %d", minimumServerNameValidationChecks, checks)
	}
}

func isValidServerNameBySpecification(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}

	for _, character := range name {
		asciiLetter := character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'

		asciiDigit := character >= '0' && character <= '9'
		if !asciiLetter && !asciiDigit {
			return false
		}
	}

	return true
}
