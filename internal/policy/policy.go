package policy

import "fmt"

// Policy enforces MCP-level guardrails before Kubernetes RBAC is checked.
type Policy struct {
	ReadOnly        bool
	AllowSecretRead bool
	AllowPodCommand bool
}

var readVerbs = map[string]bool{
	"get":   true,
	"list":  true,
	"watch": true,
}

// Check blocks unsafe operations before they reach the Kubernetes API.
func (p Policy) Check(verb, resource string) error {
	if p.ReadOnly && !readVerbs[verb] {
		return fmt.Errorf("blocked by MCP readonly policy: verb %q is not allowed", verb)
	}
	if resource == "secrets" && !p.AllowSecretRead {
		return fmt.Errorf("blocked by MCP policy: Secret reads are disabled")
	}
	if resource == "pods/exec" && !p.AllowPodCommand {
		return fmt.Errorf("blocked by MCP policy: pod command tools are disabled")
	}
	return nil
}
