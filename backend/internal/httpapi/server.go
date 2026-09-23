package httpapi

import (
	"careerquest/internal/auth"
	"careerquest/internal/engine"
	"careerquest/internal/llm"
	"careerquest/internal/model"
	"careerquest/internal/store"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Store     *store.Store
	Auth      *auth.Manager
	LLM       *llm.Client
	WebDir    string
	DevOrigin string
	rateMu    sync.Mutex
	attempts  map[string]attempt
}
type attempt struct {
	Count int
	Until time.Time
}

func send(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, code, message string) {
	send(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		fail(w, 400, "invalid_json", "Некорректный JSON запроса.")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		fail(w, 400, "invalid_json", "Ожидался один JSON объект.")
		return false
	}
	return true
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.attempts = map[string]attempt{}
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		d := s.Store.Snapshot()
		send(w, 200, map[string]any{"status": "ok", "as_of_date": d.Catalog.Meta.AsOfDate, "revision": d.Revision})
	})
	mux.HandleFunc("POST /api/session", s.login)
	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, r *http.Request) {
		if session, ok := s.require(w, r, ""); ok {
			send(w, 200, session)
		}
	})
	mux.HandleFunc("DELETE /api/session", func(w http.ResponseWriter, r *http.Request) { s.Auth.Logout(w, r); w.WriteHeader(204) })
	mux.HandleFunc("GET /api/employees", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.require(w, r, "hr"); !ok {
			return
		}
		employees := s.Store.Snapshot().Employees
		items := make([]map[string]string, 0, len(employees))
		for _, e := range employees {
			items = append(items, map[string]string{"employee_id": e.ID, "full_name": e.FullName, "department": e.Department, "role": e.Role, "grade": e.Grade})
		}
		send(w, 200, map[string]any{"items": items})
	})
	mux.HandleFunc("GET /api/employees/{id}", func(w http.ResponseWriter, r *http.Request) {
		employee, d, ok := s.employee(w, r)
		if ok {
			send(w, 200, engine.BuildProfile(d, employee))
		}
	})
	mux.HandleFunc("GET /api/employees/{id}/recommendations", func(w http.ResponseWriter, r *http.Request) {
		employee, d, ok := s.employee(w, r)
		if !ok {
			return
		}
		candidates := engine.RankCandidates(d, employee)
		result := s.LLM.Recommend(r.Context(), candidates, d.Revision)
		if len(candidates) == 0 {
			result.EmptyReason = engine.EmptyReason(d, employee)
		}
		send(w, 200, result)
	})
	mux.HandleFunc("POST /api/employees/{id}/completions", s.complete)
	mux.HandleFunc("GET /api/hr/overview", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.require(w, r, "hr"); ok {
			send(w, 200, engine.SummarizeHR(s.Store.Snapshot()))
		}
	})
	mux.HandleFunc("POST /api/hr/import", s.importData)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		fail(w, 404, "not_found", "API маршрут не найден.")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.static(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || ((u.Host != r.Host || !oneScheme(u.Scheme)) && origin != s.DevOrigin) {
					fail(w, 403, "origin_denied", "Недопустимый источник запроса.")
					return
				}
			}
		}
		mux.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
func oneScheme(s string) bool { return s == "http" || s == "https" }
func (s *Server) require(w http.ResponseWriter, r *http.Request, role string) (auth.Session, bool) {
	session, ok := s.Auth.Get(r)
	if !ok {
		fail(w, 401, "unauthorized", "Войдите в демо-аккаунт.")
		return session, false
	}
	if role != "" && session.Role != role {
		fail(w, 403, "forbidden", "Недостаточно прав.")
		return session, false
	}
	return session, true
}
func (s *Server) employee(w http.ResponseWriter, r *http.Request) (model.Employee, model.Dataset, bool) {
	session, ok := s.require(w, r, "")
	if !ok {
		return model.Employee{}, model.Dataset{}, false
	}
	id := r.PathValue("id")
	if session.Role != "hr" && session.EmployeeID != id {
		fail(w, 403, "forbidden", "Доступен только собственный профиль.")
		return model.Employee{}, model.Dataset{}, false
	}
	d := s.Store.Snapshot()
	employee, ok := engine.FindEmployee(d, id)
	if !ok {
		fail(w, 404, "not_found", "Сотрудник не найден.")
	}
	return employee, d, ok
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	s.rateMu.Lock()
	now := time.Now()
	for key, a := range s.attempts {
		if now.After(a.Until) {
			delete(s.attempts, key)
		}
	}
	a := s.attempts[ip]
	if a.Count == 0 {
		a.Until = now.Add(time.Minute)
	}
	a.Count++
	s.attempts[ip] = a
	s.rateMu.Unlock()
	if a.Count > 15 {
		fail(w, 429, "rate_limit", "Слишком много попыток. Подождите минуту.")
		return
	}
	var input struct {
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	if !s.Auth.Login(w, input.Role, input.Password) {
		fail(w, 401, "invalid_credentials", "Неверный пароль демо-аккаунта.")
		return
	}
	session := auth.Session{Role: input.Role}
	if input.Role == "employee" {
		session.EmployeeID = s.Auth.EmployeeID
	}
	send(w, 200, session)
}
func (s *Server) complete(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.employee(w, r); !ok {
		return
	}
	var input struct {
		EventID string `json:"event_id"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	profile, changed, err := s.Store.Complete(r.PathValue("id"), input.EventID)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "not_found", "Сотрудник или активность не найдены.")
		return
	}
	if errors.Is(err, store.ErrNotEligible) {
		fail(w, 409, "not_eligible", "Активность больше недоступна или не сокращает разрыв до цели. Обновите рекомендации.")
		return
	}
	if err != nil {
		log.Printf("completion persistence failed: %v", err)
		fail(w, 500, "save_failed", "Не удалось сохранить прогресс. Изменения не применены.")
		return
	}
	send(w, 200, map[string]any{"profile": profile, "changed": changed})
}
func (s *Server) importData(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.require(w, r, "hr"); !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		fail(w, 400, "invalid_upload", "Ожидаются multipart файлы размером до 12 МБ.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	readFile := func(name string) ([]byte, error) {
		f, _, err := r.FormFile(name)
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}
	employees, err := readFile("employees")
	if err != nil {
		fail(w, 400, "invalid_upload", "Не удалось прочитать employees.")
		return
	}
	history, err := readFile("history")
	if err != nil {
		fail(w, 400, "invalid_upload", "Не удалось прочитать history.")
		return
	}
	result, err := s.Store.Import(employees, history)
	if err != nil {
		fail(w, 422, "invalid_dataset", err.Error())
		return
	}
	send(w, 200, result)
}
func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	clean := filepath.Clean(filepath.FromSlash(r.URL.Path))
	path := filepath.Join(s.WebDir, strings.TrimLeft(clean, "/\\"))
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.FileServer(http.Dir(s.WebDir)).ServeHTTP(w, r)
		return
	}
	if filepath.Ext(r.URL.Path) != "" {
		http.NotFound(w, r)
		return
	}
	index := filepath.Join(s.WebDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.Error(w, "Frontend is not built. Run npm run demo or npm run dev.", 503)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, index)
}
