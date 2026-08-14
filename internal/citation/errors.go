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

package citation

import "errors"

var (
	// ErrFileRead indicates the cited file could not be read from disk.
	ErrFileRead = errors.New("failed to read cited file")

	// ErrPathTraversal indicates the resolved file path escapes the
	// workspace root boundary.
	ErrPathTraversal = errors.New(
		"file path resolves outside workspace root",
	)

	// ErrFileTooLarge indicates the cited file exceeds the maximum
	// allowed size for citation validation.
	ErrFileTooLarge = errors.New("cited file exceeds maximum allowed size")

	// ErrSectionNotFound indicates the referenced section could not be
	// located in the document.
	ErrSectionNotFound = errors.New(
		"referenced section not found in document",
	)

	// ErrInvalidLineRange indicates the line range reference is
	// malformed or out of bounds.
	ErrInvalidLineRange = errors.New("invalid line range reference")
)
