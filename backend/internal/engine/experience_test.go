package engine

import (
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"encoding/json"
	"strings"
	"testing"
)

func TestMonthlyExperienceReversalAndTreeDoNotReset(t *testing.T) {
	d := testfixture.Dataset()
	d.BusinessDate = "2026-11-01"
	d.Workflow.Ledger = []model.ExpEntry{
		{ID: "a", EmployeeID: "E0001", Amount: 80, Month: "2026-10"},
		{ID: "b", EmployeeID: "E0001", Amount: 40, Month: "2026-10"},
		{ID: "c", EmployeeID: "E0001", Amount: 20, Month: "2026-11"},
	}
	experience := ExperienceFor(d, "E0001", "")
	if experience.MonthlyEXP != 20 || experience.TotalEXP != 140 || experience.TreeLevel != 2 || experience.Energy != 40 {
		t.Fatalf("bad month rollover: %+v", experience)
	}
	d.Workflow.Ledger = append(d.Workflow.Ledger, model.ExpEntry{ID: "undo", EmployeeID: "E0001", Amount: -80, Month: "2026-10", ReversalOf: "a"})
	experience = ExperienceFor(d, "E0001", "2026-10")
	if experience.MonthlyEXP != 40 || experience.TotalEXP != 60 || experience.TreeLevel != 1 || len(experience.Entries) != 3 {
		t.Fatalf("reversal lost history: %+v", experience)
	}
}

func TestLeaderboardSharedPlacesAndPublicFields(t *testing.T) {
	d := testfixture.Dataset()
	d.Employees = append(d.Employees, model.Employee{ID: "E0003", FullName: "Third", Role: "Analyst", Department: "Data"}, model.Employee{ID: "E0004", FullName: "Zero"})
	d.Workflow.Ledger = []model.ExpEntry{
		{EmployeeID: "E0001", Amount: 60, Month: "2026-10"},
		{EmployeeID: "E0002", Amount: 60, Month: "2026-10"},
		{EmployeeID: "E0003", Amount: 20, Month: "2026-10"},
	}
	board := LeaderboardFor(d, "", "", "", false)
	if len(board.Items) != 3 || board.Items[0].Rank != 1 || board.Items[1].Rank != 1 || board.Items[2].Rank != 3 {
		t.Fatalf("shared places: %+v", board)
	}
	raw, _ := json.Marshal(board)
	if strings.Contains(string(raw), "employee_id") || strings.Contains(string(raw), "department") || strings.Contains(string(raw), "role") {
		t.Fatal("public ranking leaked profile metadata")
	}
	if len(LeaderboardFor(d, "", "Data", "Analyst", false).Items) != 3 {
		t.Fatal("public filters could reveal hidden profile fields")
	}
	hr := LeaderboardFor(d, "2026-10", "Data", "Analyst", true)
	if len(hr.Items) != 1 || hr.Items[0].EmployeeID != "E0003" {
		t.Fatal("HR filters or identity missing")
	}
	if len(LeaderboardFor(d, "2026-11", "", "", false).Items) != 0 {
		t.Fatal("no leader for an empty month")
	}
}
