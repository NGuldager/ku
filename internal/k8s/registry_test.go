package k8s

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

type preferredResourcesDiscovery struct {
	discovery.DiscoveryInterface
	lists []*metav1.APIResourceList
	err   error
}

func (d preferredResourcesDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return d.lists, d.err
}

func TestLoadRegistryFailsWhenDiscoveryReturnsNoResources(t *testing.T) {
	boom := errors.New("discovery unavailable")
	c := &Client{disco: preferredResourcesDiscovery{err: boom}}

	err := c.loadRegistry()
	if !errors.Is(err, boom) {
		t.Fatalf("loadRegistry() error = %v, want %v", err, boom)
	}
	if c.registry != nil {
		t.Fatalf("registry = %+v, want nil after total failure", c.registry)
	}
}

func TestLoadRegistryKeepsPartialResources(t *testing.T) {
	boom := errors.New("metrics API unavailable")
	c := &Client{disco: preferredResourcesDiscovery{
		err: boom,
		lists: []*metav1.APIResourceList{{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{{
				Name:         "pods",
				SingularName: "pod",
				Namespaced:   true,
				Kind:         "Pod",
				Verbs:        []string{"list"},
			}},
		}},
	}}

	err := c.loadRegistry()
	if !errors.Is(err, boom) {
		t.Fatalf("loadRegistry() error = %v, want %v", err, boom)
	}
	if c.registry == nil {
		t.Fatal("registry is nil after partial discovery")
	}
	if _, ok := c.registry.Resolve("pods"); !ok {
		t.Fatal("partial registry cannot resolve pods")
	}
}

func TestResolveDoesNotFallbackForQualifiedResource(t *testing.T) {
	reg := &Registry{byKey: map[string]ResourceInfo{}}
	reg.add(ResourceInfo{Resource: "pods", Kind: "Pod"})

	if _, ok := reg.Resolve("pods"); !ok {
		t.Fatal("Resolve(pods) failed")
	}
	if _, ok := reg.Resolve("pods.badgroup"); ok {
		t.Fatal("Resolve(pods.badgroup) fell back to core pods")
	}
}

func TestHasPodSelector(t *testing.T) {
	cases := []struct {
		group, resource string
		want            bool
	}{
		{"apps", "deployments", true},
		{"apps", "statefulsets", true},
		{"apps", "daemonsets", true},
		{"apps", "replicasets", true},
		{"batch", "jobs", true},
		{"", "pods", false},
		{"", "services", false},
		{"", "nodes", false},
		{"batch", "cronjobs", false},
	}
	for _, tc := range cases {
		ri := ResourceInfo{Group: tc.group, Resource: tc.resource}
		if got := ri.HasPodSelector(); got != tc.want {
			t.Errorf("HasPodSelector(%s/%s) = %t, want %t", tc.group, tc.resource, got, tc.want)
		}
	}
}
