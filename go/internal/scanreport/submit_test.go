package scanreport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubmitReport(t *testing.T) {
	var gotContentType string
	var gotFields map[string]string
	var gotFileSize int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ce/submit" {
			w.WriteHeader(404)
			return
		}
		gotContentType = r.Header.Get("Content-Type")

		mediaType, params, _ := mime.ParseMediaType(gotContentType)
		if mediaType != "multipart/form-data" {
			t.Errorf("expected multipart/form-data, got %s", mediaType)
		}

		gotFields = make(map[string]string)
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("reading part: %v", err)
			}
			data, _ := io.ReadAll(part)
			name := part.FormName()
			if name == "report" {
				gotFileSize = len(data)
			} else {
				gotFields[name] = string(data)
			}
			part.Close()
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"taskId": "AX-123"})
	}))
	defer srv.Close()

	cfg := SubmitConfig{
		CloudURL:   srv.URL + "/",
		ProjectKey: "my-proj",
		OrgKey:     "my-org",
		BranchName: "main",
	}

	result, err := SubmitReport(context.Background(), srv.Client(), cfg, []byte("fake-zip-data"))
	if err != nil {
		t.Fatalf("SubmitReport: %v", err)
	}
	if result.TaskID != "AX-123" {
		t.Errorf("expected taskId AX-123, got %s", result.TaskID)
	}
	if gotFileSize != len("fake-zip-data") {
		t.Errorf("expected file size %d, got %d", len("fake-zip-data"), gotFileSize)
	}
	if gotFields["projectKey"] != "my-proj" {
		t.Errorf("expected projectKey my-proj, got %s", gotFields["projectKey"])
	}
	if gotFields["organization"] != "my-org" {
		t.Errorf("expected organization my-org, got %s", gotFields["organization"])
	}
	if !strings.Contains(gotFields["properties"], "sonar.projectKey=my-proj") {
		t.Errorf("expected properties to contain projectKey, got %s", gotFields["properties"])
	}
}

func TestSubmitReportHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprint(w, `{"errors":[{"msg":"bad request"}]}`)
	}))
	defer srv.Close()

	cfg := SubmitConfig{CloudURL: srv.URL + "/", ProjectKey: "p", OrgKey: "o"}
	_, err := SubmitReport(context.Background(), srv.Client(), cfg, []byte("zip"))
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to contain 400, got: %v", err)
	}
}

func TestPollCETaskSuccess(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var status string
		if callCount < 3 {
			status = "PENDING"
		} else {
			status = "SUCCESS"
		}
		json.NewEncoder(w).Encode(map[string]any{
			"task": map[string]string{"id": "AX-123", "status": status},
		})
	}))
	defer srv.Close()

	logger := slog.Default()
	err := PollCETask(context.Background(), srv.Client(), srv.URL+"/", "AX-123", logger)
	if err != nil {
		t.Fatalf("PollCETask: %v", err)
	}
}

func TestPollCETaskFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"task": map[string]string{"id": "AX-123", "status": "FAILED", "errorMessage": "analysis error"},
		})
	}))
	defer srv.Close()

	logger := slog.Default()
	err := PollCETask(context.Background(), srv.Client(), srv.URL+"/", "AX-123", logger)
	if err == nil {
		t.Fatal("expected error for failed task")
	}
	if !strings.Contains(err.Error(), "analysis error") {
		t.Errorf("expected error message, got: %v", err)
	}
}

func TestParseTaskID(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		want  string
	}{
		{"simple", `{"taskId":"AX-1"}`, "AX-1"},
		{"nested", `{"task":{"id":"AX-2","status":"PENDING"}}`, "AX-2"},
		{"empty", `{}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTaskID([]byte(tc.body))
			if got != tc.want {
				t.Errorf("parseTaskID(%s): got %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestExtractJSONField(t *testing.T) {
	body := `{"task":{"id":"AX-1","status":"SUCCESS","errorMessage":"oops"}}`
	if got := extractJSONField([]byte(body), "task", "status"); got != "SUCCESS" {
		t.Errorf("expected SUCCESS, got %q", got)
	}
	if got := extractJSONField([]byte(body), "task", "errorMessage"); got != "oops" {
		t.Errorf("expected oops, got %q", got)
	}
	if got := extractJSONField([]byte(body), "task", "missing"); got != "" {
		t.Errorf("expected empty for missing field, got %q", got)
	}
}
