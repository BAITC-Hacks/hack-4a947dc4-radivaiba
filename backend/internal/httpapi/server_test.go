package httpapi

import (
	"bytes"
	"careerquest/internal/auth"
	"careerquest/internal/llm"
	"careerquest/internal/model"
	"careerquest/internal/store"
	"careerquest/internal/testfixture"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestAccountsPermissionsReviewAndOrigin(t *testing.T) {
	data, _ := store.New(testfixture.Dataset(), filepath.Join(t.TempDir(), "state.json"))
	creds, err := data.ProvisionAccounts("", "hr", "")
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: data, Auth: auth.New("", "", ""), LLM: &llm.Client{}, WebDir: t.TempDir(), EvidenceDir: t.TempDir()}
	handler := server.Handler()
	request := func(method, path string, body any, cookie *http.Cookie, origin string) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	login := func(name string) *http.Cookie {
		for _, c := range creds {
			if c.Login == name {
				w := request("POST", "/api/session", map[string]string{"login": c.Login, "password": c.Password}, nil, "")
				if w.Code != 200 {
					t.Fatal(w.Body.String())
				}
				return w.Result().Cookies()[0]
			}
		}
		t.Fatal("missing account")
		return nil
	}
	if w := request("GET", "/api/leaderboard", nil, nil, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	employee, other, hr := login("E0001"), login("E0002"), login("hr")
	if !employee.HttpOnly || employee.SameSite != http.SameSiteStrictMode {
		t.Fatal("cookie flags")
	}
	for _, path := range []string{"/api/employees", "/api/employees/E0002", "/api/employees/E0002/development", "/api/hr/overview", "/api/hr/submissions"} {
		if w := request("GET", path, nil, employee, ""); w.Code != 403 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if w := request("POST", "/api/hr/import", nil, employee, ""); w.Code != 403 {
		t.Fatal("employee import")
	}
	if w := request("POST", "/api/employees/E0001/completions", map[string]string{"event_id": "design-course"}, employee, ""); w.Code != 410 {
		t.Fatal("legacy completion allowed", w.Code)
	}
	if w := request("POST", "/api/employees/E0001/modules/design-course/start", nil, employee, "https://evil.example"); w.Code != 403 {
		t.Fatal("origin allowed")
	}
	started := request("POST", "/api/employees/E0001/modules/design-course/start", nil, employee, "")
	var en model.EnrollmentDetail
	if started.Code != 200 {
		t.Fatal(started.Body.String())
	}
	json.Unmarshal(started.Body.Bytes(), &en)
	if w := request("GET", "/api/enrollments/"+en.Enrollment.ID, nil, other, ""); w.Code != 403 {
		t.Fatal("other enrollment allowed")
	}
	sent := request("POST", "/api/enrollments/"+en.Enrollment.ID+"/submissions", map[string]string{"text": "I documented the architecture and tradeoffs.", "request_key": "submit-test-001"}, employee, "")
	var sub model.SubmissionDetail
	json.Unmarshal(sent.Body.Bytes(), &sub)
	if sent.Code != 200 {
		t.Fatal(sent.Body.String())
	}
	if w := request("POST", "/api/hr/submissions/"+sub.Submission.ID+"/decision", map[string]string{"action": "approve", "request_key": "review-test-001"}, employee, ""); w.Code != 403 {
		t.Fatal("self approval")
	}
	reviewed := request("POST", "/api/hr/submissions/"+sub.Submission.ID+"/decision", map[string]string{"action": "approve", "request_key": "review-test-001"}, hr, "")
	if reviewed.Code != 200 {
		t.Fatal(reviewed.Body.String())
	}
	var decisionResult model.SubmissionDetail
	if err := json.Unmarshal(reviewed.Body.Bytes(), &decisionResult); err != nil {
		t.Fatal(err)
	}
	if len(decisionResult.Decisions) != 1 || decisionResult.Decisions[0].ReviewerLogin != "hr" {
		t.Fatal("reviewer display login missing", decisionResult.Decisions)
	}
	audit := data.RawSnapshot().Workflow.Decisions[0]
	if audit.ReviewerID == "" || decisionResult.Decisions[0].ReviewerID != audit.ReviewerID || audit.ReviewerLogin != "" {
		t.Fatal("display decoration changed the persisted audit identity", audit)
	}
	loadedEnrollment := request("GET", "/api/enrollments/"+en.Enrollment.ID, nil, employee, "")
	var enrollmentResult model.EnrollmentDetail
	if err := json.Unmarshal(loadedEnrollment.Body.Bytes(), &enrollmentResult); err != nil {
		t.Fatal(err)
	}
	if loadedEnrollment.Code != 200 || len(enrollmentResult.Decisions) != 1 || enrollmentResult.Decisions[0].ReviewerLogin != "hr" || enrollmentResult.Decisions[0].ReviewerID != audit.ReviewerID {
		t.Fatal("enrollment reviewer identity", loadedEnrollment.Body.String())
	}
	if data.RawSnapshot().Workflow.Decisions[0].ReviewerLogin != "" || audit.ReviewerLogin != "" {
		t.Fatal("response mutated an immutable store snapshot")
	}
	if w := request("GET", "/api/leaderboard", nil, employee, ""); w.Code != 200 || bytes.Contains(w.Body.Bytes(), []byte("employee_id")) || bytes.Contains(w.Body.Bytes(), []byte("Engineering")) {
		t.Fatal("leaderboard leaks", w.Body.String())
	}
	if w := request("GET", "/api/hr/overview", nil, hr, ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	request("DELETE", "/api/session", nil, employee, "")
	if w := request("GET", "/api/session", nil, employee, ""); w.Code != 401 {
		t.Fatal("logout")
	}
}
