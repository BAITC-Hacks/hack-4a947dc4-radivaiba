package engine

import (
	"careerquest/internal/model"
	"fmt"
	"math"
	"sort"
	"strings"
)

func Eligible(d model.Dataset, employee model.Employee, event model.Event, levels map[string]int) bool {
	return EligibilityReason(d, employee, event, levels) == ""
}

// EligibilityReason is a stable code shared by recommendation filters and the tree.
func EligibilityReason(d model.Dataset, employee model.Employee, event model.Event, levels map[string]int) string {
	if event.Mandatory {
		return "mandatory"
	}
	goal := Target(employee)
	currentAudience := Contains(event.TargetRoles, employee.Role) && Contains(event.TargetGrades, employee.Grade)
	targetAudience := Contains(event.TargetRoles, goal.TargetRole) && Contains(event.TargetGrades, goal.TargetGrade)
	if !currentAudience && !targetAudience {
		return "audience"
	}
	for skill, required := range event.Prerequisites {
		if levels[skill] < required {
			return "prerequisites"
		}
	}
	if event.Format != "self_paced" {
		available := false
		for _, date := range event.UpcomingSessions {
			if date >= BusinessDate(d) {
				available = true
				break
			}
		}
		if !available {
			return "schedule"
		}
	}
	for _, r := range EmployeeHistory(d, employee.ID) {
		if r.EventID == event.ID && r.Status == "completed" && (event.ID != "EV_036" || r.Date == BusinessDate(d)) {
			return "completed"
		}
	}
	for _, enrollment := range d.Workflow.Enrollments {
		if enrollment.EmployeeID == employee.ID && enrollment.EventID == event.ID && enrollment.State == "pending" {
			return "pending"
		}
	}
	return ""
}
func RankCandidates(d model.Dataset, employee model.Employee) []model.Recommendation {
	return RankCandidatesLocale(d, employee, "ru")
}

func RankCandidatesLocale(d model.Dataset, employee model.Employee, locale string) []model.Recommendation {
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
				gapTexts = append(gapTexts, fmt.Sprintf(Text(locale, "%s: %d → %d, требуется %d", "%s: %d → %d, қажет деңгей: %d", "%s: %d → %d, target %d"), skill.Name, before, after, skill.Required))
			}
		}
		if benefit <= 0 {
			continue
		}
		completed, negative := 0, 0
		inProgress := false
		for _, enrollment := range d.Workflow.Enrollments {
			if enrollment.EmployeeID == employee.ID && enrollment.EventID == event.ID && (enrollment.State == "in_progress" || enrollment.State == "changes_requested") {
				inProgress = true
			}
		}
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
			{ID: event.ID + ":goal", Factor: "goal", Text: fmt.Sprintf(Text(locale, "Сейчас %s · %s. Цель: %s · %s.", "Қазір: %s · %s. Мақсат: %s · %s.", "Now: %s · %s. Goal: %s · %s."), employee.Role, employee.Grade, trajectory.Goal.TargetRole, trajectory.Goal.TargetGrade)},
			{ID: event.ID + ":gap", Factor: "gap", Text: strings.Join(gapTexts, "; ") + "."},
			{ID: event.ID + ":history", Factor: "history", Text: fmt.Sprintf(Text(locale, "Среди похожих добровольных активностей (%s, %s): завершено %d, пропущено / отклонено / брошено %d.", "Ұқсас ерікті іс-шаралар (%s, %s): аяқталғаны — %d, өткізілгені / бас тартылғаны / тоқтатылғаны — %d.", "Similar voluntary activities (%s, %s): %d completed; %d missed, declined or dropped."), event.Type, event.Format, completed, negative)},
		}
		explanation := Text(locale, "Шаг сокращает разрыв до выбранной цели с учётом истории участия и времени на обучение.", "Бұл қадам оқу уақыты мен қатысу тарихын ескеріп, мақсатыңызға жақындатады.", "This step closes a gap toward your goal while considering your participation history and learning time.")
		if inProgress {
			explanation = Text(locale, "Продолжите уже начатую активность: она закрывает требования вашей цели.", "Бастаған іс-шараңызды жалғастырыңыз: ол мақсатыңызға қажет дағдыларды дамытады.", "Continue the activity you have started: it develops skills needed for your goal.")
		}
		if crossRole {
			explanation = Text(locale, "Подготовка к смене роли: активность соответствует целевой профессии и её требованиям.", "Рөл ауыстыруға дайындық: іс-шара таңдаған мамандығыңыздың талаптарына сай келеді.", "Preparing for a role change: this activity matches the requirements of your target role.")
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
	return EmptyReasonLocale(d, employee, "ru")
}

func EmptyReasonLocale(d model.Dataset, employee model.Employee, locale string) string {
	t := BuildTrajectory(d, employee, EffectiveSkills(d, employee))
	hasGap := false
	for _, s := range t.Skills {
		if s.Gap > 0 {
			hasGap = true
			break
		}
	}
	if !hasGap {
		return Text(locale, "Требования выбранной цели уже закрыты. Обсудите следующую цель с руководителем.", "Таңдаған мақсатыңыздың талаптары орындалды. Жетекшіңізбен келесі мақсатты талқылаңыз.", "You already meet this goal's requirements. Discuss your next goal with your manager.")
	}
	return Text(locale, "Сейчас нет подходящего следующего шага: проверьте необходимые навыки, аудиторию, расписание и уже пройденное обучение.", "Қазір қолайлы келесі қадам жоқ: қажетті дағдыларды, қатысушылар тобын, кестені және аяқталған оқуды тексеріңіз.", "No suitable next step is available right now. Check required skills, audience, schedule and completed learning.")
}
