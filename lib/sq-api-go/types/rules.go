package types

// RuleParam is a parameter definition on a rule template.
type RuleParam struct {
	Key          string `json:"key"`
	HTMLDesc     string `json:"htmlDesc"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

// Rule represents a single rule returned by /api/rules/search or /api/rules/show.
type Rule struct {
	Key         string      `json:"key"`
	Repo        string      `json:"repo"`
	Name        string      `json:"name"`
	Severity    string      `json:"severity"`
	Status      string      `json:"status"`
	Type        string      `json:"type"`
	Lang        string      `json:"lang"`
	LangName    string      `json:"langName"`
	IsTemplate  bool        `json:"isTemplate"`
	TemplateKey string      `json:"templateKey"`
	Tags        []string    `json:"tags"`
	SysTags     []string    `json:"sysTags"`
	Params      []RuleParam `json:"params"`
}

// RulesSearchResponse is the response envelope for /api/rules/search.
type RulesSearchResponse struct {
	PagedResponse
	Rules []Rule `json:"rules"`
}

// RuleShowResponse is the response envelope for /api/rules/show.
type RuleShowResponse struct {
	Rule Rule `json:"rule"`
}

// RuleRepository represents a single rule repository returned by /api/rules/repositories.
type RuleRepository struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Language string `json:"language"`
}

// RepositoriesResponse is the response envelope for /api/rules/repositories.
type RepositoriesResponse struct {
	Repositories []RuleRepository `json:"repositories"`
}
