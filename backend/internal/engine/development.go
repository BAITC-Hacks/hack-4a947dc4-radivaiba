package engine

import (
	"careerquest/internal/model"
	"fmt"
	"sort"
	"strings"
)

// ModuleReward is deterministic; AI never sets rewards. Started modules keep a config snapshot.
func ModuleReward(event model.Event) int {
	if event.Mandatory {
		return 0
	}
	reward := 20
	if event.DurationHours > 12 {
		reward = 60
	} else if event.DurationHours > 4 {
		reward = 40
	}
	bonus := 0
	for _, required := range event.Prerequisites {
		if required >= 3 {
			bonus = 20
			break
		}
		if required > 0 {
			bonus = 10
		}
	}
	return reward + bonus
}

func DefaultModuleConfig(d model.Dataset, event model.Event) model.ModuleConfig {
	category, _ := moduleCategory(d, event)
	config := model.ModuleConfig{EventID: event.ID, Version: 1, Outcome: map[string]string{}, Criteria: map[string][]string{}, RewardEXP: ModuleReward(event), RepeatPolicy: "once"}
	if event.ID == "EV_036" {
		config.RepeatPolicy = "daily_one_reward_per_month"
	}
	for _, locale := range []string{"ru", "kk", "en"} {
		config.Outcome[locale] = practicalOutcome(category, locale)
		config.Criteria[locale] = []string{
			Text(locale, "Опишите рабочую ситуацию и зачем вам нужен результат.", "Жұмыс жағдайын және бұл нәтиженің сізге не үшін қажет екенін сипаттаңыз.", "Describe the work situation and why the outcome matters to you."),
			fmt.Sprintf(Text(locale, "Покажите применение навыка из модуля «%s»: приложите результат, ссылку или подробное описание.", "«%s» модуліндегі дағдыны қолданғаныңызды көрсетіңіз: нәтижені, сілтемені немесе толық сипаттаманы тіркеңіз.", "Show how you applied a skill from “%s”: attach an outcome, link or detailed description."), event.Title),
			Text(locale, "Объясните принятое решение, что получилось и что можно улучшить. HR проверит результат по этим критериям.", "Қабылдаған шешімді, нәтижені және жақсарту жолдарын түсіндіріңіз. HR нәтижені осы өлшемдер бойынша тексереді.", "Explain your decision, what worked and what could improve. HR will review the outcome against these criteria."),
		}
	}
	return config
}

func practicalOutcome(category, locale string) string {
	switch category {
	case "engineering":
		return Text(locale, "Разберите реальное техническое решение: покажите схему, код или конфигурацию и объясните выбор и ограничения.", "Нақты техникалық шешімді талдаңыз: сызбаны, кодты немесе баптауды көрсетіп, таңдауыңыз бен шектеулерді түсіндіріңіз.", "Work through a real technical decision: show a diagram, code or configuration and explain choices and limitations.")
	case "frontend":
		return Text(locale, "Улучшите небольшой элемент интерфейса и покажите, как изменились удобство, доступность или производительность.", "Интерфейстің шағын бөлігін жақсартып, қолдану ыңғайлылығы, қолжетімділігі немесе өнімділігі қалай өзгергенін көрсетіңіз.", "Improve a small interface element and demonstrate the change in usability, accessibility or performance.")
	case "quality":
		return Text(locale, "Проверьте реальный сценарий продукта: подготовьте тест, результаты проверки и вывод о рисках.", "Өнімнің нақты сценарийін тексеріңіз: тест, тексеру нәтижесі және тәуекелдер туралы қорытынды дайындаңыз.", "Test a real product scenario: provide a test, its results and a conclusion about risks.")
	case "data":
		return Text(locale, "Ответьте на рабочий вопрос с помощью данных: покажите анализ, вывод и ограничения результата.", "Деректер арқылы жұмыс сұрағына жауап беріңіз: талдауды, қорытындыны және нәтиженің шектеулерін көрсетіңіз.", "Answer a work question using data: show your analysis, conclusion and its limitations.")
	case "product":
		return Text(locale, "Разберите пользовательскую проблему и предложите проверяемое продуктовое решение с критериями успеха.", "Пайдаланушы мәселесін талдап, табыс өлшемдері бар тексерілетін өнімдік шешім ұсыныңыз.", "Explore a user problem and propose a testable product decision with success criteria.")
	case "hr":
		return Text(locale, "Улучшите один рабочий HR-сценарий: подготовьте план, критерии оценки и пример применения без персональных данных.", "Бір HR жұмыс сценарийін жақсартыңыз: жоспар, бағалау өлшемдері және жеке деректерсіз қолдану мысалын дайындаңыз.", "Improve one HR workflow: prepare a plan, assessment criteria and an example without personal data.")
	case "sales":
		return Text(locale, "Подготовьте план разговора с клиентом: его задача, уточняющие вопросы, предложение и следующий шаг.", "Клиентпен әңгімелесу жоспарын дайындаңыз: оның міндеті, нақтылау сұрақтары, ұсыныс және келесі қадам.", "Prepare a client conversation: their need, clarifying questions, a proposal and a next step.")
	case "support":
		return Text(locale, "Разберите обращение клиента: причина проблемы, шаги диагностики, решение и понятный ответ.", "Клиент өтінішін талдаңыз: мәселенің себебі, диагностика қадамдары, шешім және түсінікті жауап.", "Work through a support case: the cause, diagnostic steps, a solution and a clear response.")
	case "communication":
		return Text(locale, "Подготовьте короткое выступление, рабочее сообщение или план встречи для конкретной аудитории и цели.", "Нақты аудитория мен мақсатқа арналған қысқа баяндама, жұмыс хабарламасы немесе кездесу жоспарын дайындаңыз.", "Prepare a short presentation, work message or meeting plan for a specific audience and purpose.")
	case "leadership", "collaboration":
		return Text(locale, "Подготовьте и опробуйте план обратной связи или совместного решения задачи; опишите результат без личных данных коллег.", "Кері байланыс немесе міндетті бірлесіп шешу жоспарын дайындап, қолданып көріңіз; әріптестердің жеке деректерінсіз нәтижені сипаттаңыз.", "Prepare and try a feedback or collaboration plan; describe the outcome without colleagues' personal data.")
	case "personal_effectiveness":
		return Text(locale, "Выберите рабочие приоритеты на неделю, примените один новый приём и опишите, что помогло вам в работе.", "Апталық жұмыс басымдықтарын таңдап, бір жаңа тәсілді қолданыңыз және жұмысыңызға не көмектескенін сипаттаңыз.", "Choose your work priorities for a week, try one new approach and describe what helped you.")
	default:
		return Text(locale, "Разберите реальную рабочую задачу: сравните варианты, обоснуйте решение и покажите результат применения навыка.", "Нақты жұмыс міндетін талдаңыз: нұсқаларды салыстырып, шешімді негіздеңіз және дағдыны қолдану нәтижесін көрсетіңіз.", "Work through a real task: compare options, explain a decision and show the result of applying the skill.")
	}
}

func moduleCategory(d model.Dataset, event model.Event) (string, string) {
	categories := map[string]string{}
	for _, skill := range d.Catalog.Skills {
		categories[skill.ID] = skill.Category
	}
	weights := map[string]int{}
	for _, gain := range event.DevelopsSkills {
		category := categories[gain.SkillID]
		if category == "" {
			category = "engineering"
		}
		weights[category] += gain.Gain
	}
	category, weight := "engineering", -1
	for candidate, value := range weights {
		if value > weight || (value == weight && candidate < category) {
			category, weight = candidate, value
		}
	}
	branch := "professional"
	switch category {
	case "communication", "leadership", "collaboration":
		branch = "people"
	case "thinking", "personal_effectiveness":
		branch = "personal"
	}
	return category, branch
}

func moduleGains(d model.Dataset, employee model.Employee, event model.Event) []model.ExpectedGain {
	levels := EffectiveSkills(d, employee)
	trajectory := BuildTrajectory(d, employee, levels)
	required := map[string]int{}
	for _, skill := range trajectory.Skills {
		required[skill.SkillID] = skill.Required
	}
	names := map[string]string{}
	for _, skill := range d.Catalog.Skills {
		names[skill.ID] = skill.Name
	}
	gains := []model.ExpectedGain{}
	for _, gain := range event.DevelopsSkills {
		before := levels[gain.SkillID]
		after := Raised(before, gain)
		if after > before {
			gains = append(gains, model.ExpectedGain{SkillID: gain.SkillID, Name: names[gain.SkillID], Before: before, After: after, Required: required[gain.SkillID]})
		}
	}
	return gains
}

func BuildDevelopment(d model.Dataset, employee model.Employee, locale string) model.Development {
	date := BusinessDate(d)
	result := model.Development{Items: []model.ModuleView{}, Experience: ExperienceFor(d, employee.ID, monthOf(date)), BusinessDate: date, Revision: d.Revision}
	goal := Target(employee)
	levels := EffectiveSkills(d, employee)
	recommended := map[string]bool{}
	ranked := RankCandidatesLocale(d, employee, locale)
	for _, item := range ranked[:min(3, len(ranked))] {
		recommended[item.Event.ID] = true
	}
	configs := map[string]model.ModuleConfig{}
	for _, config := range d.Workflow.Configs {
		configs[config.EventID] = config
	}
	history := EmployeeHistory(d, employee.ID)
	for _, event := range d.Events {
		// Mandatory training is kept in history, outside the voluntary development tree.
		if event.Mandatory {
			continue
		}
		audience := (Contains(event.TargetRoles, employee.Role) && Contains(event.TargetGrades, employee.Grade)) || (Contains(event.TargetRoles, goal.TargetRole) && Contains(event.TargetGrades, goal.TargetGrade))
		var current *model.Enrollment
		for _, enrollment := range d.Workflow.Enrollments {
			if enrollment.EmployeeID != employee.ID || enrollment.EventID != event.ID {
				continue
			}
			// Match StartModule: continue the first unfinished occurrence, including an old revoked result.
			if current != nil && current.State != "completed" {
				continue
			}
			if current == nil || enrollment.State != "completed" || enrollment.BusinessDate > current.BusinessDate || (enrollment.BusinessDate == current.BusinessDate && enrollment.StartedAt > current.StartedAt) || (enrollment.BusinessDate == current.BusinessDate && enrollment.StartedAt == current.StartedAt && enrollment.ID > current.ID) {
				copy := enrollment
				current = &copy
			}
		}
		completed, started := false, false
		for _, activity := range history {
			if activity.EventID == event.ID {
				completed = completed || activity.Status == "completed"
				started = started || activity.Status == "in_progress"
			}
		}
		if !audience && current == nil && !completed && !started {
			continue
		}
		config, exists := configs[event.ID]
		if !exists {
			config = DefaultModuleConfig(d, event)
		}
		category, branch := moduleCategory(d, event)
		item := model.ModuleView{Event: event, Config: config, State: "available", Recommended: recommended[event.ID], Branch: branch, Category: category, ExpectedGains: moduleGains(d, employee, event), Enrollment: current}
		reason := EligibilityReason(d, employee, event, levels)
		if reason != "" {
			item.State = "locked"
			item.BlockedReason = blockedReason(d, event, levels, reason, locale)
		}
		if started && reason == "" {
			item.State = "in_progress"
		}
		if completed && (event.ID != "EV_036" || reason == "completed") {
			item.State, item.BlockedReason = "completed", ""
		}
		if current != nil {
			// An active review remains visible even after the employee changes career goals.
			if current.State != "completed" || event.ID != "EV_036" || current.BusinessDate == date {
				item.State, item.BlockedReason, item.Config = current.State, "", current.Config
			} else {
				item.Enrollment = nil // a repeat on another day creates a new occurrence
			}
		}
		if item.State == "completed" {
			// Do not advertise the same future gain a second time on a completed module.
			item.ExpectedGains = []model.ExpectedGain{}
			item.Recommended = false
		}
		result.Items = append(result.Items, item)
	}
	sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].Event.ID < result.Items[j].Event.ID })
	return result
}

func blockedReason(d model.Dataset, event model.Event, levels map[string]int, reason, locale string) string {
	switch reason {
	case "prerequisites":
		names := map[string]string{}
		for _, skill := range d.Catalog.Skills {
			names[skill.ID] = skill.Name
		}
		missing := []string{}
		for id, required := range event.Prerequisites {
			if levels[id] < required {
				missing = append(missing, fmt.Sprintf("%s: %d → %d", names[id], levels[id], required))
			}
		}
		sort.Strings(missing)
		return Text(locale, "Сначала развейте навыки: ", "Алдымен мына дағдыларды дамытыңыз: ", "Develop these skills first: ") + strings.Join(missing, "; ")
	case "audience":
		return Text(locale, "Модуль рассчитан на другую роль или грейд.", "Модуль басқа рөлге немесе деңгейге арналған.", "This module is intended for a different role or grade.")
	case "schedule":
		return Text(locale, "В каталоге пока нет будущих сессий этого модуля.", "Каталогта бұл модульдің алдағы сессиялары әзірге жоқ.", "No upcoming session is listed for this module yet.")
	case "pending":
		return Text(locale, "Результат уже ожидает проверки HR.", "Нәтиже HR тексеруін күтуде.", "Your outcome is already awaiting HR review.")
	case "completed":
		return Text(locale, "Модуль уже завершён; повтор клуба доступен в другой день.", "Модуль аяқталған; клубты басқа күні қайталауға болады.", "Already completed; the club can be repeated on another day.")
	default:
		return Text(locale, "Модуль пока недоступен.", "Модуль әзірге қолжетімсіз.", "This module is not available yet.")
	}
}
