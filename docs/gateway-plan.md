---
title: Gateway Hosting Plan
description: Plan for turning generic-k8s-mcp into a remote HTTP MCP gateway usable by multiple clients.
---

# Gateway Hosting Plan

This project can be turned into a shared MCP gateway, but it needs a deliberate shift from a local stdio binary to a networked service.

## Goal

Support any MCP client that can speak to a remote server over HTTPS, while preserving the core safety model:

- explicit auth
- read-only policy
- Kubernetes RBAC enforcement
- auditable access

## Recommended architecture

```text
MCP client
  -> HTTPS MCP gateway
  -> authentication and tenant resolution
  -> request policy and rate limits
  -> Kubernetes identity selection
  -> Kubernetes API
```

## Phase 1: Add remote MCP transport

Current state:

- only stdio is implemented

Required next step:

- add Streamable HTTP transport as the primary remote transport
- optionally add SSE compatibility if a target client still needs it
- keep stdio support for local developer use

Deliverables:

- `/mcp` HTTP endpoint
- health endpoint such as `/healthz`
- startup and per-request timeouts
- maximum request and response size limits

## Phase 2: Add authentication

A public gateway cannot trust anonymous callers.

Recommended first implementation:

- bearer token authentication for server-to-server clients
- optional OIDC later for user-facing environments

Required capabilities:

- validate inbound identity
- map caller to tenant or workspace
- reject unauthenticated traffic before any Kubernetes call

## Phase 3: Define the Kubernetes identity model

You need to choose how the gateway reaches clusters.

### Option A: One gateway per cluster

Best first production model.

- deploy the gateway inside a cluster
- give it a tightly scoped ServiceAccount
- expose it behind ingress or a private load balancer

Pros:

- simplest RBAC story
- clean failure domain
- no central kubeconfig broker

Cons:

- one deployment per cluster

### Option B: Central gateway for multiple clusters

Only do this after the single-cluster model works well.

- gateway stores or resolves multiple cluster credentials
- every request chooses a target cluster

Additional requirements:

- secure credential storage
- per-cluster authorization rules
- stronger audit and tenancy controls

## Phase 4: Keep the read-only safety contract

The remote gateway should preserve the current rules:

- no write operations by default
- no secret reads by default
- no pod exec by default

Required controls:

- enforce the existing MCP policy layer before Kubernetes calls
- continue running `SelfSubjectAccessReview` for the selected identity
- return clear denied responses when either policy or RBAC blocks access

## Phase 5: Add multi-client operational hardening

For a shared gateway, add:

- structured request logging
- audit logs with caller, tool, namespace, cluster, and outcome
- rate limiting
- request concurrency limits
- panic recovery and stable error responses
- Prometheus metrics
- tracing hooks

## Phase 6: Make responses safe for shared use

Before exposing this to many clients, define output controls:

- cap log output size
- cap event and list result size
- redact or block sensitive fields
- make long-running reads cancellable

## Phase 7: Deployment shape

Recommended first deployment:

1. Run in Kubernetes.
2. Use an in-cluster ServiceAccount.
3. Expose via an internal ingress or authenticated gateway.
4. Terminate TLS at ingress.
5. Restrict access by network policy and auth token.

Suggested runtime pieces:

- `Deployment`
- `Service`
- `Ingress` or gateway route
- `ServiceAccount`
- namespaced RBAC
- `NetworkPolicy`
- optional `HorizontalPodAutoscaler`

## Phase 8: Client compatibility

To be usable by any client, document:

- the remote MCP endpoint URL
- authentication method
- timeout expectations
- supported tool list
- example client configs

This should live in the public docs site once HTTP transport exists.

## Proposed implementation order

1. Add Streamable HTTP transport without changing tool behavior.
2. Add auth middleware and audit logging.
3. Package an in-cluster deployment for one-cluster-per-gateway hosting.
4. Add response size limits and rate limits.
5. Publish example configs for remote MCP clients.
6. Only then consider multi-cluster central gateway support.

## Recommended first release target

Aim for:

- one cluster
- one gateway deployment
- read-only only
- token auth
- internal or VPN-only exposure

That is the fastest path to a usable shared MCP gateway without overbuilding the control plane.
