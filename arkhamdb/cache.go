package arkhamdb

import (
	"sync"
	"time"
)

const cacheTTL = 24 * time.Hour

// timeNow is a variable so tests can override it
var timeNow = time.Now

type cardsCacheStore struct {
	mu       sync.RWMutex
	data     []map[string]interface{}
	cachedAt time.Time
}

type packsCacheStore struct {
	mu       sync.RWMutex
	data     []map[string]interface{}
	cachedAt time.Time
}

var cardsCache cardsCacheStore
var packsCache packsCacheStore
