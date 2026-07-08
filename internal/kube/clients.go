package kube

import (
	"context"
	"fmt"

	appconfig "github.com/vk7416/generic-k8s-mcp/internal/config"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Clients contains Kubernetes clients created from either kubeconfig or in-cluster config.
type Clients struct {
	Mode             string
	CurrentContext   string
	DefaultNamespace string
	REST             *rest.Config
	Clientset        kubernetes.Interface
	Dynamic          dynamic.Interface
	Discovery        discovery.DiscoveryInterface
	Mapper           meta.RESTMapper
	ServerVersion    string
}

// NewClients builds Kubernetes clients using the selected access mode.
func NewClients(ctx context.Context, cfg appconfig.Config) (*Clients, error) {
	var restCfg *rest.Config
	var err error
	currentContext := "in-cluster"
	defaultNS := cfg.Namespace
	if defaultNS == "" {
		defaultNS = "default"
	}

	switch cfg.Mode {
	case "incluster":
		restCfg, err = rest.InClusterConfig()
	case "local", "":
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		if cfg.Kubeconfig != "" {
			loadingRules.ExplicitPath = cfg.Kubeconfig
		}
		overrides := &clientcmd.ConfigOverrides{}
		if cfg.Context != "" {
			overrides.CurrentContext = cfg.Context
		}
		clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
		raw, rawErr := clientCfg.RawConfig()
		if rawErr == nil {
			currentContext = raw.CurrentContext
			if cfg.Context != "" {
				currentContext = cfg.Context
			}
			if ctxCfg, ok := raw.Contexts[currentContext]; ok && cfg.Namespace == "" && ctxCfg.Namespace != "" {
				defaultNS = ctxCfg.Namespace
			}
		}
		restCfg, err = clientCfg.ClientConfig()
	default:
		return nil, fmt.Errorf("unsupported mode %q", cfg.Mode)
	}
	if err != nil {
		return nil, err
	}

	if restCfg.QPS == 0 {
		restCfg.QPS = 50
		restCfg.Burst = 100
	}

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	disco, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))

	version := "unknown"
	if v, err := clientset.Discovery().ServerVersion(); err == nil {
		version = v.String()
	}

	return &Clients{
		Mode:             cfg.Mode,
		CurrentContext:   currentContext,
		DefaultNamespace: defaultNS,
		REST:             restCfg,
		Clientset:        clientset,
		Dynamic:          dyn,
		Discovery:        disco,
		Mapper:           mapper,
		ServerVersion:    version,
	}, nil
}

// NamespaceOrDefault returns a Kubernetes namespace value. The string "all" means all namespaces.
func (c *Clients) NamespaceOrDefault(ns string) string {
	if ns == "" {
		ns = c.DefaultNamespace
	}
	if ns == "all" || ns == "*" || ns == "-A" {
		return metav1.NamespaceAll
	}
	return ns
}
