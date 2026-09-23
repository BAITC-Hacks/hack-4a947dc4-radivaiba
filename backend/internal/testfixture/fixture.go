package testfixture

import "careerquest/internal/model"

// Small independently authored fixture: never depends on the organizers' dataset.
func Dataset() model.Dataset {
	d := model.Dataset{Revision: 1, Catalog: model.Catalog{Meta: model.Meta{Dataset: "Test", Version: "1", AsOfDate: "2026-10-01"}, Skills: []model.Skill{{ID: "design", Name: "System Design", Type: "hard"}, {ID: "speech", Name: "Public Speaking", Type: "soft"}, {ID: "cloud", Name: "Cloud", Type: "hard"}}}, History: []model.Activity{}}
	for _, grade := range []string{"Junior", "Middle", "Senior", "Lead"} {
		d.Catalog.RoleProfiles = append(d.Catalog.RoleProfiles, model.RoleProfile{Role: "Backend", Grade: grade, RequiredSkills: map[string]int{"design": 3, "speech": 2}, CriticalSkills: []string{"design"}})
	}
	d.Catalog.RoleProfiles = append(d.Catalog.RoleProfiles, model.RoleProfile{Role: "Analyst", Grade: "Middle", RequiredSkills: map[string]int{"cloud": 5}, CriticalSkills: []string{"cloud"}})
	d.Employees = []model.Employee{{ID: "E0001", FullName: "Demo One", Department: "Engineering", Role: "Backend", Grade: "Middle", HireDate: "2024-01-01", TenureMonths: 33, WorkFormat: "remote", PreferredLanguage: "ru", LastReviewDate: "2026-09-01", Skills: map[string]int{"design": 1, "speech": 0, "cloud": 4}}, {ID: "E0002", FullName: "Demo Two", Department: "Engineering", Role: "Backend", Grade: "Lead", HireDate: "2024-01-01", TenureMonths: 33, WorkFormat: "office", PreferredLanguage: "en", LastReviewDate: "2026-09-01", Skills: map[string]int{"design": 3, "speech": 2}}}
	d.Events = []model.Event{
		{ID: "design-course", Title: "System Design Lab", Type: "course", Format: "online", DurationHours: 4, TargetRoles: []string{"Backend"}, TargetGrades: []string{"Junior", "Middle", "Senior", "Lead"}, DevelopsSkills: []model.SkillGain{{SkillID: "design", Gain: 1, MaxLevel: 5}}, Prerequisites: map[string]int{"design": 1}, UpcomingSessions: []string{"2026-10-07"}},
		{ID: "EV_036", Title: "Speaking Club", Type: "meetup", Format: "offline", DurationHours: 2, TargetRoles: []string{"Backend"}, TargetGrades: []string{"Junior", "Middle", "Senior", "Lead"}, DevelopsSkills: []model.SkillGain{{SkillID: "speech", Gain: 1, MaxLevel: 4}}, UpcomingSessions: []string{"2026-10-08"}},
		{ID: "cloud-basics", Title: "Cloud Basics", Type: "course", Format: "self_paced", DurationHours: 2, TargetRoles: []string{"Backend"}, TargetGrades: []string{"Middle"}, DevelopsSkills: []model.SkillGain{{SkillID: "cloud", Gain: 1, MaxLevel: 2}}},
		{ID: "analyst-course", Title: "Analyst Cloud", Type: "course", Format: "self_paced", DurationHours: 4, TargetRoles: []string{"Analyst"}, TargetGrades: []string{"Middle"}, DevelopsSkills: []model.SkillGain{{SkillID: "cloud", Gain: 1, MaxLevel: 5}}},
	}
	return d
}
func Record(id, event, date, status string) model.Activity {
	pct := 0
	if status == "completed" {
		pct = 100
	}
	return model.Activity{ID: id, EmployeeID: "E0001", EventID: event, Date: date, Status: status, CompletionPct: pct, AssignedBy: "self"}
}
