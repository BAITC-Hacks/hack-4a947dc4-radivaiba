CREATE TABLE careerquest.accounts (
 id text PRIMARY KEY, login text NOT NULL UNIQUE, employee_id text REFERENCES careerquest.employees(employee_id),
 role text NOT NULL CHECK (role IN ('employee','hr')), data jsonb NOT NULL, position integer NOT NULL,
 CHECK (role <> 'employee' OR employee_id IS NOT NULL)
);
CREATE UNIQUE INDEX employee_account_unique ON careerquest.accounts(employee_id) WHERE role='employee';
CREATE TABLE careerquest.employee_preferences (employee_id text PRIMARY KEY REFERENCES careerquest.employees(employee_id), data jsonb NOT NULL, position integer NOT NULL);
CREATE TABLE careerquest.module_config (event_id text NOT NULL REFERENCES careerquest.events(event_id), version integer NOT NULL, data jsonb NOT NULL, position integer NOT NULL, PRIMARY KEY(event_id,version));
CREATE TABLE careerquest.enrollments (
 id text PRIMARY KEY, employee_id text NOT NULL REFERENCES careerquest.employees(employee_id), event_id text NOT NULL REFERENCES careerquest.events(event_id),
 occurrence text NOT NULL, data jsonb NOT NULL, position integer NOT NULL, UNIQUE(employee_id,event_id,occurrence)
);
CREATE TABLE careerquest.submission_versions (
 id text PRIMARY KEY, enrollment_id text NOT NULL REFERENCES careerquest.enrollments(id), version integer NOT NULL,
 data jsonb NOT NULL, position integer NOT NULL, UNIQUE(enrollment_id,version)
);
CREATE TABLE careerquest.attachments (id text PRIMARY KEY, submission_id text NOT NULL REFERENCES careerquest.submission_versions(id), data jsonb NOT NULL, position integer NOT NULL);
CREATE TABLE careerquest.review_decisions (
 id text PRIMARY KEY, submission_id text NOT NULL REFERENCES careerquest.submission_versions(id), reviewer_id text NOT NULL REFERENCES careerquest.accounts(id),
 action text NOT NULL CHECK(action IN ('approve','return','revoke')), request_key text NOT NULL UNIQUE, reversal_of text UNIQUE REFERENCES careerquest.review_decisions(id),
 completion_id text REFERENCES careerquest.activity_history(record_id), data jsonb NOT NULL, position integer NOT NULL
);
CREATE UNIQUE INDEX decision_per_submission ON careerquest.review_decisions(submission_id) WHERE action IN ('approve','return');
CREATE TABLE careerquest.exp_ledger (
 id text PRIMARY KEY, employee_id text NOT NULL REFERENCES careerquest.employees(employee_id), approval_id text NOT NULL UNIQUE REFERENCES careerquest.review_decisions(id),
 amount integer NOT NULL, month text NOT NULL CHECK(month ~ '^[0-9]{4}-[0-9]{2}$'), reversal_of text UNIQUE REFERENCES careerquest.exp_ledger(id),
 data jsonb NOT NULL, position integer NOT NULL
);
CREATE INDEX enrollment_employee_idx ON careerquest.enrollments(employee_id);
CREATE INDEX ledger_employee_month_idx ON careerquest.exp_ledger(employee_id,month);
