package model

type Meta struct {
	Dataset  string `json:"dataset"`
	Version  string `json:"version"`
	AsOfDate string `json:"as_of_date"`
}
type Skill struct {
	ID          string `json:"skill_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Category    string `json:"category"`
	Description string `json:"description"`
}
type RoleProfile struct {
	Role           string         `json:"role"`
	Grade          string         `json:"grade"`
	RequiredSkills map[string]int `json:"required_skills"`
	CriticalSkills []string       `json:"critical_skills"`
}
type Catalog struct {
	Meta             Meta              `json:"meta"`
	ProficiencyScale map[string]string `json:"proficiency_scale"`
	Skills           []Skill           `json:"skills"`
	RoleProfiles     []RoleProfile     `json:"role_profiles"`
}
type Goal struct {
	TargetRole  string `json:"target_role"`
	TargetGrade string `json:"target_grade"`
}
type Employee struct {
	ID                string         `json:"employee_id"`
	FullName          string         `json:"full_name"`
	Department        string         `json:"department"`
	Role              string         `json:"role"`
	Grade             string         `json:"grade"`
	ManagerID         *string        `json:"manager_id"`
	HireDate          string         `json:"hire_date"`
	TenureMonths      int            `json:"tenure_months"`
	WorkFormat        string         `json:"work_format"`
	PreferredLanguage string         `json:"preferred_language"`
	CareerGoal        *Goal          `json:"career_goal"`
	Skills            map[string]int `json:"skills"`
	LastReviewDate    string         `json:"last_review_date"`
}
type EmployeesFile struct {
	Meta      Meta       `json:"meta"`
	Employees []Employee `json:"employees"`
}
type SkillGain struct {
	SkillID  string `json:"skill_id"`
	Gain     int    `json:"gain"`
	MaxLevel int    `json:"max_level"`
}
type Event struct {
	ID               string         `json:"event_id"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Type             string         `json:"type"`
	Format           string         `json:"format"`
	DurationHours    float64        `json:"duration_hours"`
	Mandatory        bool           `json:"mandatory"`
	TargetRoles      []string       `json:"target_roles"`
	TargetGrades     []string       `json:"target_grades"`
	DevelopsSkills   []SkillGain    `json:"develops_skills"`
	Prerequisites    map[string]int `json:"prerequisites"`
	UpcomingSessions []string       `json:"upcoming_sessions"`
}
type EventsFile struct {
	Meta   Meta    `json:"meta"`
	Events []Event `json:"events"`
}
type Activity struct {
	ID             string `json:"record_id"`
	EmployeeID     string `json:"employee_id"`
	EventID        string `json:"event_id"`
	Date           string `json:"date"`
	DueDate        string `json:"due_date"`
	Status         string `json:"status"`
	CompletionPct  int    `json:"completion_pct"`
	Score          *int   `json:"score"`
	FeedbackRating *int   `json:"feedback_rating"`
	AssignedBy     string `json:"assigned_by"`
	Demo           bool   `json:"demo,omitempty"`
}
type Dataset struct {
	Catalog      Catalog    `json:"catalog"`
	Employees    []Employee `json:"employees"`
	Events       []Event    `json:"events"`
	History      []Activity `json:"history"`
	Revision     int64      `json:"revision"`
	Workflow     Workflow   `json:"workflow,omitempty"`
	BusinessDate string     `json:"-"`
}
type SkillProgress struct {
	SkillID  string `json:"skill_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Current  int    `json:"current"`
	Required int    `json:"required"`
	Gap      int    `json:"gap"`
	Critical bool   `json:"critical"`
}
type Trajectory struct {
	Goal         Goal            `json:"goal"`
	Inferred     bool            `json:"inferred"`
	AtTopGrade   bool            `json:"at_top_grade"`
	Progress     float64         `json:"progress"`
	CriticalGaps int             `json:"critical_gaps"`
	Skills       []SkillProgress `json:"skills"`
}
type HistoryItem struct {
	Activity
	EventTitle string `json:"event_title"`
}
type Profile struct {
	Employee        Employee       `json:"employee"`
	EffectiveSkills map[string]int `json:"effective_skills"`
	Trajectory      Trajectory     `json:"trajectory"`
	History         []HistoryItem  `json:"history"`
	AsOfDate        string         `json:"as_of_date"`
	Revision        int64          `json:"revision"`
}
type Evidence struct {
	ID     string `json:"id"`
	Factor string `json:"factor"`
	Text   string `json:"text"`
}
type ExpectedGain struct {
	SkillID  string `json:"skill_id"`
	Name     string `json:"name"`
	Before   int    `json:"before"`
	After    int    `json:"after"`
	Required int    `json:"required"`
}
type Recommendation struct {
	Event         Event          `json:"event"`
	Score         float64        `json:"score"`
	ExpectedGains []ExpectedGain `json:"expected_gains"`
	Evidence      []Evidence     `json:"evidence"`
	Explanation   string         `json:"explanation"`
	CrossRole     bool           `json:"cross_role"`
	InProgress    bool           `json:"in_progress"`
}
type RecommendationResult struct {
	Items       []Recommendation `json:"items"`
	Mode        string           `json:"mode"`
	Notice      string           `json:"notice"`
	EmptyReason string           `json:"empty_reason"`
	Revision    int64            `json:"revision"`
}
type GapSummary struct {
	SkillID       string `json:"skill_id"`
	Name          string `json:"name"`
	Count         int    `json:"count"`
	RequiredBy    int    `json:"required_by"`
	CriticalCount int    `json:"critical_count"`
}
type NoStep struct {
	EmployeeID string `json:"employee_id"`
	FullName   string `json:"full_name"`
	Department string `json:"department"`
	Reason     string `json:"reason"`
}
type Participation struct {
	EventID   string         `json:"event_id"`
	Title     string         `json:"title"`
	Mandatory bool           `json:"mandatory"`
	Total     int            `json:"total"`
	Completed int            `json:"completed"`
	Statuses  map[string]int `json:"statuses"`
}
type Overview struct {
	EmployeeCount   int             `json:"employee_count"`
	EventCount      int             `json:"event_count"`
	SkillCount      int             `json:"skill_count"`
	HistoryCount    int             `json:"history_count"`
	AverageProgress float64         `json:"average_progress"`
	Gaps            []GapSummary    `json:"gaps"`
	WithoutNextStep []NoStep        `json:"without_next_step"`
	Participation   []Participation `json:"participation"`
	AsOfDate        string          `json:"as_of_date"`
	Revision        int64           `json:"revision"`
}
