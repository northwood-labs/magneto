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

package models_test

import (
	"testing"

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/models"
)

// TestCanonicalFindingDomains verifies known domains are duplicate-free and
// returned in stable enum order without changing the source slice.
func TestCanonicalFindingDomains(t *testing.T) {
	input := []models.FindingDomain{
		models.DomainOperations,
		models.DomainSecurity,
		models.DomainOperations,
		models.FindingDomain("unknown"),
		models.DomainCorrectness,
	}

	actual := models.CanonicalFindingDomains(input)

	assert.Equal(t, []models.FindingDomain{
		models.DomainSecurity,
		models.DomainCorrectness,
		models.DomainOperations,
	}, actual)
	assert.Equal(t, models.DomainOperations, input[0])
}
