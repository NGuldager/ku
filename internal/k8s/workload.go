package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// WorkloadPodSelector resolves a workload's spec.selector into a label selector
// string suitable for listing the pods it owns. It works uniformly across the
// pod-owning kinds (deployments, statefulsets, daemonsets, replicasets, jobs),
// all of which carry a metav1.LabelSelector at spec.selector.
func (c *Client) WorkloadPodSelector(ctx context.Context, res ResourceInfo, namespace, name string) (string, error) {
	obj, err := c.dynamic.Resource(res.GVR()).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	raw, found, err := unstructured.NestedMap(obj.Object, "spec", "selector")
	if err != nil || !found {
		return "", fmt.Errorf("%s %q has no selector", res.Resource, name)
	}
	var ls metav1.LabelSelector
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, &ls); err != nil {
		return "", fmt.Errorf("decode selector: %w", err)
	}
	sel, err := metav1.LabelSelectorAsSelector(&ls)
	if err != nil {
		return "", fmt.Errorf("selector: %w", err)
	}
	if sel.Empty() {
		return "", fmt.Errorf("%s %q has empty selector", res.Resource, name)
	}
	return sel.String(), nil
}
