package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const cookieName = "careerquest_session"

type Session struct {
	Role       string    `json:"role"`
	EmployeeID string    `json:"employee_id"`
	Expires    time.Time `json:"-"`
}
type Manager struct {
	mu                                       sync.Mutex
	sessions                                 map[string]Session
	EmployeeID, EmployeePassword, HRPassword string
}

func New(employeeID, employeePassword, hrPassword string) *Manager {
	return &Manager{sessions: map[string]Session{}, EmployeeID: employeeID, EmployeePassword: employeePassword, HRPassword: hrPassword}
}
func (m *Manager) Login(w http.ResponseWriter, role, password string) bool {
	expected := ""
	switch role {
	case "employee":
		expected = m.EmployeePassword
	case "hr":
		expected = m.HRPassword
	default:
		return false
	}
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(password)) != 1 {
		return false
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return false
	}
	token := hex.EncodeToString(b)
	expires := time.Now().Add(8 * time.Hour)
	session := Session{Role: role, Expires: expires}
	if role == "employee" {
		session.EmployeeID = m.EmployeeID
	}
	m.mu.Lock()
	for key, s := range m.sessions {
		if time.Now().After(s.Expires) {
			delete(m.sessions, key)
		}
	}
	m.sessions[token] = session
	m.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 8 * 60 * 60})
	return true
}
func (m *Manager) Get(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[cookie.Value]
	if ok && time.Now().After(s.Expires) {
		delete(m.sessions, cookie.Value)
		return Session{}, false
	}
	return s, ok
}
func (m *Manager) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		m.mu.Lock()
		delete(m.sessions, cookie.Value)
		m.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}
