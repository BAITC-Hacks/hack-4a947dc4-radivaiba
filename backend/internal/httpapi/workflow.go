package httpapi

import (
	"careerquest/internal/engine"
	"careerquest/internal/model"
	"careerquest/internal/store"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func requestLocale(r *http.Request) string {
	v := strings.ToLower(r.Header.Get("Accept-Language"))
	if strings.HasPrefix(v, "kk") || strings.HasPrefix(v, "kz") {
		return "kk"
	}
	if strings.HasPrefix(v, "en") {
		return "en"
	}
	return "ru"
}
func workflowError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		fail(w, 404, "not_found", "Запись не найдена.")
	case errors.Is(err, store.ErrForbidden):
		fail(w, 403, "forbidden", "Недостаточно прав.")
	case errors.Is(err, store.ErrInvalid):
		fail(w, 400, "invalid_input", "Проверьте введённые данные.")
	case errors.Is(err, store.ErrConflict):
		fail(w, 409, "conflict", "Состояние изменилось. Обновите страницу.")
	case errors.Is(err, store.ErrNotEligible):
		fail(w, 409, "not_eligible", "Модуль пока недоступен.")
	case errors.Is(err, store.ErrStaleRevision):
		fail(w, 409, "stale_revision", "Перезапустите сервер после административного изменения БД.")
	default:
		fail(w, 500, "save_failed", "Не удалось сохранить изменения.")
	}
}
func validMonth(month string) bool {
	if month == "" {
		return true
	}
	_, err := time.Parse("2006-01", month)
	return err == nil
}
func cleanAttachments(v *model.Submission) {
	for i := range v.Attachments {
		v.Attachments[i].StorageKey = ""
	}
}
func (s *Server) cleanDecisions(decisions []model.ReviewDecision) []model.ReviewDecision {
	result := append([]model.ReviewDecision{}, decisions...)
	for i := range result {
		// Display data is derived at the HTTP boundary; never change the audit ID
		// or decorate the immutable store snapshot in place.
		result[i].ReviewerLogin = ""
		if account, ok := s.Store.Account(result[i].ReviewerID); ok {
			result[i].ReviewerLogin = account.Login
		}
	}
	return result
}
func (s *Server) cleanEnrollment(v model.EnrollmentDetail) model.EnrollmentDetail {
	v.Versions = append([]model.Submission{}, v.Versions...)
	for i := range v.Versions {
		v.Versions[i].Attachments = append([]model.Attachment{}, v.Versions[i].Attachments...)
		cleanAttachments(&v.Versions[i])
	}
	v.Decisions = s.cleanDecisions(v.Decisions)
	return v
}
func (s *Server) cleanSubmission(v model.SubmissionDetail) model.SubmissionDetail {
	v.Submission.Attachments = append([]model.Attachment{}, v.Submission.Attachments...)
	cleanAttachments(&v.Submission)
	v.Versions = s.cleanEnrollment(model.EnrollmentDetail{Versions: v.Versions}).Versions
	v.Decisions = s.cleanDecisions(v.Decisions)
	return v
}

func (s *Server) workflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/catalog/goals", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.require(w, r, ""); !ok {
			return
		}
		items := []model.Goal{}
		for _, p := range s.Store.Snapshot().Catalog.RoleProfiles {
			items = append(items, model.Goal{TargetRole: p.Role, TargetGrade: p.Grade})
		}
		send(w, 200, map[string]any{"items": items})
	})
	mux.HandleFunc("PATCH /api/me/preferences", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.require(w, r, "")
		if !ok {
			return
		}
		var in struct {
			Locale string `json:"locale"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		v, err := s.Store.SetLocale(a.AccountID, in.Locale)
		if err != nil {
			workflowError(w, err)
			return
		}
		a.Locale = v.Locale
		send(w, 200, a)
	})
	mux.HandleFunc("PATCH /api/employees/{id}/goal", func(w http.ResponseWriter, r *http.Request) {
		e, _, ok := s.employee(w, r)
		if !ok {
			return
		}
		var in struct {
			Goal *model.Goal `json:"goal"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		v, err := s.Store.SetGoal(e.ID, in.Goal)
		if err != nil {
			workflowError(w, err)
			return
		}
		send(w, 200, v)
	})
	mux.HandleFunc("GET /api/employees/{id}/development", func(w http.ResponseWriter, r *http.Request) {
		e, d, ok := s.employee(w, r)
		if !ok {
			return
		}
		send(w, 200, engine.BuildDevelopment(d, e, requestLocale(r)))
	})
	mux.HandleFunc("POST /api/employees/{id}/modules/{event_id}/start", func(w http.ResponseWriter, r *http.Request) {
		e, _, ok := s.employee(w, r)
		if !ok {
			return
		}
		a, _ := s.require(w, r, "")
		if a.EmployeeID != e.ID {
			workflowError(w, store.ErrForbidden)
			return
		}
		en, err := s.Store.StartModule(e.ID, r.PathValue("event_id"))
		if err != nil {
			workflowError(w, err)
			return
		}
		v, err := s.Store.Enrollment(en.ID)
		if err != nil {
			workflowError(w, err)
			return
		}
		send(w, 200, s.cleanEnrollment(v))
	})
	mux.HandleFunc("GET /api/enrollments/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedEnrollment(w, r, r.PathValue("id"), false)
		if ok {
			send(w, 200, s.cleanEnrollment(v))
		}
	})
	mux.HandleFunc("POST /api/enrollments/{id}/submissions", s.submit)
	mux.HandleFunc("GET /api/hr/submissions", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.require(w, r, "hr"); !ok {
			return
		}
		q := r.URL.Query()
		for _, key := range []string{"from", "to"} {
			if q.Get(key) != "" {
				if _, err := time.Parse("2006-01-02", q.Get(key)); err != nil {
					workflowError(w, store.ErrInvalid)
					return
				}
			}
		}
		items := []model.SubmissionDetail{}
		pending := 0
		for _, v := range s.Store.Submissions() {
			if v.Submission.Status == "pending" {
				pending++
			}
			if q.Get("status") != "" && v.Submission.Status != q.Get("status") {
				continue
			}
			if q.Get("employee_id") != "" && v.Employee.ID != q.Get("employee_id") {
				continue
			}
			if q.Get("event_id") != "" && v.Event.ID != q.Get("event_id") {
				continue
			}
			date := v.Submission.BusinessDate
			if q.Get("from") != "" && date < q.Get("from") {
				continue
			}
			if q.Get("to") != "" && date > q.Get("to") {
				continue
			}
			items = append(items, s.cleanSubmission(v))
		}
		send(w, 200, map[string]any{"items": items, "pending_count": pending})
	})
	mux.HandleFunc("POST /api/hr/submissions/{id}/decision", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.require(w, r, "hr")
		if !ok {
			return
		}
		var in struct {
			Action     string `json:"action"`
			Comment    string `json:"comment"`
			RequestKey string `json:"request_key"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		v, err := s.Store.Decide(a.AccountID, r.PathValue("id"), in.Action, in.Comment, in.RequestKey)
		if err != nil {
			workflowError(w, err)
			return
		}
		send(w, 200, s.cleanSubmission(v))
	})
	mux.HandleFunc("POST /api/hr/approvals/{id}/revoke", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.require(w, r, "hr")
		if !ok {
			return
		}
		var in struct {
			Comment    string `json:"comment"`
			RequestKey string `json:"request_key"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		v, err := s.Store.Revoke(a.AccountID, r.PathValue("id"), in.Comment, in.RequestKey)
		if err != nil {
			workflowError(w, err)
			return
		}
		send(w, 200, s.cleanSubmission(v))
	})
	mux.HandleFunc("GET /api/employees/{id}/experience", func(w http.ResponseWriter, r *http.Request) {
		e, d, ok := s.employee(w, r)
		if !ok {
			return
		}
		month := r.URL.Query().Get("month")
		if !validMonth(month) {
			workflowError(w, store.ErrInvalid)
			return
		}
		send(w, 200, engine.ExperienceFor(d, e.ID, month))
	})
	mux.HandleFunc("GET /api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.require(w, r, "")
		if !ok {
			return
		}
		q := r.URL.Query()
		if !validMonth(q.Get("month")) {
			workflowError(w, store.ErrInvalid)
			return
		}
		dep, role := "", ""
		if a.Role == "hr" {
			dep = q.Get("department")
			role = q.Get("role")
		}
		send(w, 200, engine.LeaderboardFor(s.Store.Snapshot(), q.Get("month"), dep, role, a.Role == "hr"))
	})
	mux.HandleFunc("POST /api/employees/{id}/assistant", func(w http.ResponseWriter, r *http.Request) {
		e, d, ok := s.employee(w, r)
		if !ok {
			return
		}
		var in model.AssistantRequest
		if !readJSON(w, r, &in) {
			return
		}
		if in.Locale == "" {
			in.Locale = requestLocale(r)
		}
		result, err := s.LLM.Assist(r.Context(), d, e, in)
		if err != nil {
			workflowError(w, store.ErrInvalid)
			return
		}
		send(w, 200, result)
	})
	mux.HandleFunc("GET /api/attachments/{id}", s.attachment)
}
func (s *Server) authorizedEnrollment(w http.ResponseWriter, r *http.Request, id string, ownerOnly bool) (model.EnrollmentDetail, bool) {
	a, ok := s.require(w, r, "")
	if !ok {
		return model.EnrollmentDetail{}, false
	}
	v, err := s.Store.Enrollment(id)
	if err != nil {
		workflowError(w, err)
		return v, false
	}
	if a.EmployeeID != v.Enrollment.EmployeeID && (ownerOnly || a.Role != "hr") {
		workflowError(w, store.ErrForbidden)
		return v, false
	}
	return v, true
}
func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	en, ok := s.authorizedEnrollment(w, r, r.PathValue("id"), true)
	if !ok {
		return
	}
	var in struct {
		Text       string `json:"text"`
		URL        string `json:"url"`
		RequestKey string `json:"request_key"`
	}
	attachments := []model.Attachment{}
	createdPaths := []string{}
	committed := false
	defer func() {
		if !committed {
			for _, p := range createdPaths {
				_ = os.Remove(p)
			}
		}
	}()
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			fail(w, 400, "invalid_upload", "Максимум три файла по 5 МБ.")
			return
		}
		defer r.MultipartForm.RemoveAll()
		in.Text = r.FormValue("text")
		in.URL = r.FormValue("url")
		in.RequestKey = r.FormValue("request_key")
		files := r.MultipartForm.File["files"]
		if len(files) > 3 {
			workflowError(w, store.ErrInvalid)
			return
		}
		for field := range r.MultipartForm.File {
			if field != "files" {
				workflowError(w, store.ErrInvalid)
				return
			}
		}
		for _, fh := range files {
			if fh.Size <= 0 || fh.Size > 5<<20 {
				fail(w, 400, "invalid_upload", "Размер файла должен быть не больше 5 МБ.")
				return
			}
			f, err := fh.Open()
			if err != nil {
				workflowError(w, err)
				return
			}
			data, err := io.ReadAll(io.LimitReader(f, (5<<20)+1))
			f.Close()
			if err != nil || len(data) > 5<<20 {
				workflowError(w, store.ErrInvalid)
				return
			}
			media := http.DetectContentType(data)
			if media != "application/pdf" && media != "image/png" && media != "image/jpeg" {
				fail(w, 400, "invalid_file_type", "Допустимы PDF, PNG и JPEG.")
				return
			}
			var token [16]byte
			if _, err = rand.Read(token[:]); err != nil {
				workflowError(w, err)
				return
			}
			id := "AT_" + hex.EncodeToString(token[:])
			dir := s.EvidenceDir
			if dir == "" {
				dir = "data/evidence"
			}
			if err = os.MkdirAll(dir, 0700); err != nil {
				workflowError(w, err)
				return
			}
			path := filepath.Join(dir, id)
			if err = os.WriteFile(path, data, 0600); err != nil {
				workflowError(w, err)
				return
			}
			createdPaths = append(createdPaths, path)
			hash := sha256.Sum256(data)
			name := filepath.Base(strings.ReplaceAll(fh.Filename, "\\", "/"))
			if len(name) > 180 {
				name = name[:180]
			}
			attachments = append(attachments, model.Attachment{ID: id, Filename: name, MediaType: media, Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:]), StorageKey: id})
		}
	} else {
		if !readJSON(w, r, &in) {
			return
		}
	}
	v, err := s.Store.Submit(en.Enrollment.ID, en.Enrollment.EmployeeID, in.Text, in.URL, in.RequestKey, attachments)
	if err != nil {
		workflowError(w, err)
		return
	}
	// A retry may return the original submission: remove the newly uploaded duplicate files.
	kept := map[string]bool{}
	for _, a := range v.Submission.Attachments {
		kept[a.StorageKey] = true
	}
	for _, path := range createdPaths {
		if !kept[filepath.Base(path)] {
			_ = os.Remove(path)
		}
	}
	committed = true
	send(w, 200, s.cleanSubmission(v))
}
func (s *Server) attachment(w http.ResponseWriter, r *http.Request) {
	a, ok := s.require(w, r, "")
	if !ok {
		return
	}
	id := r.PathValue("id")
	for _, v := range s.Store.Submissions() {
		for _, f := range v.Submission.Attachments {
			if f.ID != id {
				continue
			}
			if a.Role != "hr" && a.EmployeeID != v.Employee.ID {
				workflowError(w, store.ErrForbidden)
				return
			}
			if filepath.Base(f.StorageKey) != f.StorageKey || !strings.HasPrefix(f.StorageKey, "AT_") {
				workflowError(w, store.ErrNotFound)
				return
			}
			dir := s.EvidenceDir
			if dir == "" {
				dir = "data/evidence"
			}
			path := filepath.Join(dir, f.StorageKey)
			if _, err := os.Stat(path); err != nil {
				workflowError(w, store.ErrNotFound)
				return
			}
			w.Header().Set("Content-Type", f.MediaType)
			w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Filename}))
			http.ServeFile(w, r, path)
			return
		}
	}
	workflowError(w, fmt.Errorf("%w", store.ErrNotFound))
}
