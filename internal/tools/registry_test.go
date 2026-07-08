package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vk7416/generic-k8s-mcp/internal/authz"
	"github.com/vk7416/generic-k8s-mcp/internal/kube"
	"github.com/vk7416/generic-k8s-mcp/internal/policy"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func newTestRegistry(t *testing.T, objects ...runtime.Object) *Registry {
	t.Helper()

	client := fake.NewSimpleClientset(objects...)
	client.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
				Reason:  "allowed in test",
			},
		}, nil
	})

	return &Registry{
		ctx: context.Background(),
		Kube: &kube.Clients{
			Clientset:        client,
			DefaultNamespace: "default",
		},
		Authz:  authz.Checker{Client: client},
		Policy: policy.Policy{ReadOnly: true},
	}
}

func TestDescribePodIncludesReplicaSetDeploymentChain(t *testing.T) {
	isController := true
	registry := newTestRegistry(t,
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "api-pod",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       "api-rs",
					Controller: &isController,
				}},
			},
			Status: corev1.PodStatus{
				Phase: "Running",
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:  "api",
					Ready: true,
				}},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "api-rs",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "api",
					Controller: &isController,
				}},
			},
		},
	)

	raw, err := json.Marshal(nameArgs{Namespace: "default", Name: "api-pod"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, err := registry.describePod(raw)
	if err != nil {
		t.Fatalf("describePod returned error: %v", err)
	}

	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content type = %T, want map[string]any", result.StructuredContent)
	}
	workload, ok := structured["workload"].(map[string]any)
	if !ok {
		t.Fatalf("workload type = %T, want map[string]any", structured["workload"])
	}
	topLevel, ok := workload["topLevel"].(map[string]any)
	if !ok {
		t.Fatalf("topLevel type = %T, want map[string]any", workload["topLevel"])
	}

	if got, want := topLevel["kind"], "Deployment"; got != want {
		t.Fatalf("topLevel kind = %v, want %v", got, want)
	}
	if got, want := topLevel["name"], "api"; got != want {
		t.Fatalf("topLevel name = %v, want %v", got, want)
	}
	if got, want := result.Content[0].Text, `Top-level workload: Deployment api.`; !strings.Contains(got, want) {
		t.Fatalf("summary %q does not contain %q", got, want)
	}
}

func TestResolveWorkloadChainWithoutOwners(t *testing.T) {
	registry := newTestRegistry(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "standalone",
		},
	}

	workload := registry.resolveWorkloadChain("default", pod)
	chain, ok := workload["chain"].([]map[string]any)
	if !ok {
		t.Fatalf("chain type = %T, want []map[string]any", workload["chain"])
	}
	if len(chain) != 0 {
		t.Fatalf("chain length = %d, want 0", len(chain))
	}
	if _, exists := workload["topLevel"]; exists {
		t.Fatalf("topLevel should be absent for ownerless pod")
	}
}
