package update

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAvailableUpdate(t *testing.T) {
	t.Parallel()

	available, err := availableUpdate("1.2.3", "v1.3.0", "https://example.com/release")
	if err != nil {
		t.Fatalf("availableUpdate() error = %v", err)
	}
	if available == nil {
		t.Fatal("availableUpdate() = nil, want update")
	}
	if available.Version != "1.3.0" || available.URL != "https://example.com/release" {
		t.Errorf("availableUpdate() = %+v", available)
	}
}

func TestAvailableUpdateIgnoresCurrentAndOlderVersions(t *testing.T) {
	t.Parallel()

	for _, latest := range []string{"v1.2.3", "v1.2.2", "v0.99.99"} {
		available, err := availableUpdate("1.2.3", latest, "")
		if err != nil {
			t.Fatalf("availableUpdate(%q) error = %v", latest, err)
		}
		if available != nil {
			t.Errorf("availableUpdate(%q) = %+v, want nil", latest, available)
		}
	}
}

func TestAvailableUpdateRejectsMalformedReleaseVersion(t *testing.T) {
	t.Parallel()

	if _, err := availableUpdate("1.2.3", "latest", ""); err == nil {
		t.Fatal("availableUpdate() error = nil, want malformed version error")
	}
}

func TestFetchLatestUsesGitHubAPIHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("Accept header = %q", request.Header.Get("Accept"))
		}
		if request.Header.Get("User-Agent") != "loc-visuals-update-check" {
			t.Errorf("User-Agent header = %q", request.Header.Get("User-Agent"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"tag_name":"v2.0.0","html_url":"https://example.com/v2"}`))
	}))
	defer server.Close()

	entry, err := fetchLatest(server.Client(), server.URL)
	if err != nil {
		t.Fatalf("fetchLatest() error = %v", err)
	}
	if entry.TagName != "v2.0.0" || entry.HTMLURL != "https://example.com/v2" {
		t.Errorf("fetchLatest() = %+v", entry)
	}
}
