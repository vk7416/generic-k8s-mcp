---
title: Roadmap
description: Planned milestones for generic-k8s-mcp.
---

# Roadmap

## v0.1 MVP

- Local kubeconfig/context mode.
- In-cluster ServiceAccount mode.
- Minimal stdio MCP JSON-RPC server.
- Read-only Kubernetes tools.
- SelfSubjectAccessReview before each Kubernetes API call.
- Kubernetes manifests for a read-only ServiceAccount.

## v0.2 Transport hardening

- Add Streamable HTTP transport.
- Add optional authentication middleware for remote deployments.
- Add request and response audit logging.
- Add per-tool timeouts and output size limits.

## v0.3 Better troubleshooting

- Owner-reference traversal: Pod -> ReplicaSet -> Deployment.
- Add service-to-endpoint diagnostics.
- Add ingress backend diagnostics.
- Add HPA and PDB analysis.
- Add node pressure and scheduling explanation helpers.

## v0.4 Provider plugins

- GKE node pool and Cloud Logging links.
- EKS node group and CloudWatch links.
- AKS node pool and Azure Monitor links.

## Never by default

These should not be enabled by default:

- write operations
- Secret reads
- pod command execution
- port-forwarding
- YAML apply
- delete/patch/update actions

If write operations are added later, they should use a separate permission profile and explicit human approval.
