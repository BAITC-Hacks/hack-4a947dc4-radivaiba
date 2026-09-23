export interface Session {
  role: 'employee' | 'hr';
  employee_id: string;
}
export interface EmployeeSummary {
  employee_id: string;
  full_name: string;
  department: string;
  role: string;
  grade: string;
}
export interface Goal {
  target_role: string;
  target_grade: string;
}
export interface Employee extends EmployeeSummary {
  hire_date: string;
  tenure_months: number;
  work_format: string;
  preferred_language: string;
  career_goal: Goal | null;
  last_review_date: string;
  skills: Record<string, number>;
}
export interface SkillProgress {
  skill_id: string;
  name: string;
  type: string;
  current: number;
  required: number;
  gap: number;
  critical: boolean;
}
export interface Trajectory {
  goal: Goal;
  inferred: boolean;
  at_top_grade: boolean;
  progress: number;
  critical_gaps: number;
  skills: SkillProgress[];
}
export interface Activity {
  record_id: string;
  employee_id: string;
  event_id: string;
  date: string;
  due_date: string;
  status: string;
  completion_pct: number;
  score: number | null;
  feedback_rating: number | null;
  assigned_by: string;
  demo?: boolean;
  event_title: string;
}
export interface Profile {
  employee: Employee;
  effective_skills: Record<string, number>;
  trajectory: Trajectory;
  history: Activity[];
  as_of_date: string;
  revision: number;
}
export interface Event {
  event_id: string;
  title: string;
  description: string;
  type: string;
  format: string;
  duration_hours: number;
  mandatory: boolean;
  upcoming_sessions: string[];
}
export interface Evidence {
  id: string;
  factor: string;
  text: string;
}
export interface Recommendation {
  event: Event;
  score: number;
  expected_gains: {
    skill_id: string;
    name: string;
    before: number;
    after: number;
    required: number;
  }[];
  evidence: Evidence[];
  explanation: string;
  cross_role: boolean;
  in_progress: boolean;
}
export interface Recommendations {
  items: Recommendation[];
  mode: 'llm' | 'rules';
  notice: string;
  empty_reason: string;
  revision: number;
}
export interface Overview {
  employee_count: number;
  event_count: number;
  skill_count: number;
  history_count: number;
  average_progress: number;
  gaps: {
    skill_id: string;
    name: string;
    count: number;
    required_by: number;
    critical_count: number;
  }[];
  without_next_step: {
    employee_id: string;
    full_name: string;
    department: string;
    reason: string;
  }[];
  participation: {
    event_id: string;
    title: string;
    mandatory: boolean;
    total: number;
    completed: number;
    statuses: Record<string, number>;
  }[];
  as_of_date: string;
  revision: number;
}
export interface ImportResult {
  employees_added: number;
  employees_updated: number;
  history_added: number;
  history_updated: number;
  revision: number;
}
