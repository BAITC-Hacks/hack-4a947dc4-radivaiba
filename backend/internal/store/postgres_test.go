package store

import (
	"careerquest/internal/engine"
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Each test creates and drops only its own randomly named database.
// TEST_DATABASE_URL must point to a test server with CREATEDB permission.
func testPostgresURL(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid TEST_DATABASE_URL")
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { admin.Close() })
	var suffix [10]byte
	if _, err = rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	name := "careerquest_test_" + hex.EncodeToString(suffix[:])
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err = admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("cleanup test database: %v", err)
		}
	})
	u.Path = "/" + name
	query := u.Query()
	query.Del("dbname")
	u.RawQuery = query.Encode()
	return u.String()
}

func writeTestSeed(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	d := testfixture.Dataset()
	d.Events[0].Description = "Практический модуль: описание должно пережить PostgreSQL round-trip."
	// Forward manager reference exercises the deferred employee foreign key.
	d.Employees[0].ManagerID = &d.Employees[1].ID
	for name, value := range map[string]any{
		"skills.json":    d.Catalog,
		"employees.json": model.EmployeesFile{Meta: d.Catalog.Meta, Employees: d.Employees},
		"events.json":    model.EventsFile{Meta: d.Catalog.Meta, Events: d.Events},
	} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "activity_history.csv"), []byte("record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func openTestPostgres(t *testing.T, dsn, seed string) *Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, err := OpenPostgres(ctx, dsn, seed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPostgresPersistenceAndIdempotency(t *testing.T) {
	dsn, seed := testPostgresURL(t), writeTestSeed(t)
	s := openTestPostgres(t, dsn, seed)
	d, err := LoadSeed(seed)
	if err != nil {
		t.Fatalf("seed round trip changed the dataset: %v", err)
	}
	if err = compareLegacy(d, s.Snapshot()); err != nil {
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
	if s.Snapshot().Revision != 2 || len(s.Snapshot().History) != 1 {
		t.Fatal("concurrent completion applied more than once")
	}
	employee := d.Employees[0]
	employee.ID = "JURY_PG"
	data, _ := json.Marshal(model.EmployeesFile{Meta: d.Catalog.Meta, Employees: []model.Employee{employee}})
	history := []byte("record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\nPG_RECORD,JURY_PG,EV_036,2026-09-25,2026-10-08,completed,100,95,5,self\n")
	if _, err := s.Import(data, history); err != nil {
		t.Fatal(err)
	}
	if result, err := s.Import(data, history); err != nil || result.EmployeesUpdated != 1 || result.HistoryUpdated != 1 {
		t.Fatalf("repeat import failed: %+v %v", result, err)
	}
	expected := s.Snapshot()
	s.Close()
	// The initialized database must boot with no seed files available.
	reopened := openTestPostgres(t, dsn, filepath.Join(t.TempDir(), "missing-seed"))
	if !reflect.DeepEqual(expected, reopened.Snapshot()) {
		t.Fatal("restart changed persisted fields, ordering, nulls or demo completion")
	}
	profile, changed, err := reopened.Complete("E0001", "design-course")
	if err != nil || changed || profile.EffectiveSkills["design"] != 2 {
		t.Fatalf("completion after restart was not idempotent: changed=%v err=%v", changed, err)
	}
}

func TestPostgresTransactionFailureDoesNotPublish(t *testing.T) {
	dsn := testPostgresURL(t)
	s := openTestPostgres(t, dsn, writeTestSeed(t))
	before := s.Snapshot()
	next := before
	next.Revision++
	next.Employees = append([]model.Employee{}, before.Employees...)
	next.Employees[0].Department = "Must roll back"
	// Bypass app validation to force a database failure after employee writes.
	next.History = []model.Activity{testfixture.Record("bad-fk", "missing-event", "2026-09-25", "completed")}
	if err := s.save(next); err == nil {
		t.Fatal("foreign key violation accepted")
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("failed transaction published a new snapshot")
	}
	reopened := openTestPostgres(t, dsn, "missing-seed")
	if !reflect.DeepEqual(before, reopened.Snapshot()) {
		t.Fatal("failed transaction partially changed the database")
	}
}

func TestPostgresConcurrentStartupAndStaleWriter(t *testing.T) {
	dsn, seed := testPostgresURL(t), writeTestSeed(t)
	type opened struct {
		store *Store
		err   error
	}
	results := make(chan opened, 2)
	for i := 0; i < 2; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s, err := OpenPostgres(ctx, dsn, seed)
			results <- opened{s, err}
		}()
	}
	stores := []*Store{}
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err != nil {
			t.Error(result.err)
			continue
		}
		s := result.store
		t.Cleanup(func() { s.Close() })
		stores = append(stores, s)
	}
	if len(stores) != 2 {
		t.Fatal("concurrent startup failed")
	}
	if _, _, err := stores[0].Complete("E0001", "design-course"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := stores[1].Complete("E0001", "EV_036"); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("stale writer was not rejected: %v", err)
	}
	if stores[1].Snapshot().Revision != 1 {
		t.Fatal("stale writer published a failed write")
	}
	reopened := openTestPostgres(t, dsn, "missing-seed")
	if len(reopened.Snapshot().History) != 1 || reopened.Snapshot().History[0].EventID != "design-course" {
		t.Fatal("stale writer overwrote committed data")
	}
}

func TestPostgresInvalidSeedCanBeRetried(t *testing.T) {
	dsn, seed := testPostgresURL(t), writeTestSeed(t)
	path := filepath.Join(seed, "activity_history.csv")
	if err := os.WriteFile(path, []byte("invalid-header\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if s, err := OpenPostgres(ctx, dsn, seed); err == nil {
		s.Close()
		t.Fatal("invalid seed accepted")
	}
	s := openTestPostgres(t, dsn, writeTestSeed(t))
	if s.Snapshot().Revision != 1 || len(s.Snapshot().Employees) != 2 {
		t.Fatal("failed seed left partial data")
	}
}

func TestPostgresApprovalLedgerRollbackAndLateMonthRestart(t *testing.T) {
	dsn := testPostgresURL(t)
	s := openTestPostgres(t, dsn, writeTestSeed(t))
	credentials, err := s.ProvisionAccounts("", "hr", "")
	if err != nil {
		t.Fatal(err)
	}
	var hrID, employeePassword string
	for _, a := range s.RawSnapshot().Workflow.Accounts {
		if a.Role == "hr" {
			hrID = a.ID
		}
	}
	for _, c := range credentials {
		if c.Login == "E0001" {
			employeePassword = c.Password
		}
	}
	if hrID == "" || employeePassword == "" {
		t.Fatal("provisioned identities missing")
	}
	en, err := s.StartModule("E0001", "design-course")
	if err != nil {
		t.Fatal(err)
	}
	proof, err := s.Submit(en.ID, "E0001", "Documented architecture decision with tradeoffs.", "https://example.com/proof", "pg-proof-request", []model.Attachment{{ID: "AT_PG_PROOF", Filename: "proof.pdf", MediaType: "application/pdf", Size: 100, SHA256: "test-checksum", StorageKey: "AT_PG_PROOF"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetBusinessDate("2026-11-01"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Decide(hrID, proof.Submission.ID, "approve", "Reviewed practical evidence", "pg-approve-request"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(s.RawSnapshot().Workflow.Ledger) != 1 || len(s.RawSnapshot().History) != 1 {
		t.Fatal("parallel retries awarded twice")
	}
	beforeImport := s.RawSnapshot()
	modified := beforeImport.Employees[0]
	modified.Skills = map[string]int{"design": 5, "speech": 0, "cloud": 4}
	badJSON, _ := json.Marshal(model.EmployeesFile{Meta: beforeImport.Catalog.Meta, Employees: []model.Employee{modified}})
	if _, err = s.Import(badJSON, nil); err == nil || s.RawSnapshot().Revision != beforeImport.Revision {
		t.Fatal("import rewrote an assessed baseline with active application progress")
	}
	modified = beforeImport.Employees[0]
	modified.Department = "Updated department after November approval"
	goodJSON, _ := json.Marshal(model.EmployeesFile{Meta: beforeImport.Catalog.Meta, Employees: []model.Employee{modified}})
	if _, err = s.Import(goodJSON, nil); err != nil {
		t.Fatal("valid import rejected linked future application completion", err)
	}
	expected := s.RawSnapshot()
	s.Close()
	reopened := openTestPostgres(t, dsn, "missing-seed")
	if !reflect.DeepEqual(expected, reopened.RawSnapshot()) {
		t.Fatal("workflow fields changed on PostgreSQL restart")
	}
	if _, ok := reopened.Authenticate("E0001", employeePassword); !ok {
		t.Fatal("account hash lost on restart")
	}
	if err = reopened.SetBusinessDate("2026-11-01"); err != nil {
		t.Fatal(err)
	}
	xp := engine.ExperienceFor(reopened.Snapshot(), "E0001", "2026-11")
	if xp.MonthlyEXP != 30 || xp.TotalEXP != 30 || engine.ExperienceFor(reopened.Snapshot(), "E0001", "2026-10").MonthlyEXP != 0 {
		t.Fatalf("late-month reward: %+v", xp)
	}
	storedProof, err := reopened.Submission(proof.Submission.ID)
	if err != nil || len(storedProof.Submission.Attachments) != 1 || storedProof.Submission.Attachments[0].StorageKey != "AT_PG_PROOF" {
		t.Fatal("proof metadata lost", err)
	}
	// Fail after profile/history writes to prove workflow and baseline roll back together.
	next := reopened.nextWorkflow()
	next.Employees = append([]model.Employee{}, next.Employees...)
	next.Employees[0].Department = "This update must roll back"
	next.Workflow.Ledger = append(next.Workflow.Ledger, model.ExpEntry{ID: "XP_INVALID_FK", EmployeeID: "E0001", ApprovalID: "missing-approval", Amount: 1, Month: "2026-11"})
	if err = reopened.save(next); err == nil {
		t.Fatal("invalid workflow reference committed")
	}
	if !reflect.DeepEqual(expected, reopened.RawSnapshot()) {
		t.Fatal("failed transaction published workflow")
	}
	check := openTestPostgres(t, dsn, "missing-seed")
	if !reflect.DeepEqual(expected, check.RawSnapshot()) {
		t.Fatal("failed transaction changed durable baseline/workflow")
	}
	check.Close()
	approvalID := expected.Workflow.Ledger[0].ApprovalID
	if _, err = reopened.Revoke(hrID, approvalID, "Incorrect evidence, requires revision", "pg-revoke-request"); err != nil {
		t.Fatal(err)
	}
	afterRevoke := reopened.RawSnapshot()
	reopened.Close()
	final := openTestPostgres(t, dsn, "missing-seed")
	if !reflect.DeepEqual(afterRevoke, final.RawSnapshot()) {
		t.Fatal("revocation lost on restart")
	}
	if len(final.RawSnapshot().History) != 1 || len(final.RawSnapshot().Workflow.Decisions) != 2 || len(final.RawSnapshot().Workflow.Ledger) != 2 {
		t.Fatal("audit records were deleted")
	}
	if xp = engine.ExperienceFor(final.Snapshot(), "E0001", "2026-11"); xp.TotalEXP != 0 || xp.MonthlyEXP != 0 {
		t.Fatal("revocation did not correct original month", xp)
	}
}
