package store

import (
	"bytes"
	"careerquest/internal/model"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func DecodeJSON(data []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})))
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one JSON document")
	}
	return nil
}
func ParseHistory(data []byte) ([]model.Activity, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})))
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("activity_history.csv: header: %w", err)
	}
	columns := map[string]int{}
	for i, h := range header {
		h = strings.TrimSpace(h)
		if _, ok := columns[h]; ok {
			return nil, fmt.Errorf("activity_history.csv: duplicate column %s", h)
		}
		columns[h] = i
	}
	for _, h := range []string{"record_id", "employee_id", "event_id", "date", "due_date", "status", "completion_pct", "score", "feedback_rating", "assigned_by"} {
		if _, ok := columns[h]; !ok {
			return nil, fmt.Errorf("activity_history.csv: missing column %s", h)
		}
	}
	result := []model.Activity{}
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("activity_history.csv: row %d: %w", line, err)
		}
		get := func(k string) string { return strings.TrimSpace(row[columns[k]]) }
		pct, err := strconv.Atoi(get("completion_pct"))
		if err != nil {
			return nil, fmt.Errorf("activity_history.csv: row %d: invalid completion_pct", line)
		}
		var score, rating *int
		for key, dst := range map[string]**int{"score": &score, "feedback_rating": &rating} {
			if value := get(key); value != "" {
				n, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("activity_history.csv: row %d: invalid %s", line, key)
				}
				*dst = &n
			}
		}
		result = append(result, model.Activity{ID: get("record_id"), EmployeeID: get("employee_id"), EventID: get("event_id"), Date: get("date"), DueDate: get("due_date"), Status: get("status"), CompletionPct: pct, Score: score, FeedbackRating: rating, AssignedBy: get("assigned_by")})
	}
	return result, nil
}
