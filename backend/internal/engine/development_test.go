package engine

import (
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"reflect"
	"strings"
	"testing"
)

func TestModuleRewardsAndLocalisedCriteria(t *testing.T) {
	d := testfixture.Dataset()
	for _, tc := range []struct {
		hours        float64
		prerequisite int
		mandatory    bool
		want         int
	}{{4, 0, false, 20}, {4.1, 0, false, 40}, {12, 1, false, 50}, {12.1, 3, false, 80}, {40, 4, true, 0}} {
		event := model.Event{ID: "module", DurationHours: tc.hours, Mandatory: tc.mandatory, Prerequisites: map[string]int{"design": tc.prerequisite}}
		if got := ModuleReward(event); got != tc.want {
			t.Fatalf("reward %+v: got %d", tc, got)
		}
	}
	config := DefaultModuleConfig(d, d.Events[1])
	if config.RepeatPolicy != "daily_one_reward_per_month" {
		t.Fatal(config.RepeatPolicy)
	}
	for _, locale := range []string{"ru", "kk", "en"} {
		if config.Outcome[locale] == "" || len(config.Criteria[locale]) != 3 {
			t.Fatalf("missing locale %s", locale)
		}
	}
	if config.Outcome["ru"] == config.Outcome["kk"] || config.Criteria["ru"][0] == config.Criteria["en"][0] {
		t.Fatal("locale text must actually change")
	}
}

func TestDevelopmentUsesRealEventsAndKeepsPendingAfterGoalChange(t *testing.T) {
	d := testfixture.Dataset()
	e := d.Employees[0]
	config := DefaultModuleConfig(d, d.Events[0])
	config.RewardEXP = 99
	d.Workflow.Enrollments = []model.Enrollment{{ID: "enrollment", EmployeeID: e.ID, EventID: d.Events[0].ID, State: "pending", BusinessDate: "2026-10-01", Config: config}}
	before := EffectiveSkills(d, e)
	e.CareerGoal = &model.Goal{TargetRole: "Analyst", TargetGrade: "Middle"}
	view := BuildDevelopment(d, e, "en")
	found := false
	ids := map[string]bool{}
	for _, item := range view.Items {
		if ids[item.Event.ID] {
			t.Fatal("duplicate apple")
		}
		ids[item.Event.ID] = true
		if item.Event.ID == d.Events[0].ID {
			found = true
			if item.State != "pending" || item.Config.RewardEXP != 99 || item.Enrollment == nil {
				t.Fatalf("lost pending enrollment: %+v", item)
			}
		}
	}
	if !found || view.Experience.TotalEXP != 0 || !reflect.DeepEqual(before, EffectiveSkills(d, e)) {
		t.Fatal("pending must remain visible and must not award progress")
	}
	for _, item := range RankCandidates(d, e) {
		if item.Event.ID == d.Events[0].ID {
			t.Fatal("pending module should not be recommended again")
		}
	}
}

func TestBusinessDateAndLegacyCompletion(t *testing.T) {
	d := testfixture.Dataset()
	d.BusinessDate = "2026-10-02"
	d.History = []model.Activity{testfixture.Record("later", "design-course", "2026-10-02", "completed")}
	if EffectiveSkills(d, d.Employees[0])["design"] != 2 || BuildProfile(d, d.Employees[0]).AsOfDate != "2026-10-02" {
		t.Fatal("injected business date not used")
	}
	view := BuildDevelopment(d, d.Employees[0], "ru")
	for _, module := range view.Items {
		if module.Event.ID == "design-course" && module.State != "completed" {
			t.Fatal("legacy completion missing")
		}
	}
	if view.Experience.TotalEXP != 0 {
		t.Fatal("imported completed history must not receive EXP")
	}
	d.History[0].EventID = "EV_036"
	if Eligible(d, d.Employees[0], d.Events[1], EffectiveSkills(d, d.Employees[0])) {
		t.Fatal("same business day repeat")
	}
	d.BusinessDate = "2026-10-03"
	if !Eligible(d, d.Employees[0], d.Events[1], EffectiveSkills(d, d.Employees[0])) {
		t.Fatal("new business day repeat blocked")
	}
}

func TestRecommendationAndPrerequisiteLocales(t *testing.T) {
	d := testfixture.Dataset()
	d.Events[0].Prerequisites["design"] = 3
	for _, locale := range []string{"ru", "kk", "en"} {
		view := BuildDevelopment(d, d.Employees[0], locale)
		for _, item := range view.Items {
			if item.Event.ID == "design-course" && (item.State != "locked" || !strings.Contains(item.BlockedReason, "System Design: 1 → 3")) {
				t.Fatalf("missing useful prerequisite: %+v", item)
			}
		}
		ranked := RankCandidatesLocale(d, d.Employees[0], locale)
		if len(ranked) == 0 || len(ranked[0].Evidence) != 3 {
			t.Fatal("localized evidence missing")
		}
	}
	if Locale("KZ") != "kk" || Locale("ENG") != "en" || Locale("unknown") != "ru" {
		t.Fatal("locale normalization")
	}
}

func TestModuleBranchDoesNotMoveWithSkillsOrHistory(t *testing.T) {
	d := testfixture.Dataset()
	d.Catalog.Skills[0].Category = "engineering"
	d.Catalog.Skills[1].Category = "communication"
	d.Events[0].DevelopsSkills = append(d.Events[0].DevelopsSkills, model.SkillGain{SkillID: "speech", Gain: 1, MaxLevel: 5})
	before := BuildDevelopment(d, d.Employees[0], "ru")
	d.Employees[0].Skills["design"] = 5
	d.History = []model.Activity{testfixture.Record("new", "design-course", "2026-10-01", "completed")}
	after := BuildDevelopment(d, d.Employees[0], "ru")
	for i := range before.Items {
		if before.Items[i].Branch != after.Items[i].Branch {
			t.Fatal("tree branch moved after skill change")
		}
	}
}

func TestReopenedOlderClubOccurrenceRemainsVisible(t *testing.T) {
	d := testfixture.Dataset()
	d.BusinessDate = "2026-10-03"
	config := DefaultModuleConfig(d, d.Events[1])
	d.Workflow.Enrollments = []model.Enrollment{
		{ID: "old", EmployeeID: "E0001", EventID: "EV_036", State: "changes_requested", BusinessDate: "2026-10-01", Config: config},
		{ID: "new", EmployeeID: "E0001", EventID: "EV_036", State: "completed", BusinessDate: "2026-10-02", Config: config},
	}
	for _, module := range BuildDevelopment(d, d.Employees[0], "en").Items {
		if module.Event.ID == "EV_036" && (module.State != "changes_requested" || module.Enrollment == nil || module.Enrollment.ID != "old") {
			t.Fatal("reopened review hidden by later completed occurrence")
		}
	}
}

func TestApprovedGainsFollowDecisionTimeRatherThanRandomRecordID(t *testing.T) {
	d := testfixture.Dataset()
	d.Events[0].DevelopsSkills = []model.SkillGain{{SkillID: "design", Gain: 2, MaxLevel: 2}}
	d.Events[1].DevelopsSkills = []model.SkillGain{{SkillID: "design", Gain: 1, MaxLevel: 5}}
	d.History = []model.Activity{testfixture.Record("APP_z", "design-course", "2026-10-01", "completed"), testfixture.Record("APP_a", "EV_036", "2026-10-01", "completed")}
	d.Workflow.Decisions = []model.ReviewDecision{
		{Action: "approve", CompletionID: "APP_z", RecordedAt: "2026-09-23T10:00:00Z"},
		{Action: "approve", CompletionID: "APP_a", RecordedAt: "2026-09-23T10:00:00.1Z"},
	}
	if level := EffectiveSkills(d, d.Employees[0])["design"]; level != 3 {
		t.Fatalf("chronological gains should produce 3, got %d", level)
	}
	profile := BuildProfile(d, d.Employees[0])
	if profile.History[0].ID != "APP_a" {
		t.Fatal("profile history must show latest HR approval first")
	}
	// The store projection excludes a revoked completion; recomputation respects the remaining cap.
	d.History = d.History[1:]
	if level := EffectiveSkills(d, d.Employees[0])["design"]; level != 2 {
		t.Fatalf("recomputed after revoke: %d", level)
	}
}
