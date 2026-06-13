package k8s

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventLine is a single recent event for the cockpit.
type EventLine struct {
	Age       string
	Namespace string
	Type      string
	Reason    string
	Object    string
	Message   string
}

// ClusterOverview is the cockpit's snapshot of cluster health and usage.
type ClusterOverview struct {
	Version    string
	Nodes      int
	NodesReady int

	HasMetrics    bool
	CPUUsedMilli  int64
	CPUAllocMilli int64
	MemUsedBytes  int64
	MemAllocBytes int64

	Namespaces int

	Pods         int
	PodRunning   int
	PodPending   int
	PodFailed    int
	PodSucceeded int

	Deployments      int
	DeploymentsReady int

	Warnings []EventLine
}

// ClusterStats gathers a cluster overview. Each section is best-effort: a
// failure in one (e.g. no metrics) leaves its fields zero rather than failing
// the whole snapshot.
func (c *Client) ClusterStats(ctx context.Context) (*ClusterOverview, error) {
	o := &ClusterOverview{}

	if v, err := c.disco.ServerVersion(); err == nil {
		o.Version = v.GitVersion
	}

	if nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err == nil {
		o.Nodes = len(nodes.Items)
		for i := range nodes.Items {
			n := &nodes.Items[i]
			if nodeReady(n) {
				o.NodesReady++
			}
			if q, ok := n.Status.Allocatable[corev1.ResourceCPU]; ok {
				o.CPUAllocMilli += q.MilliValue()
			}
			if q, ok := n.Status.Allocatable[corev1.ResourceMemory]; ok {
				o.MemAllocBytes += q.Value()
			}
		}
	}

	if usage, err := c.nodeUsage(ctx); err == nil && len(usage) > 0 {
		o.HasMetrics = true
		for _, u := range usage {
			o.CPUUsedMilli += u.cpuMilli
			o.MemUsedBytes += u.memBytes
		}
	}

	if nss, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}); err == nil {
		o.Namespaces = len(nss.Items)
	}

	if pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{}); err == nil {
		o.Pods = len(pods.Items)
		for i := range pods.Items {
			switch pods.Items[i].Status.Phase {
			case corev1.PodRunning:
				o.PodRunning++
			case corev1.PodPending:
				o.PodPending++
			case corev1.PodFailed:
				o.PodFailed++
			case corev1.PodSucceeded:
				o.PodSucceeded++
			}
		}
	}

	if deps, err := c.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{}); err == nil {
		o.Deployments = len(deps.Items)
		for i := range deps.Items {
			d := &deps.Items[i]
			if d.Status.Replicas > 0 && d.Status.ReadyReplicas >= d.Status.Replicas {
				o.DeploymentsReady++
			} else if d.Status.Replicas == 0 {
				o.DeploymentsReady++ // scaled to zero counts as healthy
			}
		}
	}

	o.Warnings = c.recentWarnings(ctx)
	return o, nil
}

func nodeReady(n *corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (c *Client) recentWarnings(ctx context.Context) []EventLine {
	list, err := c.clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
		Limit:         300,
	})
	if err != nil {
		return nil
	}
	items := list.Items
	sort.Slice(items, func(i, j int) bool {
		return eventTime(&items[i]).After(eventTime(&items[j]))
	})
	if len(items) > 8 {
		items = items[:8]
	}
	out := make([]EventLine, 0, len(items))
	for i := range items {
		e := &items[i]
		obj := e.InvolvedObject.Kind
		if e.InvolvedObject.Name != "" {
			obj += "/" + e.InvolvedObject.Name
		}
		out = append(out, EventLine{
			Age:       ageString(eventTime(e)),
			Namespace: e.Namespace,
			Type:      e.Type,
			Reason:    e.Reason,
			Object:    obj,
			Message:   e.Message,
		})
	}
	return out
}

func eventTime(e *corev1.Event) time.Time {
	if !e.LastTimestamp.IsZero() {
		return e.LastTimestamp.Time
	}
	if !e.EventTime.IsZero() {
		return e.EventTime.Time
	}
	return e.CreationTimestamp.Time
}

// ageString renders a compact age like 2m, 3h, or 4d.
func ageString(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
