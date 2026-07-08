---
title: Tools
description: Tool reference for generic-k8s-mcp.
---

# Tool reference

## cluster_info

Shows access mode, current context, default namespace, and Kubernetes server version.

## can_i

Checks whether the current identity can perform an action.

Example arguments:

```json
{
  "verb": "list",
  "group": "",
  "resource": "pods",
  "namespace": "default"
}
```

## list_pods

Lists pods in one namespace or across all namespaces when RBAC allows it.

Example:

```json
{
  "namespace": "default",
  "labelSelector": "app=api"
}
```

Use `namespace: "all"` for all namespaces.

## describe_pod

Reads one Pod and returns phase, readiness, restarts, conditions, containers, node placement, warning events, and owning workload details when those reads are allowed.

## get_pod_logs

Reads Pod logs.

Example:

```json
{
  "namespace": "default",
  "name": "api-123",
  "container": "api",
  "tailLines": 100,
  "sinceSeconds": 3600
}
```

## list_events

Lists events in a namespace and returns the most recent events first.

## list_nodes / describe_node

Reads node status, conditions, taints, labels, capacity, and allocatable resources.

## list_deployments / describe_deployment

Reads Deployment readiness and rollout status.

## get_resource_usage

Reads `metrics.k8s.io/v1beta1` pod or node metrics when metrics-server or a compatible metrics API is installed.

## find_unhealthy_workloads

High-level troubleshooting helper that looks for unhealthy Pods, unavailable Deployments, and Warning events.

## explain_resource

Generic dynamic-client read for CRDs and built-in resources.

Example:

```json
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "namespace": "default",
  "name": "api"
}
```
