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

	// CriticalityRequired identifies a component whose failure blocks approval.
	CriticalityRequired ComponentCriticality = "required"

	// CriticalityOptional identifies a component whose failure remains auditable
	// without blocking approval evaluation.
	CriticalityOptional ComponentCriticality = "optional"

	// SelectionSelected identifies an artifact selected for a review session.
	SelectionSelected SelectionDecision = "selected"

	// SelectionSkipped identifies an artifact that was not selected for review.
	SelectionSkipped SelectionDecision = "skipped"

	// SelectionAmbiguous identifies an artifact selected because classification
	// evidence was incomplete or conflicting.
	SelectionAmbiguous SelectionDecision = "ambiguous"

	// HumanDecisionOverride identifies a human decision that overrides review
	// outcome handling.
	HumanDecisionOverride HumanDecision = "override"

	// HumanDecisionBlockAcceptance identifies a human decision that accepts a
	// progression block.
	HumanDecisionBlockAcceptance HumanDecision = "block_acceptance"

	// Phase3BaselineAbsent records that no pre-Phase-1 control baseline exists.
	Phase3BaselineAbsent Phase3Baseline = "absent"
)

type (
	// TerminalStatus represents the final disposition of a review session.
	TerminalStatus string

	// ComponentCriticality represents the impact of an unavailable component.
	ComponentCriticality string

	// SelectionDecision represents the deterministic selection result for an
	// artifact.
	SelectionDecision string

	// HumanDecision represents an auditable decision made by the human.
	HumanDecision string

	// Phase3Baseline represents the availability of a Phase 3 control baseline.
	Phase3Baseline string

	// SessionMetadata contains the top-level metadata for a review session.
	SessionMetadata struct {
		TaskExecutionID             string             `json:"task_execution_id,omitempty"` // lint:allow_format
		SelectionReason             string             `json:"selection_reason,omitempty"`  // lint:allow_format
		Timestamp                   string             `json:"timestamp"`
		TerminalStatus              TerminalStatus     `json:"terminal_status"`                          // lint:allow_format
		SelectionDecision           SelectionDecision  `json:"selection_decision,omitempty"`             // lint:allow_format
		Phase3Baseline              Phase3Baseline     `json:"phase_3_baseline"`                         // lint:allow_format
		ArtifactPath                string             `json:"artifact_path"`                            // lint:allow_format
		SessionID                   string             `json:"session_id,omitempty"`                     // lint:allow_format
		SpecName                    string             `json:"spec_name"`                                // lint:allow_format
		DegradedComponents          []DegradationEntry `json:"degraded_components,omitempty"`            // lint:allow_format
		TriggeredBlastRadiusDomains []string           `json:"triggered_blast_radius_domains,omitempty"` // lint:allow_format
		LoadedRubricCriteria        []string           `json:"loaded_rubric_criteria,omitempty"`         // lint:allow_format
		RoundsExecuted              int                `json:"rounds_executed"`                          // lint:allow_format
		SelectionAmbiguous          bool               `json:"selection_ambiguous"`                      // lint:allow_format
		FoundationalArtifact        bool               `json:"foundational_artifact"`                    // lint:allow_format
		LegacyScoreMigrated         bool               `json:"legacy_score_migrated,omitempty"`          // lint:allow_format
	}

	// HumanEscalation records a judgment question escalated to the human.
	HumanEscalation struct {
		CriterionName      string `json:"criterion_name"` // lint:allow_format
		Question           string `json:"question"`
		Context            string `json:"context"`
		InspectedEvidence  string `json:"inspected_evidence"`     // lint:allow_format
		RemainingAmbiguity string `json:"remaining_ambiguity"`    // lint:allow_format
		HumanAnswer        string `json:"human_answer,omitempty"` // lint:allow_format
		Resolved           bool   `json:"resolved"`
	}

	// HumanOverride records a human's decision to override a finding.
	HumanOverride struct {
		CriterionName                 string        `json:"criterion_name"`  // lint:allow_format
		HumanRationale                string        `json:"human_rationale"` // lint:allow_format
		Decision                      HumanDecision `json:"decision"`
		FindingIndex                  int           `json:"finding_index"`                   // lint:allow_format
		OriginalCriterionSatisfaction int           `json:"original_criterion_satisfaction"` // lint:allow_format
	}

	// HumanBlockAcceptance records a human decision to accept a progression
	// block without conflating it with an override.
	HumanBlockAcceptance struct {
		CriterionName   string        `json:"criterion_name"` // lint:allow_format
		Decision        HumanDecision `json:"decision"`
		EvidenceContext string        `json:"evidence_context"` // lint:allow_format
		HumanRationale  string        `json:"human_rationale"`  // lint:allow_format
	}

	// UnavailableValue identifies a terminal-record value that could not be
	// persisted and why it was unavailable.
	UnavailableValue struct {
		Field  string `json:"field"`
		Reason string `json:"reason"`
	}

	// DegradationEntry records a component failure during a review session.
	DegradationEntry struct {
		Component           string               `json:"component"`
		FailureMode         string               `json:"failure_mode"` // lint:allow_format
		Timestamp           string               `json:"timestamp"`
		Criticality         ComponentCriticality `json:"criticality"`
		UnavailableValueKey string               `json:"unavailable_value_key,omitempty"` // lint:allow_format
		AffectedCriteria    []string             `json:"affected_criteria"`               // lint:allow_format
	}

	// AttackRoundResult records the outcome of the mandatory attack round.
	AttackRoundResult struct {
		Issues         []ReviewFinding `json:"issues,omitempty"`
		NewIssuesFound bool            `json:"new_issues_found"` // lint:allow_format
	}

	// ReviewSessionOutput is the top-level structure written as the review
	// output Markdown's front matter and used for programmatic access.
	ReviewSessionOutput struct {
		AttackRoundResult      *AttackRoundResult    `json:"attack_round_result,omitempty"`      // lint:allow_format
		HumanBlockAcceptance   *HumanBlockAcceptance `json:"human_block_acceptance,omitempty"`   // lint:allow_format
		TerminalIdempotencyKey string                `json:"terminal_idempotency_key,omitempty"` // lint:allow_format
		Findings               []ReviewFinding       `json:"findings"`
		HumanEscalations       []HumanEscalation     `json:"human_escalations,omitempty"`  // lint:allow_format
		HumanOverrides         []HumanOverride       `json:"human_overrides,omitempty"`    // lint:allow_format
		UnavailableValues      []UnavailableValue    `json:"unavailable_values,omitempty"` // lint:allow_format
		DeadChecks             []string              `json:"dead_checks,omitempty"`        // lint:allow_format
		Metadata               SessionMetadata       `json:"metadata"`
	}
)
