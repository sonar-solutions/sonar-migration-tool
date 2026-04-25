package structure

// MapOrganizationStructure converts bindings to organizations.
func MapOrganizationStructure(bindings []Binding) []Organization {
	var orgs []Organization
	for _, b := range bindings {
		if b.ProjectCount < 1 {
			continue
		}
		orgs = append(orgs, Organization{
			SonarQubeOrgKey:  b.Key,
			SonarCloudOrgKey: "",
			ServerURL:        b.ServerURL,
			ALM:              b.ALM,
			URL:              b.URL,
			IsCloud:          b.IsCloud,
			ProjectCount:     b.ProjectCount,
		})
	}
	return orgs
}
