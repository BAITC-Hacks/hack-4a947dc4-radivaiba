package httpapi

import (
	"bytes"
	"careerquest/internal/auth"
	"careerquest/internal/store"
	"careerquest/internal/testfixture"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestPermissionsCompletionAndOrigin(t *testing.T) {
	s, _ := store.New(testfixture.Dataset(), filepath.Join(t.TempDir(), "state.json"))
	server := &Server{Store: s, Auth: auth.New("E0001", "employee-secret", "hr-secret"), WebDir: t.TempDir()}
	handler := server.Handler()
	request := func(method, path, body string, cookie *http.Cookie, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/hr/overview", "", nil, ""); w.Code != 401 {
		t.Fatalf("anonymous %d", w.Code)
	}
	login := request("POST", "/api/session", `{"role":"employee","password":"employee-secret"}`, nil, "")
	if login.Code != 200 {
		t.Fatal(login.Body.String())
	}
	cookie := login.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("cookie flags")
	}
	for _, path := range []string{"/api/employees", "/api/employees/E0002", "/api/employees/E0002/recommendations", "/api/hr/overview"} {
		if w := request("GET", path, "", cookie, ""); w.Code != 403 {
			t.Fatalf("%s should be forbidden: %d", path, w.Code)
		}
	}
	if w := request("POST", "/api/hr/import", "", cookie, ""); w.Code != 403 {
		t.Fatal("employee import allowed")
	}
	if w := request("POST", "/api/employees/E0002/completions", `{"event_id":"design-course"}`, cookie, ""); w.Code != 403 {
		t.Fatal("other employee mutation allowed")
	}
	if w := request("POST", "/api/employees/E0001/completions", `{"event_id":"design-course"}`, cookie, "https://evil.example"); w.Code != 403 {
		t.Fatal("cross-origin mutation allowed")
	}
	w := request("POST", "/api/employees/E0001/completions", `{"event_id":"design-course"}`, cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var result struct {
		Changed bool `json:"changed"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if !result.Changed {
		t.Fatal("completion did not change")
	}
	w = request("POST", "/api/employees/E0001/completions", `{"event_id":"design-course"}`, cookie, "")
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Changed {
		t.Fatal("duplicate completed twice")
	}
	login = request("POST", "/api/session", `{"role":"hr","password":"hr-secret"}`, nil, "")
	hrCookie := login.Result().Cookies()[0]
	if w = request("GET", "/api/hr/overview", "", hrCookie, ""); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	request("DELETE", "/api/session", "", cookie, "")
	if w = request("GET", "/api/session", "", cookie, ""); w.Code != 401 {
		t.Fatal("logout session still active")
	}
}
