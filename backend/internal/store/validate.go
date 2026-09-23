package store

import (
	"careerquest/internal/model"
	"fmt"
	"strings"
	"time"
)

func validDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}
func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if a == value {
			return true
		}
	}
	return false
}
func uniqueID(seen map[string]bool, id, location string) error {
	if strings.TrimSpace(id) == "" || seen[id] {
		return fmt.Errorf("%s: empty or duplicate ID %q", location, id)
	}
	seen[id] = true
	return nil
}

func Validate(d model.Dataset) error {
	if !validDate(d.Catalog.Meta.AsOfDate) {
		return fmt.Errorf("skills.json: meta.as_of_date must be YYYY-MM-DD")
	}
	if len(d.Catalog.Skills) == 0 || len(d.Catalog.RoleProfiles) == 0 || len(d.Employees) == 0 || len(d.Events) == 0 {
		return fmt.Errorf("dataset: skills, role profiles, employees and events must not be empty")
	}
	skills := map[string]bool{}
	profiles := map[string]bool{}
	employees := map[string]bool{}
	events := map[string]bool{}
	records := map[string]bool{}
	for _, s := range d.Catalog.Skills {
		if err := uniqueID(skills, s.ID, "skills.json"); err != nil {
			return err
		}
		if s.Name == "" || !oneOf(s.Type, "hard", "soft") {
			return fmt.Errorf("skills.json: %s: invalid name/type", s.ID)
		}
	}
	validateLevels := func(levels map[string]int, where string) error {
		for k, v := range levels {
			if !skills[k] || v < 0 || v > 5 {
				return fmt.Errorf("%s: invalid skill/level %s=%d", where, k, v)
			}
		}
		return nil
	}
	for _, p := range d.Catalog.RoleProfiles {
		key := p.Role + "/" + p.Grade
		if err := uniqueID(profiles, key, "skills.json role_profiles"); err != nil {
			return err
		}
		if p.Role == "" || !oneOf(p.Grade, "Junior", "Middle", "Senior", "Lead") || len(p.RequiredSkills) == 0 {
			return fmt.Errorf("skills.json: invalid role profile %s", key)
		}
		if err := validateLevels(p.RequiredSkills, "skills.json "+key); err != nil {
			return err
		}
		for _, k := range p.CriticalSkills {
			if p.RequiredSkills[k] <= 0 {
				return fmt.Errorf("skills.json %s: invalid critical skill %s", key, k)
			}
		}
	}
	for _, e := range d.Employees {
		where := "employees.json " + e.ID
		if err := uniqueID(employees, e.ID, where); err != nil {
			return err
		}
		if e.FullName == "" || e.Department == "" || !profiles[e.Role+"/"+e.Grade] {
			return fmt.Errorf("%s: missing name/department or unknown role/grade", where)
		}
		if !validDate(e.HireDate) || !validDate(e.LastReviewDate) || e.HireDate > d.Catalog.Meta.AsOfDate || e.LastReviewDate > d.Catalog.Meta.AsOfDate || e.LastReviewDate < e.HireDate {
			return fmt.Errorf("%s: invalid hire_date/last_review_date", where)
		}
		if e.TenureMonths < 0 || !oneOf(e.WorkFormat, "office", "hybrid", "remote") || !oneOf(e.PreferredLanguage, "ru", "en", "kk") || e.Skills == nil {
			return fmt.Errorf("%s: invalid tenure, work_format, language or missing skills", where)
		}
		if err := validateLevels(e.Skills, where); err != nil {
			return err
		}
		if e.CareerGoal != nil && !profiles[e.CareerGoal.TargetRole+"/"+e.CareerGoal.TargetGrade] {
			return fmt.Errorf("%s: unknown career_goal role/grade", where)
		}
	}
	for _, e := range d.Employees {
		if e.ManagerID != nil && (!employees[*e.ManagerID] || *e.ManagerID == e.ID) {
			return fmt.Errorf("employees.json %s: unknown or self manager_id", e.ID)
		}
	}
	for _, e := range d.Events {
		where := "events.json " + e.ID
		if err := uniqueID(events, e.ID, where); err != nil {
			return err
		}
		if e.Title == "" || e.DurationHours <= 0 || !oneOf(e.Format, "online", "offline", "self_paced") || !oneOf(e.Type, "compliance", "onboarding", "course", "workshop", "mentoring", "certification", "meetup") {
			return fmt.Errorf("%s: invalid title, duration, format or type", where)
		}
		if err := validateLevels(e.Prerequisites, where); err != nil {
			return err
		}
		gainIDs := map[string]bool{}
		for _, g := range e.DevelopsSkills {
			if !skills[g.SkillID] || g.Gain <= 0 || g.MaxLevel < 0 || g.MaxLevel > 5 || gainIDs[g.SkillID] {
				return fmt.Errorf("%s: invalid/duplicate develops_skills", where)
			}
			gainIDs[g.SkillID] = true
		}
		for _, date := range e.UpcomingSessions {
			if !validDate(date) {
				return fmt.Errorf("%s: invalid session date", where)
			}
		}
	}
	approvedDates := map[string]string{}
	for _, decision := range d.Workflow.Decisions {
		if decision.Action == "approve" && strings.HasPrefix(decision.CompletionID, "APP_") {
			approvedDates[decision.CompletionID] = decision.BusinessDate
		}
	}
	for _, r := range d.History {
		where := "activity_history.csv " + r.ID
		if err := uniqueID(records, r.ID, where); err != nil {
			return err
		}
		if !employees[r.EmployeeID] || !events[r.EventID] {
			return fmt.Errorf("%s: unknown employee_id/event_id", where)
		}
		appOwned := approvedDates[r.ID] == r.Date && r.Status == "completed" && r.AssignedBy == "hr"
		if !validDate(r.Date) || (r.Date > d.Catalog.Meta.AsOfDate && !appOwned) || (r.DueDate != "" && !validDate(r.DueDate)) {
			return fmt.Errorf("%s: invalid or future date", where)
		}
		if !oneOf(r.Status, "completed", "in_progress", "dropped", "no_show", "declined", "overdue") || !oneOf(r.AssignedBy, "self", "manager", "hr") {
			return fmt.Errorf("%s: invalid status/assigned_by", where)
		}
		if r.CompletionPct < 0 || r.CompletionPct > 100 || (r.Status == "completed" && r.CompletionPct != 100) || (oneOf(r.Status, "no_show", "declined") && r.CompletionPct != 0) || (oneOf(r.Status, "in_progress", "dropped", "overdue") && r.CompletionPct > 95) || (r.Status == "dropped" && r.CompletionPct < 5) {
			return fmt.Errorf("%s: invalid completion_pct for status", where)
		}
		if r.Score != nil && (*r.Score < 0 || *r.Score > 100) || r.FeedbackRating != nil && (*r.FeedbackRating < 1 || *r.FeedbackRating > 5) {
			return fmt.Errorf("%s: invalid score/feedback_rating", where)
		}
	}
	return nil
}
