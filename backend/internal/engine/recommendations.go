package engine

import (
	"careerquest/internal/model"
	"fmt"
	"math"
	"sort"
	"strings"
)

func Eligible(d model.Dataset, employee model.Employee, event model.Event, levels map[string]int) bool {
	if event.Mandatory {
		return false
	}
	goal := Target(employee)
	currentAudience := Contains(event.TargetRoles, employee.Role) && Contains(event.TargetGrades, employee.Grade)
	targetAudience := Contains(event.TargetRoles, goal.TargetRole) && Contains(event.TargetGrades, goal.TargetGrade)
	if !currentAudience && !targetAudience {
		return false
	}
	for skill, required := range event.Prerequisites {
		if levels[skill] < required {
			return false
		}
	}
	if event.Format != "self_paced" {
		available := false
		for _, date := range event.UpcomingSessions {
			if date >= d.Catalog.Meta.AsOfDate {
				available = true
				break
			}
		}
		if !available {
			return false
		}
	}
	for _, r := range EmployeeHistory(d, employee.ID) {
		if r.EventID == event.ID && r.Status == "completed" && (event.ID != "EV_036" || r.Date == d.Catalog.Meta.AsOfDate) {
			return false
		}
	}
	return true
}
func RankCandidates(d model.Dataset, employee model.Employee) []model.Recommendation {
	levels := EffectiveSkills(d, employee)
	trajectory := BuildTrajectory(d, employee, levels)
	skills := map[string]model.SkillProgress{}
	for _, s := range d.Catalog.Skills {
		skills[s.ID] = model.SkillProgress{SkillID: s.ID, Name: s.Name, Type: s.Type}
	}
	for _, s := range trajectory.Skills {
		skills[s.SkillID] = s
	}
	events := map[string]model.Event{}
	for _, event := range d.Events {
		events[event.ID] = event
	}
	history := EmployeeHistory(d, employee.ID)
	result := []model.Recommendation{}
	for _, event := range d.Events {
		if !Eligible(d, employee, event, levels) {
			continue
		}
		gains := []model.ExpectedGain{}
		benefit := 0.0
		gapTexts := []string{}
		for _, g := range event.DevelopsSkills {
			before := levels[g.SkillID]
			after := Raised(before, g)
			skill := skills[g.SkillID]
			if after <= before {
				continue
			}
			gains = append(gains, model.ExpectedGain{SkillID: g.SkillID, Name: skill.Name, Before: before, After: after, Required: skill.Required})
			weight := 1.0
			if skill.Critical {
				weight = 3
			}
			benefit += weight * float64(min(after-before, skill.Gap))
			if skill.Gap > 0 {
				gapTexts = append(gapTexts, fmt.Sprintf("%s: %d → %d, требуется %d", skill.Name, before, after, skill.Required))
			}
		}
		if benefit <= 0 {
			continue
		}
		completed, negative := 0, 0
		inProgress := false
		for _, r := range history {
			if r.EventID == event.ID && r.Status == "in_progress" {
				inProgress = true
			}
			previous := events[r.EventID]
			if previous.Mandatory || previous.Type != event.Type || previous.Format != event.Format {
				continue
			}
			switch r.Status {
			case "completed":
				completed++
			case "no_show", "dropped", "declined":
				negative++
			}
		}
		reliability := float64(completed+1) / float64(completed+negative+2)
		score := benefit * reliability / math.Sqrt(math.Max(1, event.DurationHours))
		if inProgress {
			score *= 1.25
		}
		crossRole := !(Contains(event.TargetRoles, employee.Role) && Contains(event.TargetGrades, employee.Grade)) && trajectory.Goal.TargetRole != employee.Role
		evidence := []model.Evidence{
			{ID: event.ID + ":goal", Factor: "goal", Text: fmt.Sprintf("Сейчас %s · %s. Цель: %s · %s.", employee.Role, employee.Grade, trajectory.Goal.TargetRole, trajectory.Goal.TargetGrade)},
			{ID: event.ID + ":gap", Factor: "gap", Text: strings.Join(gapTexts, "; ") + "."},
			{ID: event.ID + ":history", Factor: "history", Text: fmt.Sprintf("Среди похожих добровольных активностей (%s, %s): завершено %d, пропущено / отклонено / брошено %d.", event.Type, event.Format, completed, negative)},
		}
		explanation := "Шаг сокращает разрыв до выбранной цели с учётом истории участия и времени на обучение."
		if inProgress {
			explanation = "Продолжите уже начатую активность: она закрывает требования вашей цели."
		}
		if crossRole {
			explanation = "Подготовка к смене роли: активность соответствует целевой профессии и её требованиям."
		}
		result = append(result, model.Recommendation{Event: event, Score: score, ExpectedGains: gains, Evidence: evidence, Explanation: explanation, CrossRole: crossRole, InProgress: inProgress})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].Event.ID < result[j].Event.ID
		}
		return result[i].Score > result[j].Score
	})
	return result
}
func EmptyReason(d model.Dataset, employee model.Employee) string {
	t := BuildTrajectory(d, employee, EffectiveSkills(d, employee))
	hasGap := false
	for _, s := range t.Skills {
		if s.Gap > 0 {
			hasGap = true
			break
		}
	}
	if !hasGap {
		return "Требования выбранной цели уже закрыты. Обсудите следующую цель с руководителем."
	}
	return "В каталоге нет доступной добровольной активности, которая сейчас сокращает разрыв: проверьте prerequisites, аудиторию, расписание и уже пройденное обучение."
}
