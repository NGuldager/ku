package k8s

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func deploymentsRes() ResourceInfo {
	return ResourceInfo{Group: "apps", Version: "v1", Resource: "deployments", Kind: "Deployment", Namespaced: true}
}

func newWorkloadObj(matchLabels map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "api", "namespace": "default"},
		"spec":       map[string]interface{}{},
	}}
	if matchLabels != nil {
		_ = unstructured.SetNestedMap(obj.Object, map[string]interface{}{"matchLabels": matchLabels}, "spec", "selector")
	}
	return obj
}

func TestWorkloadPodSelector(t *testing.T) {
	obj := newWorkloadObj(map[string]interface{}{"app": "api", "tier": "backend"})
	c := &Client{dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), obj)}

	got, err := c.WorkloadPodSelector(context.Background(), deploymentsRes(), "default", "api")
	if err != nil {
		t.Fatalf("WorkloadPodSelector() error = %v", err)
	}
	// LabelSelectorAsSelector sorts keys, so the string is deterministic.
	if want := "app=api,tier=backend"; got != want {
		t.Fatalf("WorkloadPodSelector() = %q, want %q", got, want)
	}
}

func TestWorkloadPodSelectorEmpty(t *testing.T) {
	obj := newWorkloadObj(nil) // no spec.selector
	c := &Client{dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), obj)}

	_, err := c.WorkloadPodSelector(context.Background(), deploymentsRes(), "default", "api")
	if err == nil || !strings.Contains(err.Error(), "no selector") {
		t.Fatalf("WorkloadPodSelector() error = %v, want a no-selector error", err)
	}
}
