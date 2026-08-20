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

import "errors"

var (
	// ErrOutputDirCreate indicates the output directory could not be created.
	ErrOutputDirCreate = errors.New("failed to create output directory")

	// ErrOutputFileStat indicates a review output filename could not be checked.
	ErrOutputFileStat = errors.New("failed to inspect review output filename")

	// ErrOutputIdempotencyKeyRequired indicates terminal persistence received
	// no idempotency key.
	ErrOutputIdempotencyKeyRequired = errors.New("terminal review record requires an idempotency key")

	// ErrOutputPathInvalid indicates an output location or spec name would be
	// unsafe for review-record persistence.
	ErrOutputPathInvalid = errors.New("review output path is invalid")

	// ErrOutputRecordWrite indicates a terminal record could not be written.
	ErrOutputRecordWrite = errors.New("failed to write terminal review record")

	// ErrOutputSessionRequired indicates terminal persistence received no
	// session.
	ErrOutputSessionRequired = errors.New("terminal review session is required")

	// ErrOutputTerminalStatus indicates persistence received a nonterminal
	// session status.
	ErrOutputTerminalStatus = errors.New("terminal review record requires a terminal status")

	// ErrOutputTimestampInvalid indicates terminal persistence received an
	// unsupported session timestamp.
	ErrOutputTimestampInvalid = errors.New("terminal review record timestamp is invalid")
)
