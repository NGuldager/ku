package ui

import (
	"testing"

	"github.com/bjarneo/kli/internal/k8s"
)

func TestKubernetesDocsURL(t *testing.T) {
	tests := []struct {
		name string
		res  k8s.ResourceInfo
		want string
		ok   bool
	}{
		{
			name: "ingress",
			res:  k8s.ResourceInfo{Resource: "ingresses", Kind: "Ingress"},
			want: "https://kubernetes.io/docs/concepts/services-networking/ingress/",
			ok:   true,
		},
		{
			name: "deployment",
			res:  k8s.ResourceInfo{Resource: "deployments", Kind: "Deployment"},
			want: "https://kubernetes.io/docs/concepts/workloads/controllers/deployment/",
			ok:   true,
		},
		{
			name: "unknown crd",
			res:  k8s.ResourceInfo{Group: "example.com", Resource: "widgets", Kind: "Widget"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := kubernetesDocsURL(tt.res)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("kubernetesDocsURL(%+v) = %q, %t; want %q, %t", tt.res, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestDocsResourceUsesCurrentScreen(t *testing.T) {
	deploy := k8s.ResourceInfo{Resource: "deployments", Kind: "Deployment"}
	ingress := k8s.ResourceInfo{Resource: "ingresses", Kind: "Ingress"}
	secret := k8s.ResourceInfo{Resource: "secrets", Kind: "Secret"}

	tests := []struct {
		name string
		app  App
		want string
		ok   bool
	}{
		{
			name: "table",
			app:  App{screen: screenTable, res: deploy},
			want: "deployments",
			ok:   true,
		},
		{
			name: "config target",
			app:  App{screen: screenConfig, res: deploy, configTarget: target{res: ingress}},
			want: "ingresses",
			ok:   true,
		},
		{
			name: "detail target",
			app:  App{screen: screenDetail, res: deploy, detailTarget: target{res: secret}},
			want: "secrets",
			ok:   true,
		},
		{
			name: "logs",
			app:  App{screen: screenLogs, res: deploy},
			want: "pods",
			ok:   true,
		},
		{
			name: "cockpit",
			app:  App{screen: screenCockpit, res: deploy},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.app.docsResource()
			if ok != tt.ok || got.Resource != tt.want {
				t.Fatalf("docsResource() = %+v, %t; want resource %q, %t", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestValidateDocsURL(t *testing.T) {
	if err := validateDocsURL("https://kubernetes.io/docs/"); err != nil {
		t.Fatalf("valid URL rejected: %v", err)
	}
	for _, raw := range []string{"file:///etc/passwd", "javascript:alert(1)", "https://"} {
		if err := validateDocsURL(raw); err == nil {
			t.Fatalf("invalid URL %q accepted", raw)
		}
	}
}
