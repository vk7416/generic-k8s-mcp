package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vk7416/generic-k8s-mcp/internal/authz"
	appconfig "github.com/vk7416/generic-k8s-mcp/internal/config"
	"github.com/vk7416/generic-k8s-mcp/internal/kube"
	"github.com/vk7416/generic-k8s-mcp/internal/mcp"
	"github.com/vk7416/generic-k8s-mcp/internal/policy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Registry exposes Kubernetes tools over MCP.
type Registry struct {
	ctx    context.Context
	Kube   *kube.Clients
	Authz  authz.Checker
	Policy policy.Policy
}

// NewRegistry creates a Kubernetes MCP tool registry.
func NewRegistry(ctx context.Context, cfg appconfig.Config, clients *kube.Clients) *Registry {
	return &Registry{
		ctx:   ctx,
		Kube:  clients,
		Authz: authz.Checker{Client: clients.Clientset},
		Policy: policy.Policy{
			ReadOnly:        cfg.ReadOnly,
			AllowSecretRead: cfg.AllowSecretRead,
			AllowPodCommand: cfg.AllowPodCommand,
		},
	}
}

// ListTools returns tool definitions.
func (r *Registry) ListTools() []mcp.Tool {
	return []mcp.Tool{
		tool("cluster_info", "Show Kubernetes access mode, context, namespace, and server version.", nil),
		tool("can_i", "Check whether the current Kubernetes identity can perform an action.", map[string]any{"verb": str("Verb such as get, list, watch"), "resource": str("Resource such as pods or deployments"), "group": str("API group, for example apps"), "namespace": str("Namespace, optional"), "name": str("Resource name, optional")}),
		tool("list_namespaces", "List namespaces visible to the current context.", nil),
		tool("list_nodes", "List nodes and high-level readiness/resource details.", nil),
		tool("describe_node", "Describe one Kubernetes node.", map[string]any{"name": str("Node name")}, "name"),
		tool("list_pods", "List pods by namespace, label selector, or field selector.", map[string]any{"namespace": str("Namespace, or all"), "labelSelector": str("Optional label selector"), "fieldSelector": str("Optional field selector")}),
		tool("describe_pod", "Describe one pod with container and warning event details.", map[string]any{"namespace": str("Namespace"), "name": str("Pod name")}, "name"),
		tool("get_pod_logs", "Read pod logs with optional container, tail, and sinceSeconds.", map[string]any{"namespace": str("Namespace"), "name": str("Pod name"), "container": str("Container name, optional"), "tailLines": num("Tail lines, optional"), "sinceSeconds": num("Only logs newer than this many seconds, optional")}, "name"),
		tool("list_events", "List Kubernetes events in a namespace.", map[string]any{"namespace": str("Namespace, or all"), "fieldSelector": str("Optional field selector"), "limit": num("Maximum events to return")}),
		tool("list_deployments", "List deployments and readiness in a namespace.", map[string]any{"namespace": str("Namespace, or all"), "labelSelector": str("Optional label selector")}),
		tool("describe_deployment", "Describe one deployment.", map[string]any{"namespace": str("Namespace"), "name": str("Deployment name")}, "name"),
		tool("get_resource_usage", "Read pod or node metrics from metrics.k8s.io when installed.", map[string]any{"scope": str("pods or nodes"), "namespace": str("Namespace for pod metrics, or all")}),
		tool("find_unhealthy_workloads", "Find pods/deployments with unhealthy status and recent warnings.", map[string]any{"namespace": str("Namespace, or all")}),
		tool("explain_resource", "Read any resource dynamically by apiVersion, kind, namespace, and name.", map[string]any{"apiVersion": str("API version, for example apps/v1"), "kind": str("Kind, for example Deployment"), "namespace": str("Namespace for namespaced resources"), "name": str("Resource name")}, "apiVersion", "kind", "name"),
	}
}

// CallTool dispatches a tool call.
func (r *Registry) CallTool(name string, arguments json.RawMessage) (mcp.ToolResult, error) {
	switch name {
	case "cluster_info":
		return r.clusterInfo()
	case "can_i":
		return r.canI(arguments)
	case "list_namespaces":
		return r.listNamespaces()
	case "list_nodes":
		return r.listNodes()
	case "describe_node":
		return r.describeNode(arguments)
	case "list_pods":
		return r.listPods(arguments)
	case "describe_pod":
		return r.describePod(arguments)
	case "get_pod_logs":
		return r.getPodLogs(arguments)
	case "list_events":
		return r.listEvents(arguments)
	case "list_deployments":
		return r.listDeployments(arguments)
	case "describe_deployment":
		return r.describeDeployment(arguments)
	case "get_resource_usage":
		return r.getResourceUsage(arguments)
	case "find_unhealthy_workloads":
		return r.findUnhealthy(arguments)
	case "explain_resource":
		return r.explainResource(arguments)
	default:
		return mcp.ToolResult{}, fmt.Errorf("unknown tool %q", name)
	}
}

func tool(name, desc string, props map[string]any, required ...string) mcp.Tool {
	if props == nil {
		props = map[string]any{}
	}
	return mcp.Tool{Name: name, Description: desc, InputSchema: map[string]any{"type": "object", "properties": props, "required": required}}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any { return map[string]any{"type": "number", "description": desc} }

func decode[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 || string(raw) == "null" {
		return out, nil
	}
	return out, json.Unmarshal(raw, &out)
}

func ok(summary string, data any) mcp.ToolResult {
	return mcp.ToolResult{Content: []mcp.Content{{Type: "text", Text: summary}}, StructuredContent: data}
}

func (r *Registry) authorize(namespace, group, resource, verb, name string) error {
	if err := r.Policy.Check(verb, resource); err != nil {
		return err
	}
	allowed, reason, err := r.Authz.CanI(r.ctx, namespace, group, resource, verb, name)
	if err != nil {
		return err
	}
	if !allowed {
		if reason == "" {
			reason = "Kubernetes RBAC denied the request"
		}
		return fmt.Errorf("access denied: current context cannot %s resource %s/%s in namespace %q: %s", verb, group, resource, namespace, reason)
	}
	return nil
}

func authNamespace(ns string) string {
	if ns == metav1.NamespaceAll {
		return ""
	}
	return ns
}

func age(t metav1.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return time.Since(t.Time).Round(time.Second).String()
}

func podReady(p corev1.Pod) string {
	ready := 0
	total := len(p.Status.ContainerStatuses)
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}
	return fmt.Sprintf("%d/%d", ready, total)
}

func podRestarts(p corev1.Pod) int32 {
	var n int32
	for _, cs := range p.Status.ContainerStatuses {
		n += cs.RestartCount
	}
	return n
}

func podReason(p corev1.Pod) string {
	if p.Status.Reason != "" {
		return p.Status.Reason
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			return cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil {
			return cs.State.Terminated.Reason
		}
	}
	return ""
}

func ownerRefMap(ref metav1.OwnerReference) map[string]any {
	return map[string]any{
		"apiVersion": ref.APIVersion,
		"kind":       ref.Kind,
		"name":       ref.Name,
		"controller": ref.Controller != nil && *ref.Controller,
	}
}

func controllerOwner(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return &refs[i]
		}
	}
	if len(refs) == 0 {
		return nil
	}
	return &refs[0]
}

func workloadSummary(chain []map[string]any) string {
	if len(chain) == 0 {
		return ""
	}
	last := chain[len(chain)-1]
	kind, _ := last["kind"].(string)
	name, _ := last["name"].(string)
	if kind == "" || name == "" {
		return ""
	}
	return fmt.Sprintf("%s %s", kind, name)
}

func (r *Registry) resolveWorkloadChain(namespace string, pod *corev1.Pod) map[string]any {
	chain := []map[string]any{}
	for _, ref := range pod.OwnerReferences {
		chain = append(chain, ownerRefMap(ref))
	}
	current := controllerOwner(pod.OwnerReferences)
	if current == nil {
		return map[string]any{"chain": chain}
	}

	visited := map[string]bool{}
	lookupErrors := []string{}
	for depth := 0; current != nil && depth < 4; depth++ {
		key := current.APIVersion + "/" + current.Kind + "/" + current.Name
		if visited[key] {
			lookupErrors = append(lookupErrors, "owner reference cycle detected")
			break
		}
		visited[key] = true

		switch {
		case current.APIVersion == "apps/v1" && current.Kind == "ReplicaSet":
			if err := r.authorize(namespace, "apps", "replicasets", "get", current.Name); err != nil {
				lookupErrors = append(lookupErrors, err.Error())
				current = nil
				continue
			}
			rs, err := r.Kube.Clientset.AppsV1().ReplicaSets(namespace).Get(r.ctx, current.Name, metav1.GetOptions{})
			if err != nil {
				lookupErrors = append(lookupErrors, err.Error())
				current = nil
				continue
			}
			current = controllerOwner(rs.OwnerReferences)
		case current.APIVersion == "batch/v1" && current.Kind == "Job":
			if err := r.authorize(namespace, "batch", "jobs", "get", current.Name); err != nil {
				lookupErrors = append(lookupErrors, err.Error())
				current = nil
				continue
			}
			job, err := r.Kube.Clientset.BatchV1().Jobs(namespace).Get(r.ctx, current.Name, metav1.GetOptions{})
			if err != nil {
				lookupErrors = append(lookupErrors, err.Error())
				current = nil
				continue
			}
			current = controllerOwner(job.OwnerReferences)
		default:
			current = nil
		}

		if current != nil {
			chain = append(chain, ownerRefMap(*current))
		}
	}

	data := map[string]any{"chain": chain}
	if len(chain) > 0 {
		data["topLevel"] = chain[len(chain)-1]
	}
	if len(lookupErrors) > 0 {
		data["lookupErrors"] = lookupErrors
	}
	return data
}

func isPodHealthy(p corev1.Pod) bool {
	if p.Status.Phase == corev1.PodFailed || p.Status.Phase == corev1.PodUnknown {
		return false
	}
	if p.Status.Phase == corev1.PodSucceeded {
		return true
	}
	for _, cs := range p.Status.ContainerStatuses {
		if !cs.Ready || cs.RestartCount > 0 || cs.State.Waiting != nil || cs.State.Terminated != nil {
			return false
		}
	}
	return true
}

func (r *Registry) clusterInfo() (mcp.ToolResult, error) {
	data := map[string]any{"mode": r.Kube.Mode, "context": r.Kube.CurrentContext, "namespace": r.Kube.DefaultNamespace, "serverVersion": r.Kube.ServerVersion}
	return ok(fmt.Sprintf("Connected to %s using mode %s. Default namespace: %s. Server version: %s.", r.Kube.CurrentContext, r.Kube.Mode, r.Kube.DefaultNamespace, r.Kube.ServerVersion), data), nil
}

type canIArgs struct{ Verb, Group, Resource, Namespace, Name string }

func (r *Registry) canI(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[canIArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	allowed, reason, err := r.Authz.CanI(r.ctx, args.Namespace, args.Group, args.Resource, args.Verb, args.Name)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	data := map[string]any{"allowed": allowed, "reason": reason, "namespace": args.Namespace, "group": args.Group, "resource": args.Resource, "verb": args.Verb, "name": args.Name}
	return ok(fmt.Sprintf("allowed=%t for %s %s/%s in namespace %q", allowed, args.Verb, args.Group, args.Resource, args.Namespace), data), nil
}

func (r *Registry) listNamespaces() (mcp.ToolResult, error) {
	if err := r.authorize("", "", "namespaces", "list", ""); err != nil {
		return mcp.ToolResult{}, err
	}
	list, err := r.Kube.Clientset.CoreV1().Namespaces().List(r.ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	items := make([]map[string]any, 0, len(list.Items))
	for _, ns := range list.Items {
		items = append(items, map[string]any{"name": ns.Name, "status": string(ns.Status.Phase), "age": age(ns.CreationTimestamp), "labels": ns.Labels})
	}
	return ok(fmt.Sprintf("Found %d namespaces.", len(items)), map[string]any{"namespaces": items}), nil
}

func (r *Registry) listNodes() (mcp.ToolResult, error) {
	if err := r.authorize("", "", "nodes", "list", ""); err != nil {
		return mcp.ToolResult{}, err
	}
	list, err := r.Kube.Clientset.CoreV1().Nodes().List(r.ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	items := make([]map[string]any, 0, len(list.Items))
	for _, n := range list.Items {
		ready := "Unknown"
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady {
				ready = string(c.Status)
			}
		}
		items = append(items, map[string]any{"name": n.Name, "ready": ready, "age": age(n.CreationTimestamp), "taints": n.Spec.Taints, "capacity": n.Status.Capacity, "allocatable": n.Status.Allocatable, "labels": n.Labels})
	}
	return ok(fmt.Sprintf("Found %d nodes.", len(items)), map[string]any{"nodes": items}), nil
}

type nameArgs struct{ Namespace, Name string }

func (r *Registry) describeNode(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[nameArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	if args.Name == "" {
		return mcp.ToolResult{}, fmt.Errorf("name is required")
	}
	if err := r.authorize("", "", "nodes", "get", args.Name); err != nil {
		return mcp.ToolResult{}, err
	}
	n, err := r.Kube.Clientset.CoreV1().Nodes().Get(r.ctx, args.Name, metav1.GetOptions{})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	data := map[string]any{"name": n.Name, "age": age(n.CreationTimestamp), "labels": n.Labels, "taints": n.Spec.Taints, "conditions": n.Status.Conditions, "capacity": n.Status.Capacity, "allocatable": n.Status.Allocatable, "addresses": n.Status.Addresses}
	return ok(fmt.Sprintf("Node %s has %d conditions and %d taints.", n.Name, len(n.Status.Conditions), len(n.Spec.Taints)), data), nil
}

type listPodsArgs struct{ Namespace, LabelSelector, FieldSelector string }

func (r *Registry) listPods(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[listPodsArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if err := r.authorize(authNamespace(ns), "", "pods", "list", ""); err != nil {
		return mcp.ToolResult{}, err
	}
	list, err := r.Kube.Clientset.CoreV1().Pods(ns).List(r.ctx, metav1.ListOptions{LabelSelector: args.LabelSelector, FieldSelector: args.FieldSelector})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	items := make([]map[string]any, 0, len(list.Items))
	for _, p := range list.Items {
		items = append(items, map[string]any{"namespace": p.Namespace, "name": p.Name, "phase": string(p.Status.Phase), "ready": podReady(p), "restarts": podRestarts(p), "reason": podReason(p), "node": p.Spec.NodeName, "age": age(p.CreationTimestamp)})
	}
	return ok(fmt.Sprintf("Found %d pods.", len(items)), map[string]any{"pods": items}), nil
}

func (r *Registry) describePod(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[nameArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	if args.Name == "" {
		return mcp.ToolResult{}, fmt.Errorf("name is required")
	}
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if err := r.authorize(ns, "", "pods", "get", args.Name); err != nil {
		return mcp.ToolResult{}, err
	}
	p, err := r.Kube.Clientset.CoreV1().Pods(ns).Get(r.ctx, args.Name, metav1.GetOptions{})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	warnings := []map[string]any{}
	if r.authorize(ns, "", "events", "list", "") == nil {
		events, _ := r.Kube.Clientset.CoreV1().Events(ns).List(r.ctx, metav1.ListOptions{FieldSelector: "involvedObject.name=" + args.Name})
		for _, e := range events.Items {
			if e.Type == corev1.EventTypeWarning {
				warnings = append(warnings, map[string]any{"reason": e.Reason, "message": e.Message, "count": e.Count, "lastTimestamp": e.LastTimestamp.String()})
			}
		}
	}
	workload := r.resolveWorkloadChain(ns, p)
	data := map[string]any{"namespace": p.Namespace, "name": p.Name, "phase": string(p.Status.Phase), "ready": podReady(*p), "restarts": podRestarts(*p), "reason": podReason(*p), "node": p.Spec.NodeName, "podIP": p.Status.PodIP, "conditions": p.Status.Conditions, "containers": p.Status.ContainerStatuses, "warningEvents": warnings, "workload": workload}
	summary := fmt.Sprintf("Pod %s/%s is %s, ready %s, restarts %d, reason %q.", p.Namespace, p.Name, p.Status.Phase, podReady(*p), podRestarts(*p), podReason(*p))
	if chain, ok := workload["chain"].([]map[string]any); ok {
		if owner := workloadSummary(chain); owner != "" {
			summary += " Top-level workload: " + owner + "."
		}
	}
	return ok(summary, data), nil
}

type logsArgs struct {
	Namespace, Name, Container string
	TailLines, SinceSeconds    int64
}

func (r *Registry) getPodLogs(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[logsArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	if args.Name == "" {
		return mcp.ToolResult{}, fmt.Errorf("name is required")
	}
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if err := r.authorize(ns, "", "pods/log", "get", args.Name); err != nil {
		return mcp.ToolResult{}, err
	}
	if args.TailLines == 0 {
		args.TailLines = 200
	}
	opts := &corev1.PodLogOptions{Container: args.Container, TailLines: &args.TailLines}
	if args.SinceSeconds > 0 {
		opts.SinceSeconds = &args.SinceSeconds
	}
	rawLogs, err := r.Kube.Clientset.CoreV1().Pods(ns).GetLogs(args.Name, opts).DoRaw(r.ctx)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	text := string(rawLogs)
	return ok(fmt.Sprintf("Logs for pod %s/%s returned %d bytes.", ns, args.Name, len(text)), map[string]any{"namespace": ns, "pod": args.Name, "container": args.Container, "logs": text}), nil
}

type eventsArgs struct {
	Namespace, FieldSelector string
	Limit                    int
}

func (r *Registry) listEvents(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[eventsArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if err := r.authorize(authNamespace(ns), "", "events", "list", ""); err != nil {
		return mcp.ToolResult{}, err
	}
	list, err := r.Kube.Clientset.CoreV1().Events(ns).List(r.ctx, metav1.ListOptions{FieldSelector: args.FieldSelector})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].CreationTimestamp.After(list.Items[j].CreationTimestamp.Time)
	})
	limit := args.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	items := []map[string]any{}
	for i, e := range list.Items {
		if i >= limit {
			break
		}
		items = append(items, map[string]any{"namespace": e.Namespace, "type": e.Type, "reason": e.Reason, "message": e.Message, "count": e.Count, "object": e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name, "age": age(e.CreationTimestamp)})
	}
	return ok(fmt.Sprintf("Found %d events, returning %d.", len(list.Items), len(items)), map[string]any{"events": items}), nil
}

type deployArgs struct {
	Namespace, LabelSelector string
	Name                     string
}

func (r *Registry) listDeployments(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[deployArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if err := r.authorize(authNamespace(ns), "apps", "deployments", "list", ""); err != nil {
		return mcp.ToolResult{}, err
	}
	list, err := r.Kube.Clientset.AppsV1().Deployments(ns).List(r.ctx, metav1.ListOptions{LabelSelector: args.LabelSelector})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	items := make([]map[string]any, 0, len(list.Items))
	for _, d := range list.Items {
		replicas := int32(0)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		items = append(items, map[string]any{"namespace": d.Namespace, "name": d.Name, "desired": replicas, "ready": d.Status.ReadyReplicas, "available": d.Status.AvailableReplicas, "updated": d.Status.UpdatedReplicas, "age": age(d.CreationTimestamp)})
	}
	return ok(fmt.Sprintf("Found %d deployments.", len(items)), map[string]any{"deployments": items}), nil
}

func (r *Registry) describeDeployment(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[deployArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	if args.Name == "" {
		return mcp.ToolResult{}, fmt.Errorf("name is required")
	}
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if err := r.authorize(ns, "apps", "deployments", "get", args.Name); err != nil {
		return mcp.ToolResult{}, err
	}
	d, err := r.Kube.Clientset.AppsV1().Deployments(ns).Get(r.ctx, args.Name, metav1.GetOptions{})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	replicas := int32(0)
	if d.Spec.Replicas != nil {
		replicas = *d.Spec.Replicas
	}
	data := map[string]any{"namespace": d.Namespace, "name": d.Name, "desired": replicas, "ready": d.Status.ReadyReplicas, "available": d.Status.AvailableReplicas, "updated": d.Status.UpdatedReplicas, "conditions": d.Status.Conditions, "selector": d.Spec.Selector, "strategy": d.Spec.Strategy}
	return ok(fmt.Sprintf("Deployment %s/%s desired=%d ready=%d available=%d updated=%d.", d.Namespace, d.Name, replicas, d.Status.ReadyReplicas, d.Status.AvailableReplicas, d.Status.UpdatedReplicas), data), nil
}

type usageArgs struct{ Scope, Namespace string }

func (r *Registry) getResourceUsage(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[usageArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	if args.Scope == "" {
		args.Scope = "pods"
	}
	switch args.Scope {
	case "pods":
		ns := r.Kube.NamespaceOrDefault(args.Namespace)
		if err := r.authorize(authNamespace(ns), "metrics.k8s.io", "pods", "list", ""); err != nil {
			return mcp.ToolResult{}, err
		}
		gvr := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
		list, err := r.Kube.Dynamic.Resource(gvr).Namespace(ns).List(r.ctx, metav1.ListOptions{})
		if err != nil {
			return mcp.ToolResult{}, err
		}
		return ok(fmt.Sprintf("Found %d pod metrics.", len(list.Items)), map[string]any{"scope": "pods", "items": list.Items}), nil
	case "nodes":
		if err := r.authorize("", "metrics.k8s.io", "nodes", "list", ""); err != nil {
			return mcp.ToolResult{}, err
		}
		gvr := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
		list, err := r.Kube.Dynamic.Resource(gvr).List(r.ctx, metav1.ListOptions{})
		if err != nil {
			return mcp.ToolResult{}, err
		}
		return ok(fmt.Sprintf("Found %d node metrics.", len(list.Items)), map[string]any{"scope": "nodes", "items": list.Items}), nil
	default:
		return mcp.ToolResult{}, fmt.Errorf("scope must be pods or nodes")
	}
}

func (r *Registry) findUnhealthy(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[listPodsArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if err := r.authorize(authNamespace(ns), "", "pods", "list", ""); err != nil {
		return mcp.ToolResult{}, err
	}
	pods, err := r.Kube.Clientset.CoreV1().Pods(ns).List(r.ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	badPods := []map[string]any{}
	for _, p := range pods.Items {
		if !isPodHealthy(p) {
			badPods = append(badPods, map[string]any{"namespace": p.Namespace, "name": p.Name, "phase": string(p.Status.Phase), "ready": podReady(p), "restarts": podRestarts(p), "reason": podReason(p), "node": p.Spec.NodeName})
		}
	}
	badDeployments := []map[string]any{}
	if r.authorize(authNamespace(ns), "apps", "deployments", "list", "") == nil {
		deploys, _ := r.Kube.Clientset.AppsV1().Deployments(ns).List(r.ctx, metav1.ListOptions{})
		for _, d := range deploys.Items {
			replicas := int32(0)
			if d.Spec.Replicas != nil {
				replicas = *d.Spec.Replicas
			}
			if d.Status.ReadyReplicas < replicas || d.Status.AvailableReplicas < replicas {
				badDeployments = append(badDeployments, map[string]any{"namespace": d.Namespace, "name": d.Name, "desired": replicas, "ready": d.Status.ReadyReplicas, "available": d.Status.AvailableReplicas, "updated": d.Status.UpdatedReplicas})
			}
		}
	}
	warnings := []map[string]any{}
	if r.authorize(authNamespace(ns), "", "events", "list", "") == nil {
		events, _ := r.Kube.Clientset.CoreV1().Events(ns).List(r.ctx, metav1.ListOptions{})
		for _, e := range events.Items {
			if e.Type == corev1.EventTypeWarning {
				warnings = append(warnings, map[string]any{"namespace": e.Namespace, "reason": e.Reason, "message": e.Message, "object": e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name, "count": e.Count})
			}
		}
	}
	summary := fmt.Sprintf("Found %d unhealthy pods, %d unhealthy deployments, and %d warning events.", len(badPods), len(badDeployments), len(warnings))
	return ok(summary, map[string]any{"unhealthyPods": badPods, "unhealthyDeployments": badDeployments, "warningEvents": warnings}), nil
}

type explainArgs struct{ APIVersion, Kind, Namespace, Name string }

func (r *Registry) explainResource(raw json.RawMessage) (mcp.ToolResult, error) {
	args, err := decode[explainArgs](raw)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	if args.APIVersion == "" || args.Kind == "" || args.Name == "" {
		return mcp.ToolResult{}, fmt.Errorf("apiVersion, kind, and name are required")
	}
	gv, err := schema.ParseGroupVersion(args.APIVersion)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	mapping, err := r.Kube.Mapper.RESTMapping(gv.WithKind(args.Kind).GroupKind(), gv.Version)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	res := mapping.Resource.Resource
	group := mapping.Resource.Group
	ns := r.Kube.NamespaceOrDefault(args.Namespace)
	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		ns = ""
	}
	if err := r.authorize(ns, group, res, "get", args.Name); err != nil {
		return mcp.ToolResult{}, err
	}
	var obj *unstructured.Unstructured
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		obj, err = r.Kube.Dynamic.Resource(mapping.Resource).Namespace(ns).Get(r.ctx, args.Name, metav1.GetOptions{})
	} else {
		obj, err = r.Kube.Dynamic.Resource(mapping.Resource).Get(r.ctx, args.Name, metav1.GetOptions{})
	}
	if err != nil {
		return mcp.ToolResult{}, err
	}
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	data := map[string]any{"apiVersion": obj.GetAPIVersion(), "kind": obj.GetKind(), "namespace": obj.GetNamespace(), "name": obj.GetName(), "labels": obj.GetLabels(), "annotations": obj.GetAnnotations(), "status": status}
	return ok(fmt.Sprintf("Read %s %s/%s.", obj.GetKind(), obj.GetNamespace(), obj.GetName()), data), nil
}

func compactReasons(items []map[string]any, key string) string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		if v, ok := item[key].(string); ok && v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return strings.Join(out, ", ")
}
