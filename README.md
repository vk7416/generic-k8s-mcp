# Generic Kubernetes MCP Server

A read-only, context-aware Model Context Protocol (MCP) server for Kubernetes.

The goal is simple: let an AI assistant inspect Kubernetes clusters using the same access model as `kubectl` and K9s.

```text
Natural language
  -> AI client
  -> MCP tools
  -> current kubeconfig context or in-cluster ServiceAccount
  -> Kubernetes API server
  -> RBAC-controlled read-only results
```

## Status

This is an MVP scaffold. It implements a minimal JSON-RPC/MCP stdio server in Go and exposes read-only Kubernetes tools.

## Design principles

1. **Use existing Kubernetes auth**: local mode uses kubeconfig/context; in-cluster mode uses the Pod ServiceAccount.
2. **RBAC is the source of truth**: every Kubernetes API call is checked with `SelfSubjectAccessReview` before running.
3. **Read-only by default**: no create, update, patch, delete, exec, port-forward, scale, apply, or secret reads by default.
4. **Generic Kubernetes first**: works with GKE, EKS, AKS, kubeadm, kind, minikube, and on-prem clusters.
5. **Cloud-specific integrations later**: GKE/EKS/AKS plugins can be added later without changing the core.

## Current tools

| Tool | Purpose |
|---|---|
| `cluster_info` | Show current access mode, context, namespace, and Kubernetes server version. |
| `can_i` | Check whether the current identity can perform a Kubernetes action. |
| `list_namespaces` | List visible namespaces. |
| `list_nodes` | List nodes, readiness, taints, capacity, and allocatable resources. |
| `describe_node` | Inspect a node's labels, taints, conditions, and resource info. |
| `list_pods` | List pods by namespace, label selector, and field selector. |
| `describe_pod` | Inspect pod phase, readiness, conditions, containers, and warning events. |
| `get_pod_logs` | Read pod logs with optional container, tail, and since options. |
| `list_events` | List events in a namespace. |
| `list_deployments` | List deployment readiness and rollout status. |
| `describe_deployment` | Inspect one deployment. |
| `get_resource_usage` | Read pod or node usage from `metrics.k8s.io` when metrics-server is installed. |
| `find_unhealthy_workloads` | Summarize unhealthy pods/deployments and warning events. |
| `explain_resource` | Dynamically read any Kubernetes resource by apiVersion/kind/name. |

## Local usage

Build:

```bash
go build -o bin/k8s-mcp-server ./cmd/k8s-mcp-server
```

Run against your current kubeconfig:

```bash
./bin/k8s-mcp-server \
  --mode=local \
  --kubeconfig="$HOME/.kube/config" \
  --context="" \
  --namespace=default \
  --readonly=true \
  --allow-secret-read=false \
  --allow-pod-command=false
```

Run against a specific context:

```bash
./bin/k8s-mcp-server \
  --mode=local \
  --context=my-cluster-context \
  --namespace=kube-system
```

## MCP client config example

For a stdio MCP client:

```json
{
  "mcpServers": {
    "generic-k8s": {
      "command": "/path/to/k8s-mcp-server",
      "args": [
        "--mode=local",
        "--kubeconfig=/Users/YOU/.kube/config",
        "--context=YOUR_CONTEXT",
        "--namespace=default",
        "--readonly=true",
        "--allow-secret-read=false",
        "--allow-pod-command=false"
      ]
    }
  }
}
```

## In-cluster usage

Deploy with read-only RBAC:

```bash
kubectl apply -f deploy/namespace.yaml
kubectl apply -f deploy/rbac-readonly.yaml
kubectl apply -f deploy/deployment.yaml
```

In-cluster mode uses `rest.InClusterConfig()` and the Pod's ServiceAccount token.

## Security defaults

The server blocks dangerous operations at the MCP policy layer and then asks Kubernetes RBAC before every read. This gives two layers of control:

```text
MCP readonly policy
  +
Kubernetes RBAC
```

By default, the server does **not** expose tools for:

- reading Secrets
- exec into Pods
- port-forward
- applying YAML
- patching resources
- deleting resources
- scaling or restarting workloads

## Repository layout

```text
cmd/k8s-mcp-server/      CLI entrypoint
internal/mcp/            Minimal MCP JSON-RPC server
internal/kube/           Kubernetes client/context loading
internal/authz/          SelfSubjectAccessReview checks
internal/policy/         Read-only guardrails
internal/tools/          Kubernetes MCP tools
deploy/                  Kubernetes deployment manifests
examples/                MCP client examples
docs/                    Architecture and security notes
```

## Example questions

Once connected to an MCP-capable assistant, ask:

```text
Show unhealthy pods in namespace payments.
Why is deployment checkout-api not ready?
List nodes with pressure conditions.
Show warning events in kube-system.
Get the last 100 logs from pod api-123 in prod.
Can my current context list pods across all namespaces?
```

## MVP limitations

- The stdio transport is implemented directly with newline-delimited JSON-RPC.
- HTTP/SSE transport can be added later.
- No write operations are implemented.
- Cloud provider integrations are intentionally out of scope for v1.

## License

Apache-2.0
