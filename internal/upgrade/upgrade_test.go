package upgrade

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type rewriter struct {
	target *url.URL
	rt     http.RoundTripper
}

func (r rewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = r.target.Scheme
	clone.URL.Host = r.target.Host
	clone.Host = r.target.Host
	return r.rt.RoundTrip(clone)
}

func installTestClient(t *testing.T, serverURL string) {
	t.Helper()
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	old := httpClient
	httpClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: rewriter{target: u, rt: http.DefaultTransport},
	}
	t.Cleanup(func() { httpClient = old })
}

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch string
		want         string
	}{
		{goos: "linux", goarch: "amd64", want: "kli-linux-amd64"},
		{goos: "darwin", goarch: "arm64", want: "kli-darwin-arm64"},
		{goos: "windows", goarch: "amd64", want: "kli-windows-amd64.exe"},
		{goos: "windows", goarch: "arm64", want: "kli-windows-arm64.exe"},
	}
	for _, tt := range tests {
		got, err := assetName(tt.goos, tt.goarch)
		if err != nil {
			t.Fatalf("assetName(%q, %q): %v", tt.goos, tt.goarch, err)
		}
		if got != tt.want {
			t.Errorf("assetName(%q, %q) = %q; want %q", tt.goos, tt.goarch, got, tt.want)
		}
	}
}

func TestLatestVersionSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/") || !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer srv.Close()
	installTestClient(t, srv.URL)

	tag, err := latestVersion()
	if err != nil {
		t.Fatalf("latestVersion: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("tag = %q; want v1.2.3", tag)
	}
}

func TestLatestChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "abc123  kli-linux-amd64\n")
	}))
	defer srv.Close()
	installTestClient(t, srv.URL)

	got, err := latestChecksum(srv.URL+"/checksums.txt", "kli-linux-amd64")
	if err != nil {
		t.Fatalf("latestChecksum: %v", err)
	}
	if got != "abc123" {
		t.Errorf("checksum = %q; want abc123", got)
	}
}

func TestDownloadAndReplace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "kli")
	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	newContent := []byte("NEW BINARY BYTES")
	sum := sha256.Sum256(newContent)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(newContent)
	}))
	defer srv.Close()
	installTestClient(t, srv.URL)

	if err := downloadAndReplace(srv.URL+"/kli-linux-amd64", target, fmt.Sprintf("%x", sum)); err != nil {
		t.Fatalf("downloadAndReplace: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(newContent) {
		t.Errorf("target content = %q; want %q", got, newContent)
	}
}

func TestDownloadAndReplaceChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "kli")
	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "NEW")
	}))
	defer srv.Close()
	installTestClient(t, srv.URL)

	err := downloadAndReplace(srv.URL+"/kli", target, strings.Repeat("0", 64))
	if err == nil {
		t.Fatal("downloadAndReplace should fail on checksum mismatch")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "OLD" {
		t.Errorf("target content = %q; want OLD", got)
	}
}
