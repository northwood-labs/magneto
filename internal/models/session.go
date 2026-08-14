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

package models // lint:allow_bad_package_name

const (
	// TerminalNotApproved indicates the review did not approve the artifact.
	TerminalNotApproved TerminalStatus = "not_approved"

	// TerminalApproved indicates the review approved the artifact after passing
	// all criteria and the attack round.
	TerminalApproved TerminalStatus = "approved"

	// TerminalHumanOverride indicates a human overrode the review outcome.
	TerminalHumanOverride TerminalStatus = "human_override"

	// TerminalPartialReview indicates the review completed partially due to
	// component degradation or early termination.
	TerminalPartialReview TerminalStatus = "partial_review"
)

type (
	// TerminalStatus represents the final disposition of a review session.
	TerminalStatus string

	// SessionMetadata contains the top-level metadata for a review session.
	SessionMetadata struct {
		SpecName           string             `json:"spec_name"`     // lint:allow_format
		ArtifactPath       string             `json:"artifact_path"` // lint:allow_format
		Timestamp          string             `json:"timestamp"`
		TerminalStatus     TerminalStatus     `json:"terminal_status"`               // lint:allow_format
		DegradedComponents []DegradationEntry `json:"degraded_components,omitempty"` // lint:allow_format
		RoundsExecuted     int                `json:"rounds_executed"`               // lint:allow_format
	}

	// HumanEscalation records a judgment question escalated to the human.
	HumanEscalation struct {
		CriterionName string `json:"criterion_name"` // lint:allow_format
		Question      string `json:"question"`
		Context       string `json:"context"`
		HumanAnswer   string `json:"human_answer,omitempty"` // lint:allow_format
		Resolved      bool   `json:"resolved"`
	}

	// HumanOverride records a human's decision to override a finding.
	HumanOverride struct {
		CriterionName  string `json:"criterion_name"`  // lint:allow_format
		HumanRationale string `json:"human_rationale"` // lint:allow_format
		FindingIndex   int    `json:"finding_index"`   // lint:allow_format
		OriginalScore  int    `json:"original_score"`  // lint:allow_format
	}

	// DegradationEntry records a component failure during a review session.
	DegradationEntry struct {
		Component        string   `json:"component"`
		FailureMode      string   `json:"failure_mode"` // lint:allow_format
		Timestamp        string   `json:"timestamp"`
		AffectedCriteria []string `json:"affected_criteria"` // lint:allow_format
	}

	// AttackRoundResult records the outcome of the mandatory attack round.
	AttackRoundResult struct {
		Issues         []ReviewFinding `json:"issues,omitempty"`
		NewIssuesFound bool            `json:"new_issues_found"` // lint:allow_format
	}

	// ReviewSessionOutput is the top-level structure written as the review
	// output Markdown's front matter and used for programmatic access.
	ReviewSessionOutput struct {
		AttackRoundResult *AttackRoundResult `json:"attack_round_result,omitempty"` // lint:allow_format
		Findings          []ReviewFinding    `json:"findings"`
		HumanEscalations  []HumanEscalation  `json:"human_escalations,omitempty"` // lint:allow_format
		HumanOverrides    []HumanOverride    `json:"human_overrides,omitempty"`   // lint:allow_format
		DeadChecks        []string           `json:"dead_checks,omitempty"`       // lint:allow_format
		Metadata          SessionMetadata    `json:"metadata"`
	}
)
