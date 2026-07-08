---
title: Architecture
description: Local and in-cluster architecture for generic-k8s-mcp.
---

# Architecture

`generic-k8s-mcp` follows the same core access idea as `kubectl` and K9s: it uses the current Kubernetes identity and lets Kubernetes RBAC decide what is allowed.

## Local mode

```text
AI client
  -> stdio MCP server process
  -> kubeconfig/current-context
  -> Kubernetes API server
```

Local mode is best for individual engineers. The server uses the selected kubeconfig context and has the same access as `kubectl` for that context.

## In-cluster mode

```text
AI client
  -> MCP server running in Kubernetes
  -> ServiceAccount token
  -> Kubernetes API server
```

In-cluster mode is best for shared/team deployments. The server uses `rest.InClusterConfig()` and the Pod ServiceAccount identity.

## Request flow

```text
User asks a natural-language Kubernetes question
  -> AI client chooses an MCP tool
  -> MCP server validates tool input
  -> MCP read-only policy checks the request
  -> Kubernetes SelfSubjectAccessReview checks RBAC
  -> Kubernetes API call runs if allowed
  -> MCP returns structured JSON plus a text summary
```

## Tool design

Tools are intentionally small and explicit. For example, `describe_pod` does not mutate anything. It reads the Pod, reads related warning events when allowed, resolves owner workload context when RBAC permits it, and returns structured evidence.

## Why not use cluster-admin?

The server should not bypass normal Kubernetes access. Using cluster-admin would make the MCP server a privileged automation endpoint. Instead, use normal user kubeconfig permissions locally or a tightly scoped ServiceAccount in-cluster.
