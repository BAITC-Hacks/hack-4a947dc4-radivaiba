package store

import (
	"careerquest/internal/model"
	"fmt"
	"os"
	"path/filepath"
)

// LoadSeed validates the four source files without reading or changing runtime storage.
func LoadSeed(seedDir string) (model.Dataset, error) {
	d := model.Dataset{Revision: 1}
	read := func(name string, out any) error {
		b, err := os.ReadFile(filepath.Join(seedDir, name))
		if err != nil {
			return err
		}
		if err = DecodeJSON(b, out); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if err := read("skills.json", &d.Catalog); err != nil {
		return d, err
	}
	var employees model.EmployeesFile
	var events model.EventsFile
	if err := read("employees.json", &employees); err != nil {
		return d, err
	}
	if err := read("events.json", &events); err != nil {
		return d, err
	}
	if employees.Meta.AsOfDate != d.Catalog.Meta.AsOfDate || events.Meta.AsOfDate != d.Catalog.Meta.AsOfDate {
		return d, fmt.Errorf("dataset meta.as_of_date values must match")
	}
	d.Employees = employees.Employees
	d.Events = events.Events
	b, err := os.ReadFile(filepath.Join(seedDir, "activity_history.csv"))
	if err != nil {
		return d, err
	}
	d.History, err = ParseHistory(b)
	if err != nil {
		return d, err
	}
	return d, Validate(d)
}
