package store

import (
	"careerquest/internal/engine"
	"careerquest/internal/model"
	"careerquest/internal/testfixture"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func testWorkflow(t *testing.T) (*Store, string) {
	t.Helper()
	s, err := New(testfixture.Dataset(), filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ProvisionAccounts("", "hr", ""); err != nil {
		t.Fatal(err)
	}
	for _, a := range s.Snapshot().Workflow.Accounts {
		if a.Role == "hr" {
			return s, a.ID
		}
	}
	t.Fatal("hr missing")
	return nil, ""
}
func submitModule(t *testing.T, s *Store, event, key string) model.SubmissionDetail {
	t.Helper()
	en, err := s.StartModule("E0001", event)
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.Submit(en.ID, "E0001", "A useful work outcome with evidence.", "https://example.com/evidence", key, nil)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestWorkflowReviewReturnConcurrencyRevocationAndRestart(t *testing.T) {
	s, hr := testWorkflow(t)
	baseline := s.Snapshot()
	before := engine.EffectiveSkills(baseline, baseline.Employees[0])["design"]
	sub := submitModule(t, s, "design-course", "first-result")
	if len(s.Snapshot().History) != len(baseline.History) || engine.ExperienceFor(s.Snapshot(), "E0001", "").TotalEXP != 0 {
		t.Fatal("pending grants reward")
	}
	if _, err := s.Decide(hr, sub.Submission.ID, "return", "", "return-empty"); !errors.Is(err, ErrInvalid) {
		t.Fatal("empty comment", err)
	}
	returned, err := s.Decide(hr, sub.Submission.ID, "return", "Add tradeoffs", "return-result")
	if err != nil {
		t.Fatal(err)
	}
	newer, err := s.Submit(returned.Enrollment.ID, "E0001", "Revised architecture with documented tradeoffs.", "", "second-result", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(newer.Versions) != 2 || newer.Submission.Version != 2 {
		t.Fatal("versions lost")
	}
	if _, err = s.SetGoal("E0001", &model.Goal{TargetRole: "Analyst", TargetGrade: "Middle"}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	success := 0
	var mu sync.Mutex
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.Decide(hr, newer.Submission.ID, "approve", "", fmt.Sprintf("approve-%03d", i))
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			} else if !errors.Is(err, ErrConflict) {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if success != 1 {
		t.Fatal("concurrent decisions", success)
	}
	d := s.Snapshot()
	xp := engine.ExperienceFor(d, "E0001", "")
	if xp.TotalEXP != 30 || engine.EffectiveSkills(d, d.Employees[0])["design"] != before+1 {
		t.Fatal("approval did not apply once", xp)
	}
	if len(baseline.Workflow.Submissions) != 0 || len(baseline.History) != 0 {
		t.Fatal("mutated old snapshot")
	}
	reopened, err := Open("", s.path)
	if err != nil {
		t.Fatal(err)
	}
	if engine.ExperienceFor(reopened.Snapshot(), "E0001", "").TotalEXP != 30 {
		t.Fatal("restart lost EXP")
	}
	var approval model.ReviewDecision
	for _, v := range d.Workflow.Decisions {
		if v.Action == "approve" {
			approval = v
		}
	}
	if _, err = s.Revoke(hr, approval.ID, "Incorrect evidence", "revoke-result"); err != nil {
		t.Fatal(err)
	}
	d = s.Snapshot()
	if engine.ExperienceFor(d, "E0001", "").TotalEXP != 0 || engine.EffectiveSkills(d, d.Employees[0])["design"] != before {
		t.Fatal("revocation projection")
	}
	if len(s.RawSnapshot().History) != 1 {
		t.Fatal("audit history deleted")
	}
	if _, err = s.Revoke(hr, approval.ID, "Incorrect evidence", "revoke-result"); err != nil {
		t.Fatal("retry", err)
	}
}
func TestWorkflowFailedSaveAndSelfReview(t *testing.T) {
	s, hr := testWorkflow(t)
	sub := submitModule(t, s, "design-course", "proof-save-fail")
	snap := s.Snapshot()
	oldPath := s.path
	s.path = filepath.Dir(s.path)
	if _, err := s.Decide(hr, sub.Submission.ID, "approve", "", "failed-save-key"); err == nil {
		t.Fatal("expected save failure")
	}
	if s.Snapshot().Revision != snap.Revision || len(s.Snapshot().Workflow.Ledger) != 0 || s.Snapshot().Workflow.Submissions[0].Status != "pending" {
		t.Fatal("failed save published")
	}
	s.path = oldPath
	s.data.Workflow.Accounts = append(s.data.Workflow.Accounts, model.Account{ID: "hr-linked", Login: "hr-linked", Role: "hr", EmployeeID: "E0001", Active: true})
	if _, err := s.Decide("hr-linked", sub.Submission.ID, "approve", "", "self-review-key"); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
}
func TestWorkflowRepeatMonthAndLateApproval(t *testing.T) {
	s, hr := testWorkflow(t)
	sub := submitModule(t, s, "EV_036", "repeat-first")
	if _, err := s.Decide(hr, sub.Submission.ID, "approve", "", "repeat-approve-1"); err != nil {
		t.Fatal(err)
	}
	s.SetBusinessDate("2026-10-02")
	sub = submitModule(t, s, "EV_036", "repeat-second")
	if _, err := s.Decide(hr, sub.Submission.ID, "approve", "", "repeat-approve-2"); err != nil {
		t.Fatal(err)
	}
	if xp := engine.ExperienceFor(s.Snapshot(), "E0001", ""); xp.TotalEXP != 20 {
		t.Fatal("repeat farming", xp)
	}
	sub = submitModule(t, s, "design-course", "late-month-proof")
	s.SetBusinessDate("2026-11-01")
	if _, err := s.Decide(hr, sub.Submission.ID, "approve", "", "late-month-review"); err != nil {
		t.Fatal(err)
	}
	xp := engine.ExperienceFor(s.Snapshot(), "E0001", "")
	if xp.MonthlyEXP != 30 || xp.TotalEXP != 50 {
		t.Fatal("wrong credit month", xp)
	}
}
