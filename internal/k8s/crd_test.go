package k8s

import (
	"testing"
)

func ver(name string, served, storage bool) map[string]any {
	return map[string]any{"name": name, "served": served, "storage": storage}
}

func crdSpec(group, plural, kind, scope string, versions ...map[string]any) map[string]any {
	vs := make([]any, len(versions))
	for i, v := range versions {
		vs[i] = v
	}
	return map[string]any{
		"spec": map[string]any{
			"group":    group,
			"scope":    scope,
			"names":    map[string]any{"plural": plural, "kind": kind},
			"versions": vs,
		},
	}
}

func TestCRDResourcePrefersRegistryEntry(t *testing.T) {
	reg := NewRegistry([]ResourceInfo{
		{Group: "keda.sh", Version: "v1alpha1", Resource: "scaledobjects", Kind: "ScaledObject", Namespaced: true},
	})
	ri, ok := reg.crdResource(crdSpec("keda.sh", "scaledobjects", "ScaledObject", "Namespaced", ver("v1", true, true)))
	if !ok {
		t.Fatal("expected the CRD to resolve")
	}
	if ri.Version != "v1alpha1" {
		t.Fatalf("should keep the registry's discovered version, got %q", ri.Version)
	}
}

func TestCRDResourceFallsBackToSpec(t *testing.T) {
	reg := NewRegistry(nil) // empty: the CRD is not already known
	ri, ok := reg.crdResource(crdSpec("widgets.io", "widgets", "Widget", "Cluster",
		ver("v1beta1", true, false), ver("v1", true, true)))
	if !ok {
		t.Fatal("expected the spec fallback to resolve")
	}
	if ri.Version != "v1" {
		t.Fatalf("expected the storage version v1, got %q", ri.Version)
	}
	if ri.Namespaced {
		t.Fatal("a Cluster-scoped CRD should not be namespaced")
	}
	if ri.Kind != "Widget" || ri.Resource != "widgets" {
		t.Fatalf("unexpected resource: %+v", ri)
	}
}

func TestCRDResourceSkipsUnservedAndMalformed(t *testing.T) {
	reg := NewRegistry(nil)
	if _, ok := reg.crdResource(crdSpec("x.io", "xs", "X", "Cluster", ver("v1", false, false))); ok {
		t.Fatal("a CRD with no served version should be skipped")
	}
	if _, ok := reg.crdResource(map[string]any{"spec": map[string]any{}}); ok {
		t.Fatal("a CRD missing group/plural should be skipped")
	}
}

func TestRegistryMergeAddsNewSkipsExisting(t *testing.T) {
	reg := NewRegistry([]ResourceInfo{{Resource: "pods", Kind: "Pod"}})
	reg.Merge([]ResourceInfo{
		{Resource: "pods", Kind: "Pod"},                              // existing -> skipped
		{Group: "keda.sh", Version: "v1", Resource: "scaledobjects"}, // new
		{Group: "keda.sh", Version: "v1", Resource: "scaledobjects"}, // duplicate within the merge -> skipped
	})
	if got := len(reg.All()); got != 2 {
		t.Fatalf("expected 2 resources after merge, got %d", got)
	}
	if _, ok := reg.Resolve("scaledobjects.keda.sh"); !ok {
		t.Fatal("a merged CRD must be resolvable so the : jump can find it")
	}
}
