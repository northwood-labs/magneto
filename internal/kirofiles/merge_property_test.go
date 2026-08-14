// Copyright ((20\d\d\-2026)|(2026)), Northwood Labs, LLC <license@northwood-labs.com>
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
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/kirofiles"
)

const minimumMCPMergeChecks = 100

// TestProperty_MCPMergePreservesNonMagnetoEntries verifies Property 1: MCP
// merge preserves non-Magneto entries.
//
// For every valid configuration with arbitrary non-Magneto server entries and
// top-level metadata, merging preserves each value's raw JSON tokens. Values
// are compacted before comparison because the merge contract requires the full
// output document to be re-indented.
//
// **Validates: Requirements 5.5, 5.6**.
func TestProperty_MCPMergePreservesNonMagnetoEntries(t *testing.T) {
	checks := 0

	rapid.Check(t, func(rt *rapid.T) {
		checks++

		existing, expectedServers, expectedMetadata := drawMCPConfiguration(rt)
		serverName := "installed" + rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, "server_name")

		result, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
			Existing:   existing,
			ServerName: serverName,
			Definition: kirofiles.MCPTemplate(),
		})
		if mergeErr != nil {
			rt.Fatal(mergeErr)
		}

		merged := make(map[string]json.RawMessage)

		unmarshalErr := json.Unmarshal(result.Output, &merged)
		if unmarshalErr != nil {
			rt.Fatal(unmarshalErr)
		}

		rawServers, found := merged["mcpServers"]
		if !found {
			rt.Fatal("merged configuration did not contain mcpServers")
		}

		actualServers := make(map[string]json.RawMessage)

		unmarshalErr = json.Unmarshal(rawServers, &actualServers)
		if unmarshalErr != nil {
			rt.Fatal(unmarshalErr)
		}

		for name, expected := range expectedServers {
			actual, found := actualServers[name]
			if !found {
				rt.Fatalf("preserved MCP server %q was removed", name)
			}

			assertRawJSONEqual(rt, name, expected, actual)
		}

		for name, expected := range expectedMetadata {
			actual, found := merged[name]
			if !found {
				rt.Fatalf("top-level key %q was removed", name)
			}

			assertRawJSONEqual(rt, name, expected, actual)
		}
	})

	if checks < minimumMCPMergeChecks {
		t.Fatalf("expected at least %d Rapid checks, ran %d", minimumMCPMergeChecks, checks)
	}
}

// TestProperty_MCPMergeRemovesLegacyMagnetoKeys verifies Property 2: MCP
// merge removes all Magneto-related keys and adds the new entry.
//
// Each generated configuration includes the exact selected key plus legacy keys
// with random casing and surrounding text that embeds "magneto". The merge must
// leave only the selected key, replaced with the current definition.
//
// **Validates: Requirements 5.3, 5.4**.
func TestProperty_MCPMergeRemovesLegacyMagnetoKeys(t *testing.T) {
	checks := 0

	rapid.Check(t, func(rt *rapid.T) {
		checks++

		serverName := "installed" + rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, "server_name")
		existing := drawLegacyMCPConfiguration(rt, serverName)
		result := mergeLegacyMCPConfiguration(rt, existing, serverName)

		assertLegacyMCPServersRemoved(rt, result.Output, serverName)
	})

	if checks < minimumMCPMergeChecks {
		t.Fatalf("expected at least %d Rapid checks, ran %d", minimumMCPMergeChecks, checks)
	}
}

// TestProperty_MCPMergeOutputFormatting verifies Property 3: MCP merge output
// formatting.
//
// For every valid merge input, the output must be valid JSON with canonical
// two-space indentation and exactly one trailing newline.
//
// **Validates: Requirements 5.7**.
func TestProperty_MCPMergeOutputFormatting(t *testing.T) {
	checks := 0

	rapid.Check(t, func(rt *rapid.T) {
		checks++

		existing, _, _ := drawMCPConfiguration(rt)
		if rapid.Bool().Draw(rt, "missing_configuration") {
			existing = nil
		}

		serverName := "installed" + rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, "server_name")
		result := mergeLegacyMCPConfiguration(rt, existing, serverName)

		assertMCPMergeFormatting(rt, result.Output)
	})

	if checks < minimumMCPMergeChecks {
		t.Fatalf("expected at least %d Rapid checks, ran %d", minimumMCPMergeChecks, checks)
	}
}

// TestProperty_MCPMergeIsIdempotent verifies Property 4: MCP merge
// idempotence.
//
// For every valid merge input, applying the same server name and definition to
// the merged output produces byte-identical output.
//
// **Validates: Requirements 7.1**.
func TestProperty_MCPMergeIsIdempotent(t *testing.T) {
	checks := 0

	rapid.Check(t, func(rt *rapid.T) {
		checks++

		existing, _, _ := drawMCPConfiguration(rt)
		if rapid.Bool().Draw(rt, "missing_configuration") {
			existing = nil
		}

		serverName := "installed" + rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, "server_name")
		first := mergeLegacyMCPConfiguration(rt, existing, serverName)
		second := mergeLegacyMCPConfiguration(rt, first.Output, serverName)

		if !bytes.Equal(first.Output, second.Output) {
			rt.Fatalf(
				"merging the same definition twice changed output: first %q, second %q",
				first.Output,
				second.Output,
			)
		}
	})

	if checks < minimumMCPMergeChecks {
		t.Fatalf("expected at least %d Rapid checks, ran %d", minimumMCPMergeChecks, checks)
	}
}

func assertMCPMergeFormatting(rt *rapid.T, output []byte) {
	if !json.Valid(output) {
		rt.Fatalf("merged output is not valid JSON: %q", output)
	}

	if !bytes.HasSuffix(output, []byte{'\n'}) || bytes.HasSuffix(output, []byte{'\n', '\n'}) {
		rt.Fatalf("merged output must end with exactly one newline: %q", output)
	}

	var compact bytes.Buffer

	compactErr := json.Compact(&compact, output)
	if compactErr != nil {
		rt.Fatal(compactErr)
	}

	var expected bytes.Buffer

	indentErr := json.Indent(&expected, compact.Bytes(), "", "  ")
	if indentErr != nil {
		rt.Fatal(indentErr)
	}

	expected.WriteByte('\n')

	if !bytes.Equal(output, expected.Bytes()) {
		rt.Fatalf("merged output is not formatted with two-space indentation: %q", output)
	}
}

func drawLegacyMCPConfiguration(rt *rapid.T, serverName string) []byte {
	servers := map[string]json.RawMessage{
		serverName: drawMCPValue(rt, "selected_server_value"),
	}
	legacyCount := rapid.IntRange(1, 5).Draw(rt, "legacy_server_count")

	for index := range legacyCount {
		legacyName := drawLegacyMagnetoKey(rt, "legacy_server_"+strconv.Itoa(index))

		servers[legacyName] = drawMCPValue(rt, "legacy_server_value")
	}

	serversJSON, marshalErr := json.Marshal(servers)
	if marshalErr != nil {
		rt.Fatal(marshalErr)
	}

	existing, marshalErr := json.Marshal(map[string]json.RawMessage{"mcpServers": serversJSON})
	if marshalErr != nil {
		rt.Fatal(marshalErr)
	}

	return existing
}

func mergeLegacyMCPConfiguration(rt *rapid.T, existing []byte, serverName string) *kirofiles.MergeResult {
	result, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
		Existing:   existing,
		ServerName: serverName,
		Definition: kirofiles.MCPTemplate(),
	})
	if mergeErr != nil {
		rt.Fatal(mergeErr)
	}

	return result
}

func assertLegacyMCPServersRemoved(rt *rapid.T, output []byte, serverName string) {
	merged := make(map[string]json.RawMessage)

	unmarshalErr := json.Unmarshal(output, &merged)
	if unmarshalErr != nil {
		rt.Fatal(unmarshalErr)
	}

	actualServers := make(map[string]json.RawMessage)

	unmarshalErr = json.Unmarshal(merged["mcpServers"], &actualServers)
	if unmarshalErr != nil {
		rt.Fatal(unmarshalErr)
	}

	if len(actualServers) != 1 {
		rt.Fatalf("expected only selected MCP server %q, got %d entries", serverName, len(actualServers))
	}

	actualDefinition, found := actualServers[serverName]
	if !found {
		rt.Fatalf("selected MCP server %q was not written", serverName)
	}

	assertRawJSONEqual(rt, serverName, kirofiles.MCPTemplate(), actualDefinition)
}

func drawLegacyMagnetoKey(rt *rapid.T, label string) string {
	var name strings.Builder

	name.WriteString(rapid.StringMatching(`[A-Za-z0-9]{0,12}`).Draw(rt, label+"_prefix"))

	for index, letter := range "magneto" {
		if rapid.Bool().Draw(rt, label+"_case_"+strconv.Itoa(index)) {
			name.WriteRune(letter - ('a' - 'A'))
			continue
		}

		name.WriteRune(letter)
	}

	name.WriteString(rapid.StringMatching(`[A-Za-z0-9]{0,12}`).Draw(rt, label+"_suffix"))

	return name.String()
}

func drawMCPConfiguration(rt *rapid.T) ([]byte, map[string]json.RawMessage, map[string]json.RawMessage) {
	serverCount := rapid.IntRange(1, 5).Draw(rt, "server_count")
	metadataCount := rapid.IntRange(1, 4).Draw(rt, "metadata_count")
	servers := make(map[string]json.RawMessage, serverCount)
	metadata := make(map[string]json.RawMessage, metadataCount)
	configuration := make(map[string]json.RawMessage, metadataCount+1)

	for index := range serverCount {
		nameSuffix := rapid.StringMatching(`[A-LN-Za-ln-z0-9]{1,16}`).Draw(rt, "server_name")
		name := "external" + strconv.Itoa(index) + nameSuffix

		servers[name] = drawMCPValue(rt, "server_value")
	}

	serversJSON, marshalErr := json.Marshal(servers)
	if marshalErr != nil {
		rt.Fatal(marshalErr)
	}

	configuration["mcpServers"] = serversJSON

	for index := range metadataCount {
		name := "metadata" + strconv.Itoa(index)
		value := drawMetadataValue(rt, "metadata_value")

		metadata[name] = value
		configuration[name] = value
	}

	existing, marshalErr := json.Marshal(configuration)
	if marshalErr != nil {
		rt.Fatal(marshalErr)
	}

	return existing, servers, metadata
}

func drawMCPValue(rt *rapid.T, label string) json.RawMessage {
	command := rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, label+"_command")
	argument := rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, label+"_argument")
	environmentValue := rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, label+"_environment")
	disabled := strconv.FormatBool(rapid.Bool().Draw(rt, label+"_disabled"))

	return json.RawMessage(
		`{"command":"` + command + `","args":["` + argument + `"],"env":{"TOKEN":"` + environmentValue +
			`"},"disabled":` + disabled + `}`,
	)
}

func drawMetadataValue(rt *rapid.T, label string) json.RawMessage {
	owner := rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, label+"_owner")
	tag := rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, label+"_tag")
	priority := strconv.Itoa(rapid.IntRange(0, 100).Draw(rt, label+"_priority"))

	return json.RawMessage(`{"owner":"` + owner + `","tags":["` + tag + `"],"priority":` + priority + `}`)
}

func assertRawJSONEqual(rt *rapid.T, name string, expected, actual json.RawMessage) {
	var expectedCompact bytes.Buffer

	compactErr := json.Compact(&expectedCompact, expected)
	if compactErr != nil {
		rt.Fatal(compactErr)
	}

	var actualCompact bytes.Buffer

	compactErr = json.Compact(&actualCompact, actual)
	if compactErr != nil {
		rt.Fatal(compactErr)
	}

	if !bytes.Equal(expectedCompact.Bytes(), actualCompact.Bytes()) {
		rt.Fatalf(
			"raw JSON value for %q changed: expected %s, got %s",
			name,
			expectedCompact.Bytes(),
			actualCompact.Bytes(),
		)
	}
}
