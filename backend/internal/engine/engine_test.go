package engine

import (
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"testing"
)

func TestSkillsAssessmentBoundaryAndCaps(t *testing.T) {
	d := testfixture.Dataset()
	d.History = []model.Activity{testfixture.Record("old", "design-course", "2026-08-01", "completed"), testfixture.Record("same-day", "design-course", "2026-09-01", "completed"), testfixture.Record("new", "design-course", "2026-09-02", "completed"), testfixture.Record("cap", "cloud-basics", "2026-09-03", "completed"), testfixture.Record("unfinished", "design-course", "2026-09-04", "in_progress"), testfixture.Record("future", "design-course", "2026-10-02", "completed")}
	levels := EffectiveSkills(d, d.Employees[0])
	if levels["design"] != 2 || levels["cloud"] != 4 {
		t.Fatalf("unexpected skills: %v", levels)
	}
	if Raised(5, model.SkillGain{Gain: 2, MaxLevel: 3}) != 5 || Raised(4, model.SkillGain{Gain: 3, MaxLevel: 5}) != 5 {
		t.Fatal("skill caps must not lower skill or exceed 5")
	}
}
func TestCriticalGapOutranksRepeatedlySkippedSpeaking(t *testing.T) {
	d := testfixture.Dataset()
	for i, date := range []string{"2026-07-01", "2026-08-01", "2026-09-01"} {
		d.History = append(d.History, testfixture.Record(string(rune('a'+i)), "EV_036", date, "no_show"))
	}
	ranked := RankCandidates(d, d.Employees[0])
	if len(ranked) != 2 || ranked[0].Event.ID != "design-course" {
		t.Fatalf("critical technical step must rank first: %+v", ranked)
	}
	for _, r := range ranked {
		if len(r.Evidence) < 3 {
			t.Fatal("missing explanation factors")
		}
	}
}
func TestTargetsAndEligibility(t *testing.T) {
	d := testfixture.Dataset()
	e := d.Employees[0]
	if Target(e).TargetGrade != "Senior" {
		t.Fatal("next grade")
	}
	if Target(d.Employees[1]).TargetGrade != "Lead" {
		t.Fatal("lead stays lead")
	}
	e.CareerGoal = &model.Goal{TargetRole: "Analyst", TargetGrade: "Middle"}
	ranked := RankCandidates(d, e)
	if len(ranked) != 1 || ranked[0].Event.ID != "analyst-course" || !ranked[0].CrossRole {
		t.Fatalf("cross-role recommendation: %+v", ranked)
	}
	e = d.Employees[0]
	levels := EffectiveSkills(d, e)
	event := d.Events[0]
	for _, name := range []string{"mandatory", "prerequisite", "audience", "past-session", "completed"} {
		t.Run(name, func(t *testing.T) {
			copy := event
			local := d
			switch name {
			case "mandatory":
				copy.Mandatory = true
			case "prerequisite":
				copy.Prerequisites = map[string]int{"design": 5}
			case "audience":
				copy.TargetRoles = []string{"Unknown"}
			case "past-session":
				copy.UpcomingSessions = []string{"2026-09-01"}
			case "completed":
				local.History = []model.Activity{testfixture.Record("a", copy.ID, "2026-09-20", "completed")}
			}
			if Eligible(local, e, copy, levels) {
				t.Fatal("invalid candidate was allowed")
			}
		})
	}
	if len(RankCandidates(d, d.Employees[1])) != 0 || EmptyReason(d, d.Employees[1]) == "" {
		t.Fatal("covered goal should have an empty explanation")
	}
}
func TestRecurringClubAndHR(t *testing.T) {
	d := testfixture.Dataset()
	e := d.Employees[0]
	event := d.Events[1]
	d.History = []model.Activity{testfixture.Record("a", event.ID, "2026-09-20", "completed")}
	if !Eligible(d, e, event, EffectiveSkills(d, e)) {
		t.Fatal("recurring club may be repeated on another date")
	}
	d.History = append(d.History, testfixture.Record("b", event.ID, "2026-10-01", "completed"))
	if Eligible(d, e, event, EffectiveSkills(d, e)) {
		t.Fatal("same-day club repetition allowed")
	}
	hr := SummarizeHR(d)
	if hr.EmployeeCount != 2 || len(hr.WithoutNextStep) != 1 || hr.Participation[1].Completed != 2 {
		t.Fatalf("bad aggregation: %+v", hr)
	}
}
