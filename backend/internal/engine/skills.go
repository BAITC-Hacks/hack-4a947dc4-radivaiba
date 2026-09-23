package engine

import (
	"careerquest/internal/model"
	"sort"
	"time"
)

func Contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func Raised(old int, gain model.SkillGain) int { return max(old, min(5, gain.MaxLevel, old+gain.Gain)) }

// BusinessDate is supplied by the application clock; pure calculations never use wall time.
func BusinessDate(d model.Dataset) string {
	if d.BusinessDate != "" {
		return d.BusinessDate
	}
	return d.Catalog.Meta.AsOfDate
}

func EffectiveSkills(d model.Dataset, employee model.Employee) map[string]int {
	levels := make(map[string]int, len(employee.Skills))
	for k, v := range employee.Skills {
		levels[k] = v
	}
	events := make(map[string]model.Event, len(d.Events))
	for _, event := range d.Events {
		events[event.ID] = event
	}
	history := EmployeeHistory(d, employee.ID)
	sortActivityHistory(d, history)
	for _, record := range history {
		// A demo completion on the assessment date is a new action, not part of that assessment.
		if record.Status != "completed" || record.Date > BusinessDate(d) || (record.Date <= employee.LastReviewDate && !record.Demo) || record.Date < employee.LastReviewDate {
			continue
		}
		for _, gain := range events[record.EventID].DevelopsSkills {
			levels[gain.SkillID] = Raised(levels[gain.SkillID], gain)
		}
	}
	return levels
}

func sortActivityHistory(d model.Dataset, history []model.Activity) {
	approvedAt := map[string]time.Time{}
	for _, decision := range d.Workflow.Decisions {
		if decision.Action == "approve" && decision.CompletionID != "" {
			if timestamp, err := time.Parse(time.RFC3339Nano, decision.RecordedAt); err == nil {
				approvedAt[decision.CompletionID] = timestamp
			}
		}
	}
	sort.Slice(history, func(i, j int) bool {
		a, b := history[i], history[j]
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		aTime, aApp := approvedAt[a.ID]
		bTime, bApp := approvedAt[b.ID]
		if aApp != bApp {
			return !aApp // baseline records precede new HR decisions on the same business day
		}
		if aApp && !aTime.Equal(bTime) {
			return aTime.Before(bTime)
		}
		return a.ID < b.ID
	})
}
func EmployeeHistory(d model.Dataset, id string) []model.Activity {
	result := []model.Activity{}
	for _, r := range d.History {
		if r.EmployeeID == id && r.Date <= BusinessDate(d) {
			result = append(result, r)
		}
	}
	return result
}
func FindEmployee(d model.Dataset, id string) (model.Employee, bool) {
	for _, employee := range d.Employees {
		if employee.ID == id {
			return employee, true
		}
	}
	return model.Employee{}, false
}
func FindEvent(d model.Dataset, id string) (model.Event, bool) {
	for _, event := range d.Events {
		if event.ID == id {
			return event, true
		}
	}
	return model.Event{}, false
}
