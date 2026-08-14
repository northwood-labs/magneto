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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	mcpServersKey       = "mcpServers"
	magnetoKeySubstring = "magneto"
)

var errMCPConfigurationMalformed = errors.New("malformed MCP configuration")

type (
	// MergeInput contains the MCP configuration values to merge.
	MergeInput struct {
		Existing   []byte
		ServerName string
		Definition json.RawMessage
	}

	// MergeResult contains the formatted merged MCP configuration.
	MergeResult struct {
		Output []byte
	}
)

// MergeMCPConfig merges the Magneto definition into an MCP configuration.
func MergeMCPConfig(input *MergeInput) (*MergeResult, error) {
	configuration, configurationErr := parseMCPConfiguration(input.Existing)
	if configurationErr != nil {
		return nil, fmt.Errorf("failed to parse MCP configuration: %w", configurationErr)
	}

	servers, serversErr := parseMCPServers(configuration)
	if serversErr != nil {
		return nil, fmt.Errorf("failed to parse MCP servers: %w", serversErr)
	}

	removeMagnetoServers(servers, input.ServerName)

	servers[input.ServerName] = input.Definition

	configuration[mcpServersKey], configurationErr = json.Marshal(servers)
	if configurationErr != nil {
		return nil, fmt.Errorf("failed to marshal MCP servers: %w", configurationErr)
	}

	output, marshalErr := json.MarshalIndent(configuration, "", "  ")
	if marshalErr != nil {
		return nil, fmt.Errorf("failed to marshal MCP configuration: %w", marshalErr)
	}

	return &MergeResult{Output: append(output, '\n')}, nil
}

func parseMCPConfiguration(existing []byte) (map[string]json.RawMessage, error) {
	configuration := make(map[string]json.RawMessage)
	if len(existing) == 0 {
		return configuration, nil
	}

	unmarshalErr := json.Unmarshal(existing, &configuration)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("malformed MCP configuration: %w", unmarshalErr)
	}

	if configuration == nil {
		return nil, fmt.Errorf("%w: expected a JSON object", errMCPConfigurationMalformed)
	}

	return configuration, nil
}

func parseMCPServers(configuration map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	rawServers, found := configuration[mcpServersKey]
	if !found {
		return make(map[string]json.RawMessage), nil
	}

	servers := make(map[string]json.RawMessage)

	unmarshalErr := json.Unmarshal(rawServers, &servers)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("malformed MCP configuration mcpServers value: %w", unmarshalErr)
	}

	if servers == nil {
		return make(map[string]json.RawMessage), nil
	}

	return servers, nil
}

func removeMagnetoServers(servers map[string]json.RawMessage, serverName string) {
	delete(servers, serverName)

	for name := range servers {
		if strings.Contains(strings.ToLower(name), magnetoKeySubstring) {
			delete(servers, name)
		}
	}
}
