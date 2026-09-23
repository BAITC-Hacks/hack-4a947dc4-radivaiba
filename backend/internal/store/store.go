package store

import (
	"careerquest/internal/engine"
	"careerquest/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var ErrNotFound = errors.New("not found")
var ErrNotEligible = errors.New("activity is not an available recommended step")

type Store struct {
	mu   sync.RWMutex
	data model.Dataset
	path string
}
type ImportResult struct {
	EmployeesAdded   int   `json:"employees_added"`
	EmployeesUpdated int   `json:"employees_updated"`
	HistoryAdded     int   `json:"history_added"`
	HistoryUpdated   int   `json:"history_updated"`
	Revision         int64 `json:"revision"`
}

func Open(seedDir, statePath string) (*Store, error) {
	s := &Store{path: statePath}
	if data, err := os.ReadFile(statePath); err == nil {
		if err = DecodeJSON(data, &s.data); err != nil {
			return nil, fmt.Errorf("state file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	} else {
		read := func(name string, out any) error {
			b, e := os.ReadFile(filepath.Join(seedDir, name))
			if e != nil {
				return e
			}
			if e = DecodeJSON(b, out); e != nil {
				return fmt.Errorf("%s: %w", name, e)
			}
			return nil
		}
		if err = read("skills.json", &s.data.Catalog); err != nil {
			return nil, err
		}
		var employees model.EmployeesFile
		var events model.EventsFile
		if err = read("employees.json", &employees); err != nil {
			return nil, err
		}
		if err = read("events.json", &events); err != nil {
			return nil, err
		}
		if employees.Meta.AsOfDate != s.data.Catalog.Meta.AsOfDate || events.Meta.AsOfDate != s.data.Catalog.Meta.AsOfDate {
			return nil, fmt.Errorf("dataset meta.as_of_date values must match")
		}
		s.data.Employees = employees.Employees
		s.data.Events = events.Events
		b, err := os.ReadFile(filepath.Join(seedDir, "activity_history.csv"))
		if err != nil {
			return nil, err
		}
		s.data.History, err = ParseHistory(b)
		if err != nil {
			return nil, err
		}
		s.data.Revision = 1
	}
	if err := Validate(s.data); err != nil {
		return nil, err
	}
	return s, nil
}
func New(d model.Dataset, path string) (*Store, error) {
	if err := Validate(d); err != nil {
		return nil, err
	}
	return &Store{data: d, path: path}, nil
}

// Datasets are immutable after publication. Mutations replace slices instead of changing shared maps.
func (s *Store) Snapshot() model.Dataset { s.mu.RLock(); defer s.mu.RUnlock(); return s.data }
func (s *Store) save(next model.Dataset) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".state-*.json")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err = json.NewEncoder(f).Encode(next); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(temp, s.path); err != nil {
		return err
	}
	s.data = next
	return nil
}
func (s *Store) Complete(employeeID, eventID string) (model.Profile, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	employee, ok := engine.FindEmployee(s.data, employeeID)
	if !ok {
		return model.Profile{}, false, ErrNotFound
	}
	event, ok := engine.FindEvent(s.data, eventID)
	if !ok {
		return model.Profile{}, false, ErrNotFound
	}
	for _, r := range engine.EmployeeHistory(s.data, employeeID) {
		if r.EventID == eventID && r.Status == "completed" && (eventID != "EV_036" || r.Date == s.data.Catalog.Meta.AsOfDate) {
			return engine.BuildProfile(s.data, employee), false, nil
		}
	}
	allowed := false
	for _, candidate := range engine.RankCandidates(s.data, employee) {
		if candidate.Event.ID == event.ID {
			allowed = true
			break
		}
	}
	if !allowed {
		return model.Profile{}, false, ErrNotEligible
	}
	next := s.data
	next.History = append([]model.Activity{}, s.data.History...)
	next.Revision++
	updated := false
	for i, r := range next.History {
		if r.EmployeeID == employeeID && r.EventID == eventID && r.Status == "in_progress" {
			next.History[i].Status = "completed"
			next.History[i].CompletionPct = 100
			next.History[i].Date = next.Catalog.Meta.AsOfDate
			next.History[i].Demo = true
			updated = true
			break
		}
	}
	if !updated {
		id := fmt.Sprintf("DEMO_%s_%s_%s", employeeID, eventID, next.Catalog.Meta.AsOfDate)
		for _, r := range next.History {
			if r.ID == id {
				return model.Profile{}, false, fmt.Errorf("demo record ID collision")
			}
		}
		next.History = append(next.History, model.Activity{ID: id, EmployeeID: employeeID, EventID: eventID, Date: next.Catalog.Meta.AsOfDate, Status: "completed", CompletionPct: 100, AssignedBy: "self", Demo: true})
	}
	if err := s.save(next); err != nil {
		return model.Profile{}, false, err
	}
	return engine.BuildProfile(next, employee), true, nil
}
func (s *Store) Import(employeeJSON, historyCSV []byte) (ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := ImportResult{}
	next := s.data
	if len(employeeJSON) == 0 && len(historyCSV) == 0 {
		return result, fmt.Errorf("upload employees JSON and/or history CSV")
	}
	if len(employeeJSON) > 0 {
		var file model.EmployeesFile
		if err := DecodeJSON(employeeJSON, &file); err != nil {
			return result, fmt.Errorf("employees.json: %w", err)
		}
		if file.Meta.AsOfDate != next.Catalog.Meta.AsOfDate {
			return result, fmt.Errorf("employees.json: meta.as_of_date must be %s", next.Catalog.Meta.AsOfDate)
		}
		if len(file.Employees) == 0 {
			return result, fmt.Errorf("employees.json: employees must not be empty")
		}
		seen := map[string]bool{}
		byID := map[string]model.Employee{}
		for _, e := range next.Employees {
			byID[e.ID] = e
		}
		for _, e := range file.Employees {
			if err := uniqueID(seen, e.ID, "employees.json import"); err != nil {
				return result, err
			}
			if _, ok := byID[e.ID]; ok {
				result.EmployeesUpdated++
			} else {
				result.EmployeesAdded++
			}
			byID[e.ID] = e
		}
		next.Employees = make([]model.Employee, 0, len(byID))
		for _, e := range byID {
			next.Employees = append(next.Employees, e)
		}
		sort.Slice(next.Employees, func(i, j int) bool { return next.Employees[i].ID < next.Employees[j].ID })
	}
	if len(historyCSV) > 0 {
		records, err := ParseHistory(historyCSV)
		if err != nil {
			return result, err
		}
		seen := map[string]bool{}
		byID := map[string]model.Activity{}
		for _, r := range next.History {
			byID[r.ID] = r
		}
		for _, r := range records {
			if err := uniqueID(seen, r.ID, "activity_history.csv import"); err != nil {
				return result, err
			}
			if strings.HasPrefix(r.ID, "DEMO_") {
				return result, fmt.Errorf("activity_history.csv %s: DEMO_ prefix is reserved", r.ID)
			}
			if _, ok := byID[r.ID]; ok {
				result.HistoryUpdated++
			} else {
				result.HistoryAdded++
			}
			byID[r.ID] = r
		}
		next.History = make([]model.Activity, 0, len(byID))
		for _, r := range byID {
			next.History = append(next.History, r)
		}
		sort.Slice(next.History, func(i, j int) bool {
			if next.History[i].Date == next.History[j].Date {
				return next.History[i].ID < next.History[j].ID
			}
			return next.History[i].Date < next.History[j].Date
		})
	}
	if err := Validate(next); err != nil {
		return ImportResult{}, err
	}
	next.Revision++
	if err := s.save(next); err != nil {
		return ImportResult{}, err
	}
	result.Revision = next.Revision
	return result, nil
}
