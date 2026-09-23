package engine

import (
	"careerquest/internal/model"
	"math"
	"sort"
)

func Target(employee model.Employee) model.Goal {
	if employee.CareerGoal != nil {
		return *employee.CareerGoal
	}
	grades := []string{"Junior", "Middle", "Senior", "Lead"}
	for i, grade := range grades {
		if employee.Grade == grade {
			return model.Goal{TargetRole: employee.Role, TargetGrade: grades[min(i+1, 3)]}
		}
	}
	return model.Goal{TargetRole: employee.Role, TargetGrade: employee.Grade}
}
func BuildTrajectory(d model.Dataset, employee model.Employee, levels map[string]int) model.Trajectory {
	goal := Target(employee)
	result := model.Trajectory{Goal: goal, Inferred: employee.CareerGoal == nil, AtTopGrade: employee.CareerGoal == nil && employee.Grade == "Lead", Skills: []model.SkillProgress{}}
	var target model.RoleProfile
	for _, p := range d.Catalog.RoleProfiles {
		if p.Role == goal.TargetRole && p.Grade == goal.TargetGrade {
			target = p
			break
		}
	}
	var total, covered float64
	for _, skill := range d.Catalog.Skills {
		required := target.RequiredSkills[skill.ID]
		current := levels[skill.ID]
		if required == 0 && current == 0 {
			continue
		}
		critical := Contains(target.CriticalSkills, skill.ID)
		gap := max(0, required-current)
		weight := 1.0
		if critical {
			weight = 3
			if gap > 0 {
				result.CriticalGaps++
			}
		}
		total += weight * float64(required)
		covered += weight * float64(min(current, required))
		result.Skills = append(result.Skills, model.SkillProgress{SkillID: skill.ID, Name: skill.Name, Type: skill.Type, Current: current, Required: required, Gap: gap, Critical: critical})
	}
	if total > 0 {
		result.Progress = math.Round(covered/total*1000) / 10
	} else {
		result.Progress = 100
	}
	sort.Slice(result.Skills, func(i, j int) bool {
		a, b := result.Skills[i], result.Skills[j]
		if (a.Critical && a.Gap > 0) != (b.Critical && b.Gap > 0) {
			return a.Critical && a.Gap > 0
		}
		if a.Gap != b.Gap {
			return a.Gap > b.Gap
		}
		return a.Name < b.Name
	})
	return result
}
func BuildProfile(d model.Dataset, employee model.Employee) model.Profile {
	levels := EffectiveSkills(d, employee)
	titles := map[string]string{}
	for _, e := range d.Events {
		titles[e.ID] = e.Title
	}
	history := []model.HistoryItem{}
	activities := EmployeeHistory(d, employee.ID)
	sortActivityHistory(d, activities)
	for i := len(activities) - 1; i >= 0; i-- {
		r := activities[i]
		history = append(history, model.HistoryItem{Activity: r, EventTitle: titles[r.EventID]})
	}
	return model.Profile{Employee: employee, EffectiveSkills: levels, Trajectory: BuildTrajectory(d, employee, levels), History: history, AsOfDate: BusinessDate(d), Revision: d.Revision}
}
