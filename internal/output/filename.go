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

package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type (
	// FilenameInput contains the parameters for generating a review output
	// filename.
	FilenameInput struct {
		Timestamp     time.Time
		SpecName      string
		WorkspaceRoot string
	}
)

// GenerateFilename produces a unique filename in the format
// {spec-name}-{ISO-8601-date}-{sequence-number}.md within the .kiro/reviews/
// directory. It creates the output directory if it does not exist and
// disambiguates by incrementing the sequence number.
func GenerateFilename(input *FilenameInput) (string, error) {
	dir := filepath.Join(input.WorkspaceRoot, ".kiro", "reviews")

	mkdirErr := os.MkdirAll(dir, 0o0755) // lint:allow_755
	if mkdirErr != nil {
		return "", fmt.Errorf("%w: %s", ErrOutputDirCreate, dir)
	}

	date := input.Timestamp.Format("2006-01-02")
	seq := 1

	for {
		name := fmt.Sprintf("%s-%s-%d.md", input.SpecName, date, seq)
		fullPath := filepath.Join(dir, name)

		_, statErr := os.Stat(fullPath)
		if os.IsNotExist(statErr) {
			return fullPath, nil
		}

		seq++
	}
}
