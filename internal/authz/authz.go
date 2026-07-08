package authz

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Checker performs SelfSubjectAccessReview checks using the current Kubernetes identity.
type Checker struct {
	Client kubernetes.Interface
}

// CanI asks the Kubernetes API server whether the current identity can perform an action.
func (c Checker) CanI(ctx context.Context, namespace, group, resource, verb, name string) (bool, string, error) {
	review := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespace,
				Group:     group,
				Resource:  resource,
				Verb:      verb,
				Name:      name,
			},
		},
	}

	resp, err := c.Client.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return false, "", err
	}
	return resp.Status.Allowed, resp.Status.Reason, nil
}
