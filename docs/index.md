---
title: Generic Kubernetes MCP Server
description: Read-only Kubernetes MCP server with kubeconfig or in-cluster access.
---

# Generic Kubernetes MCP Server

A read-only, context-aware Model Context Protocol (MCP) server for Kubernetes.

The goal is simple: let an AI assistant inspect Kubernetes clusters using the same access model as `kubectl` and K9s.

## Status

This project is currently best used as a local stdio MCP server. It already supports:

- local kubeconfig and context access
- in-cluster ServiceAccount access
- read-only Kubernetes troubleshooting tools
- `SelfSubjectAccessReview` checks before Kubernetes API calls
- pod diagnostics that resolve workload ownership such as `Pod -> ReplicaSet -> Deployment`

What it does not provide yet is a production-ready remote gateway. That requires HTTP transport, authentication, tenancy boundaries, and operational hardening.

Read the gateway plan here: [Gateway Hosting Plan](gateway-plan.html)

## Architecture

### Local mode

```text
AI client
  -> MCP tool call
  -> generic-k8s-mcp (stdio)
  -> kubeconfig / selected context
  -> Kubernetes API
```

### Future remote gateway mode

```text
Any MCP client
  -> HTTPS MCP gateway
  -> authn/authz/policy/audit
  -> Kubernetes API
```

## What this gives you

You can ask questions like:

```text
Show unhealthy pods in namespace payments.
Why is deployment checkout-api not ready?
List nodes with pressure conditions.
Show warning events in kube-system.
Get the last 100 logs from pod api-123 in prod.
Can my current context list pods across all namespaces?
```

The server does not create admin access. It uses either your existing kubeconfig context or an in-cluster ServiceAccount, and then relies on Kubernetes RBAC.

## Design principles

1. Use existing Kubernetes auth.
2. Treat Kubernetes RBAC as the source of truth.
3. Stay read-only by default.
4. Work across generic Kubernetes environments first.
5. Add cloud-specific integrations later without changing the core.

## Current tools

| Tool | Purpose |
|---|---|
| `cluster_info` | Show current access mode, context, namespace, and Kubernetes server version. |
| `can_i` | Check whether the current identity can perform a Kubernetes action. |
| `list_namespaces` | List visible namespaces. |
| `list_nodes` | List nodes, readiness, taints, capacity, and allocatable resources. |
| `describe_node` | Inspect a node's labels, taints, conditions, and resource info. |
| `list_pods` | List pods by namespace, label selector, and field selector. |
| `describe_pod` | Inspect pod phase, readiness, conditions, containers, warning events, and owning workload. |
| `get_pod_logs` | Read pod logs with optional container, tail, and since options. |
| `list_events` | List events in a namespace. |
| `list_deployments` | List deployment readiness and rollout status. |
| `describe_deployment` | Inspect one deployment. |
| `get_resource_usage` | Read pod or node usage from `metrics.k8s.io` when available. |
| `find_unhealthy_workloads` | Summarize unhealthy pods, deployments, and warning events. |
| `explain_resource` | Dynamically read any Kubernetes resource by apiVersion, kind, and name. |

## Quick start

Clone, build, and run locally:

```bash
git clone https://github.com/vk7416/generic-k8s-mcp.git
cd generic-k8s-mcp
go mod tidy
make build
./bin/k8s-mcp-server --mode=local --namespace=default
```

Then connect your MCP client to the binary over stdio.

For the full setup flow, see [Quickstart](quickstart.html).

## Documentation

- [Quickstart](quickstart.html)
- [Architecture](architecture.html)
- [Tools](tools.html)
- [Security](security.html)
- [Roadmap](roadmap.html)
- [Gateway Hosting Plan](gateway-plan.html)
