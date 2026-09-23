package store

import (
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestCompareLegacyDetectsFieldLoss(t *testing.T) {
	want := testfixture.Dataset()
	want.Events[0].Description = "Important existing description"
	got := want
	got.Events = append([]model.Event{}, want.Events...)
	got.Events[0].Description = ""
	if err := compareLegacy(want, got); err == nil {
		t.Fatal("same counts concealed a lost event description")
	}
}

func TestPostgresLegacyDryRunApplyAndIdempotency(t *testing.T) {
	dsn := testPostgresURL(t)
	d := testfixture.Dataset()
	d.Revision = 7
	d.Events[0].Description = "Full legacy catalog description"
	d.Employees[0].Department = "Imported legacy department"
	r := testfixture.Record("DEMO_LEGACY", "design-course", "2026-10-01", "completed")
	r.Demo = true
	d.History = []model.Activity{r}
	path := filepath.Join(t.TempDir(), "state.json")
	raw, _ := json.Marshal(d)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	report, err := MigrateLegacy(ctx, dsn, path, false)
	if err != nil || report.Applied || report.AlreadyImported || report.DemoCompletions != 1 {
		t.Fatalf("dry-run: %+v %v", report, err)
	}
	// An untouched target has no application schema, even after a dry-run.
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*config)
	defer db.Close()
	var hasSchema bool
	if err = db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='careerquest')").Scan(&hasSchema); err != nil || hasSchema {
		t.Fatalf("dry-run wrote schema: %v %v", hasSchema, err)
	}
	report, err = MigrateLegacy(ctx, dsn, path, true)
	if err != nil || !report.Applied || report.Revision != 7 {
		t.Fatalf("apply: %+v %v", report, err)
	}
	s := openTestPostgres(t, dsn, "missing-seed")
	if err = compareLegacy(d, s.Snapshot()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Import([]byte(`{"meta":{"as_of_date":"2026-10-01"},"employees":[]}`), nil); err == nil {
		t.Fatal("invalid import accepted")
	}
	for _, apply := range []bool{false, true} {
		report, err = MigrateLegacy(ctx, dsn, path, apply)
		if err != nil || !report.AlreadyImported || report.Applied {
			t.Fatalf("repeat: %+v %v", report, err)
		}
	}
	changed := append(append([]byte{}, raw...), '\n')
	if err = os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = MigrateLegacy(ctx, dsn, path, true); err == nil {
		t.Fatal("different checksum overwrote an initialized database")
	}
	if err = compareLegacy(d, openTestPostgres(t, dsn, "missing-seed").Snapshot()); err != nil {
		t.Fatal(err)
	}
}
