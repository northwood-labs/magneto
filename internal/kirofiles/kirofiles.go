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
	"embed"
	"fmt"
	"io/fs"
)

const (
	sourcePrefix = "source/"

	hookTriggerFile  = "hooks/adversarial-review-trigger.json"
	mcpTemplateFile  = "settings/mcp.json"
	antiPatternsFile = "steering/adversarial-review-anti-patterns.md"
	architectureFile = "steering/adversarial-review-architecture-constraints.md"
	rubricFile       = "steering/adversarial-review-rubric.md"
)

//go:embed source/hooks/adversarial-review-trigger.json
//go:embed source/settings/mcp.json
//go:embed source/steering/adversarial-review-anti-patterns.md
//go:embed source/steering/adversarial-review-architecture-constraints.md
//go:embed source/steering/adversarial-review-rubric.md
var content embed.FS

// Content returns the embedded bytes for a file in the fixed Kiro manifest.
func Content(path string) ([]byte, error) {
	switch path {
	case hookTriggerFile, mcpTemplateFile, antiPatternsFile, architectureFile, rubricFile:
		data, readErr := content.ReadFile(sourcePrefix + path)
		if readErr != nil {
			return nil, fmt.Errorf("%w: %s", readErr, path)
		}

		return data, nil
	default:
		return nil, fs.ErrNotExist
	}
}

// Files returns the fixed Kiro asset manifest in installation order.
func Files() []string {
	return []string{
		hookTriggerFile,
		mcpTemplateFile,
		antiPatternsFile,
		architectureFile,
		rubricFile,
	}
}

// MCPTemplate returns the embedded Magneto MCP server definition.
func MCPTemplate() []byte {
	template, readErr := Content(mcpTemplateFile)
	if readErr != nil {
		return nil
	}

	return template
}
