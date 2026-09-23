// Command demo-fixture creates a small independently authored dataset for public
// screenshots and automated demos. It never reads the organizers' dataset and
// never connects to a database. Employee names and work examples are fictional.
// Run from the repository root:
//
//	go run ./backend/cmd/demo-fixture --output tmp/screenshot-seed
package main

import (
	"careerquest/internal/model"
	"careerquest/internal/store"
	"careerquest/internal/testfixture"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	output := flag.String("output", "tmp/screenshot-seed", "empty output directory for fictional screenshot data")
	flag.Parse()
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, "demo fixture:", err)
		os.Exit(1)
	}
}

func run(output string) error {
	d := screenshotDataset()
	if err := store.Validate(d); err != nil {
		return fmt.Errorf("validate independently authored fixture: %w", err)
	}
	files := []struct {
		name string
		data any
	}{
		{"skills.json", d.Catalog},
		{"employees.json", model.EmployeesFile{Meta: d.Catalog.Meta, Employees: d.Employees}},
		{"events.json", model.EventsFile{Meta: d.Catalog.Meta, Events: d.Events}},
	}
	// Refuse to replace an existing package, including an accidentally selected seed directory.
	for _, name := range []string{"skills.json", "employees.json", "events.json", "activity_history.csv"} {
		if _, err := os.Stat(filepath.Join(output, name)); err == nil {
			return fmt.Errorf("%s already exists; choose a new output directory", filepath.Join(output, name))
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		return err
	}
	for _, item := range files {
		file, err := os.OpenFile(filepath.Join(output, item.name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		encodeErr := encoder.Encode(item.data)
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	file, err := os.OpenFile(filepath.Join(output, "activity_history.csv"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	if err = writer.Write([]string{"record_id", "employee_id", "event_id", "date", "due_date", "status", "completion_pct", "score", "feedback_rating", "assigned_by"}); err != nil {
		file.Close()
		return err
	}
	writer.Flush()
	writeErr, closeErr := writer.Error(), file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	// Exercise the same parser used at startup before reporting a usable seed package.
	loaded, err := store.LoadSeed(output)
	if err != nil {
		return fmt.Errorf("verify generated package: %w", err)
	}
	abs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	fmt.Printf("Fictional screenshot seed: %s\nEmployees: %d; events: %d; skills: %d; history: %d; date: %s\nNo organizer data was read. No database was changed.\n", abs, len(loaded.Employees), len(loaded.Events), len(loaded.Catalog.Skills), len(loaded.History), loaded.Catalog.Meta.AsOfDate)
	return nil
}

func screenshotDataset() model.Dataset {
	d := testfixture.Dataset()
	d.Catalog.Meta = model.Meta{Dataset: "Career Quest — независимый демонстрационный набор", Version: "demo-1", AsOfDate: "2026-10-01"}
	d.Catalog.ProficiencyScale = map[string]string{"0": "Пока не применял", "1": "Знаком с основами", "2": "Решаю задачи с поддержкой", "3": "Применяю самостоятельно", "4": "Разбираю сложные случаи", "5": "Помогаю другим освоить навык"}
	d.Catalog.Skills = []model.Skill{
		{ID: "design", Name: "Проектирование систем", Type: "hard", Category: "engineering", Description: "Выбор компонентов сервиса и объяснение технических компромиссов."},
		{ID: "speech", Name: "Ясное выступление", Type: "soft", Category: "communication", Description: "Понятное объяснение идеи с учётом вопросов аудитории."},
		{ID: "cloud", Name: "Надёжная поставка", Type: "hard", Category: "engineering", Description: "Подготовка безопасного изменения приложения и плана восстановления."},
		{ID: "priorities", Name: "Рабочие приоритеты", Type: "soft", Category: "personal_effectiveness", Description: "Выбор посильного объёма работы и оценка результата."},
	}
	for i := range d.Catalog.RoleProfiles {
		if d.Catalog.RoleProfiles[i].Role == "Backend" {
			d.Catalog.RoleProfiles[i].RequiredSkills["priorities"] = 2
		}
	}
	d.Employees[0].FullName = "Алия Демонстрационная"
	d.Employees[0].Department = "Демонстрационная команда продукта"
	d.Employees[0].Skills["priorities"] = 0
	d.Employees[0].CareerGoal = &model.Goal{TargetRole: "Backend", TargetGrade: "Senior"}
	d.Employees[1].FullName = "Данияр Пример"
	d.Employees[1].Department = "Демонстрационная команда продукта"
	d.Employees[1].PreferredLanguage = "ru"
	d.Employees[1].Skills["priorities"] = 2
	allGrades := []string{"Junior", "Middle", "Senior", "Lead"}
	d.Events = []model.Event{
		{ID: "design-course", Title: "Архитектурный разбор рабочего сервиса", Description: "Выберите небольшой сервис, нарисуйте его основные компоненты и объясните, какое ограничение повлияло на ваше решение. Итог — схема и короткое описание двух рассмотренных вариантов.", Type: "course", Format: "self_paced", DurationHours: 4, TargetRoles: []string{"Backend"}, TargetGrades: allGrades, DevelopsSkills: []model.SkillGain{{SkillID: "design", Gain: 1, MaxLevel: 5}}, Prerequisites: map[string]int{"design": 1}},
		{ID: "EV_036", Title: "Клуб коротких выступлений", Description: "Подготовьте объяснение одной рабочей идеи, выступите перед небольшой группой и соберите вопросы. Для проверки сохраните план выступления и вывод о том, что стало понятнее аудитории.", Type: "meetup", Format: "online", DurationHours: 2, TargetRoles: []string{"Backend", "Analyst"}, TargetGrades: allGrades, DevelopsSkills: []model.SkillGain{{SkillID: "speech", Gain: 1, MaxLevel: 4}}, UpcomingSessions: []string{"2026-10-08", "2026-10-15", "2026-10-22", "2026-11-05"}},
		{ID: "cloud-basics", Title: "Спокойный выпуск новой версии", Description: "Составьте план небольшого выпуска: необходимые проверки, сигнал об ошибке и способ вернуться к предыдущей версии. Обсудите, какой риск удалось уменьшить.", Type: "workshop", Format: "self_paced", DurationHours: 3, TargetRoles: []string{"Backend"}, TargetGrades: allGrades, DevelopsSkills: []model.SkillGain{{SkillID: "cloud", Gain: 1, MaxLevel: 5}}, Prerequisites: map[string]int{"cloud": 2}},
		{ID: "analyst-course", Title: "Оценка технических ограничений", Description: "Разберите пример изменения продукта: отделите предположения от фактов и запишите, какие данные нужны для выбора решения. Итог — краткая записка с ограничениями и следующей проверкой.", Type: "course", Format: "self_paced", DurationHours: 6, TargetRoles: []string{"Analyst"}, TargetGrades: []string{"Middle", "Senior", "Lead"}, DevelopsSkills: []model.SkillGain{{SkillID: "cloud", Gain: 1, MaxLevel: 5}}},
		{ID: "focus-plan", Title: "Неделя осознанных приоритетов", Description: "Выберите три результата на рабочую неделю, обозначьте первый небольшой шаг и резерв времени. После применения плана объясните, что помогло сосредоточиться и что стоит изменить.", Type: "workshop", Format: "self_paced", DurationHours: 2, TargetRoles: []string{"Backend", "Analyst"}, TargetGrades: allGrades, DevelopsSkills: []model.SkillGain{{SkillID: "priorities", Gain: 1, MaxLevel: 5}}},
		{ID: "design-review", Title: "Обсуждение архитектурных компромиссов", Description: "Подготовьте разбор двух архитектурных подходов к одной задаче. Сравните их по поддержке, надёжности и простоте, затем защитите выбранный вариант на совместном разборе.", Type: "workshop", Format: "online", DurationHours: 8, TargetRoles: []string{"Backend"}, TargetGrades: []string{"Middle", "Senior", "Lead"}, DevelopsSkills: []model.SkillGain{{SkillID: "design", Gain: 1, MaxLevel: 5}}, Prerequisites: map[string]int{"design": 3}, UpcomingSessions: []string{"2026-10-20", "2026-11-12"}},
		{ID: "feedback-lab", Title: "Понятная обратная связь", Description: "Перепишите рабочий комментарий так, чтобы в нём были наблюдение, объяснение последствий и конкретное предложение. Проверьте формулировку на вымышленном примере, не раскрывая данные коллег.", Type: "workshop", Format: "self_paced", DurationHours: 3, TargetRoles: []string{"Backend", "Analyst"}, TargetGrades: allGrades, DevelopsSkills: []model.SkillGain{{SkillID: "speech", Gain: 1, MaxLevel: 4}}},
	}
	d.History = []model.Activity{}
	return d
}
