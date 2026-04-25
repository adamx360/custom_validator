package validation

type InvalidField struct {
	Path    string   `json:"path"`
	Reasons []Reason `json:"reasons"`
}

type Reason struct {
	Scopes      []string `json:"scopes"`
	Code        string   `json:"code"`
	Description string   `json:"description"`
}
