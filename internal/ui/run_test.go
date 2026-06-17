package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPreflightsEmptyKubeconfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}

	err := Run(Options{Kubeconfig: path})
	if err == nil {
		t.Fatal("Run succeeded for empty kubeconfig")
	}
	if !strings.Contains(err.Error(), "kubeconfig is empty or missing") {
		t.Fatalf("error = %q, want friendly kubeconfig message", err)
	}
}
