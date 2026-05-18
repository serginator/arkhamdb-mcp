package arkhamdb

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// LevelRange represents an XP level range in a deck option
type LevelRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// DeckOption represents one rule in an investigator's deck_options array
type DeckOption struct {
	Faction []string    `json:"faction"`
	Level   *LevelRange `json:"level"`
	Limit   int         `json:"limit"`
	Error   string      `json:"error"`
	Tag     []string    `json:"tag"`
	Text    string      `json:"text"`
	Type    []string    `json:"type"`
	Trait   []string    `json:"trait"`
	Not     bool        `json:"not"`
	Size    int         `json:"size"`
}

// DeckRequirements represents the deck_requirements field on an investigator card
type DeckRequirements struct {
	Size   int                          `json:"size"`
	Card   map[string]map[string]string `json:"card"`
	Random []map[string]interface{}     `json:"random"`
}

// parseDeckOptions parses the deck_options interface{} value from a card map
func parseDeckOptions(raw interface{}) []DeckOption {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var opts []DeckOption
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil
	}
	return opts
}

// parseDeckRequirements parses the deck_requirements interface{} value from a card map
func parseDeckRequirements(raw interface{}) *DeckRequirements {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var req DeckRequirements
	if err := json.Unmarshal(data, &req); err != nil {
		return nil
	}
	return &req
}

// isCardAllowedByOptions returns true if the card satisfies at least one DeckOption
func isCardAllowedByOptions(card map[string]interface{}, options []DeckOption) bool {
	if len(options) == 0 {
		return false
	}
	cardFaction, _ := card["faction_code"].(string)
	xpVal, _ := card["xp"].(float64)
	cardXP := int(xpVal)
	cardType, _ := card["type_code"].(string)
	cardText := getCardText(card)

	// Pre-compile text regexes to avoid recompiling per card
	textRegexes := make(map[string]*regexp.Regexp)
	for _, opt := range options {
		if opt.Text != "" {
			if _, ok := textRegexes[opt.Text]; !ok {
				if re, err := regexp.Compile("(?i)" + opt.Text); err == nil {
					textRegexes[opt.Text] = re
				}
			}
		}
	}

	// Note: DeckOption.Limit is a count-based constraint (e.g. "max 5 survivor cards in deck"),
	// not a card eligibility filter. It is enforced during deck construction in BuildStarterDeck
	// and ValidateDeck via per-option counting, not here.
	for _, opt := range options {
		if opt.Size > 0 && len(opt.Faction) == 0 && opt.Level == nil {
			continue // size-only option
		}
		if cardMatchesDeckOption(card, opt, cardFaction, cardXP, cardType, cardText, textRegexes) {
			return true
		}
	}
	return false
}

// cardMatchesDeckOption returns true if all non-empty constraints in opt are satisfied
func cardMatchesDeckOption(card map[string]interface{}, opt DeckOption, faction string, xp int, cardType string, text string, textRegexes map[string]*regexp.Regexp) bool {
	// Faction check
	if len(opt.Faction) > 0 {
		factionMatch := false
		for _, f := range opt.Faction {
			if strings.EqualFold(f, faction) {
				factionMatch = true
				break
			}
		}
		if opt.Not {
			if factionMatch {
				return false
			}
		} else {
			if !factionMatch {
				return false
			}
		}
	}

	// Level check
	if opt.Level != nil {
		if xp < opt.Level.Min || xp > opt.Level.Max {
			return false
		}
	}

	// Type check
	if len(opt.Type) > 0 {
		typeMatch := false
		for _, t := range opt.Type {
			if strings.EqualFold(t, cardType) {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	// Trait check (card must have at least one of the listed traits)
	if len(opt.Trait) > 0 {
		traitMatch := false
		for _, t := range opt.Trait {
			if hasTrait(card, t) {
				traitMatch = true
				break
			}
		}
		if !traitMatch {
			return false
		}
	}

	// Tag check (e.g., "hd" = heals damage, "hh" = heals horror)
	if len(opt.Tag) > 0 {
		tagVal, _ := card["tags"].(string)
		tagMatch := false
		for _, tag := range opt.Tag {
			if strings.Contains(tagVal, tag) {
				tagMatch = true
				break
			}
		}
		if !tagMatch {
			return false
		}
	}

	// Text regex check — use pre-compiled regex if available
	if opt.Text != "" {
		if re, ok := textRegexes[opt.Text]; ok {
			if !re.MatchString(text) {
				return false
			}
		} else {
			matched, _ := regexp.MatchString("(?i)"+opt.Text, text)
			if !matched {
				return false
			}
		}
	}

	return true
}

// deckOptionsDescription produces a human-readable description of a DeckOption
func deckOptionsDescription(opt DeckOption) string {
	if opt.Size > 0 {
		return fmt.Sprintf("Deck size: %d", opt.Size)
	}
	parts := []string{}
	if len(opt.Faction) > 0 {
		prefix := ""
		if opt.Not {
			prefix = "NOT "
		}
		parts = append(parts, prefix+strings.Join(opt.Faction, "/"))
	}
	if len(opt.Trait) > 0 {
		parts = append(parts, "trait:"+strings.Join(opt.Trait, "/"))
	}
	if len(opt.Tag) > 0 {
		parts = append(parts, "tag:"+strings.Join(opt.Tag, "/"))
	}
	if len(opt.Type) > 0 {
		parts = append(parts, "type:"+strings.Join(opt.Type, "/"))
	}
	if opt.Level != nil {
		parts = append(parts, fmt.Sprintf("level %d-%d", opt.Level.Min, opt.Level.Max))
	}
	if opt.Limit > 0 {
		parts = append(parts, fmt.Sprintf("max %d cards", opt.Limit))
	}
	if len(parts) == 0 {
		return "any card"
	}
	return strings.Join(parts, ", ")
}
