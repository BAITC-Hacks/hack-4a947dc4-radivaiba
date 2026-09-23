package engine

import (
	"careerquest/internal/model"
	"sort"
)

func monthOf(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return date
}

func ExperienceFor(d model.Dataset, employeeID, month string) model.Experience {
	if month == "" {
		month = monthOf(BusinessDate(d))
	}
	result := model.Experience{Month: month, TreeLevel: 1, Entries: []model.ExpEntry{}}
	for _, entry := range d.Workflow.Ledger {
		if entry.EmployeeID != employeeID {
			continue
		}
		result.TotalEXP += entry.Amount
		if entry.Month == month {
			result.MonthlyEXP += entry.Amount
			result.Entries = append(result.Entries, entry)
		}
	}
	result.TreeLevel = 1 + max(0, result.TotalEXP)/100
	result.Energy = max(0, result.TotalEXP) % 100
	sort.Slice(result.Entries, func(i, j int) bool {
		if result.Entries[i].RecordedAt == result.Entries[j].RecordedAt {
			return result.Entries[i].ID > result.Entries[j].ID
		}
		return result.Entries[i].RecordedAt > result.Entries[j].RecordedAt
	})
	return result
}

// LeaderboardFor exposes only names/rank/EXP outside HR, never employee profile IDs.
func LeaderboardFor(d model.Dataset, month, department, role string, hr bool) model.Leaderboard {
	if !hr {
		department, role = "", ""
	}
	currentMonth := monthOf(BusinessDate(d))
	if month == "" {
		month = currentMonth
	}
	result := model.Leaderboard{Month: month, Months: []string{}, Items: []model.LeaderboardEntry{}}
	months := map[string]bool{currentMonth: true}
	amounts := map[string]int{}
	for _, entry := range d.Workflow.Ledger {
		months[entry.Month] = true
		if entry.Month == month {
			amounts[entry.EmployeeID] += entry.Amount
		}
	}
	for value := range months {
		result.Months = append(result.Months, value)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(result.Months)))
	for _, employee := range d.Employees {
		if amounts[employee.ID] <= 0 || (department != "" && employee.Department != department) || (role != "" && employee.Role != role) {
			continue
		}
		entry := model.LeaderboardEntry{FullName: employee.FullName, EXP: amounts[employee.ID], EmployeeID: employee.ID}
		if hr {
			entry.Department, entry.Role = employee.Department, employee.Role
		}
		result.Items = append(result.Items, entry)
	}
	sort.Slice(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]
		if a.EXP != b.EXP {
			return a.EXP > b.EXP
		}
		return a.EmployeeID < b.EmployeeID
	})
	for index := range result.Items {
		rank := index + 1
		if index > 0 && result.Items[index].EXP == result.Items[index-1].EXP {
			rank = result.Items[index-1].Rank
		}
		result.Items[index].Rank = rank
		if !hr {
			result.Items[index].EmployeeID = ""
		}
	}
	return result
}
