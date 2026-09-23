package engine

import (
	"careerquest/internal/model"
	"math"
	"sort"
)

func SummarizeHR(d model.Dataset) model.Overview {
	return SummarizeHRLocale(d, "ru")
}

func SummarizeHRLocale(d model.Dataset, locale string) model.Overview {
	result := model.Overview{EmployeeCount: len(d.Employees), EventCount: len(d.Events), SkillCount: len(d.Catalog.Skills), HistoryCount: len(d.History), Gaps: []model.GapSummary{}, WithoutNextStep: []model.NoStep{}, Participation: []model.Participation{}, AsOfDate: BusinessDate(d), Revision: d.Revision}
	gaps := map[string]model.GapSummary{}
	for _, e := range d.Employees {
		t := BuildTrajectory(d, e, EffectiveSkills(d, e))
		result.AverageProgress += t.Progress
		for _, s := range t.Skills {
			if s.Required == 0 {
				continue
			}
			g := gaps[s.SkillID]
			g.SkillID = s.SkillID
			g.Name = s.Name
			g.RequiredBy++
			if s.Gap > 0 {
				g.Count++
				if s.Critical {
					g.CriticalCount++
				}
			}
			gaps[s.SkillID] = g
		}
		if len(RankCandidates(d, e)) == 0 {
			result.WithoutNextStep = append(result.WithoutNextStep, model.NoStep{EmployeeID: e.ID, FullName: e.FullName, Department: e.Department, Reason: EmptyReasonLocale(d, e, locale)})
		}
	}
	if len(d.Employees) > 0 {
		result.AverageProgress = math.Round(result.AverageProgress/float64(len(d.Employees))*10) / 10
	}
	for _, g := range gaps {
		if g.Count > 0 {
			result.Gaps = append(result.Gaps, g)
		}
	}
	sort.Slice(result.Gaps, func(i, j int) bool {
		if result.Gaps[i].Count == result.Gaps[j].Count {
			return result.Gaps[i].SkillID < result.Gaps[j].SkillID
		}
		return result.Gaps[i].Count > result.Gaps[j].Count
	})
	for _, e := range d.Events {
		p := model.Participation{EventID: e.ID, Title: e.Title, Mandatory: e.Mandatory, Statuses: map[string]int{}}
		for _, r := range d.History {
			if r.EventID != e.ID || r.Date > BusinessDate(d) {
				continue
			}
			p.Total++
			p.Statuses[r.Status]++
			if r.Status == "completed" {
				p.Completed++
			}
		}
		result.Participation = append(result.Participation, p)
	}
	return result
}
