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
	SuggestDeckImprovements(deckID *int, decklistID *int, maxResults int) (string, error)
	GetPacksAndCycles() (string, error)
}
