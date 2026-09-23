export type Locale = 'ru' | 'kk' | 'en';
export interface Session {
  account_id: string;
  login: string;
  role: 'employee' | 'hr';
  employee_id: string;
  locale: Locale;
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
export interface ExpectedGain {
  skill_id: string;
  name: string;
  before: number;
  after: number;
  required: number;
}
export interface Recommendation {
  event: Event;
  score: number;
  expected_gains: ExpectedGain[];
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
export interface ModuleConfig {
  event_id: string;
  version: number;
  outcome: Record<Locale, string>;
  criteria: Record<Locale, string[]>;
  reward_exp: number;
  repeat_policy: string;
}
export type ModuleState =
  'available' | 'in_progress' | 'pending' | 'changes_requested' | 'completed' | 'locked';
export interface Enrollment {
  id: string;
  employee_id: string;
  event_id: string;
  occurrence: string;
  config: ModuleConfig;
  state: ModuleState;
  started_at: string;
  business_date: string;
  completion_id?: string;
}
export interface Attachment {
  id: string;
  submission_id: string;
  filename: string;
  media_type: string;
  size: number;
  sha256: string;
}
export interface Submission {
  id: string;
  enrollment_id: string;
  version: number;
  text: string;
  url: string;
  attachments: Attachment[];
  status: string;
  submitted_at: string;
  business_date: string;
}
export interface ReviewDecision {
  id: string;
  submission_id: string;
  enrollment_id: string;
  reviewer_id: string;
  reviewer_login?: string;
  action: string;
  comment: string;
  recorded_at: string;
  business_date: string;
  reversal_of?: string;
  completion_id?: string;
}
export interface ExpEntry {
  id: string;
  employee_id: string;
  event_id: string;
  approval_id: string;
  amount: number;
  month: string;
  recorded_at: string;
  reversal_of?: string;
}
export interface Experience {
  month: string;
  monthly_exp: number;
  total_exp: number;
  tree_level: number;
  energy: number;
  entries: ExpEntry[];
}
export interface ModuleView {
  event: Event;
  config: ModuleConfig;
  state: ModuleState;
  recommended: boolean;
  blocked_reason: string;
  branch: string;
  category: string;
  expected_gains: ExpectedGain[];
  enrollment: Enrollment | null;
}
export interface Development {
  items: ModuleView[];
  experience: Experience;
  business_date: string;
  revision: number;
}
export interface EnrollmentDetail {
  enrollment: Enrollment;
  event: Event;
  versions: Submission[];
  decisions: ReviewDecision[];
}
export interface SubmissionDetail extends EnrollmentDetail {
  submission: Submission;
  employee: Employee;
}
export interface LeaderboardEntry {
  rank: number;
  full_name: string;
  exp: number;
  employee_id?: string;
  department?: string;
  role?: string;
}
export interface Leaderboard {
  month: string;
  months: string[];
  items: LeaderboardEntry[];
}
export type AssistantAction = 'why' | 'fifteen_minutes' | 'simplify' | 'alternative' | 'start';
export interface AssistantResponse {
  benefit: string;
  first_step: string;
  application: string;
  mode: 'llm' | 'rules';
  notice: string;
  evidence: Evidence[];
  alternative_event_id?: string;
}
