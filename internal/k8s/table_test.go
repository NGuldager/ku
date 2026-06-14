package k8s

import (
	"strings"
	"testing"
)

func TestTableAcceptExcludesPlainJSON(t *testing.T) {
	parts := strings.Split(tableAccept, ",")
	for _, part := range parts {
		if strings.TrimSpace(part) == "application/json" {
			t.Fatal("tableAccept must not include plain application/json")
		}
	}
}

func TestDecodeTableRejectsNormalList(t *testing.T) {
	_, err := decodeTable([]byte(`{"kind":"PodList","items":[]}`), "pods")
	if err == nil {
		t.Fatal("decodeTable succeeded for PodList")
	}
	if !strings.Contains(err.Error(), "PodList") {
		t.Fatalf("decodeTable error = %q, want PodList", err)
	}
}
