package llm

import (
	"careerquest/internal/engine"
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAssistantActionsFallbackInThreeLanguages(t *testing.T) {
	d := testfixture.Dataset()
	var client *Client
	firstSteps := map[string]string{}
	for _, locale := range []string{"ru", "kk", "en"} {
		for _, action := range []string{"why", "fifteen_minutes", "simplify", "alternative", "start"} {
			before := engine.EffectiveSkills(d, d.Employees[0])
			result, err := client.Assist(context.Background(), d, d.Employees[0], model.AssistantRequest{EventID: "design-course", Action: action, Locale: locale})
			if err != nil || result.Mode != "rules" || result.Benefit == "" || result.Application == "" || result.FirstStep == "" || len(result.Evidence) != 3 {
				t.Fatalf("%s/%s: %+v %v", locale, action, result, err)
			}
			if action == "fifteen_minutes" {
				if !strings.Contains(result.FirstStep, "15") {
					t.Fatal("missing practical timebox")
				}
				firstSteps[locale] = result.FirstStep
			}
			if action == "alternative" && result.AlternativeEventID != "EV_036" {
				t.Fatal("alternative must be another eligible event")
			}
			if engine.EffectiveSkills(d, d.Employees[0])["design"] != before["design"] || len(d.Workflow.Ledger) != 0 {
				t.Fatal("helper must not award skill or EXP")
			}
		}
	}
	if firstSteps["ru"] == firstSteps["kk"] || firstSteps["en"] == firstSteps["ru"] {
		t.Fatal("language toggle must change advice")
	}
	for _, request := range []model.AssistantRequest{{EventID: "FAKE", Action: "start"}, {EventID: "design-course", Action: "award_exp"}} {
		if _, err := client.Assist(context.Background(), d, d.Employees[0], request); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
}

func TestAssistantValidatesFactsAndNeverSendsProofsOrIdentity(t *testing.T) {
	d := testfixture.Dataset()
	d.Employees[0].FullName = "NAME_MUST_NOT_LEAVE"
	d.Workflow.Submissions = []model.Submission{{ID: "private", Text: "PROOF_MUST_NOT_LEAVE", URL: "https://private.example/proof"}}
	for _, tc := range []struct {
		name     string
		choice   assistantChoice
		delay    time.Duration
		wantMode string
	}{
		{name: "valid", choice: assistantChoice{Benefit: "Useful for your goal", FirstStep: "Choose one example", Application: "Apply the skill at work", EvidenceIDs: []string{"design-course:goal", "design-course:gap", "design-course:history"}}, wantMode: "llm"},
		{name: "foreign-evidence", choice: assistantChoice{Benefit: "Useful", FirstStep: "Start", Application: "Apply", EvidenceIDs: []string{"FAKE:goal", "design-course:gap", "design-course:history"}}, wantMode: "rules"},
		{name: "invented-alternative", choice: assistantChoice{Benefit: "Useful", FirstStep: "Start", Application: "Apply", EvidenceIDs: []string{"design-course:goal", "design-course:gap", "design-course:history"}, AlternativeEventID: "FAKE"}, wantMode: "rules"},
		{name: "timeout", delay: 100 * time.Millisecond, wantMode: "rules"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				text := string(body)
				if strings.Contains(text, "NAME_MUST_NOT_LEAVE") || strings.Contains(text, "PROOF_MUST_NOT_LEAVE") || strings.Contains(text, "private.example") {
					t.Error("sensitive data was sent to model")
				}
				if !strings.Contains(text, "English") || !strings.Contains(text, "design-course:goal") {
					t.Error("missing language or grounded facts")
				}
				if tc.delay > 0 {
					time.Sleep(tc.delay)
				}
				content, _ := json.Marshal(tc.choice)
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(content)}}}})
			}))
			defer server.Close()
			client := &Client{BaseURL: server.URL, APIKey: "test", Model: "test", Timeout: 30 * time.Millisecond}
			result, err := client.Assist(context.Background(), d, d.Employees[0], model.AssistantRequest{EventID: "design-course", Action: "start", Locale: "en"})
			if err != nil || result.Mode != tc.wantMode || len(result.Evidence) != 3 {
				t.Fatalf("result: %+v err=%v", result, err)
			}
		})
	}
}

func TestLocalizedRecommendationPromptAndFallback(t *testing.T) {
	d := testfixture.Dataset()
	for _, locale := range []string{"ru", "kk", "en"} {
		candidates := engine.RankCandidatesLocale(d, d.Employees[0], locale)
		var disabled *Client
		result := disabled.RecommendLocale(context.Background(), candidates, 1, locale)
		if result.Mode != "rules" || result.Notice == "" {
			t.Fatal("localized fallback missing")
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "in "+languageName(locale)) {
				t.Error("wrong prompt language")
			}
			w.Write([]byte(`{"choices":[{"message":{"content":"invalid JSON"}}]}`))
		}))
		client := &Client{BaseURL: server.URL, APIKey: "test", Model: "test"}
		if result := client.RecommendLocale(context.Background(), candidates, 1, locale); result.Mode != "rules" {
			t.Fatal("invalid JSON must fall back")
		}
		server.Close()
	}
}
