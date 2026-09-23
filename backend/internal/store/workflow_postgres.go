package store

import (
	"careerquest/internal/model"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func writeWorkflow(ctx context.Context, tx *sql.Tx, w model.Workflow) error {
	put := func(query string, payload any, args ...any) error {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		args = append(args, string(raw))
		_, err = tx.ExecContext(ctx, query, args...)
		return err
	}
	for i, v := range w.Accounts {
		if err := put(`INSERT INTO careerquest.accounts(id,login,employee_id,role,position,data) VALUES($1,$2,$3,$4,$5,$6::jsonb) ON CONFLICT(id) DO UPDATE SET data=EXCLUDED.data,login=EXCLUDED.login,employee_id=EXCLUDED.employee_id,role=EXCLUDED.role`, v, v.ID, v.Login, nullable(v.EmployeeID), v.Role, i); err != nil {
			return err
		}
	}
	for i, v := range w.Preferences {
		if err := put(`INSERT INTO careerquest.employee_preferences(employee_id,position,data) VALUES($1,$2,$3::jsonb) ON CONFLICT(employee_id) DO UPDATE SET data=EXCLUDED.data`, v, v.EmployeeID, i); err != nil {
			return err
		}
	}
	for i, v := range w.Configs {
		if err := put(`INSERT INTO careerquest.module_config(event_id,version,position,data) VALUES($1,$2,$3,$4::jsonb) ON CONFLICT(event_id,version) DO NOTHING`, v, v.EventID, v.Version, i); err != nil {
			return err
		}
	}
	for i, v := range w.Enrollments {
		if err := put(`INSERT INTO careerquest.enrollments(id,employee_id,event_id,occurrence,position,data) VALUES($1,$2,$3,$4,$5,$6::jsonb) ON CONFLICT(id) DO UPDATE SET data=EXCLUDED.data`, v, v.ID, v.EmployeeID, v.EventID, v.Occurrence, i); err != nil {
			return err
		}
	}
	for i, v := range w.Submissions {
		if err := put(`INSERT INTO careerquest.submission_versions(id,enrollment_id,version,position,data) VALUES($1,$2,$3,$4,$5::jsonb) ON CONFLICT(id) DO UPDATE SET data=EXCLUDED.data`, v, v.ID, v.EnrollmentID, v.Version, i); err != nil {
			return err
		}
		for j, a := range v.Attachments {
			if err := put(`INSERT INTO careerquest.attachments(id,submission_id,position,data) VALUES($1,$2,$3,$4::jsonb) ON CONFLICT(id) DO NOTHING`, a, a.ID, v.ID, i*3+j); err != nil {
				return err
			}
		}
	}
	for i, v := range w.Decisions {
		if err := put(`INSERT INTO careerquest.review_decisions(id,submission_id,reviewer_id,action,request_key,reversal_of,completion_id,position,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb) ON CONFLICT(id) DO NOTHING`, v, v.ID, v.SubmissionID, v.ReviewerID, v.Action, v.RequestKey, nullable(v.ReversalOf), nullable(v.CompletionID), i); err != nil {
			return err
		}
	}
	for i, v := range w.Ledger {
		if err := put(`INSERT INTO careerquest.exp_ledger(id,employee_id,approval_id,amount,month,reversal_of,position,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb) ON CONFLICT(id) DO NOTHING`, v, v.ID, v.EmployeeID, v.ApprovalID, v.Amount, v.Month, nullable(v.ReversalOf), i); err != nil {
			return err
		}
	}
	return nil
}
func loadWorkflow(ctx context.Context, tx *sql.Tx) (model.Workflow, error) {
	var w model.Workflow
	entries := []struct {
		table string
		dest  any
	}{{"accounts", &w.Accounts}, {"employee_preferences", &w.Preferences}, {"module_config", &w.Configs}, {"enrollments", &w.Enrollments}, {"submission_versions", &w.Submissions}, {"review_decisions", &w.Decisions}, {"exp_ledger", &w.Ledger}}
	for _, e := range entries {
		// Names above are fixed server constants, never request data.
		var raw []byte
		if err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT COALESCE(jsonb_agg(data ORDER BY position),'[]'::jsonb) FROM careerquest.%s`, e.table)).Scan(&raw); err != nil {
			return w, err
		}
		if string(raw) == "[]" {
			continue
		}
		if err := json.Unmarshal(raw, e.dest); err != nil {
			return w, err
		}
	}
	return w, nil
}
