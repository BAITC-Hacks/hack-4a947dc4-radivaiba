package store

import (
	"careerquest/internal/engine"
	"careerquest/internal/testfixture"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestLiveClockCrossesOrganizationMonthWithoutRestart(t *testing.T) {
	d := testfixture.Dataset()
	d.Events[0].UpcomingSessions = []string{"2026-11-05"}
	d.Events[1].UpcomingSessions = []string{"2026-11-05"}
	s, err := New(d, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ProvisionAccounts("E0001", "hr-calendar", ""); err != nil {
		t.Fatal(err)
	}
	hr := ""
	for _, account := range s.Snapshot().Workflow.Accounts {
		if account.Login == "hr-calendar" {
			hr = account.ID
		}
	}
	if hr == "" {
		t.Fatal("missing HR account")
	}
	// UTC is still October 31 when Asia/Qyzylorda reaches November 1.
	before := time.Date(2026, 10, 31, 18, 59, 50, 0, time.UTC)
	boundary := time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC)
	var clock atomic.Int64
	clock.Store(before.UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	if err = s.ConfigureLiveClock("Asia/Qyzylorda", now); err != nil {
		t.Fatal(err)
	}
	october := submitModule(t, s, "EV_036", "october-proof")
	octoberApproved, err := s.Decide(hr, october.Submission.ID, "approve", "", "october-decision")
	if err != nil {
		t.Fatal(err)
	}
	late := submitModule(t, s, "design-course", "last-october-proof")
	oldSnapshot := s.Snapshot()
	oldXP := engine.ExperienceFor(oldSnapshot, "E0001", "")
	if oldSnapshot.BusinessDate != "2026-10-31" || oldXP.MonthlyEXP != 20 || oldXP.TotalEXP != 20 {
		t.Fatalf("wrong October setup: %+v", oldXP)
	}
	if late.Submission.BusinessDate != "2026-10-31" || late.Submission.SubmittedAt != before.Format(time.RFC3339Nano) {
		t.Fatal("submission did not use the injected clock")
	}
	clock.Store(boundary.UnixNano())
	newSnapshot := s.Snapshot()
	newXP := engine.ExperienceFor(newSnapshot, "E0001", "")
	if newSnapshot.BusinessDate != "2026-11-01" || newXP.Month != "2026-11" || newXP.MonthlyEXP != 0 {
		t.Fatalf("snapshot did not advance across local midnight: %+v", newXP)
	}
	if newXP.TotalEXP != oldXP.TotalEXP || newXP.TreeLevel != oldXP.TreeLevel || newXP.Energy != oldXP.Energy {
		t.Fatal("month change erased tree or overall experience")
	}
	if oldSnapshot.BusinessDate != "2026-10-31" || len(oldSnapshot.Workflow.Ledger) != 1 {
		t.Fatal("a later snapshot mutated the previous snapshot")
	}
	approved, err := s.Decide(hr, late.Submission.ID, "approve", "", "november-late-decision")
	if err != nil {
		t.Fatal(err)
	}
	decision := approved.Decisions[len(approved.Decisions)-1]
	if decision.BusinessDate != "2026-11-01" || decision.RecordedAt != boundary.Format(time.RFC3339Nano) {
		t.Fatal("approval did not use the live organization date and matching UTC instant")
	}
	xp := engine.ExperienceFor(s.Snapshot(), "E0001", "")
	if xp.MonthlyEXP != 30 || xp.TotalEXP != 50 || xp.Energy != 50 {
		t.Fatalf("late approval entered the wrong month: %+v", xp)
	}
	if engine.ExperienceFor(s.Snapshot(), "E0001", "2026-10").MonthlyEXP != 20 {
		t.Fatal("October ledger was rewritten")
	}
	novemberClub := submitModule(t, s, "EV_036", "november-club-proof")
	if novemberClub.Enrollment.Occurrence != "2026-11-01" || novemberClub.Enrollment.StartedAt != boundary.Format(time.RFC3339Nano) {
		t.Fatal("new enrollment used a stale startup date")
	}
	if _, err = s.Decide(hr, novemberClub.Submission.ID, "approve", "", "november-club-decision"); err != nil {
		t.Fatal(err)
	}
	if xp := engine.ExperienceFor(s.Snapshot(), "E0001", ""); xp.MonthlyEXP != 50 || xp.TotalEXP != 70 {
		t.Fatal("club reward did not reset for the new month", xp)
	}
	octoberApproval := octoberApproved.Decisions[0].ID
	revoked, err := s.Revoke(hr, octoberApproval, "Incorrect earlier evidence", "november-revoke-october")
	if err != nil {
		t.Fatal(err)
	}
	reversal := revoked.Decisions[len(revoked.Decisions)-1]
	if reversal.BusinessDate != "2026-11-01" || reversal.RecordedAt != boundary.Format(time.RFC3339Nano) {
		t.Fatal("revoke used stale calendar")
	}
	if engine.ExperienceFor(s.Snapshot(), "E0001", "2026-10").MonthlyEXP != 0 || engine.ExperienceFor(s.Snapshot(), "E0001", "2026-11").MonthlyEXP != 50 {
		t.Fatal("reversal must correct the original award month")
	}
	reopened, err := Open("", s.path)
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.ConfigureLiveClock("Asia/Qyzylorda", now); err != nil {
		t.Fatal(err)
	}
	if xp := engine.ExperienceFor(reopened.Snapshot(), "E0001", ""); xp.MonthlyEXP != 50 || xp.TotalEXP != 50 {
		t.Fatal("restart lost month-boundary progress", xp)
	}
}

func TestLiveClockValidationAndFixedDemoOverride(t *testing.T) {
	s, err := New(testfixture.Dataset(), filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetBusinessDate("2026-10-17"); err != nil {
		t.Fatal(err)
	}
	tooEarly := func() time.Time { return time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC) }
	if err = s.ConfigureLiveClock("Asia/Qyzylorda", tooEarly); err == nil {
		t.Fatal("live startup before dataset date must fail")
	}
	if err = s.ConfigureLiveClock("Not/A_Timezone", time.Now); err == nil {
		t.Fatal("invalid organization timezone accepted")
	}
	if s.Snapshot().BusinessDate != "2026-10-17" {
		t.Fatal("invalid configuration changed the existing demo clock")
	}
	var clock atomic.Int64
	clock.Store(time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC).UnixNano())
	if err = s.ConfigureLiveClock("Asia/Qyzylorda", func() time.Time { return time.Unix(0, clock.Load()) }); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().BusinessDate != "2026-11-01" {
		t.Fatal("timezone not applied")
	}
	if err = s.SetBusinessDate("2026-10-17"); err != nil {
		t.Fatal(err)
	}
	clock.Store(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC).UnixNano())
	if s.Snapshot().BusinessDate != "2026-10-17" {
		t.Fatal("demo date kept following the live clock")
	}
}

func TestReviewCapturesOneClockInstantAtMidnight(t *testing.T) {
	s, hr := testWorkflow(t)
	submission := submitModule(t, s, "design-course", "midnight-proof")
	before := time.Date(2026, 10, 31, 18, 59, 59, 999999999, time.UTC)
	after := before.Add(time.Nanosecond)
	var calls atomic.Int64
	now := func() time.Time {
		if calls.Add(1) > 1 {
			return after
		}
		return before
	}
	if err := s.ConfigureLiveClock("Asia/Qyzylorda", now); err != nil {
		t.Fatal(err)
	}
	calls.Store(0)
	result, err := s.Decide(hr, submission.Submission.ID, "approve", "", "midnight-decision")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("review sampled the clock %d times", calls.Load())
	}
	decision := result.Decisions[0]
	if decision.BusinessDate != "2026-10-31" || decision.RecordedAt != before.Format(time.RFC3339Nano) {
		t.Fatal("one review straddled two calendar dates")
	}
	if s.Snapshot().BusinessDate != "2026-11-01" {
		t.Fatal("following request did not see new day")
	}
}
