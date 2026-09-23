package engine

import (
	"careerquest/internal/model"
	"sort"
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
	sort.Slice(history, func(i, j int) bool {
		if history[i].Date == history[j].Date {
			return history[i].ID < history[j].ID
		}
		return history[i].Date < history[j].Date
	})
	for _, record := range history {
		// A demo completion on the assessment date is a new action, not part of that assessment.
		if record.Status != "completed" || record.Date > d.Catalog.Meta.AsOfDate || (record.Date <= employee.LastReviewDate && !record.Demo) || record.Date < employee.LastReviewDate {
			continue
		}
		for _, gain := range events[record.EventID].DevelopsSkills {
			levels[gain.SkillID] = Raised(levels[gain.SkillID], gain)
		}
	}
	return levels
}
func EmployeeHistory(d model.Dataset, id string) []model.Activity {
	result := []model.Activity{}
	for _, r := range d.History {
		if r.EmployeeID == id && r.Date <= d.Catalog.Meta.AsOfDate {
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
