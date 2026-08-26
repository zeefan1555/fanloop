package traceconfig

type RegistryProfile string

const (
	RegistryProduction RegistryProfile = "production"
	RegistryTest       RegistryProfile = "test"
)

type Registry struct {
	Profile   RegistryProfile
	URL       string
	BaseToken string
	TableID   string
	ViewID    string
}

var registries = map[RegistryProfile]map[string]Registry{
	RegistryProduction: {
		"": {
			Profile:   RegistryProduction,
			URL:       "https://example.invalid/redacted",
			BaseToken: "EXAMPLEBASEID",
			TableID:   "tblAT0YSny5i6ffv",
			ViewID:    "vew5zFcUtJ",
		},
		"fanloop-maintainer": {
			Profile:   RegistryProduction,
			URL:       "https://example.invalid/redacted",
			BaseToken: "EXAMPLEBASEID",
			TableID:   "tblW1KFyrKtUeNF4",
			ViewID:    "vew5zFcUtJ",
		},
	},
	RegistryTest: {
		"": {
			Profile:   RegistryTest,
			URL:       "https://example.invalid/redacted",
			BaseToken: "EXAMPLEBASEID",
			TableID:   "tblGrG1epTVE9tHs",
			ViewID:    "vew5zFcUtJ",
		},
	},
}

func Resolve(profile RegistryProfile, workflowID string) (Registry, bool) {
	if profile == "" {
		profile = RegistryProduction
	}
	endpoints, ok := registries[profile]
	if !ok {
		return Registry{}, false
	}
	if registry, ok := endpoints[workflowID]; ok {
		return registry, true
	}
	registry, ok := endpoints[""]
	return registry, ok
}

func Valid(profile RegistryProfile) bool {
	_, ok := registries[profile]
	return ok
}

func IsMaintainerProduction(profile RegistryProfile, workflowID string) bool {
	return profile == RegistryProduction && workflowID == "fanloop-maintainer"
}
