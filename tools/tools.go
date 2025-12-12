package tools

// ArkhamDBTool is the interface for the ArkhamDB tools
// It defines the methods that can be used to interact with the ArkhamDB API.
type ArkhamDBTool interface {
	GetCard(cardCode string) (string, error)
	SearchCardsByName(name string) (string, error)
	GetDeck(deckID int) (string, error)
	GetDecklist(decklistID int) (string, error)
	FindCardSynergies(cardCode string, maxResults int) (string, error)
}
