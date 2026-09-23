package store

import (
	"bytes"
	"careerquest/internal/model"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

type LegacyReport struct {
	SHA256          string `json:"sha256"`
	Revision        int64  `json:"revision"`
	Employees       int    `json:"employees"`
	Events          int    `json:"events"`
	Skills          int    `json:"skills"`
	History         int    `json:"history"`
	DemoCompletions int    `json:"demo_completions"`
	AlreadyImported bool   `json:"already_imported"`
	Applied         bool   `json:"applied"`
}

// MigrateLegacy preserves the complete runtime snapshot, including application
// completions not present in the organizer seed. Dry-run never changes the DB.
// Apply deliberately refuses an initialized database; it is not a merge tool.
func MigrateLegacy(ctx context.Context, databaseURL, statePath string, apply bool) (LegacyReport, error) {
	var report LegacyReport
	raw, err := os.ReadFile(statePath)
	if err != nil {
		return report, fmt.Errorf("read legacy snapshot: %w", err)
	}
	var legacy model.Dataset
	if err = DecodeJSON(raw, &legacy); err != nil {
		return report, fmt.Errorf("decode legacy snapshot: %w", err)
	}
	if err = Validate(legacy); err != nil {
		return report, fmt.Errorf("validate legacy snapshot: %w", err)
	}
	if legacy.Revision < 1 {
		return report, errors.New("legacy revision must be positive")
	}
	hash := sha256.Sum256(raw)
	report = LegacyReport{SHA256: hex.EncodeToString(hash[:]), Revision: legacy.Revision, Employees: len(legacy.Employees), Events: len(legacy.Events), Skills: len(legacy.Catalog.Skills), History: len(legacy.History)}
	for _, activity := range legacy.History {
		if activity.Demo && activity.Status == "completed" {
			report.DemoCompletions++
		}
	}
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return report, errors.New("invalid DATABASE_URL")
	}
	config.ConnectTimeout = 5 * time.Second
	db := stdlib.OpenDB(*config)
	defer db.Close()
	if err = db.PingContext(ctx); err != nil {
		return report, fmt.Errorf("connect to migration target: %w", err)
	}
	// Reject a populated target before even applying schema migrations.
	preflight, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, err
	}
	report.AlreadyImported, err = legacyTargetState(ctx, preflight, report.SHA256)
	preflight.Rollback()
	if err != nil || report.AlreadyImported || !apply {
		return report, err
	}
	// Schema creation is allowed only for --apply. Imported records and the
	// checksum marker are committed together in the following transaction.
	if err = migrate(ctx, db); err != nil {
		return report, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731904812)"); err != nil {
		return report, err
	}
	report.AlreadyImported, err = legacyTargetState(ctx, tx, report.SHA256)
	if err != nil || report.AlreadyImported {
		return report, err
	}
	if err = seedPostgres(ctx, tx, legacy); err != nil {
		return report, fmt.Errorf("import legacy snapshot: %w", err)
	}
	loaded, err := loadPostgres(ctx, tx)
	if err != nil {
		return report, err
	}
	if err = compareLegacy(legacy, loaded); err != nil {
		return report, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO careerquest.legacy_imports (source_sha256, revision) VALUES ($1, $2)", report.SHA256, legacy.Revision); err != nil {
		return report, err
	}
	if err = tx.Commit(); err != nil {
		return report, err
	}
	report.Applied = true
	return report, nil
}

func legacyTargetState(ctx context.Context, tx *sql.Tx, checksum string) (bool, error) {
	var hasImports bool
	if err := tx.QueryRowContext(ctx, "SELECT to_regclass('careerquest.legacy_imports') IS NOT NULL").Scan(&hasImports); err != nil {
		return false, err
	}
	if hasImports {
		var imported bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM careerquest.legacy_imports WHERE source_sha256=$1)", checksum).Scan(&imported); err != nil {
			return false, err
		}
		if imported {
			return true, nil
		}
	}
	for _, table := range []string{"dataset_metadata", "employees", "events", "skills", "role_profiles", "activity_history"} {
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", "careerquest."+table).Scan(&exists); err != nil {
			return false, err
		}
		if !exists {
			continue
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM careerquest."+table).Scan(&count); err != nil {
			return false, err
		}
		if count > 0 {
			return false, fmt.Errorf("target database is not empty (%s has %d records); refusing to overwrite it", table, count)
		}
	}
	return false, nil
}

func compareLegacy(want, got model.Dataset) error {
	// Compare every persisted field, not just counts. BusinessDate is a derived
	// runtime view and is not a field in the original JSON snapshot.
	for _, section := range []struct {
		name string
		a, b any
	}{
		{"catalog", want.Catalog, got.Catalog}, {"employees", want.Employees, got.Employees},
		{"events", want.Events, got.Events}, {"history", want.History, got.History},
		{"revision", want.Revision, got.Revision},
		{"workflow", want.Workflow, got.Workflow},
	} {
		a, err := json.Marshal(section.a)
		if err != nil {
			return err
		}
		b, err := json.Marshal(section.b)
		if err != nil {
			return err
		}
		if !bytes.Equal(a, b) {
			return fmt.Errorf("legacy verification failed: %s did not round-trip exactly", section.name)
		}
	}
	return nil
}
