package runner

type Config struct {
	Terraform `json:"terraform,omitempty"`
}

type Terraform struct {
	Backend map[string]*Backend `json:"backend,omitempty"`
}

type Backend struct {
	Namespace       string `json:"namespace,omitempty"`
	SecretSuffix    string `json:"secret_suffix,omitempty"`
	InClusterConfig string `json:"in_cluster_config,omitempty"`
}
