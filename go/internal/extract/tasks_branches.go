package extract

func branchTasks() []TaskDef {
	return []TaskDef{
		{Name: "getBranches", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: perProjectArray("getBranches", "api/project_branches/list", "branches", "project", "projectKey")},
		{Name: "getProjectPullRequests", Editions: []Edition{EditionDeveloper, EditionEnterprise, EditionDatacenter},
			Dependencies: []string{"getProjects"},
			Run:          perProjectArray("getProjectPullRequests", "api/project_pull_requests/list", "pullRequests", "project", "projectKey")},
	}
}
