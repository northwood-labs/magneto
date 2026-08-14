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

package kirofiles

import (
	"errors"
	"fmt"
)

// ErrMCPServerNameInvalid indicates an MCP server name does not conform to
// VS Code naming conventions.
var ErrMCPServerNameInvalid = errors.New("invalid MCP server name format")

// ValidateServerName verifies that name starts with a lowercase ASCII letter
// and contains only ASCII letters and digits.
func ValidateServerName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name must not be empty", ErrMCPServerNameInvalid)
	}

	if !isLowercaseASCIILetter(rune(name[0])) {
		return fmt.Errorf("%w: name must start with a lowercase ASCII letter", ErrMCPServerNameInvalid)
	}

	for _, character := range name {
		if !isASCIIAlphaNumeric(character) {
			return fmt.Errorf("%w: name may contain only ASCII letters and digits", ErrMCPServerNameInvalid)
		}
	}

	return nil
}

func isLowercaseASCIILetter(character rune) bool {
	return character >= 'a' && character <= 'z'
}

func isASCIIAlphaNumeric(character rune) bool {
	return isASCIIUppercaseLetter(character) || isLowercaseASCIILetter(character) || isASCIIDigit(character)
}

func isASCIIUppercaseLetter(character rune) bool {
	return character >= 'A' && character <= 'Z'
}

func isASCIIDigit(character rune) bool {
	return character >= '0' && character <= '9'
}
