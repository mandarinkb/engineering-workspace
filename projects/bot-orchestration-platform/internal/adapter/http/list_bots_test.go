package http_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestHandleListBots ทดสอบว่า GET /bots คืนชื่อ + config schema ของ bot ที่ register ไว้
func TestHandleListBots(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/bots")
	if err != nil {
		t.Fatalf("GET /bots: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var bots []struct {
		Name         string `json:"name"`
		ConfigSchema []struct {
			Key      string `json:"key"`
			Required bool   `json:"required"`
		} `json:"config_schema"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bots); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(bots) != 1 || bots[0].Name != "stub-bot" {
		t.Fatalf("expected exactly [stub-bot], got %+v", bots)
	}
	if len(bots[0].ConfigSchema) != 1 || bots[0].ConfigSchema[0].Key != "url" || !bots[0].ConfigSchema[0].Required {
		t.Errorf("expected config schema with required \"url\" field, got %+v", bots[0].ConfigSchema)
	}
}
