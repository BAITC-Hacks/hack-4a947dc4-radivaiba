package model

// Workflow is application-owned data, separate from the organizer dataset.
type Workflow struct {
	Accounts    []Account            `json:"accounts"`
	Preferences []EmployeePreference `json:"preferences"`
	Configs     []ModuleConfig       `json:"configs"`
	Enrollments []Enrollment         `json:"enrollments"`
	Submissions []Submission         `json:"submissions"`
	Decisions   []ReviewDecision     `json:"decisions"`
	Ledger      []ExpEntry           `json:"ledger"`
}
type Account struct {
	ID           string `json:"id"`
	Login        string `json:"login"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	EmployeeID   string `json:"employee_id"`
	Locale       string `json:"locale"`
	Active       bool   `json:"active"`
}
type EmployeePreference struct {
	EmployeeID string `json:"employee_id"`
	Goal       *Goal  `json:"goal"`
	GoalSet    bool   `json:"goal_set"`
}
type ModuleConfig struct {
	EventID      string              `json:"event_id"`
	Version      int                 `json:"version"`
	Outcome      map[string]string   `json:"outcome"`
	Criteria     map[string][]string `json:"criteria"`
	RewardEXP    int                 `json:"reward_exp"`
	RepeatPolicy string              `json:"repeat_policy"`
}
type Enrollment struct {
	ID           string       `json:"id"`
	EmployeeID   string       `json:"employee_id"`
	EventID      string       `json:"event_id"`
	Occurrence   string       `json:"occurrence"`
	Config       ModuleConfig `json:"config"`
	State        string       `json:"state"`
	StartedAt    string       `json:"started_at"`
	BusinessDate string       `json:"business_date"`
	CompletionID string       `json:"completion_id,omitempty"`
}
type Attachment struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	Filename     string `json:"filename"`
	MediaType    string `json:"media_type"`
	Size         int64  `json:"size"`
	SHA256       string `json:"sha256"`
	StorageKey   string `json:"storage_key,omitempty"`
}
type Submission struct {
	ID           string       `json:"id"`
	EnrollmentID string       `json:"enrollment_id"`
	Version      int          `json:"version"`
	Text         string       `json:"text"`
	URL          string       `json:"url"`
	Attachments  []Attachment `json:"attachments"`
	Status       string       `json:"status"`
	SubmittedAt  string       `json:"submitted_at"`
	BusinessDate string       `json:"business_date"`
	RequestKey   string       `json:"request_key,omitempty"`
}
type ReviewDecision struct {
	ID            string `json:"id"`
	SubmissionID  string `json:"submission_id"`
	EnrollmentID  string `json:"enrollment_id"`
	ReviewerID    string `json:"reviewer_id"`
	ReviewerLogin string `json:"reviewer_login,omitempty"`
	Action        string `json:"action"`
	Comment       string `json:"comment"`
	RecordedAt    string `json:"recorded_at"`
	BusinessDate  string `json:"business_date"`
	RequestKey    string `json:"request_key"`
	ReversalOf    string `json:"reversal_of,omitempty"`
	CompletionID  string `json:"completion_id,omitempty"`
}
type ExpEntry struct {
	ID         string `json:"id"`
	EmployeeID string `json:"employee_id"`
	EventID    string `json:"event_id"`
	ApprovalID string `json:"approval_id"`
	Amount     int    `json:"amount"`
	Month      string `json:"month"`
	RecordedAt string `json:"recorded_at"`
	ReversalOf string `json:"reversal_of,omitempty"`
}
type Experience struct {
	Month      string     `json:"month"`
	MonthlyEXP int        `json:"monthly_exp"`
	TotalEXP   int        `json:"total_exp"`
	TreeLevel  int        `json:"tree_level"`
	Energy     int        `json:"energy"`
	Entries    []ExpEntry `json:"entries"`
}
type ModuleView struct {
	Event         Event          `json:"event"`
	Config        ModuleConfig   `json:"config"`
	State         string         `json:"state"`
	Recommended   bool           `json:"recommended"`
	BlockedReason string         `json:"blocked_reason"`
	Branch        string         `json:"branch"`
	Category      string         `json:"category"`
	ExpectedGains []ExpectedGain `json:"expected_gains"`
	Enrollment    *Enrollment    `json:"enrollment"`
}
type Development struct {
	Items        []ModuleView `json:"items"`
	Experience   Experience   `json:"experience"`
	BusinessDate string       `json:"business_date"`
	Revision     int64        `json:"revision"`
}
type SubmissionDetail struct {
	Submission Submission       `json:"submission"`
	Enrollment Enrollment       `json:"enrollment"`
	Employee   Employee         `json:"employee"`
	Event      Event            `json:"event"`
	Versions   []Submission     `json:"versions"`
	Decisions  []ReviewDecision `json:"decisions"`
}
type EnrollmentDetail struct {
	Enrollment Enrollment       `json:"enrollment"`
	Event      Event            `json:"event"`
	Versions   []Submission     `json:"versions"`
	Decisions  []ReviewDecision `json:"decisions"`
}
type LeaderboardEntry struct {
	Rank       int    `json:"rank"`
	FullName   string `json:"full_name"`
	EXP        int    `json:"exp"`
	EmployeeID string `json:"employee_id,omitempty"`
	Department string `json:"department,omitempty"`
	Role       string `json:"role,omitempty"`
}
type Leaderboard struct {
	Month  string             `json:"month"`
	Months []string           `json:"months"`
	Items  []LeaderboardEntry `json:"items"`
}
type AssistantRequest struct {
	EventID string `json:"event_id"`
	Action  string `json:"action"`
	Locale  string `json:"locale"`
}
type AssistantResponse struct {
	Benefit            string     `json:"benefit"`
	FirstStep          string     `json:"first_step"`
	Application        string     `json:"application"`
	Mode               string     `json:"mode"`
	Notice             string     `json:"notice"`
	Evidence           []Evidence `json:"evidence"`
	AlternativeEventID string     `json:"alternative_event_id,omitempty"`
}
