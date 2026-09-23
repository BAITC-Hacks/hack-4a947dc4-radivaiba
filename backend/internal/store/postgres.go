package store

import (
	"careerquest/internal/model"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrations embed.FS

var ErrStaleRevision = errors.New("database changed in another process; restart this server to reload the dataset")

// OpenPostgres migrates the schema and seeds only an uninitialized database.
// Existing databases are authoritative and do not require the seed files.
func OpenPostgres(ctx context.Context, databaseURL, seedDir string) (*Store, error) {
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		// Parse errors can contain the connection string, including its password.
		return nil, errors.New("invalid DATABASE_URL: expected a PostgreSQL connection string")
	}
	config.ConnectTimeout = 5 * time.Second
	db := stdlib.OpenDB(*config)
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	s := &Store{db: db}
	ready := false
	defer func() {
		if !ready {
			db.Close()
		}
	}()
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	if err = migrate(ctx, db); err != nil {
		return nil, fmt.Errorf("PostgreSQL migration: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Serialize first startup, including the empty-database check and all seed inserts.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731904812)"); err != nil {
		return nil, err
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM careerquest.dataset_metadata)").Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		seed, err := LoadSeed(seedDir)
		if err != nil {
			return nil, fmt.Errorf("initial PostgreSQL seed: %w", err)
		}
		if err = seedPostgres(ctx, tx, seed); err != nil {
			return nil, fmt.Errorf("initial PostgreSQL seed: %w", err)
		}
	}
	// Read all tables from a coherent view while preventing app writes.
	if _, err = tx.ExecContext(ctx, "SELECT revision FROM careerquest.dataset_metadata WHERE singleton FOR UPDATE"); err != nil {
		return nil, err
	}
	s.data, err = loadPostgres(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err = Validate(s.data); err != nil {
		return nil, fmt.Errorf("PostgreSQL dataset: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	ready = true
	return s, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731904812)"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS careerquest;
		CREATE TABLE IF NOT EXISTS careerquest.schema_migrations (
			version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, file := range files {
		var applied bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM careerquest.schema_migrations WHERE version = $1)", file.Name()).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		query, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(query)); err != nil {
			return fmt.Errorf("%s: %w", file.Name(), err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO careerquest.schema_migrations (version) VALUES ($1)", file.Name()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func seedPostgres(ctx context.Context, tx *sql.Tx, d model.Dataset) error {
	meta, err := json.Marshal(d.Catalog.Meta)
	if err != nil {
		return err
	}
	scale, err := json.Marshal(d.Catalog.ProficiencyScale)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO careerquest.dataset_metadata (meta, proficiency_scale, revision)
		VALUES ($1::jsonb, $2::jsonb, $3)`, string(meta), string(scale), d.Revision); err != nil {
		return err
	}
	for _, entry := range []struct {
		table string
		data  any
	}{
		{"skills", d.Catalog.Skills},
		{"role_profiles", d.Catalog.RoleProfiles},
		{"events", d.Events},
		{"employees", d.Employees},
		{"activity_history", d.History},
	} {
		if err = writeRows(ctx, tx, entry.table, entry.data); err != nil {
			return err
		}
	}
	return nil
}

// Table and column names come exclusively from these constants, never from input.
var postgresTables = map[string]struct{ key, columns string }{
	"skills":           {"skill_id", "skill_id,name,type,category,description,position"},
	"role_profiles":    {"role,grade", "role,grade,required_skills,critical_skills,position"},
	"employees":        {"employee_id", "employee_id,full_name,department,role,grade,manager_id,hire_date,tenure_months,work_format,preferred_language,career_goal,skills,last_review_date,position"},
	"events":           {"event_id", "event_id,title,type,format,duration_hours,mandatory,target_roles,target_grades,develops_skills,prerequisites,upcoming_sessions,position"},
	"activity_history": {"record_id", "record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by,demo,position"},
}

func writeRows(ctx context.Context, tx *sql.Tx, table string, rows any) error {
	spec, ok := postgresTables[table]
	if !ok {
		return errors.New("unknown PostgreSQL dataset table")
	}
	data, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	if string(data) == "null" {
		data = []byte("[]")
	}
	value := "value || jsonb_build_object('position', ordinality - 1)"
	if table == "activity_history" {
		value += " || jsonb_build_object('due_date', NULLIF(value->>'due_date', ''), 'demo', COALESCE((value->>'demo')::boolean, false))"
	}
	assignments := []string{}
	for _, column := range strings.Split(spec.columns, ",") {
		assignments = append(assignments, column+" = EXCLUDED."+column)
	}
	query := fmt.Sprintf(`INSERT INTO careerquest.%s AS stored (%s)
		SELECT (jsonb_populate_record(NULL::careerquest.%s, %s)).*
		FROM jsonb_array_elements($1::jsonb) WITH ORDINALITY AS source(value, ordinality)
		ON CONFLICT (%s) DO UPDATE SET %s
		WHERE stored IS DISTINCT FROM EXCLUDED`, table, spec.columns, table, value, spec.key, strings.Join(assignments, ","))
	_, err = tx.ExecContext(ctx, query, string(data))
	if err != nil {
		return fmt.Errorf("write %s: %w", table, err)
	}
	return nil
}

func readRows[T any](ctx context.Context, tx *sql.Tx, table string) ([]T, error) {
	if _, ok := postgresTables[table]; !ok {
		return nil, errors.New("unknown PostgreSQL dataset table")
	}
	rows, err := tx.QueryContext(ctx, "SELECT to_jsonb(t) FROM careerquest."+table+" AS t ORDER BY position")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []T{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func loadPostgres(ctx context.Context, tx *sql.Tx) (model.Dataset, error) {
	var d model.Dataset
	var meta, scale []byte
	err := tx.QueryRowContext(ctx, "SELECT meta, proficiency_scale, revision FROM careerquest.dataset_metadata WHERE singleton").Scan(&meta, &scale, &d.Revision)
	if err != nil {
		return d, err
	}
	if err = json.Unmarshal(meta, &d.Catalog.Meta); err != nil {
		return d, err
	}
	if err = json.Unmarshal(scale, &d.Catalog.ProficiencyScale); err != nil {
		return d, err
	}
	if d.Catalog.Skills, err = readRows[model.Skill](ctx, tx, "skills"); err != nil {
		return d, err
	}
	if d.Catalog.RoleProfiles, err = readRows[model.RoleProfile](ctx, tx, "role_profiles"); err != nil {
		return d, err
	}
	if d.Employees, err = readRows[model.Employee](ctx, tx, "employees"); err != nil {
		return d, err
	}
	if d.Events, err = readRows[model.Event](ctx, tx, "events"); err != nil {
		return d, err
	}
	d.History, err = readRows[model.Activity](ctx, tx, "activity_history")
	return d, err
}

func (s *Store) savePostgres(next model.Dataset) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// This both serializes writes and rejects a stale process without losing committed data.
	result, err := tx.ExecContext(ctx, `UPDATE careerquest.dataset_metadata SET revision = $1
		WHERE singleton AND revision = $2`, next.Revision, s.data.Revision)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrStaleRevision
	}
	if err = writeRows(ctx, tx, "employees", next.Employees); err != nil {
		return err
	}
	if err = writeRows(ctx, tx, "activity_history", next.History); err != nil {
		return err
	}
	return tx.Commit()
}
