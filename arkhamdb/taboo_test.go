package arkhamdb

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTabooListParsing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Two versions; only the last (most recent) should be used.
		w.Write([]byte(`[
			{"cards": []},
			{"cards": [{"code":"01001","deck_limit":0},{"code":"01002","xp":1}]}
		]`))
	}))
	defer ts.Close()

	c := NewArkhamDBClient(ts.URL)
	list, err := c.fetchTabooList()
	if err != nil {
		t.Fatalf("fetchTabooList: %v", err)
	}

	e, ok := list["01001"]
	if !ok || !e.Banned {
		t.Errorf("01001 should be banned, got %+v", e)
	}

	e, ok = list["01002"]
	if !ok || e.XPCost != 1 {
		t.Errorf("01002 should have XPCost=1, got %+v", e)
	}

	// First version's empty cards should not appear
	if len(list) != 2 {
		t.Errorf("expected 2 entries, got %d", len(list))
	}
}

func TestFetchTabooListEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	c := NewArkhamDBClient(ts.URL)
	list, err := c.fetchTabooList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil for empty versions, got %v", list)
	}
}
