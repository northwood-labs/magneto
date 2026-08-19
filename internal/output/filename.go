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
	"strings"
	"sync"
	"time"

	"go.nwlabs.dev/magneto/internal/models"
)

var terminalRecordPaths = &terminalRecordPathRegistry{
	paths: make(map[string]string),
}

type (
	// FilenameInput contains the parameters for generating a review output
	// filename.
	FilenameInput struct {
		Timestamp     time.Time
		SpecName      string
		WorkspaceRoot string
	}

	// PersistSessionInput contains a terminal session and its workspace-local
	// review-record destination.
	PersistSessionInput struct {
		Session       *models.ReviewSessionOutput
		WorkspaceRoot string
	}

	// PersistSessionResult identifies the terminal record and whether this call
	// created it rather than returning the result of an idempotent retry.
	PersistSessionResult struct {
		RecordPath string
		Created    bool
	}

	terminalRecordPathRegistry struct {
		paths map[string]string
		mu    sync.Mutex
	}
)

// GenerateFilename produces a unique filename in the format
// {spec-name}-{ISO-8601-date}-{sequence-number}.md within the .kiro/reviews/
// directory. It creates the output directory if it does not exist and
// disambiguates by incrementing the sequence number.
func GenerateFilename(input *FilenameInput) (string, error) {
	if input == nil {
		return "", ErrOutputPathInvalid
	}

	dir, directoryErr := reviewDirectory(input.WorkspaceRoot)
	if directoryErr != nil {
		return "", fmt.Errorf("prepare review output directory: %w", directoryErr)
	}

	specName, specNameErr := reviewSpecName(input.SpecName)
	if specNameErr != nil {
		return "", fmt.Errorf("validate review output name: %w", specNameErr)
	}

	date := input.Timestamp.Format(time.DateOnly)
	seq := 1

	for {
		name := fmt.Sprintf("%s-%s-%d.md", specName, date, seq)
		fullPath := filepath.Join(dir, name)

		_, statErr := os.Stat(fullPath)
		if os.IsNotExist(statErr) {
			return fullPath, nil
		}

		if statErr != nil {
			return "", fmt.Errorf("%w: %w", ErrOutputFileStat, statErr)
		}

		seq++
	}
}

// PersistSession renders and writes exactly one terminal review record for an
// idempotency key. Retries made to the same MCP process return the original
// record path without creating a second record.
func PersistSession(input *PersistSessionInput) (*PersistSessionResult, error) {
	if input == nil || input.Session == nil {
		return nil, ErrOutputSessionRequired
	}

	if !isTerminalRecordStatus(input.Session.Metadata.TerminalStatus) {
		return nil, ErrOutputTerminalStatus
	}

	idempotencyKey := strings.TrimSpace(input.Session.TerminalIdempotencyKey)
	if idempotencyKey == "" {
		return nil, ErrOutputIdempotencyKeyRequired
	}

	timestamp, timestampErr := terminalRecordTimestamp(input.Session.Metadata.Timestamp)
	if timestampErr != nil {
		return nil, fmt.Errorf("parse terminal review timestamp: %w", timestampErr)
	}

	terminalRecordPaths.mu.Lock()
	defer terminalRecordPaths.mu.Unlock()

	filename, filenameErr := GenerateFilename(&FilenameInput{
		Timestamp:     timestamp,
		SpecName:      input.Session.Metadata.SpecName,
		WorkspaceRoot: input.WorkspaceRoot,
	})
	if filenameErr != nil {
		return nil, fmt.Errorf("generate terminal review filename: %w", filenameErr)
	}

	registryKey := filepath.Dir(filepath.Dir(filepath.Dir(filename))) + "\x00" + idempotencyKey
	if recordPath, found := terminalRecordPaths.paths[registryKey]; found {
		return &PersistSessionResult{RecordPath: recordPath}, nil
	}

	writeErr := writeTerminalRecord(filename, RenderSession(input.Session))
	if writeErr != nil {
		return nil, fmt.Errorf("persist terminal review record: %w", writeErr)
	}

	terminalRecordPaths.paths[registryKey] = filename

	return &PersistSessionResult{RecordPath: filename, Created: true}, nil
}

// isTerminalRecordStatus reports whether status can be persisted as a terminal
// review record.
func isTerminalRecordStatus(status models.TerminalStatus) bool {
	switch status {
	case models.TerminalApproved,
		models.TerminalNotApproved,
		models.TerminalPartialReview,
		models.TerminalHumanOverride:
		return true
	default:
		return false
	}
}

// reviewDirectory creates and validates the local review directory, including
// symlink resolution, before any terminal record is written.
func reviewDirectory(workspaceRoot string) (string, error) {
	if strings.TrimSpace(workspaceRoot) == "" {
		return "", ErrOutputPathInvalid
	}

	absoluteRoot, absoluteErr := filepath.Abs(workspaceRoot)
	if absoluteErr != nil {
		return "", fmt.Errorf("%w: %w", ErrOutputPathInvalid, absoluteErr)
	}

	resolvedRoot, resolveRootErr := filepath.EvalSymlinks(absoluteRoot)
	if resolveRootErr != nil {
		return "", fmt.Errorf("%w: %w", ErrOutputPathInvalid, resolveRootErr)
	}

	dir := filepath.Join(absoluteRoot, ".kiro", "reviews")

	mkdirErr := os.MkdirAll(dir, 0o0755) // lint:allow_755
	if mkdirErr != nil {
		return "", fmt.Errorf("%w: %w", ErrOutputDirCreate, mkdirErr)
	}

	resolvedDir, resolveDirErr := filepath.EvalSymlinks(dir)
	if resolveDirErr != nil {
		return "", fmt.Errorf("%w: %w", ErrOutputPathInvalid, resolveDirErr)
	}

	containmentErr := verifyContainedDirectory(resolvedRoot, resolvedDir)
	if containmentErr != nil {
		return "", fmt.Errorf("validate review output containment: %w", containmentErr)
	}

	return dir, nil
}

// reviewSpecName ensures the supplied spec name cannot escape the review
// directory through a generated filename.
func reviewSpecName(specName string) (string, error) {
	trimmed := strings.TrimSpace(specName)
	if trimmed == "" || trimmed == "." || trimmed == ".." || filepath.Base(trimmed) != trimmed ||
		strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", ErrOutputPathInvalid
	}

	return trimmed, nil
}

// verifyContainedDirectory reports whether target resolves under root.
func verifyContainedDirectory(root, target string) error {
	relativePath, relativeErr := filepath.Rel(root, target)
	if relativeErr != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return ErrOutputPathInvalid
	}

	return nil
}

// terminalRecordTimestamp parses the two timestamp representations accepted by
// existing review sessions.
func terminalRecordTimestamp(value string) (time.Time, error) {
	timestamp, timestampErr := time.Parse(time.RFC3339, value)
	if timestampErr == nil {
		return timestamp, nil
	}

	date, dateErr := time.Parse(time.DateOnly, value)
	if dateErr != nil {
		return time.Time{}, fmt.Errorf("%w: %w", ErrOutputTimestampInvalid, dateErr)
	}

	return date, nil
}

// writeTerminalRecord creates a new record without overwriting an existing
// file. The caller holds the terminal-record registry lock.
func writeTerminalRecord(path, rendered string) error {
	file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o0666) // lint:allow_666
	if openErr != nil {
		return fmt.Errorf("%w: %w", ErrOutputRecordWrite, openErr)
	}

	_, writeErr := file.WriteString(rendered)
	if writeErr != nil {
		return fmt.Errorf("clean incomplete terminal record: %w", removeIncompleteTerminalRecord(file, path, writeErr))
	}

	closeErr := file.Close()
	if closeErr != nil {
		return fmt.Errorf("clean incomplete terminal record: %w", removeIncompleteTerminalRecord(nil, path, closeErr))
	}

	return nil
}

// removeIncompleteTerminalRecord closes and removes a partial record before
// returning the original write failure.
func removeIncompleteTerminalRecord(file *os.File, path string, cause error) error {
	if file != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("%w: %w", ErrOutputRecordWrite, closeErr)
		}
	}

	removeErr := os.Remove(path)
	if removeErr != nil {
		return fmt.Errorf("%w: %w", ErrOutputRecordWrite, removeErr)
	}

	return fmt.Errorf("%w: %w", ErrOutputRecordWrite, cause)
}
