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

package novelty

import "go.nwlabs.dev/magneto/internal/models"

const (
	// ReasonNewCriterion indicates the finding references a criterion not
	// present in any prior round.
	ReasonNewCriterion = "new criterion not seen in prior rounds"

	// ReasonNewEvidence indicates the finding presents evidence not seen in any
	// prior finding for the same criterion.
	ReasonNewEvidence = "new evidence for existing criterion"

	// ReasonNewSatisfactionWithEvidence indicates the finding differs in
	// satisfaction and includes new evidence.
	ReasonNewSatisfactionWithEvidence = "different satisfaction with new evidence"
)

type (
	// CheckInput contains the parameters for a novelty check.
	CheckInput struct {
		CurrentFindings []models.ReviewFinding
		PriorFindings   []models.ReviewFinding
	}

	// CheckResult contains the outcome of a novelty check.
	CheckResult struct {
		NovelItems []NovelItem
		Novel      bool
	}

	// NovelItem describes a single novel finding with a reason.
	NovelItem struct {
		CriterionName string
		Reason        string
	}

	// priorEntry aggregates evidence and scores seen for a single
	// criterion across all prior rounds.
	priorEntry struct {
		excerpts map[string]struct{}
		scores   map[int]struct{}
	}
)

// Check compares current round findings against prior round findings to
// determine if the current round surfaces new failure modes. A round is
// non-novel if every finding references a criterion and evidence already
// present in prior rounds. New criterion, new evidence, or different score with
// new evidence counts as novel.
func Check(input *CheckInput) *CheckResult {
	result := &CheckResult{}

	priorByCriterion := buildPriorIndex(input.PriorFindings)

	for i := range input.CurrentFindings {
		current := &input.CurrentFindings[i]
		reason := classifyFinding(current, priorByCriterion)

		if reason != "" {
			result.Novel = true
			result.NovelItems = append(result.NovelItems, NovelItem{
				CriterionName: current.CriterionName,
				Reason:        reason,
			})
		}
	}

	return result
}

func buildPriorIndex(findings []models.ReviewFinding) map[string]*priorEntry {
	index := make(map[string]*priorEntry, len(findings))

	for i := range findings {
		f := &findings[i]
		entry, exists := index[f.CriterionName]

		if !exists {
			entry = &priorEntry{
				excerpts: make(map[string]struct{}),
				scores:   make(map[int]struct{}),
			}
			index[f.CriterionName] = entry
		}

		entry.excerpts[f.QuotedExcerpt] = struct{}{}
		entry.scores[f.CriterionSatisfaction] = struct{}{}
	}

	return index
}

func classifyFinding(current *models.ReviewFinding, priorIndex map[string]*priorEntry) string {
	entry, exists := priorIndex[current.CriterionName]
	if !exists {
		return ReasonNewCriterion
	}

	_, evidenceSeen := entry.excerpts[current.QuotedExcerpt]
	_, scoreSeen := entry.scores[current.CriterionSatisfaction]

	if !evidenceSeen && !scoreSeen {
		return ReasonNewSatisfactionWithEvidence
	}

	if !evidenceSeen {
		return ReasonNewEvidence
	}

	return ""
}
