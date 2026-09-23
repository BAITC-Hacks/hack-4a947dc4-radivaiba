package store

import (
	"careerquest/internal/auth"
	"careerquest/internal/engine"
	"careerquest/internal/model"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var ErrConflict = errors.New("state conflict")
var ErrForbidden = errors.New("access denied")
var ErrInvalid = errors.New("invalid request")

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b[:])
}

// Clock helpers are called while holding s.mu. A workflow operation captures one
// instant so its audit timestamp and business month cannot straddle midnight.
func (s *Store) clockNow() time.Time {
	if s.liveNow != nil {
		return s.liveNow()
	}
	return time.Now()
}

func (s *Store) businessDateAt(now time.Time) string {
	if s.liveNow != nil && s.businessZone != nil {
		return now.In(s.businessZone).Format("2006-01-02")
	}
	if s.businessDate != "" {
		return s.businessDate
	}
	return s.data.Catalog.Meta.AsOfDate
}

// ConfigureLiveClock selects the organization calendar. now must be safe for
// concurrent calls; production uses time.Now and tests can supply a fake clock.
func (s *Store) ConfigureLiveClock(timezone string, now func() time.Time) error {
	zone, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid organization timezone: %w", err)
	}
	if now == nil {
		now = time.Now
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if now().In(zone).Format("2006-01-02") < s.data.Catalog.Meta.AsOfDate {
		return fmt.Errorf("business date precedes dataset")
	}
	s.liveNow, s.businessZone, s.businessDate = now, zone, ""
	return nil
}

func (s *Store) SetBusinessDate(date string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if date < s.data.Catalog.Meta.AsOfDate {
		return fmt.Errorf("business date precedes dataset")
	}
	s.businessDate = date
	s.liveNow, s.businessZone = nil, nil
	return nil
}
func workflowView(d model.Dataset, date string) model.Dataset {
	if date == "" {
		date = d.Catalog.Meta.AsOfDate
	}
	d.BusinessDate = date
	d.Employees = append([]model.Employee{}, d.Employees...)
	for i := range d.Employees {
		for _, p := range d.Workflow.Preferences {
			if p.EmployeeID == d.Employees[i].ID && p.GoalSet {
				d.Employees[i].CareerGoal = p.Goal
			}
		}
	}
	revoked := map[string]bool{}
	for _, v := range d.Workflow.Decisions {
		if v.Action == "revoke" {
			for _, a := range d.Workflow.Decisions {
				if a.ID == v.ReversalOf {
					revoked[a.CompletionID] = true
				}
			}
		}
	}
	d.History = append([]model.Activity{}, d.History...)
	out := d.History[:0]
	for _, h := range d.History {
		if !revoked[h.ID] {
			out = append(out, h)
		}
	}
	d.History = out
	return d
}
func (s *Store) nextWorkflow() model.Dataset {
	d := s.data
	b, err := json.Marshal(d.Workflow)
	if err != nil {
		panic(err)
	}
	var copied model.Workflow
	if err = json.Unmarshal(b, &copied); err != nil {
		panic(err)
	}
	d.Workflow = copied
	d.History = append([]model.Activity{}, d.History...)
	d.Revision++
	return d
}
func (s *Store) EnsureConfigs() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.nextWorkflow()
	changed := false
	for _, e := range s.data.Events {
		found := false
		for _, c := range next.Workflow.Configs {
			if c.EventID == e.ID {
				found = true
				break
			}
		}
		if !found {
			next.Workflow.Configs = append(next.Workflow.Configs, engine.DefaultModuleConfig(s.data, e))
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.save(next)
}
func accountByID(w model.Workflow, id string) (model.Account, bool) {
	for _, a := range w.Accounts {
		if a.ID == id && a.Active {
			return a, true
		}
	}
	return model.Account{}, false
}
func (s *Store) Account(id string) (model.Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return accountByID(s.data.Workflow, id)
}
func (s *Store) Authenticate(login, password string) (model.Account, bool) {
	s.mu.RLock()
	var found model.Account
	for _, a := range s.data.Workflow.Accounts {
		if a.Active && a.Login == login {
			found = a
			break
		}
	}
	s.mu.RUnlock()
	if found.ID == "" {
		return found, false
	}
	return found, auth.CheckPassword(found.PasswordHash, password)
}

type Credential struct {
	Login      string `json:"login"`
	Password   string `json:"password"`
	Role       string `json:"role"`
	EmployeeID string `json:"employee_id"`
}

func (s *Store) ProvisionAccounts(employeeID, hrLogin, hrEmployee string) ([]Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.nextWorkflow()
	created := []Credential{}
	exists := func(login, role, id string) bool {
		for _, a := range next.Workflow.Accounts {
			if a.Login == login || (role == "employee" && a.Role == role && a.EmployeeID == id) {
				return true
			}
		}
		return false
	}
	add := func(login, role, id, locale string) error {
		if exists(login, role, id) {
			return nil
		}
		password := auth.NewPassword()
		hash, err := auth.HashPassword(password)
		if err != nil {
			return err
		}
		next.Workflow.Accounts = append(next.Workflow.Accounts, model.Account{ID: newID("AC_"), Login: login, PasswordHash: hash, Role: role, EmployeeID: id, Locale: locale, Active: true})
		created = append(created, Credential{login, password, role, id})
		return nil
	}
	if employeeID != "" {
		if _, ok := engine.FindEmployee(s.data, employeeID); !ok {
			return nil, ErrNotFound
		}
	}
	for _, e := range s.data.Employees {
		if employeeID == "" || employeeID == e.ID {
			if err := add(e.ID, "employee", e.ID, e.PreferredLanguage); err != nil {
				return nil, err
			}
		}
	}
	if hrLogin != "" {
		if hrEmployee != "" {
			if _, ok := engine.FindEmployee(s.data, hrEmployee); !ok {
				return nil, ErrNotFound
			}
		}
		if err := add(hrLogin, "hr", hrEmployee, "ru"); err != nil {
			return nil, err
		}
	}
	if len(created) == 0 {
		return created, nil
	}
	if err := s.save(next); err != nil {
		return nil, err
	}
	return created, nil
}
func (s *Store) SetLocale(accountID, locale string) (model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if locale != "ru" && locale != "kk" && locale != "en" {
		return model.Account{}, ErrInvalid
	}
	next := s.nextWorkflow()
	for i, a := range next.Workflow.Accounts {
		if a.ID == accountID && a.Active {
			next.Workflow.Accounts[i].Locale = locale
			if err := s.save(next); err != nil {
				return a, err
			}
			return next.Workflow.Accounts[i], nil
		}
	}
	return model.Account{}, ErrNotFound
}
func (s *Store) SetGoal(employeeID string, goal *model.Goal) (model.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := engine.FindEmployee(s.data, employeeID); !ok {
		return model.Profile{}, ErrNotFound
	}
	if goal != nil {
		valid := false
		for _, r := range s.data.Catalog.RoleProfiles {
			if r.Role == goal.TargetRole && r.Grade == goal.TargetGrade {
				valid = true
			}
		}
		if !valid {
			return model.Profile{}, ErrInvalid
		}
	}
	next := s.nextWorkflow()
	found := false
	for i, p := range next.Workflow.Preferences {
		if p.EmployeeID == employeeID {
			next.Workflow.Preferences[i] = model.EmployeePreference{EmployeeID: employeeID, Goal: goal, GoalSet: true}
			found = true
		}
	}
	if !found {
		next.Workflow.Preferences = append(next.Workflow.Preferences, model.EmployeePreference{EmployeeID: employeeID, Goal: goal, GoalSet: true})
	}
	if err := s.save(next); err != nil {
		return model.Profile{}, err
	}
	d := workflowView(next, s.businessDateAt(s.clockNow()))
	e, _ := engine.FindEmployee(d, employeeID)
	return engine.BuildProfile(d, e), nil
}
func (s *Store) StartModule(employeeID, eventID string) (model.Enrollment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clockNow()
	d := workflowView(s.data, s.businessDateAt(now))
	e, ok := engine.FindEmployee(d, employeeID)
	if !ok {
		return model.Enrollment{}, ErrNotFound
	}
	event, ok := engine.FindEvent(d, eventID)
	if !ok {
		return model.Enrollment{}, ErrNotFound
	}
	occurrence := "once"
	if eventID == "EV_036" {
		occurrence = d.BusinessDate
	}
	for _, en := range d.Workflow.Enrollments {
		if en.EmployeeID == employeeID && en.EventID == eventID && (en.Occurrence == occurrence || en.State != "completed") {
			return en, nil
		}
	}
	if !engine.Eligible(d, e, event, engine.EffectiveSkills(d, e)) {
		return model.Enrollment{}, ErrNotEligible
	}
	config := engine.DefaultModuleConfig(d, event)
	for _, c := range d.Workflow.Configs {
		if c.EventID == eventID && c.Version >= config.Version {
			config = c
		}
	}
	en := model.Enrollment{ID: newID("EN_"), EmployeeID: employeeID, EventID: eventID, Occurrence: occurrence, Config: config, State: "in_progress", StartedAt: now.UTC().Format(time.RFC3339Nano), BusinessDate: d.BusinessDate}
	next := s.nextWorkflow()
	next.Workflow.Enrollments = append(next.Workflow.Enrollments, en)
	if err := s.save(next); err != nil {
		return en, err
	}
	return en, nil
}
func enrollmentIndex(w model.Workflow, id string) int {
	for i, e := range w.Enrollments {
		if e.ID == id {
			return i
		}
	}
	return -1
}
func submissionIndex(w model.Workflow, id string) int {
	for i, v := range w.Submissions {
		if v.ID == id {
			return i
		}
	}
	return -1
}
func (s *Store) Enrollment(id string) (model.EnrollmentDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return enrollmentDetail(s.data, id)
}
func enrollmentDetail(d model.Dataset, id string) (model.EnrollmentDetail, error) {
	i := enrollmentIndex(d.Workflow, id)
	if i < 0 {
		return model.EnrollmentDetail{}, ErrNotFound
	}
	en := d.Workflow.Enrollments[i]
	event, _ := engine.FindEvent(d, en.EventID)
	result := model.EnrollmentDetail{Enrollment: en, Event: event, Versions: []model.Submission{}, Decisions: []model.ReviewDecision{}}
	for _, v := range d.Workflow.Submissions {
		if v.EnrollmentID == id {
			result.Versions = append(result.Versions, v)
		}
	}
	for _, v := range d.Workflow.Decisions {
		if v.EnrollmentID == id {
			result.Decisions = append(result.Decisions, v)
		}
	}
	return result, nil
}
func submissionDetail(d model.Dataset, id string) (model.SubmissionDetail, error) {
	i := submissionIndex(d.Workflow, id)
	if i < 0 {
		return model.SubmissionDetail{}, ErrNotFound
	}
	v := d.Workflow.Submissions[i]
	en, err := enrollmentDetail(d, v.EnrollmentID)
	if err != nil {
		return model.SubmissionDetail{}, err
	}
	e, _ := engine.FindEmployee(d, en.Enrollment.EmployeeID)
	return model.SubmissionDetail{Submission: v, Enrollment: en.Enrollment, Employee: e, Event: en.Event, Versions: en.Versions, Decisions: en.Decisions}, nil
}
func (s *Store) Submission(id string) (model.SubmissionDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return submissionDetail(s.data, id)
}
func (s *Store) Submissions() []model.SubmissionDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []model.SubmissionDetail{}
	for i := len(s.data.Workflow.Submissions) - 1; i >= 0; i-- {
		d, err := submissionDetail(s.data, s.data.Workflow.Submissions[i].ID)
		if err == nil {
			items = append(items, d)
		}
	}
	return items
}
func (s *Store) Submit(enrollmentID, employeeID, text, link, key string, attachments []model.Attachment) (model.SubmissionDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := enrollmentIndex(s.data.Workflow, enrollmentID)
	if idx < 0 {
		return model.SubmissionDetail{}, ErrNotFound
	}
	en := s.data.Workflow.Enrollments[idx]
	if en.EmployeeID != employeeID {
		return model.SubmissionDetail{}, ErrForbidden
	}
	if len(key) < 8 || len(key) > 100 {
		return model.SubmissionDetail{}, ErrInvalid
	}
	for _, v := range s.data.Workflow.Submissions {
		if v.EnrollmentID == enrollmentID && v.RequestKey == key {
			return submissionDetail(s.data, v.ID)
		}
	}
	if en.State != "in_progress" && en.State != "changes_requested" {
		return model.SubmissionDetail{}, ErrConflict
	}
	text = strings.TrimSpace(text)
	link = strings.TrimSpace(link)
	if len(text) > 12000 || len(link) > 2048 || len(attachments) > 3 || (len(text) < 10 && link == "" && len(attachments) == 0) {
		return model.SubmissionDetail{}, ErrInvalid
	}
	if link != "" {
		u, err := url.Parse(link)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return model.SubmissionDetail{}, ErrInvalid
		}
	}
	now := s.clockNow()
	v := model.Submission{ID: newID("SU_"), EnrollmentID: enrollmentID, Text: text, URL: link, Attachments: attachments, Status: "pending", SubmittedAt: now.UTC().Format(time.RFC3339Nano), BusinessDate: s.businessDateAt(now), RequestKey: key, Version: 1}
	if v.Attachments == nil {
		v.Attachments = []model.Attachment{}
	}
	for _, old := range s.data.Workflow.Submissions {
		if old.EnrollmentID == enrollmentID && old.Version >= v.Version {
			v.Version = old.Version + 1
		}
	}
	for i := range v.Attachments {
		v.Attachments[i].SubmissionID = v.ID
	}
	next := s.nextWorkflow()
	next.Workflow.Submissions = append(next.Workflow.Submissions, v)
	next.Workflow.Enrollments[idx].State = "pending"
	if err := s.save(next); err != nil {
		return model.SubmissionDetail{}, err
	}
	return submissionDetail(next, v.ID)
}
func (s *Store) Decide(reviewerID, submissionID, action, comment, key string) (model.SubmissionDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reviewer, ok := accountByID(s.data.Workflow, reviewerID)
	if !ok || reviewer.Role != "hr" {
		return model.SubmissionDetail{}, ErrForbidden
	}
	if len(key) < 8 || len(key) > 100 || len(comment) > 4000 || (action != "approve" && action != "return") {
		return model.SubmissionDetail{}, ErrInvalid
	}
	for _, a := range s.data.Workflow.Decisions {
		if a.RequestKey == key {
			if a.ReviewerID != reviewerID || a.SubmissionID != submissionID || a.Action != action {
				return model.SubmissionDetail{}, ErrConflict
			}
			return submissionDetail(s.data, submissionID)
		}
	}
	idx := submissionIndex(s.data.Workflow, submissionID)
	if idx < 0 {
		return model.SubmissionDetail{}, ErrNotFound
	}
	sub := s.data.Workflow.Submissions[idx]
	ei := enrollmentIndex(s.data.Workflow, sub.EnrollmentID)
	en := s.data.Workflow.Enrollments[ei]
	if en.EmployeeID == reviewer.EmployeeID {
		return model.SubmissionDetail{}, ErrForbidden
	}
	if sub.Status != "pending" || en.State != "pending" {
		return model.SubmissionDetail{}, ErrConflict
	}
	if action == "return" && strings.TrimSpace(comment) == "" {
		return model.SubmissionDetail{}, ErrInvalid
	}
	now := s.clockNow()
	d := workflowView(s.data, s.businessDateAt(now))
	date := d.BusinessDate
	month := date[:7]
	decision := model.ReviewDecision{ID: newID("RV_"), SubmissionID: submissionID, EnrollmentID: en.ID, ReviewerID: reviewerID, Action: action, Comment: strings.TrimSpace(comment), RecordedAt: now.UTC().Format(time.RFC3339Nano), BusinessDate: date, RequestKey: key}
	next := s.nextWorkflow()
	if action == "return" {
		next.Workflow.Submissions[idx].Status = "changes_requested"
		next.Workflow.Enrollments[ei].State = "changes_requested"
	} else {
		for _, h := range d.History {
			if h.EmployeeID == en.EmployeeID && h.EventID == en.EventID && h.Status == "completed" && (en.EventID != "EV_036" || h.Date == date) {
				return model.SubmissionDetail{}, ErrConflict
			}
		}
		decision.CompletionID = newID("APP_")
		next.History = append(next.History, model.Activity{ID: decision.CompletionID, EmployeeID: en.EmployeeID, EventID: en.EventID, Date: date, Status: "completed", CompletionPct: 100, AssignedBy: "hr", Demo: true})
		amount := en.Config.RewardEXP
		if en.EventID == "EV_036" {
			net := 0
			for _, x := range next.Workflow.Ledger {
				if x.EmployeeID == en.EmployeeID && x.EventID == en.EventID && x.Month == month {
					net += x.Amount
				}
			}
			if net > 0 {
				amount = 0
			}
		}
		next.Workflow.Ledger = append(next.Workflow.Ledger, model.ExpEntry{ID: newID("XP_"), EmployeeID: en.EmployeeID, EventID: en.EventID, ApprovalID: decision.ID, Amount: amount, Month: month, RecordedAt: decision.RecordedAt})
		next.Workflow.Submissions[idx].Status = "completed"
		next.Workflow.Enrollments[ei].State = "completed"
		next.Workflow.Enrollments[ei].CompletionID = decision.CompletionID
	}
	next.Workflow.Decisions = append(next.Workflow.Decisions, decision)
	if err := s.save(next); err != nil {
		return model.SubmissionDetail{}, err
	}
	return submissionDetail(next, submissionID)
}
func (s *Store) Revoke(reviewerID, approvalID, comment, key string) (model.SubmissionDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reviewer, ok := accountByID(s.data.Workflow, reviewerID)
	if !ok || reviewer.Role != "hr" {
		return model.SubmissionDetail{}, ErrForbidden
	}
	if len(key) < 8 || len(key) > 100 || strings.TrimSpace(comment) == "" || len(comment) > 4000 {
		return model.SubmissionDetail{}, ErrInvalid
	}
	var approval model.ReviewDecision
	for _, v := range s.data.Workflow.Decisions {
		if v.RequestKey == key {
			if v.ReviewerID != reviewerID || v.ReversalOf != approvalID || v.Action != "revoke" {
				return model.SubmissionDetail{}, ErrConflict
			}
			return submissionDetail(s.data, v.SubmissionID)
		}
		if v.ID == approvalID && v.Action == "approve" {
			approval = v
		}
	}
	if approval.ID == "" {
		return model.SubmissionDetail{}, ErrNotFound
	}
	for _, v := range s.data.Workflow.Decisions {
		if v.ReversalOf == approvalID {
			return model.SubmissionDetail{}, ErrConflict
		}
	}
	ei := enrollmentIndex(s.data.Workflow, approval.EnrollmentID)
	en := s.data.Workflow.Enrollments[ei]
	if reviewer.EmployeeID == en.EmployeeID {
		return model.SubmissionDetail{}, ErrForbidden
	}
	next := s.nextWorkflow()
	now := s.clockNow()
	v := model.ReviewDecision{ID: newID("RV_"), SubmissionID: approval.SubmissionID, EnrollmentID: en.ID, ReviewerID: reviewerID, Action: "revoke", Comment: strings.TrimSpace(comment), RecordedAt: now.UTC().Format(time.RFC3339Nano), BusinessDate: s.businessDateAt(now), RequestKey: key, ReversalOf: approvalID}
	for _, x := range s.data.Workflow.Ledger {
		if x.ApprovalID == approvalID {
			next.Workflow.Ledger = append(next.Workflow.Ledger, model.ExpEntry{ID: newID("XP_"), EmployeeID: x.EmployeeID, EventID: x.EventID, ApprovalID: v.ID, Amount: -x.Amount, Month: x.Month, RecordedAt: v.RecordedAt, ReversalOf: x.ID})
		}
	}
	next.Workflow.Decisions = append(next.Workflow.Decisions, v)
	next.Workflow.Enrollments[ei].State = "changes_requested"
	next.Workflow.Enrollments[ei].CompletionID = ""
	si := submissionIndex(next.Workflow, approval.SubmissionID)
	next.Workflow.Submissions[si].Status = "revoked"
	if err := s.save(next); err != nil {
		return model.SubmissionDetail{}, err
	}
	return submissionDetail(next, approval.SubmissionID)
}
