package runner

type Config struct {
	Terraform `json:"terraform,omitempty"`
}

type Terraform struct {
	Backend map[string]*Backend `json:"backend,omitempty"`
}

type Backend struct {
	Namespace      string `json:"namespace,omitempty"`
	Key            string `json:"key,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"`
}
