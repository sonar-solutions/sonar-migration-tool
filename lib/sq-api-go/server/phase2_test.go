package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

const (
	testGroupSonarUsers = "sonar-users"
	testRuleKeyJava     = "java:S1234"
	testGateSonarWay    = "Sonar way"
	testAPIServerVer    = "/api/server/version"
)

// --- Projects ---

func TestProjectsSearch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects/search", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.ProjectsSearchResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 2, PageIndex: 1, PageSize: 500}},
			Components:    []types.Project{{Key: "proj1", Name: "Project One"}, {Key: "proj2", Name: "Project Two"}},
		})
	})
	sc := newTestServer(t, mux)

	projects, err := sc.Projects.Search(context.Background()).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "proj1", projects[0].Key)
}

func TestProjectsSearchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects/search", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Projects.Search(context.Background()).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestProjectsGetDetails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/navigation/component", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "myproject", r.URL.Query().Get("component"))
		writeJSON(w, types.NavigationComponentResponse{
			Key: "myproject", Name: "My Project", Qualifier: "TRK", Visibility: "public",
		})
	})
	sc := newTestServer(t, mux)

	details, err := sc.Projects.GetDetails(context.Background(), "myproject")
	require.NoError(t, err)
	assert.Equal(t, "myproject", details.Key)
	assert.Equal(t, "TRK", details.Qualifier)
}

func TestProjectsGetDetailsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/navigation/component", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Not found"}]}`, http.StatusNotFound)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Projects.GetDetails(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, sqapi.IsNotFound(err))
}

func TestProjectsLicenseUsage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects/license_usage", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.ProjectsLicenseUsageResponse{
			Projects: []types.Project{{Key: "proj1", Name: "Project One"}},
		})
	})
	sc := newTestServer(t, mux)

	projects, err := sc.Projects.LicenseUsage(context.Background())
	require.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Equal(t, "proj1", projects[0].Key)
}

func TestProjectsLicenseUsageError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects/license_usage", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Projects.LicenseUsage(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestProjectsTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/components/show", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "myproject", r.URL.Query().Get("component"))
		writeJSON(w, types.ComponentShowResponse{
			Component: types.ComponentDetails{Key: "myproject", Tags: []string{"java", "backend"}},
		})
	})
	sc := newTestServer(t, mux)

	tags, err := sc.Projects.Tags(context.Background(), "myproject")
	require.NoError(t, err)
	assert.Equal(t, []string{"java", "backend"}, tags)
}

func TestProjectsTagsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/components/show", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Not found"}]}`, http.StatusNotFound)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Projects.Tags(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, sqapi.IsNotFound(err))
}

// --- Users ---

func TestUsersSearch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users/search", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.UsersSearchResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 100}},
			Users:         []types.User{{Login: "admin", Name: "Admin", Active: true}},
		})
	})
	sc := newTestServer(t, mux)

	users, err := sc.Users.Search(context.Background()).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "admin", users[0].Login)
}

func TestUsersSearchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users/search", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Users.Search(context.Background()).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

// --- Groups ---

func TestGroupsSearch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/permissions/groups", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.GroupsSearchResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 100}},
			Groups:        []types.Group{{ID: 1, Name: testGroupSonarUsers, MembersCount: 5}},
		})
	})
	sc := newTestServer(t, mux)

	groups, err := sc.Groups.Search(context.Background()).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, testGroupSonarUsers, groups[0].Name)
}

func TestGroupsSearchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/permissions/groups", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Groups.Search(context.Background()).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestGroupsUsers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user_groups/users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testGroupSonarUsers, r.URL.Query().Get("name"))
		writeJSON(w, types.GroupUsersResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 500}},
			Users:         []types.GroupUser{{Login: "alice", Name: "Alice"}},
		})
	})
	sc := newTestServer(t, mux)

	users, err := sc.Groups.Users(context.Background(), testGroupSonarUsers).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Login)
}

func TestGroupsUsersError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user_groups/users", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Groups.Users(context.Background(), testGroupSonarUsers).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

// --- Rules ---

func TestRulesSearch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rules/search", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.RulesSearchResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 500}},
			Rules:         []types.Rule{{Key: testRuleKeyJava, Name: "Do something"}},
		})
	})
	sc := newTestServer(t, mux)

	rules, err := sc.Rules.Search(context.Background()).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, testRuleKeyJava, rules[0].Key)
}

func TestRulesSearchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rules/search", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Rules.Search(context.Background()).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestRulesShow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rules/show", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testRuleKeyJava, r.URL.Query().Get("key"))
		writeJSON(w, types.RuleShowResponse{
			Rule: types.Rule{Key: testRuleKeyJava, Name: "Do something", Severity: "MAJOR"},
		})
	})
	sc := newTestServer(t, mux)

	rule, err := sc.Rules.Show(context.Background(), testRuleKeyJava)
	require.NoError(t, err)
	assert.Equal(t, testRuleKeyJava, rule.Key)
	assert.Equal(t, "MAJOR", rule.Severity)
}

func TestRulesShowError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rules/show", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Not found"}]}`, http.StatusNotFound)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Rules.Show(context.Background(), "nonexistent:rule")
	require.Error(t, err)
	assert.True(t, sqapi.IsNotFound(err))
}

func TestRulesRepositories(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rules/repositories", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.RepositoriesResponse{
			Repositories: []types.RuleRepository{{Key: "java", Name: "Java", Language: "java"}},
		})
	})
	sc := newTestServer(t, mux)

	repos, err := sc.Rules.Repositories(context.Background())
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "java", repos[0].Key)
}

func TestRulesRepositoriesError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rules/repositories", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.Rules.Repositories(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

// --- System ---

func TestSystemInfo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"System": map[string]string{"Edition": "Community"},
		})
	})
	sc := newTestServer(t, mux)

	info, err := sc.System.Info(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, info)
}

func TestSystemInfoError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/info", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.System.Info(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestSystemVersion(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(testAPIServerVer, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("10.7.0.96311"))
	})
	sc := newTestServer(t, mux)

	version, err := sc.System.Version(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, 10.7, version, 1e-9)
}

func TestSystemVersionError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(testAPIServerVer, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	sc := newTestServer(t, mux)

	_, err := sc.System.Version(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsUnauthorized(err))
}

// --- QualityGates ---

func TestQualityGatesList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.QualityGatesListResponse{
			QualityGates: []types.QualityGate{{ID: 1, Name: testGateSonarWay, IsDefault: true}},
		})
	})
	sc := newTestServer(t, mux)

	gates, err := sc.QualityGates.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, gates, 1)
	assert.Equal(t, testGateSonarWay, gates[0].Name)
}

func TestQualityGatesListError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/list", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityGates.List(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestQualityGatesShow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/show", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testGateSonarWay, r.URL.Query().Get("name"))
		writeJSON(w, types.QualityGate{ID: 1, Name: testGateSonarWay, IsDefault: true})
	})
	sc := newTestServer(t, mux)

	gate, err := sc.QualityGates.Show(context.Background(), testGateSonarWay)
	require.NoError(t, err)
	assert.Equal(t, testGateSonarWay, gate.Name)
}

func TestQualityGatesShowError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/show", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Not found"}]}`, http.StatusNotFound)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityGates.Show(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, sqapi.IsNotFound(err))
}

func TestQualityGatesSearchGroups(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/search_groups", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testGateSonarWay, r.URL.Query().Get("gateName"))
		writeJSON(w, types.QualityGateGroupsResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 1000}},
			Groups:        []types.QualityGateGroup{{Name: testGroupSonarUsers}},
		})
	})
	sc := newTestServer(t, mux)

	groups, err := sc.QualityGates.SearchGroups(context.Background(), testGateSonarWay).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, testGroupSonarUsers, groups[0].Name)
}

func TestQualityGatesSearchGroupsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/search_groups", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityGates.SearchGroups(context.Background(), testGateSonarWay).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestQualityGatesSearchUsers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/search_users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testGateSonarWay, r.URL.Query().Get("gateName"))
		writeJSON(w, types.QualityGateUsersResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 1000}},
			Users:         []types.QualityGateUser{{Login: "admin", Name: "Admin"}},
		})
	})
	sc := newTestServer(t, mux)

	users, err := sc.QualityGates.SearchUsers(context.Background(), testGateSonarWay).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "admin", users[0].Login)
}

func TestQualityGatesSearchUsersError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualitygates/search_users", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityGates.SearchUsers(context.Background(), testGateSonarWay).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

// --- QualityProfiles ---

func TestQualityProfilesSearch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/search", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, types.QualityProfilesSearchResponse{
			Profiles: []types.QualityProfile{{Key: "qp1", Name: testGateSonarWay, Language: "java"}},
		})
	})
	sc := newTestServer(t, mux)

	profiles, err := sc.QualityProfiles.Search(context.Background())
	require.NoError(t, err)
	assert.Len(t, profiles, 1)
	assert.Equal(t, "qp1", profiles[0].Key)
}

func TestQualityProfilesSearchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/search", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityProfiles.Search(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestQualityProfilesBackup(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/backup", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "java", r.URL.Query().Get("language"))
		assert.Equal(t, testGateSonarWay, r.URL.Query().Get("qualityProfile"))
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><profile/>`))
	})
	sc := newTestServer(t, mux)

	data, err := sc.QualityProfiles.Backup(context.Background(), "java", testGateSonarWay)
	require.NoError(t, err)
	assert.Contains(t, string(data), "<profile")
}

func TestQualityProfilesBackupError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/backup", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Not found"}]}`, http.StatusNotFound)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityProfiles.Backup(context.Background(), "java", "nonexistent")
	require.Error(t, err)
	assert.True(t, sqapi.IsNotFound(err))
}

func TestQualityProfilesSearchGroups(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/search_groups", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "java", r.URL.Query().Get("language"))
		assert.Equal(t, testGateSonarWay, r.URL.Query().Get("qualityProfile"))
		writeJSON(w, types.ProfileGroupsResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 500}},
			Groups:        []types.ProfileGroup{{Name: testGroupSonarUsers}},
		})
	})
	sc := newTestServer(t, mux)

	groups, err := sc.QualityProfiles.SearchGroups(context.Background(), "java", testGateSonarWay).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, testGroupSonarUsers, groups[0].Name)
}

func TestQualityProfilesSearchGroupsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/search_groups", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityProfiles.SearchGroups(context.Background(), "java", testGateSonarWay).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

func TestQualityProfilesSearchUsers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/search_users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "java", r.URL.Query().Get("language"))
		assert.Equal(t, testGateSonarWay, r.URL.Query().Get("qualityProfile"))
		writeJSON(w, types.ProfileUsersResponse{
			PagedResponse: types.PagedResponse{Paging: types.Paging{Total: 1, PageIndex: 1, PageSize: 500}},
			Users:         []types.ProfileUser{{Login: "alice", Name: "Alice"}},
		})
	})
	sc := newTestServer(t, mux)

	users, err := sc.QualityProfiles.SearchUsers(context.Background(), "java", testGateSonarWay).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Login)
}

func TestQualityProfilesSearchUsersError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qualityprofiles/search_users", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Forbidden"}]}`, http.StatusForbidden)
	})
	sc := newTestServer(t, mux)

	_, err := sc.QualityProfiles.SearchUsers(context.Background(), "java", testGateSonarWay).All(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsForbidden(err))
}

// --- getBytes error path (via System.Version bad response body) ---

func TestGetBytesErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(testAPIServerVer, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
	sc := newTestServer(t, mux)

	_, err := sc.System.Version(context.Background())
	require.Error(t, err)
	assert.True(t, sqapi.IsUnauthorized(err))
}
