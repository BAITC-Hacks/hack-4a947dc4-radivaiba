CREATE TABLE careerquest.dataset_metadata (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    meta jsonb NOT NULL,
    proficiency_scale jsonb NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0)
);

CREATE TABLE careerquest.skills (
    skill_id text PRIMARY KEY,
    name text NOT NULL,
    type text NOT NULL CHECK (type IN ('hard', 'soft')),
    category text NOT NULL,
    description text NOT NULL,
    position integer NOT NULL
);

CREATE TABLE careerquest.role_profiles (
    role text NOT NULL,
    grade text NOT NULL CHECK (grade IN ('Junior', 'Middle', 'Senior', 'Lead')),
    required_skills jsonb NOT NULL CHECK (jsonb_typeof(required_skills) = 'object'),
    critical_skills jsonb,
    position integer NOT NULL,
    PRIMARY KEY (role, grade)
);

CREATE TABLE careerquest.employees (
    employee_id text PRIMARY KEY,
    full_name text NOT NULL,
    department text NOT NULL,
    role text NOT NULL,
    grade text NOT NULL,
    manager_id text REFERENCES careerquest.employees(employee_id) DEFERRABLE INITIALLY DEFERRED,
    hire_date date NOT NULL,
    tenure_months integer NOT NULL CHECK (tenure_months >= 0),
    work_format text NOT NULL CHECK (work_format IN ('office', 'hybrid', 'remote')),
    preferred_language text NOT NULL CHECK (preferred_language IN ('ru', 'en', 'kk')),
    career_goal jsonb,
    skills jsonb NOT NULL CHECK (jsonb_typeof(skills) = 'object'),
    last_review_date date NOT NULL CHECK (last_review_date >= hire_date),
    position integer NOT NULL,
    FOREIGN KEY (role, grade) REFERENCES careerquest.role_profiles(role, grade),
    CHECK (manager_id IS NULL OR manager_id <> employee_id)
);

CREATE TABLE careerquest.events (
    event_id text PRIMARY KEY,
    title text NOT NULL,
    type text NOT NULL CHECK (type IN ('compliance', 'onboarding', 'course', 'workshop', 'mentoring', 'certification', 'meetup')),
    format text NOT NULL CHECK (format IN ('online', 'offline', 'self_paced')),
    duration_hours double precision NOT NULL CHECK (duration_hours > 0),
    mandatory boolean NOT NULL,
    target_roles jsonb,
    target_grades jsonb,
    develops_skills jsonb,
    prerequisites jsonb,
    upcoming_sessions jsonb,
    position integer NOT NULL
);

CREATE TABLE careerquest.activity_history (
    record_id text PRIMARY KEY,
    employee_id text NOT NULL REFERENCES careerquest.employees(employee_id),
    event_id text NOT NULL REFERENCES careerquest.events(event_id),
    date date NOT NULL,
    due_date date,
    status text NOT NULL CHECK (status IN ('completed', 'in_progress', 'dropped', 'no_show', 'declined', 'overdue')),
    completion_pct integer NOT NULL CHECK (completion_pct BETWEEN 0 AND 100),
    score integer CHECK (score BETWEEN 0 AND 100),
    feedback_rating integer CHECK (feedback_rating BETWEEN 1 AND 5),
    assigned_by text NOT NULL CHECK (assigned_by IN ('self', 'manager', 'hr')),
    demo boolean NOT NULL DEFAULT false,
    position integer NOT NULL,
    CHECK ((status = 'completed' AND completion_pct = 100)
        OR (status IN ('no_show', 'declined') AND completion_pct = 0)
        OR (status IN ('in_progress', 'overdue') AND completion_pct BETWEEN 0 AND 95)
        OR (status = 'dropped' AND completion_pct BETWEEN 5 AND 95))
);

CREATE INDEX employees_department_idx ON careerquest.employees(department);
CREATE INDEX employees_manager_idx ON careerquest.employees(manager_id);
CREATE INDEX history_employee_date_idx ON careerquest.activity_history(employee_id, date, record_id);
CREATE INDEX history_event_idx ON careerquest.activity_history(event_id);
