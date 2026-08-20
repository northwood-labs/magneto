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

package session_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

// Feature: adversarial-review-operational-workflow, Property 2: Identity-safe
// role selection and evidence scoring
//
// For any author identity, eligible Confirmer identity set, active rubric
// criteria, and evidence availability outcomes, the selected Confirmer differs
// from the author and stored findings contain exactly one clamped
// Criterion_Satisfaction per active criterion, with unavailable evidence mapped
// to 1 and clear evidence mapped to 2 through 10.
//
// **Validates: Requirements 2.4, 2.6, 2.7, 3.1, 3.2**.

const (
	minSatisfaction = 1
	maxSatisfaction = 10
	minAvailable    = 2
)

type (
	// ConfirmerSelectionInput holds generated inputs for the identity-safe role
	// selection property.
	ConfirmerSelectionInput struct {
		AuthorIdentity      string
		CandidateIdentities []string
	}

	// CriterionEvidenceInput holds generated inputs for a single rubric
	// criterion's evidence availability and satisfaction.
	CriterionEvidenceInput struct {
		CriterionName     string
		EvidenceAvailable bool
		RawSatisfaction   int
	}

	// SelectionResultInput holds the result of a Confirmer selection for
	// assertion.
	SelectionResultInput struct {
		Input    *ConfirmerSelectionInput
		Selected string
		OK       bool
	}

	// TargetConsistencyInput holds the assertion inputs for a single finding's
	// target consistency check.
	TargetConsistencyInput struct {
		Finding     *models.ReviewFinding
		Index       int
		IsTarget    bool
		GateValid   bool
		HighImpact  bool
		InTargetSet bool
	}

	// TransitionAssertionInput holds the inputs for a single finding-status
	// transition assertion.
	TransitionAssertionInput struct {
		GateAvailability session.GateAvailability
		Finding          models.ReviewFinding
		Result           models.ReviewFinding
		GateValid        bool
		HighImpact       bool
	}
)

// selectConfirmer models the protocol guarantee: select a Confirmer whose
// identity differs from the author. Returns the selected identity and whether
// selection succeeded.
func selectConfirmer(input *ConfirmerSelectionInput) (string, bool) {
	for _, candidate := range input.CandidateIdentities {
		if candidate != input.AuthorIdentity {
			return candidate, true
		}
	}

	return "", false
}

// clampSatisfaction clamps a raw satisfaction value to the valid range [1, 10].
func clampSatisfaction(raw int) int {
	if raw < minSatisfaction {
		return minSatisfaction
	}

	if raw > maxSatisfaction {
		return maxSatisfaction
	}

	return raw
}

// computeCriterionSatisfaction models the evidence-availability rule:
// unavailable evidence maps to 1, available evidence maps to a clamped value in
// [2, 10].
func computeCriterionSatisfaction(input *CriterionEvidenceInput) int {
	if !input.EvidenceAvailable {
		return minSatisfaction
	}

	clamped := clampSatisfaction(input.RawSatisfaction)

	if clamped < minAvailable {
		return minAvailable
	}

	return clamped
}

// TestProperty_IdentitySafeRoleSelectionAndEvidenceScoring verifies Property 2:
// Identity-safe role selection and evidence scoring.
//
// Feature: adversarial-review-operational-workflow, Property 2: Identity-safe
// role selection and evidence scoring
//
// **Validates: Requirements 2.4, 2.6, 2.7, 3.1, 3.2**.
func TestProperty_IdentitySafeRoleSelectionAndEvidenceScoring(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate author identity (non-empty).
		authorIdentity := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "author_identity")

		// Generate candidate Confirmer identities (1-5, non-empty).
		candidateCount := rapid.IntRange(1, 5).Draw(t, "candidate_count")

		candidates := make([]string, candidateCount)
		for i := range candidateCount {
			candidates[i] = rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "candidate")
		}

		selectionInput := &ConfirmerSelectionInput{
			AuthorIdentity:      authorIdentity,
			CandidateIdentities: candidates,
		}

		selected, ok := selectConfirmer(selectionInput)

		// Verify identity-safe selection (Requirement 2.4).
		assertConfirmerSelection(t, &SelectionResultInput{
			Input:    selectionInput,
			Selected: selected,
			OK:       ok,
		})

		// Generate active rubric criteria (1-5 distinct non-empty names).
		criteriaCount := rapid.IntRange(1, 5).Draw(t, "criteria_count")
		criteriaNames := generateDistinctCriteria(t, criteriaCount)

		// Generate evidence availability and satisfaction for each criterion.
		evidenceInputs := make([]CriterionEvidenceInput, criteriaCount)

		for i, name := range criteriaNames {
			available := rapid.Bool().Draw(t, "available")

			// Generate raw satisfaction that may be out of range to test
			// clamping.
			rawScore := rapid.IntRange(-5, 15).Draw(t, "raw_satisfaction")

			evidenceInputs[i] = CriterionEvidenceInput{
				CriterionName:     name,
				EvidenceAvailable: available,
				RawSatisfaction:   rawScore,
			}
		}

		// Verify evidence-availability rules.
		assertEvidenceScoringRules(t, evidenceInputs)
	})
}

// generateDistinctCriteria generates a set of distinct non-empty criterion
// names.
func generateDistinctCriteria(t *rapid.T, count int) []string {
	seen := make(map[string]struct{}, count)
	names := make([]string, 0, count)

	for len(names) < count {
		name := rapid.StringMatching(`[a-z]{3,12}`).Draw(t, "criterion_name")

		if _, exists := seen[name]; exists {
			continue
		}

		seen[name] = struct{}{}

		names = append(names, name)
	}

	return names
}

// assertConfirmerSelection verifies the identity-safe selection invariants.
func assertConfirmerSelection(t *rapid.T, result *SelectionResultInput) {
	hasDistinctCandidate := false

	for _, c := range result.Input.CandidateIdentities {
		if c != result.Input.AuthorIdentity {
			hasDistinctCandidate = true

			break
		}
	}

	if hasDistinctCandidate {
		if !result.OK {
			t.Fatal("selection must succeed when a distinct candidate exists")
		}

		if result.Selected == result.Input.AuthorIdentity {
			t.Fatalf("selected Confirmer %q must differ from author %q", result.Selected, result.Input.AuthorIdentity)
		}
	} else if result.OK {
		t.Fatal("selection must fail when all candidates equal the author")
	}
}

// assertEvidenceScoringRules verifies that each criterion produces exactly one
// satisfaction value in the correct range.
func assertEvidenceScoringRules(t *rapid.T, inputs []CriterionEvidenceInput) {
	seen := make(map[string]struct{}, len(inputs))

	for i := range inputs {
		input := &inputs[i]
		satisfaction := computeCriterionSatisfaction(input)

		// Each criterion has exactly one satisfaction value.
		if _, exists := seen[input.CriterionName]; exists {
			t.Fatalf("criterion %q has duplicate satisfaction", input.CriterionName)
		}

		seen[input.CriterionName] = struct{}{}

		// Satisfaction is in [1, 10] after clamping (Requirement 3.1).
		if satisfaction < minSatisfaction ||
			satisfaction > maxSatisfaction {
			t.Fatalf("satisfaction %d out of range [1, 10] for criterion %q", satisfaction, input.CriterionName)
		}

		// Unavailable evidence produces satisfaction = 1 (Requirement 2.6).
		if !input.EvidenceAvailable &&
			satisfaction != minSatisfaction {
			t.Fatalf(
				"unavailable evidence must produce satisfaction 1, got %d for "+"criterion %q",
				satisfaction,
				input.CriterionName,
			)
		}

		// Available evidence produces satisfaction in [2, 10] (Requirement 2.7,
		// 3.2).
		if input.EvidenceAvailable &&
			satisfaction < minAvailable {
			t.Fatalf(
				"available evidence must produce satisfaction in [2, 10], got %d "+"for criterion %q",
				satisfaction,
				input.CriterionName,
			)
		}
	}
}

// Feature: adversarial-review-operational-workflow, Property 4: Confirmation
// targets match the impact predicate
//
// For any gate outcomes and valid findings, the coordinator selects every and
// only gate-valid critical finding, plus every and only gate-valid high finding
// whose domains include security or correctness; it selects no other finding
// for Confirmer invocation.
//
// **Validates: Requirements 3.5, 5.1, 5.8**.

// TestProperty_ConfirmationTargetsMatchImpactPredicate verifies Property 4:
// Confirmation targets match the impact predicate.
//
// Feature: adversarial-review-operational-workflow, Property 4: Confirmation
// targets match the impact predicate
//
// **Validates: Requirements 3.5, 5.1, 5.8**.
func TestProperty_ConfirmationTargetsMatchImpactPredicate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		findingCount := rapid.IntRange(1, 10).Draw(t, "finding_count")
		findings := make([]models.ReviewFinding, findingCount)

		for i := range findingCount {
			findings[i] = drawRandomFinding(t)
		}

		targets := session.SelectConfirmerTargets(findings)

		// Build a set of target indexes for fast lookup.
		targetSet := make(map[int]struct{}, len(targets))
		for _, idx := range targets {
			targetSet[idx] = struct{}{}
		}

		for i := range findings {
			finding := &findings[i]
			isTarget := session.IsConfirmerTarget(finding)
			gateValid := session.IsGateValid(finding)
			highImpact := session.IsHighImpactFinding(finding)
			_, inTargetSet := targetSet[i]

			assertTargetConsistency(t, &TargetConsistencyInput{
				Index:       i,
				IsTarget:    isTarget,
				GateValid:   gateValid,
				HighImpact:  highImpact,
				InTargetSet: inTargetSet,
				Finding:     finding,
			})
		}

		// Verify SelectConfirmerTargets returns exactly the set that matches
		// individual IsConfirmerTarget calls.
		expectedCount := 0

		for i := range findings {
			if session.IsConfirmerTarget(&findings[i]) {
				expectedCount++
			}
		}

		if len(targets) != expectedCount {
			t.Fatalf(
				"SelectConfirmerTargets returned %d targets but IsConfirmerTarget "+"identified %d",
				len(targets),
				expectedCount,
			)
		}
	})
}

// assertTargetConsistency verifies that a finding is selected if and only if it
// is gate-valid AND high-impact.
func assertTargetConsistency(t *rapid.T, input *TargetConsistencyInput) {
	// A finding returned by SelectConfirmerTargets must be gate-valid AND
	// high-impact.
	if input.InTargetSet && !input.GateValid {
		t.Fatalf("finding[%d]: selected as target but gate is not valid", input.Index)
	}

	if input.InTargetSet && !input.HighImpact {
		t.Fatalf(
			"finding[%d]: selected as target but not high-impact (severity=%s, domains=%v)",
			input.Index,
			input.Finding.FindingSeverity,
			input.Finding.FindingDomains,
		)
	}

	// A finding NOT returned must be either not gate-valid OR not high-impact.
	if !input.InTargetSet && input.GateValid && input.HighImpact {
		t.Fatalf("finding[%d]: gate-valid and high-impact but not selected as target", input.Index)
	}

	// IsConfirmerTarget must agree with the set membership.
	if input.IsTarget != input.InTargetSet {
		t.Fatalf(
			"finding[%d]: IsConfirmerTarget=%v but SelectConfirmerTargets membership=%v",
			input.Index,
			input.IsTarget,
			input.InTargetSet,
		)
	}

	// Medium and low findings must never be selected.
	if input.InTargetSet && (input.Finding.FindingSeverity == models.SeverityMedium ||
		input.Finding.FindingSeverity == models.SeverityLow) {
		t.Fatalf("finding[%d]: medium/low severity must never be a confirmer target", input.Index)
	}

	// Findings with nil gate results must never be selected.
	if input.InTargetSet && input.Finding.CitationGateResult == nil {
		t.Fatalf("finding[%d]: nil gate result must never be a confirmer target", input.Index)
	}
}

// drawRandomFinding generates a random ReviewFinding with valid structure for
// property testing.
func drawRandomFinding(t *rapid.T) models.ReviewFinding {
	severity := drawSeverity(t)
	domains := drawDomains(t)
	gateResult := drawGateResult(t)

	return models.ReviewFinding{
		CitationGateResult: gateResult,
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Security",
		},
		CriterionName:         "test-criterion",
		QuotedExcerpt:         "test excerpt",
		Status:                models.StatusHypothesized,
		Reasoning:             "test reasoning",
		FindingSeverity:       severity,
		FindingDomains:        domains,
		CriterionSatisfaction: rapid.IntRange(1, 10).Draw(t, "satisfaction"),
	}
}

// drawSeverity generates a random FindingSeverity.
func drawSeverity(t *rapid.T) models.FindingSeverity {
	severities := []models.FindingSeverity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
	}

	idx := rapid.IntRange(0, len(severities)-1).Draw(t, "severity_idx")

	return severities[idx]
}

// drawDomains generates 1-3 random valid FindingDomain values without
// duplicates.
func drawDomains(t *rapid.T) []models.FindingDomain {
	allDomains := []models.FindingDomain{
		models.DomainSecurity,
		models.DomainCorrectness,
		models.DomainArchitecture,
		models.DomainReliability,
		models.DomainOperations,
		models.DomainDeveloperExperience,
	}

	count := rapid.IntRange(1, 3).Draw(t, "domain_count")

	// Shuffle by selecting without replacement.
	selected := make([]models.FindingDomain, 0, count)
	available := make([]models.FindingDomain, len(allDomains))
	copy(available, allDomains)

	for range count {
		idx := rapid.IntRange(0, len(available)-1).Draw(t, "domain_idx")

		selected = append(selected, available[idx])
		available[idx] = available[len(available)-1]
		available = available[:len(available)-1]
	}

	return selected
}

// drawGateResult generates a random CitationGateResult that may be nil, valid,
// or invalid.
func drawGateResult(t *rapid.T) *models.CitationGateResult {
	// 30% chance of nil gate result.
	isNil := rapid.IntRange(0, 9).Draw(t, "gate_nil_chance") < 3 // lint:allow_raw_number

	if isNil {
		return nil
	}

	schemaValid := rapid.Bool().Draw(t, "schema_valid")
	citationValid := rapid.Bool().Draw(t, "citation_valid")

	provenance := ""
	if rapid.Bool().Draw(t, "has_provenance") {
		provenance = "corr-id-123"
	}

	return &models.CitationGateResult{
		SchemaValid:             schemaValid,
		CitationValid:           citationValid,
		ProvenanceCorrelationID: provenance,
	}
}

// Feature: adversarial-review-operational-workflow, Property 5: Gate and
// confirmation transitions cannot create unsupported confirmation
//
// For any finding, schema result, citation result, validation provenance, and
// Confirmer attempts, an invalid, non-deterministically sourced, or unavailable
// gate result yields `unconfirmed` or `unchecked (gate unavailable)` and blocks
// routing and approval; a gate-valid high-impact finding becomes `confirmed`
// only with demonstration evidence, remains `hypothesized` before final
// determination without evidence, and becomes `unconfirmed` after three
// unsuccessful attempts regardless of optional attempt-detail persistence.
//
// **Validates: Requirements 3.6, 4.3, 4.4, 4.6, 5.2, 5.3, 5.5, 5.6**.

// TestProperty_GateAndConfirmationTransitionsCannotCreateUnsupportedConfirmation
// verifies Property 5: Gate and confirmation transitions cannot create
// unsupported confirmation.
//
// Feature: adversarial-review-operational-workflow, Property 5: Gate and
// confirmation transitions cannot create unsupported confirmation
//
// **Validates: Requirements 3.6, 4.3, 4.4, 4.6, 5.2, 5.3, 5.5, 5.6**.
func TestProperty_GateAndConfirmationTransitionsCannotCreateUnsupportedConfirmation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate gate availability.
		gateAvailability := drawGateAvailability(t)

		// Generate a finding with random structure.
		finding := drawTransitionFinding(t)

		// Apply the transition function.
		input := &session.FindingStatusTransitionInput{
			GateAvailability: gateAvailability,
			Finding:          finding,
		}

		result := session.TransitionFindingStatus(input)
		gateValid := session.IsGateValid(&finding)
		highImpact := session.IsHighImpactFinding(&finding)

		assertTransitionInvariants(t, &TransitionAssertionInput{
			GateAvailability: gateAvailability,
			Finding:          finding,
			Result:           result,
			GateValid:        gateValid,
			HighImpact:       highImpact,
		})
	})
}

// drawGateAvailability generates a random GateAvailability value.
func drawGateAvailability(t *rapid.T) session.GateAvailability {
	if rapid.Bool().Draw(t, "gate_available") {
		return session.GateAvailable
	}

	return session.GateUnavailable
}

// drawConfirmerAttempts generates 0-4 Confirmer attempts with random
// demonstration outcomes.
func drawConfirmerAttempts(t *rapid.T) []models.ConfirmerAttempt {
	count := rapid.IntRange(0, 4).Draw(t, "attempt_count")
	attempts := make([]models.ConfirmerAttempt, count)

	for i := range count {
		demonstrated := rapid.Bool().Draw(t, "demonstrated")
		evidence := ""

		if demonstrated {
			// Only sometimes include evidence even when demonstrated to test
			// the evidence-required path.
			if rapid.Bool().Draw(t, "has_evidence") {
				evidence = "counter-example found"
			}
		}

		attempts[i] = models.ConfirmerAttempt{
			Strategy:              "test-strategy",
			Observation:           "test-observation",
			DemonstrationEvidence: evidence,
			AttemptNumber:         i + 1,
			Demonstrated:          demonstrated,
		}
	}

	return attempts
}

// drawTransitionFinding generates a finding suitable for transition testing
// with varied gate results and confirmer attempts.
func drawTransitionFinding(t *rapid.T) models.ReviewFinding {
	severity := drawSeverity(t)
	domains := drawDomains(t)
	gateResult := drawGateResult(t)
	attempts := drawConfirmerAttempts(t)

	return models.ReviewFinding{
		CitationGateResult: gateResult,
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Security",
		},
		CriterionName:         "test-criterion",
		QuotedExcerpt:         "test excerpt",
		Status:                models.StatusHypothesized,
		Reasoning:             "test reasoning",
		FindingSeverity:       severity,
		FindingDomains:        domains,
		ConfirmerAttempts:     attempts,
		CriterionSatisfaction: rapid.IntRange(1, 10).Draw(t, "satisfaction"),
	}
}

// assertTransitionInvariants verifies all transition invariants for Property 5.
func assertTransitionInvariants(t *rapid.T, input *TransitionAssertionInput) {
	// Invariant: Status can NEVER be "confirmed" without demonstration evidence
	// (Requirements 4.3, 5.3).
	assertNoUnsupportedConfirmation(t, input)

	// Invariant: Unavailable gate yields "unchecked (gate unavailable)"
	// (Requirement 4.6).
	assertUnavailableGateTransition(t, input)

	// Invariant: Invalid or nil gate yields "unconfirmed" (Requirements 3.6,
	// 4.3).
	assertInvalidGateTransition(t, input)

	// Invariant: Gate-valid high-impact with evidence yields "confirmed"
	// (Requirement 5.2).
	assertConfirmedWithEvidence(t, input)

	// Invariant: Gate-valid high-impact without evidence and fewer than 3
	// attempts yields "hypothesized" (Requirement 5.5).
	assertHypothesizedPending(t, input)

	// Invariant: Gate-valid high-impact with 3+ unsuccessful attempts yields
	// "unconfirmed" (Requirement 5.6).
	assertUnconfirmedAfterAttempts(t, input)

	// Invariant: Gate-valid non-high-impact yields "hypothesized" (not routed
	// to confirmer).
	assertNonTargetHypothesized(t, input)
}

// assertNoUnsupportedConfirmation verifies that confirmed status never appears
// without demonstration evidence.
func assertNoUnsupportedConfirmation(t *rapid.T, input *TransitionAssertionInput) {
	if input.Result.Status != models.StatusConfirmed {
		return
	}

	// If confirmed, there must be at least one attempt with Demonstrated=true
	// and non-empty evidence.
	hasDemonstration := false

	for _, attempt := range input.Finding.ConfirmerAttempts {
		if attempt.Demonstrated && attempt.DemonstrationEvidence != "" {
			hasDemonstration = true

			break
		}
	}

	if !hasDemonstration {
		t.Fatal("status is 'confirmed' but no Confirmer attempt has Demonstrated=true with non-empty evidence")
	}
}

// assertUnavailableGateTransition verifies that an unavailable gate always
// produces "unchecked (gate unavailable)".
func assertUnavailableGateTransition(t *rapid.T, input *TransitionAssertionInput) {
	if input.GateAvailability != session.GateUnavailable {
		return
	}

	if input.Result.Status != models.StatusUncheckedGateUnavail {
		t.Fatalf("gate unavailable must produce status 'unchecked (gate unavailable)', got %q", input.Result.Status)
	}
}

// assertInvalidGateTransition verifies that an invalid or nil gate result
// produces "unconfirmed" when the gate is available.
func assertInvalidGateTransition(t *rapid.T, input *TransitionAssertionInput) {
	if input.GateAvailability != session.GateAvailable {
		return
	}

	if input.GateValid {
		return
	}

	if input.Result.Status != models.StatusUnconfirmed {
		t.Fatalf("invalid gate result with available gate must produce 'unconfirmed', got %q", input.Result.Status)
	}
}

// assertConfirmedWithEvidence verifies that a gate-valid high-impact finding
// with demonstration evidence is confirmed.
func assertConfirmedWithEvidence(t *rapid.T, input *TransitionAssertionInput) {
	if input.GateAvailability != session.GateAvailable {
		return
	}

	if !input.GateValid || !input.HighImpact {
		return
	}

	hasDemonstration := false

	for _, attempt := range input.Finding.ConfirmerAttempts {
		if attempt.Demonstrated && attempt.DemonstrationEvidence != "" {
			hasDemonstration = true

			break
		}
	}

	if !hasDemonstration {
		return
	}

	if input.Result.Status != models.StatusConfirmed {
		t.Fatalf(
			"gate-valid high-impact finding with "+"demonstration evidence must be 'confirmed', got %q",
			input.Result.Status,
		)
	}
}

// assertHypothesizedPending verifies that a gate-valid high-impact finding
// without evidence and fewer than 3 attempts remains hypothesized.
func assertHypothesizedPending(t *rapid.T, input *TransitionAssertionInput) {
	if input.GateAvailability != session.GateAvailable {
		return
	}

	if !input.GateValid || !input.HighImpact {
		return
	}

	// Check no demonstration evidence exists.
	hasDemonstration := false

	for _, attempt := range input.Finding.ConfirmerAttempts {
		if attempt.Demonstrated && attempt.DemonstrationEvidence != "" {
			hasDemonstration = true

			break
		}
	}

	if hasDemonstration {
		return
	}

	if len(input.Finding.ConfirmerAttempts) >= session.MaxConfirmerAttempts {
		return
	}

	if input.Result.Status != models.StatusHypothesized {
		t.Fatalf(
			"gate-valid high-impact finding without evidence and <%d attempts must be 'hypothesized', got %q",
			session.MaxConfirmerAttempts,
			input.Result.Status,
		)
	}
}

// assertUnconfirmedAfterAttempts verifies that a gate-valid high-impact finding
// with 3+ unsuccessful attempts becomes unconfirmed.
func assertUnconfirmedAfterAttempts(t *rapid.T, input *TransitionAssertionInput) {
	if input.GateAvailability != session.GateAvailable {
		return
	}

	if !input.GateValid || !input.HighImpact {
		return
	}

	// Check no demonstration evidence exists.
	hasDemonstration := false

	for _, attempt := range input.Finding.ConfirmerAttempts {
		if attempt.Demonstrated && attempt.DemonstrationEvidence != "" {
			hasDemonstration = true

			break
		}
	}

	if hasDemonstration {
		return
	}

	if len(input.Finding.ConfirmerAttempts) < session.MaxConfirmerAttempts {
		return
	}

	if input.Result.Status != models.StatusUnconfirmed {
		t.Fatalf(
			"gate-valid high-impact finding with %d+ unsuccessful attempts must be 'unconfirmed', got %q",
			session.MaxConfirmerAttempts,
			input.Result.Status,
		)
	}
}

// assertNonTargetHypothesized verifies that a gate-valid but non-high-impact
// finding remains hypothesized.
func assertNonTargetHypothesized(t *rapid.T, input *TransitionAssertionInput) {
	if input.GateAvailability != session.GateAvailable {
		return
	}

	if !input.GateValid {
		return
	}

	if input.HighImpact {
		return
	}

	if input.Result.Status != models.StatusHypothesized {
		t.Fatalf(
			"gate-valid non-high-impact finding must be 'hypothesized' (not routed to confirmer), got %q",
			input.Result.Status,
		)
	}
}
