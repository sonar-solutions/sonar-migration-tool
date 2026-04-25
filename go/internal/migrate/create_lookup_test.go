package migrate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

func newLookupServer(handlers map[string]string) *httptest.Server {
	mux := http.NewServeMux()
	for path, body := range handlers {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, body)
		})
	}
	return httptest.NewServer(mux)
}

func TestLookupExistingProject(t *testing.T) {
	got := lookupExistingProject("my-org", "my-proj")
	if got != "my-org_my-proj" {
		t.Errorf("got %q, want %q", got, "my-org_my-proj")
	}
}

func TestLookupExistingProfile(t *testing.T) {
	srv := newLookupServer(map[string]string{
		"/api/qualityprofiles/search": `{"profiles":[{"key":"prof-1","name":"MyProfile","language":"java"}]}`,
	})
	defer srv.Close()
	raw := common.NewRawClient(srv.Client(), srv.URL+"/")

	key, err := lookupExistingProfile(context.Background(), raw, "MyProfile", "java", "org")
	if err != nil {
		t.Fatal(err)
	}
	if key != "prof-1" {
		t.Errorf("got %q, want %q", key, "prof-1")
	}
}

func TestLookupExistingProfileNotFound(t *testing.T) {
	srv := newLookupServer(map[string]string{
		"/api/qualityprofiles/search": `{"profiles":[{"key":"prof-1","name":"Other","language":"java"}]}`,
	})
	defer srv.Close()
	raw := common.NewRawClient(srv.Client(), srv.URL+"/")

	_, err := lookupExistingProfile(context.Background(), raw, "Missing", "java", "org")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestLookupExistingGate(t *testing.T) {
	srv := newLookupServer(map[string]string{
		"/api/qualitygates/list": `{"qualitygates":[{"id":42,"name":"Better"}]}`,
	})
	defer srv.Close()
	raw := common.NewRawClient(srv.Client(), srv.URL+"/")

	id, err := lookupExistingGate(context.Background(), raw, "Better", "org")
	if err != nil {
		t.Fatal(err)
	}
	if id != "42" {
		t.Errorf("got %q, want %q", id, "42")
	}
}

func TestLookupExistingGroup(t *testing.T) {
	srv := newLookupServer(map[string]string{
		"/api/user_groups/search": `{"groups":[{"id":7,"name":"Admins"}]}`,
	})
	defer srv.Close()
	raw := common.NewRawClient(srv.Client(), srv.URL+"/")

	id, err := lookupExistingGroup(context.Background(), raw, "Admins", "org")
	if err != nil {
		t.Fatal(err)
	}
	if id != "7" {
		t.Errorf("got %q, want %q", id, "7")
	}
}

func TestLookupExistingTemplate(t *testing.T) {
	srv := newLookupServer(map[string]string{
		"/api/permissions/search_templates": `{"permissionTemplates":[{"id":"tpl-1","name":"Default Template"}]}`,
	})
	defer srv.Close()
	raw := common.NewRawClient(srv.Client(), srv.URL+"/")

	id, err := lookupExistingTemplate(context.Background(), raw, "default template", "org")
	if err != nil {
		t.Fatal(err)
	}
	if id != "tpl-1" {
		t.Errorf("got %q, want %q", id, "tpl-1")
	}
}

func TestLookupExistingTemplateNotFound(t *testing.T) {
	srv := newLookupServer(map[string]string{
		"/api/permissions/search_templates": `{"permissionTemplates":[]}`,
	})
	defer srv.Close()
	raw := common.NewRawClient(srv.Client(), srv.URL+"/")

	_, err := lookupExistingTemplate(context.Background(), raw, "Missing", "org")
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}
