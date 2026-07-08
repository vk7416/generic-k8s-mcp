---
title: Security
description: Security defaults and control model for generic-k8s-mcp.
---

# Security model

The server is designed to be safe by default.

## Defaults

- Read-only mode is enabled by default.
- Secret reads are disabled by default.
- Pod command tools are disabled by default.
- No write tools are implemented in v1.
- Every Kubernetes API read is checked with `SelfSubjectAccessReview` before execution.

## Two-layer control

```text
MCP policy layer
  +
Kubernetes RBAC layer
```

The MCP policy layer blocks unsafe tool categories before they reach the API server. Kubernetes RBAC remains the final source of truth.

## Recommended local usage

Use your normal kubeconfig context. The MCP server should have the same permissions as your `kubectl` context.

Validate access with:

```bash
kubectl auth can-i list pods -A
kubectl auth can-i get pods/log -n default
kubectl auth can-i get secrets -n default
```

## Recommended in-cluster usage

Use a dedicated ServiceAccount and bind only read permissions.

Avoid granting:

- `secrets`
- `pods/exec`
- `pods/portforward`
- write verbs such as `create`, `update`, `patch`, or `delete`

## Production hardening checklist

- Run as non-root.
- Use a dedicated namespace.
- Use read-only RBAC.
- Add network policy.
- Add audit logging for every tool call.
- Put any remote transport behind mTLS, OAuth, IAP, or an internal zero-trust proxy.
- Do not expose the server directly to the public internet.
