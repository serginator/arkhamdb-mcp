package arkhamdb

import (
	"fmt"
	"regexp"
	"strings"
)

// SynergyScore represents a synergy match with scoring
type SynergyScore struct {
	Card    map[string]interface{}
	Score   int
	Reasons []string
}

// extractTraitReferences extracts trait references from card text (e.g., [[Item]], [[Spell]])
func extractTraitReferences(text string) []string {
	if text == "" {
		return nil
	}

	// Pattern to match [[Trait]] references
	pattern := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	traits := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			trait := strings.TrimSpace(match[1])
			traitLower := strings.ToLower(trait)
			if !seen[traitLower] {
				traits = append(traits, traitLower)
				seen[traitLower] = true
			}
		}
	}

	return traits
}

// extractMechanicKeywords extracts mechanic keywords from card text
func extractMechanicKeywords(text string) []string {
	if text == "" {
		return nil
	}

	textLower := strings.ToLower(text)
	keywords := []string{}

	// Define keyword patterns and their normalized names
	keywordPatterns := map[string]string{
		"discard pile":      "discard",
		"from your discard": "discard",
		"discard":           "discard",
		"investigate":       "investigation",
		"investigation":     "investigation",
		"clue":              "clue",
		"clues":             "clue",
		"discover.*clue":    "clue",
		"damage":            "damage",
		"deal damage":       "damage",
		"take damage":       "damage",
		"resource":          "resource",
		"resources":         "resource",
		"spend":             "resource",
		"gain.*resource":    "resource",
		"draw":              "draw",
		"draw cards":        "draw",
		"move":              "movement",
		"location":          "movement",
		"at your location":  "movement",
		"skill test":        "skill_test",
		"test":              "skill_test",
		"evidence":          "evidence",
	}

	seen := make(map[string]bool)

	// Check for each keyword pattern
	for pattern, normalized := range keywordPatterns {
		matched, _ := regexp.MatchString(pattern, textLower)
		if matched && !seen[normalized] {
			keywords = append(keywords, normalized)
			seen[normalized] = true
		}
	}

	return keywords
}

// getCardText returns the card text, preferring real_text (English) over text (localized)
func getCardText(card map[string]interface{}) string {
	if realText, ok := card["real_text"].(string); ok && realText != "" {
		return realText
	}
	if text, ok := card["text"].(string); ok && text != "" {
		return text
	}
	return ""
}

// getCardTraits returns the card traits, preferring real_traits (English) over traits (localized)
func getCardTraits(card map[string]interface{}) string {
	if realTraits, ok := card["real_traits"].(string); ok && realTraits != "" {
		return realTraits
	}
	if traits, ok := card["traits"].(string); ok && traits != "" {
		return traits
	}
	return ""
}

// getCardName returns the card name, preferring real_name (English) over name (localized)
func getCardName(card map[string]interface{}) string {
	if realName, ok := card["real_name"].(string); ok && realName != "" {
		return realName
	}
	if name, ok := card["name"].(string); ok && name != "" {
		return name
	}
	return ""
}

// normalizeTrait normalizes a trait string for comparison
func normalizeTrait(trait string) string {
	return strings.ToLower(strings.TrimSpace(trait))
}

// hasTrait checks if a card has a specific trait
func hasTrait(card map[string]interface{}, targetTrait string) bool {
	traitsStr := getCardTraits(card)
	if traitsStr == "" {
		return false
	}

	targetTraitLower := normalizeTrait(targetTrait)
	traitsLower := strings.ToLower(traitsStr)

	// Split traits by common delimiters (. and ,)
	traitParts := regexp.MustCompile(`[.,\s]+`).Split(traitsLower, -1)

	for _, part := range traitParts {
		if normalizeTrait(part) == targetTraitLower {
			return true
		}
	}

	return false
}

// scoreSynergy calculates a synergy score between target card and candidate card
func scoreSynergy(targetCard, candidateCard map[string]interface{}) (int, []string) {
	score := 0
	reasons := []string{}

	targetText := getCardText(targetCard)
	candidateText := getCardText(candidateCard)
	targetTraits := getCardTraits(targetCard)
	candidateTraits := getCardTraits(candidateCard)

	// Extract trait references from target card
	traitRefs := extractTraitReferences(targetText)

	// Check if candidate card matches referenced traits (high score)
	for _, refTrait := range traitRefs {
		if hasTrait(candidateCard, refTrait) {
			score += 50
			candidateName := getCardName(candidateCard)
			reasons = append(reasons, fmt.Sprintf("Target card references [[%s]], candidate '%s' has this trait", refTrait, candidateName))
		}
	}

	// Extract trait references from candidate card and check if target matches
	candidateTraitRefs := extractTraitReferences(candidateText)
	for _, refTrait := range candidateTraitRefs {
		if hasTrait(targetCard, refTrait) {
			score += 40
			candidateName := getCardName(candidateCard)
			reasons = append(reasons, fmt.Sprintf("Candidate '%s' references [[%s]], target card has this trait", candidateName, refTrait))
		}
	}

	// Check for shared traits (medium score)
	if targetTraits != "" && candidateTraits != "" {
		targetTraitParts := regexp.MustCompile(`[.,\s]+`).Split(strings.ToLower(targetTraits), -1)
		candidateTraitParts := regexp.MustCompile(`[.,\s]+`).Split(strings.ToLower(candidateTraits), -1)

		for _, targetTrait := range targetTraitParts {
			targetTraitNorm := normalizeTrait(targetTrait)
			if targetTraitNorm == "" {
				continue
			}
			for _, candidateTrait := range candidateTraitParts {
				candidateTraitNorm := normalizeTrait(candidateTrait)
				if targetTraitNorm == candidateTraitNorm {
					score += 20
					reasons = append(reasons, fmt.Sprintf("Both cards share the '%s' trait", targetTraitNorm))
					break
				}
			}
		}
	}

	// Check for matching mechanic keywords (lower score)
	targetKeywords := extractMechanicKeywords(targetText)
	candidateKeywords := extractMechanicKeywords(candidateText)

	keywordMatches := 0
	for _, targetKw := range targetKeywords {
		for _, candidateKw := range candidateKeywords {
			if targetKw == candidateKw {
				keywordMatches++
				break
			}
		}
	}

	if keywordMatches > 0 {
		score += keywordMatches * 10
		reasons = append(reasons, fmt.Sprintf("Both cards interact with similar mechanics (%d matching keywords)", keywordMatches))
	}

	// Check for slot compatibility
	targetSlot, targetHasSlot := targetCard["real_slot"].(string)
	if !targetHasSlot {
		targetSlot, targetHasSlot = targetCard["slot"].(string)
	}
	candidateSlot, candidateHasSlot := candidateCard["real_slot"].(string)
	if !candidateHasSlot {
		candidateSlot, candidateHasSlot = candidateCard["slot"].(string)
	}

	if targetHasSlot && candidateHasSlot && targetSlot != "" && candidateSlot != "" {
		if strings.ToLower(targetSlot) == strings.ToLower(candidateSlot) {
			score += 15
			reasons = append(reasons, fmt.Sprintf("Both cards use the same slot: %s", targetSlot))
		}
	}

	// Avoid self-matches
	targetCode, targetHasCode := targetCard["code"].(string)
	candidateCode, candidateHasCode := candidateCard["code"].(string)
	if targetHasCode && candidateHasCode && targetCode == candidateCode {
		score = 0
		reasons = []string{"Same card"}
	}

	return score, reasons
}
