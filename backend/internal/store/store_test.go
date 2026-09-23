package store

import (
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentCompletionIdempotentAndPersistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := New(testfixture.Dataset(), path)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := s.Complete("E0001", "design-course"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(s.Snapshot().History) != 1 || s.Snapshot().Revision != 2 {
		t.Fatalf("double completion: %+v", s.Snapshot().History)
	}
	loaded, err := Open("does-not-exist", path)
	if err != nil {
		t.Fatal(err)
	}
	profile, changed, err := loaded.Complete("E0001", "design-course")
	if err != nil || changed || profile.EffectiveSkills["design"] != 2 {
		t.Fatalf("restart/idempotency failed: %+v %v", profile, err)
	}
}
func TestSaveFailureDoesNotPublish(t *testing.T) {
	dir := t.TempDir()
	block := filepath.Join(dir, "file")
	if err := os.WriteFile(block, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	s, _ := New(testfixture.Dataset(), filepath.Join(block, "state.json"))
	if _, _, err := s.Complete("E0001", "design-course"); err == nil {
		t.Fatal("expected failed persistence")
	}
	if len(s.Snapshot().History) != 0 {
		t.Fatal("failed save published mutation")
	}
}
func TestImportAtomicRepeatableAndNewProfile(t *testing.T) {
	s, _ := New(testfixture.Dataset(), filepath.Join(t.TempDir(), "state.json"))
	d := s.Snapshot()
	e := d.Employees[0]
	e.ID = "JURY_42"
	e.FullName = "Additional Profile"
	data, _ := json.Marshal(model.EmployeesFile{Meta: d.Catalog.Meta, Employees: []model.Employee{e}})
	badCSV := []byte("record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\nNEW,JURY_42,unknown,2026-09-25,,completed,100,,,self\n")
	if _, err := s.Import(data, badCSV); err == nil {
		t.Fatal("invalid event accepted")
	}
	if len(s.Snapshot().Employees) != 2 || s.Snapshot().Revision != 1 {
		t.Fatal("partial import applied")
	}
	goodCSV := []byte("record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\nNEW,JURY_42,design-course,2026-09-25,,completed,100,95,5,self\n")
	r, err := s.Import(data, goodCSV)
	if err != nil || r.EmployeesAdded != 1 || r.HistoryAdded != 1 {
		t.Fatalf("valid import failed: %+v %v", r, err)
	}
	r, err = s.Import(data, goodCSV)
	if err != nil || len(s.Snapshot().Employees) != 3 || len(s.Snapshot().History) != 1 || r.EmployeesUpdated != 1 {
		t.Fatalf("repeat import failed %+v %v", r, err)
	}
	badJSON := []byte(`{"meta":{"as_of_date":"2026-10-01"},"employees":[]}`)
	if _, err = s.Import(badJSON, nil); err == nil {
		t.Fatal("empty employees accepted")
	}
}
func TestCSVValidation(t *testing.T) {
	if _, err := ParseHistory([]byte("record_id,employee_id\na,b\n")); err == nil {
		t.Fatal("missing columns accepted")
	}
	d := testfixture.Dataset()
	d.History = []model.Activity{testfixture.Record("bad", "design-course", "2026-10-10", "completed")}
	if err := Validate(d); err == nil {
		t.Fatal("future history accepted")
	}
	d = testfixture.Dataset()
	d.Employees[0].Skills["unknown"] = 2
	if err := Validate(d); err == nil {
		t.Fatal("unknown skill accepted")
	}
}
func TestCompletionOnAssessmentDate(t *testing.T) {
	d := testfixture.Dataset()
	d.Employees[0].LastReviewDate = d.Catalog.Meta.AsOfDate
	s, err := New(d, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, _, err := s.Complete("E0001", "design-course")
	if err != nil || p.EffectiveSkills["design"] != 2 {
		t.Fatalf("new demo completion lost on same-day assessment: %v %+v", err, p)
	}
}
