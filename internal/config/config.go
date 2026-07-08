package config

import "flag"

// Config contains runtime options for the MCP server.
type Config struct {
	Mode            string
	Kubeconfig      string
	Context         string
	Namespace       string
	ReadOnly        bool
	AllowSecretRead bool
	AllowPodCommand bool
	Transport       string
	Listen          string
}

// FromFlags parses CLI flags.
func FromFlags() Config {
	var c Config
	flag.StringVar(&c.Mode, "mode", "local", "Access mode: local or incluster")
	flag.StringVar(&c.Kubeconfig, "kubeconfig", "", "Path to kubeconfig. Empty uses default kubeconfig loading rules")
	flag.StringVar(&c.Context, "context", "", "Kubeconfig context to use. Empty uses current-context")
	flag.StringVar(&c.Namespace, "namespace", "default", "Default namespace for namespaced tools")
	flag.BoolVar(&c.ReadOnly, "readonly", true, "Block non-read Kubernetes operations at the MCP layer")
	flag.BoolVar(&c.AllowSecretRead, "allow-secret-read", false, "Allow Secret reads. Disabled by default")
	flag.BoolVar(&c.AllowPodCommand, "allow-pod-command", false, "Allow pod command tools. Disabled by default")
	flag.StringVar(&c.Transport, "transport", "stdio", "MCP transport. Currently only stdio is implemented")
	flag.StringVar(&c.Listen, "listen", ":8080", "HTTP listen address for future transports")
	flag.Parse()
	return c
}
