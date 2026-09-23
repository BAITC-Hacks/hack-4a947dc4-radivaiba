package llm

import (
	"careerquest/internal/engine"
	"careerquest/internal/testfixture"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidatedSelectionAndFallbacks(t *testing.T) {
	d := testfixture.Dataset()
	candidates := engine.RankCandidates(d, d.Employees[0])
	id := candidates[0].Event.ID
	valid := map[string]any{"recommendations": []map[string]any{{"event_id": id, "explanation": "Закройте критический разрыв, учитывая историю участия.", "evidence_ids": []string{id + ":goal", id + ":gap", id + ":history"}}}}
	validBytes, _ := json.Marshal(valid)
	for _, tc := range []struct {
		name, content string
		delay         time.Duration
		wantMode      string
	}{{"valid", string(validBytes), 0, "llm"}, {"bad-json", "not json", 0, "rules"}, {"invented-id", `{"recommendations":[{"event_id":"FAKE","explanation":"x","evidence_ids":[]}]}`, 0, "rules"}, {"missing-factors", `{"recommendations":[{"event_id":"` + id + `","explanation":"x","evidence_ids":["` + id + `:gap"]}]}`, 0, "rules"}, {"timeout", string(validBytes), 100 * time.Millisecond, "rules"}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test" {
					t.Error("wrong API request")
				}
				if tc.delay > 0 {
					time.Sleep(tc.delay)
				}
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": tc.content}}}})
			}))
			defer server.Close()
			c := &Client{BaseURL: server.URL, APIKey: "test", Model: "test", Timeout: 30 * time.Millisecond}
			r := c.Recommend(context.Background(), candidates, 1)
			if r.Mode != tc.wantMode || len(r.Items) == 0 {
				t.Fatalf("got %+v", r)
			}
		})
	}
	var disabled *Client
	if r := disabled.Recommend(context.Background(), candidates, 1); r.Mode != "rules" || r.Notice == "" {
		t.Fatal("missing key must be explicit")
	}
}
