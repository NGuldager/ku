package k8s

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestConvertTableDimmedRows(t *testing.T) {
	tests := []struct {
		name   string
		res    ResourceInfo
		status string
		want   bool
	}{
		{name: "completed pod", res: ResourceInfo{Resource: "pods"}, status: "Completed", want: true},
		{name: "running pod", res: ResourceInfo{Resource: "pods"}, status: "Running"},
		{name: "completed non-pod", res: ResourceInfo{Resource: "jobs", Group: "batch"}, status: "Completed"},
	}

	for _, tt := range tests {
		tbl := convertTable(tableWithStatus(tt.status), tt.res, true)
		if len(tbl.Rows) != 1 {
			t.Fatalf("%s: rows = %d, want 1", tt.name, len(tbl.Rows))
		}
		if got := tbl.Rows[0].Dimmed; got != tt.want {
			t.Fatalf("%s: dimmed = %t, want %t", tt.name, got, tt.want)
		}
	}
}

func tableWithStatus(status string) *metav1.Table {
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name"},
			{Name: "Ready"},
			{Name: "Status"},
		},
		Rows: []metav1.TableRow{
			{Cells: []interface{}{"job-pod", "0/1", status}},
		},
	}
}
