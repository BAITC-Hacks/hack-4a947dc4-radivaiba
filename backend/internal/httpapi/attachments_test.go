package httpapi

import (
	"bytes"
	"careerquest/internal/auth"
	"careerquest/internal/llm"
	"careerquest/internal/model"
	"careerquest/internal/store"
	"careerquest/internal/testfixture"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type attachmentFixture struct {
	t           *testing.T
	store       *store.Store
	handler     http.Handler
	dir         string
	credentials []store.Credential
	cookies     map[string]*http.Cookie
	enrollment  model.EnrollmentDetail
}

func newAttachmentFixture(t *testing.T) *attachmentFixture {
	t.Helper()
	data, err := store.New(testfixture.Dataset(), filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	credentials, err := data.ProvisionAccounts("", "reviewer", "E0002")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &attachmentFixture{t: t, store: data, dir: t.TempDir(), credentials: credentials, cookies: map[string]*http.Cookie{}}
	server := &Server{Store: data, Auth: auth.New("", "", ""), LLM: &llm.Client{}, WebDir: t.TempDir(), EvidenceDir: fixture.dir}
	fixture.handler = server.Handler()
	started := fixture.json("POST", "/api/employees/E0001/modules/design-course/start", nil, "E0001")
	if started.Code != http.StatusOK {
		t.Fatal(started.Code, started.Body.String())
	}
	if err := json.Unmarshal(started.Body.Bytes(), &fixture.enrollment); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func (f *attachmentFixture) login(name string) *http.Cookie {
	f.t.Helper()
	if cookie := f.cookies[name]; cookie != nil {
		return cookie
	}
	for _, credential := range f.credentials {
		if credential.Login != name {
			continue
		}
		body, _ := json.Marshal(map[string]string{"login": credential.Login, "password": credential.Password})
		request := httptest.NewRequest("POST", "/api/session", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		f.handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || len(response.Result().Cookies()) == 0 {
			f.t.Fatal(response.Code, response.Body.String())
		}
		f.cookies[name] = response.Result().Cookies()[0]
		return f.cookies[name]
	}
	f.t.Fatalf("missing test account %s", name)
	return nil
}

func (f *attachmentFixture) request(method, path, contentType string, body io.Reader, account string) *httptest.ResponseRecorder {
	f.t.Helper()
	request := httptest.NewRequest(method, path, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if account != "" {
		request.AddCookie(f.login(account))
	}
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	return response
}

func (f *attachmentFixture) json(method, path string, body any, account string) *httptest.ResponseRecorder {
	f.t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		f.t.Fatal(err)
	}
	return f.request(method, path, "application/json", bytes.NewReader(raw), account)
}

type proofFile struct {
	name string
	data []byte
}

func (f *attachmentFixture) upload(key, text, link string, files []proofFile, account string) *httptest.ResponseRecorder {
	f.t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range map[string]string{"request_key": key, "text": text, "url": link} {
		if err := writer.WriteField(name, value); err != nil {
			f.t.Fatal(err)
		}
	}
	for _, file := range files {
		part, err := writer.CreateFormFile("files", file.name)
		if err != nil {
			f.t.Fatal(err)
		}
		if _, err := part.Write(file.data); err != nil {
			f.t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		f.t.Fatal(err)
	}
	return f.request("POST", "/api/enrollments/"+f.enrollment.Enrollment.ID+"/submissions", writer.FormDataContentType(), &body, account)
}

func proofPNG(t *testing.T) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	picture.Set(0, 0, color.RGBA{R: 40, G: 160, B: 90, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func (f *attachmentFixture) diskFiles() []os.DirEntry {
	f.t.Helper()
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		f.t.Fatal(err)
	}
	return entries
}

func decodeProofSubmission(t *testing.T, response *httptest.ResponseRecorder) model.SubmissionDetail {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatal(response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "storage_key") {
		t.Fatal("API exposed internal attachment storage key")
	}
	var result model.SubmissionDetail
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestProofUploadDownloadPermissionsAndIdempotentRetry(t *testing.T) {
	f := newAttachmentFixture(t)
	pngData := proofPNG(t)
	files := []proofFile{{name: `..\screenshot.png`, data: pngData}, {name: "result.pdf", data: []byte("%PDF-1.4\n% Test evidence\n%%EOF\n")}}
	created := decodeProofSubmission(t, f.upload("proof-request-001", "Architecture diagram and decision rationale.", "https://example.com/review", files, "E0001"))
	if len(created.Submission.Attachments) != 2 || len(f.diskFiles()) != 2 {
		t.Fatal("proof files were not stored")
	}
	if created.Submission.Attachments[0].Filename != "screenshot.png" {
		t.Fatal("unsafe filename was not normalized")
	}
	for index, attachment := range created.Submission.Attachments {
		if attachment.StorageKey != "" || attachment.Size != int64(len(files[index].data)) {
			t.Fatal("invalid public attachment metadata")
		}
		for _, account := range []string{"E0001", "reviewer"} {
			download := f.request("GET", "/api/attachments/"+attachment.ID, "", nil, account)
			if download.Code != http.StatusOK || !bytes.Equal(download.Body.Bytes(), files[index].data) {
				t.Fatalf("download as %s: %d", account, download.Code)
			}
			kind, params, err := mime.ParseMediaType(download.Header().Get("Content-Disposition"))
			if err != nil || kind != "attachment" || params["filename"] != attachment.Filename {
				t.Fatal("proof must be served as a download with safe filename")
			}
			if download.Header().Get("X-Content-Type-Options") != "nosniff" || download.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("private download headers missing")
			}
		}
		if response := f.request("GET", "/api/attachments/"+attachment.ID, "", nil, "E0002"); response.Code != http.StatusForbidden {
			t.Fatal("another employee could download evidence", response.Code)
		}
		if response := f.request("GET", "/api/attachments/"+attachment.ID, "", nil, ""); response.Code != http.StatusUnauthorized {
			t.Fatal("anonymous evidence access", response.Code)
		}
	}
	if response := f.upload("foreign-request-001", "Trying another employee's enrollment.", "", files[:1], "E0002"); response.Code != http.StatusForbidden {
		t.Fatal("another employee submitted evidence", response.Code)
	}
	if len(f.diskFiles()) != 2 {
		t.Fatal("forbidden upload wrote files")
	}
	before := f.store.Snapshot().Revision
	retry := decodeProofSubmission(t, f.upload("proof-request-001", "Architecture diagram and decision rationale.", "https://example.com/review", files, "E0001"))
	if retry.Submission.ID != created.Submission.ID || len(retry.Versions) != 1 || f.store.Snapshot().Revision != before {
		t.Fatal("retry created a second submission")
	}
	if len(f.diskFiles()) != 2 {
		t.Fatal("retry left duplicate proof files")
	}
	// A different key while pending fails; files written before the state check must be cleaned up.
	if response := f.upload("proof-request-002", "Unexpected second submission while pending.", "", files, "E0001"); response.Code != http.StatusConflict {
		t.Fatal("parallel pending submission accepted", response.Code)
	}
	if len(f.diskFiles()) != 2 {
		t.Fatal("failed submission left orphan files")
	}
	for _, route := range []struct{ path, account string }{{"/api/enrollments/" + f.enrollment.Enrollment.ID, "E0001"}, {"/api/hr/submissions", "reviewer"}} {
		response := f.request("GET", route.path, "", nil, route.account)
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "storage_key") {
			t.Fatal("detail or HR queue exposed storage key")
		}
	}
	// Cleaning the API response must not erase the internal key used to download the proof later.
	raw, err := f.store.Submission(created.Submission.ID)
	if err != nil || raw.Submission.Attachments[0].StorageKey == "" {
		t.Fatal("response cleaning mutated storage")
	}
}

func TestProofRejectsUnsupportedTypesCountAndSizeWithoutOrphans(t *testing.T) {
	f := newAttachmentFixture(t)
	small := []byte("%PDF-1.4\n%%EOF\n")
	large := append([]byte("%PDF-1.4\n"), make([]byte, 5<<20)...)
	for _, tc := range []struct {
		name  string
		files []proofFile
	}{
		{"extension-cannot-disguise-html", []proofFile{{"proof.pdf", []byte("<!DOCTYPE html><script>alert('proof')</script>")}}},
		{"svg-is-not-an-accepted-image", []proofFile{{"proof.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)}}},
		{"four-files", []proofFile{{"one.pdf", small}, {"two.pdf", small}, {"three.pdf", small}, {"four.pdf", small}}},
		{"over-five-megabytes", []proofFile{{"large.pdf", large}}},
		{"empty-file", []proofFile{{"empty.pdf", nil}}},
		{"valid-first-then-invalid", []proofFile{{"valid.pdf", small}, {"invalid.pdf", []byte("ordinary text")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			revision := f.store.Snapshot().Revision
			response := f.upload("invalid-proof-"+tc.name, "A result for review.", "", tc.files, "E0001")
			if response.Code != http.StatusBadRequest {
				t.Fatal(response.Code, response.Body.String())
			}
			if len(f.diskFiles()) != 0 || len(f.store.Snapshot().Workflow.Submissions) != 0 || f.store.Snapshot().Revision != revision {
				t.Fatal("invalid package changed files or persisted workflow")
			}
		})
	}
	// The limit is inclusive, and three valid files are accepted together.
	exactLimit := append([]byte("%PDF-1.4\n"), make([]byte, (5<<20)-len("%PDF-1.4\n"))...)
	accepted := decodeProofSubmission(t, f.upload("valid-proof-boundary", "", "", []proofFile{{"limit.pdf", exactLimit}, {"two.pdf", small}, {"three.png", proofPNG(t)}}, "E0001"))
	if len(accepted.Submission.Attachments) != 3 || len(f.diskFiles()) != 3 {
		t.Fatal("valid three-file boundary rejected")
	}
}

func TestProofJSONValidationAndHRIdentitySelfReview(t *testing.T) {
	f := newAttachmentFixture(t)
	path := "/api/enrollments/" + f.enrollment.Enrollment.ID + "/submissions"
	for _, tc := range []struct {
		name string
		body any
	}{
		{"empty-evidence", map[string]string{"request_key": "invalid-evidence", "text": ""}},
		{"short-text", map[string]string{"request_key": "invalid-short-text", "text": "tiny"}},
		{"http-link", map[string]string{"request_key": "invalid-http-link", "text": "A detailed result.", "url": "http://example.com/proof"}},
		{"script-link", map[string]string{"request_key": "invalid-script-link", "text": "A detailed result.", "url": "javascript:alert(1)"}},
		{"credentials-in-link", map[string]string{"request_key": "invalid-user-link", "text": "A detailed result.", "url": "https://user:password@example.com/proof"}},
		{"missing-link-host", map[string]string{"request_key": "invalid-host-link", "text": "A detailed result.", "url": "https:///proof"}},
		{"too-long-text", map[string]string{"request_key": "invalid-long-text", "text": strings.Repeat("x", 12001)}},
		{"client-injected-attachments", map[string]any{"request_key": "invalid-fake-file", "text": "A detailed result.", "attachments": []map[string]string{{"storage_key": "../../private"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := f.json("POST", path, tc.body, "E0001")
			if response.Code != http.StatusBadRequest {
				t.Fatal(response.Code, response.Body.String())
			}
			if len(f.store.Snapshot().Workflow.Submissions) != 0 {
				t.Fatal("invalid JSON evidence created a submission")
			}
		})
	}
	created := decodeProofSubmission(t, f.json("POST", path, map[string]string{"request_key": "valid-json-proof", "text": "Explained architecture tradeoffs and practical outcome.", "url": "https://example.com/proof?version=1"}, "E0001"))
	if created.Submission.URL != "https://example.com/proof?version=1" || created.Submission.Status != "pending" {
		t.Fatal("valid text and HTTPS link not stored")
	}
	for _, login := range []string{"owner-hr-one", "owner-hr-two"} {
		credentials, err := f.store.ProvisionAccounts("E0001", login, "E0001")
		if err != nil {
			t.Fatal(err)
		}
		f.credentials = append(f.credentials, credentials...)
		response := f.json("POST", "/api/hr/submissions/"+created.Submission.ID+"/decision", map[string]string{"action": "approve", "request_key": "review-" + login}, login)
		if response.Code != http.StatusForbidden {
			t.Fatal("HR account linked to the submitter approved own work", login, response.Code)
		}
	}
	if len(f.store.Snapshot().Workflow.Ledger) != 0 || len(f.store.Snapshot().History) != 0 {
		t.Fatal("self-review changed progress")
	}
	reviewed := decodeProofSubmission(t, f.json("POST", "/api/hr/submissions/"+created.Submission.ID+"/decision", map[string]string{"action": "approve", "request_key": "review-independent-hr"}, "reviewer"))
	if reviewed.Submission.Status != "completed" || len(reviewed.Decisions) != 1 {
		t.Fatal("independent HR could not approve")
	}
	approvalID := reviewed.Decisions[0].ID
	if response := f.json("POST", "/api/hr/approvals/"+approvalID+"/revoke", map[string]string{"comment": "Attempt to change my own record", "request_key": "revoke-own-approval"}, "owner-hr-two"); response.Code != http.StatusForbidden {
		t.Fatal("linked HR account changed its own completion via revoke", response.Code)
	}
}
