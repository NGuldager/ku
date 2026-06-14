package k8s

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nodeDebugImage is the image used for the node debug pod. Override with
// $KLI_DEBUG_IMAGE. It only needs a shell and chroot (busybox suffices).
func nodeDebugImage() string {
	if img := os.Getenv("KLI_DEBUG_IMAGE"); img != "" {
		return img
	}
	return "busybox"
}

// NodeShellCommand drops into the host's root filesystem from the debug
// container (the host is mounted at /host), the same idea as `kubectl debug
// node`. It prefers the host's bash, then sh, falling back to the container's
// shell.
var NodeShellCommand = []string{"/bin/sh", "-c", "chroot /host bash || chroot /host sh || exec /bin/sh"}

// CreateNodeDebugPod creates a privileged pod pinned to nodeName with the host
// namespaces and root filesystem mounted at /host, then waits for it to run.
// It returns the pod name and container to exec into. This is how a node shell
// is obtained, since nodes cannot be exec'd directly.
func (c *Client) CreateNodeDebugPod(ctx context.Context, namespace, nodeName string) (string, string, error) {
	priv := true
	deadline := int64(3600)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kli-node-debug-",
			Labels:       map[string]string{"app.kubernetes.io/managed-by": "kli"},
		},
		Spec: corev1.PodSpec{
			NodeName:              nodeName,
			HostPID:               true,
			HostNetwork:           true,
			HostIPC:               true,
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: &deadline, // self-terminate if left behind
			Tolerations:           []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
			Containers: []corev1.Container{{
				Name:            "debug",
				Image:           nodeDebugImage(),
				Command:         []string{"sleep", "3600"},
				Stdin:           true,
				TTY:             true,
				SecurityContext: &corev1.SecurityContext{Privileged: &priv},
				VolumeMounts:    []corev1.VolumeMount{{Name: "host", MountPath: "/host"}},
			}},
			Volumes: []corev1.Volume{{
				Name:         "host",
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}},
			}},
		},
	}

	created, err := c.clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return "", "", err
	}
	if err := c.waitPodRunning(ctx, namespace, created.Name); err != nil {
		// Best-effort cleanup if it never became usable. Use a fresh bounded
		// context because ctx is often already cancelled here.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if cleanupErr := c.DeletePod(cleanupCtx, namespace, created.Name); cleanupErr != nil {
			return "", "", fmt.Errorf("%w; cleanup debug pod: %v", err, cleanupErr)
		}
		return "", "", err
	}
	return created.Name, "debug", nil
}

func (c *Client) waitPodRunning(ctx context.Context, namespace, name string) error {
	for {
		p, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		switch p.Status.Phase {
		case corev1.PodRunning:
			return nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return fmt.Errorf("debug pod %s", p.Status.Phase)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// DeletePod removes a pod immediately (used to clean up debug pods).
func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	grace := int64(0)
	return c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{GracePeriodSeconds: &grace})
}
