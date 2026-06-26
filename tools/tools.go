package tools

// ArkhamDBTool is the interface for the ArkhamDB tools
// It defines the methods that can be used to interact with the ArkhamDB API.
type ArkhamDBTool interface {
	GetCard(cardCode string) (string, error)
	SearchCardsByName(name string) (string, error)
	SearchCardsAdvanced(chapter int, cycleCode string, factionCode string, typeCode string, xpMin int, xpMax int, costMin int, costMax int, traits []string, tags []string, maxResults int) (string, error)
	GetDeck(deckID int) (string, error)
	GetDecklist(decklistID int) (string, error)
	FindCardSynergies(cardCode string, maxResults int) (string, error)
	SuggestDeckImprovements(deckID *int, decklistID *int, maxResults int, strategy string) (string, error)
	GetPacksAndCycles() (string, error)
	GetInvestigatorConstraints(investigatorCode string) (string, error)
	BuildStarterDeck(investigatorCode string, chapter int, cycleCodes []string, xpBudget int, strategy string) (string, error)
	SearchReferenceDecks(investigatorCode string, xpMin int, xpMax int, tags string, daysBack int, maxResults int) (string, error)
	GetUpgradePath(deckID *int, decklistID *int, xpBudget int) (string, error)
	ValidateDeck(deckID *int, decklistID *int) (string, error)
	GetCollection() (string, error)
	SetCollection(ownedCycles []string, language string, useTaboo bool) (string, error)
	AdaptDeckToCollection(decklistID int) (string, error)
}
