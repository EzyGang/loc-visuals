package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	releaseURL    = "https://api.github.com/repos/EzyGang/loc-visuals/releases/latest"
	releasePage   = "https://github.com/EzyGang/loc-visuals/releases/latest"
	cacheName     = "update.json"
	checkInterval = 24 * time.Hour
)

// Available describes a newer published release.
type Available struct {
	Version string
	URL     string
}

type cacheEntry struct {
	CheckedAt int64  `json:"checked_at"`
	TagName   string `json:"tag_name"`
	HTMLURL   string `json:"html_url"`
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// Check returns the latest release when it is newer than current. Results are
// cached for a day to avoid adding a GitHub API request to every invocation.
func Check(current string) (*Available, error) {
	path, err := cachePath()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	entry, fresh, err := readFreshCache(path, now)
	if err != nil {
		return nil, err
	}
	if !fresh {
		entry, err = fetchLatest(&http.Client{Timeout: 2 * time.Second}, releaseURL)
		if err != nil {
			return nil, err
		}
		entry.CheckedAt = now.Unix()
		if err := writeCache(path, entry); err != nil {
			return nil, err
		}
	}

	return availableUpdate(current, entry.TagName, entry.HTMLURL)
}

func fetchLatest(client *http.Client, url string) (cacheEntry, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return cacheEntry{}, fmt.Errorf("create GitHub release request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "loc-visuals-update-check")

	response, err := client.Do(request)
	if err != nil {
		return cacheEntry{}, fmt.Errorf("request latest GitHub release: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return cacheEntry{}, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return cacheEntry{}, fmt.Errorf("request latest GitHub release: GitHub API returned %s", response.Status)
	}

	var release releaseResponse
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return cacheEntry{}, fmt.Errorf("decode latest GitHub release: %w", err)
	}
	return cacheEntry{TagName: release.TagName, HTMLURL: release.HTMLURL}, nil
}

func availableUpdate(current string, latest string, url string) (*Available, error) {
	if latest == "" {
		return nil, nil
	}
	currentVersion, err := parseVersion(current)
	if err != nil {
		return nil, fmt.Errorf("parse current version %q: %w", current, err)
	}
	latestVersion, err := parseVersion(latest)
	if err != nil {
		return nil, fmt.Errorf("parse release version %q: %w", latest, err)
	}
	if compareVersion(latestVersion, currentVersion) <= 0 {
		return nil, nil
	}
	if url == "" {
		url = releasePage
	}
	return &Available{Version: strings.TrimPrefix(latest, "v"), URL: url}, nil
}

func parseVersion(value string) ([3]int, error) {
	parts := strings.Split(strings.TrimPrefix(value, "v"), ".")
	if len(parts) != 3 {
		return [3]int{}, errors.New("expected MAJOR.MINOR.PATCH")
	}
	var parsed [3]int
	for index, part := range parts {
		if part == "" {
			return [3]int{}, errors.New("expected numeric version components")
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return [3]int{}, errors.New("expected numeric version components")
		}
		parsed[index] = number
	}
	return parsed, nil
}

func compareVersion(left [3]int, right [3]int) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}

func cachePath() (string, error) {
	if directory := os.Getenv("LOC_VISUALS_UPDATE_CACHE_DIR"); directory != "" {
		return filepath.Join(directory, cacheName), nil
	}
	directory, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve update cache directory: %w", err)
	}
	return filepath.Join(directory, "loc-visuals", cacheName), nil
}

func readFreshCache(path string, now time.Time) (cacheEntry, bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cacheEntry{}, false, nil
	}
	if err != nil {
		return cacheEntry{}, false, fmt.Errorf("read update cache %s: %w", path, err)
	}
	var entry cacheEntry
	if err := json.Unmarshal(content, &entry); err != nil {
		return cacheEntry{}, false, nil
	}
	checkedAt := time.Unix(entry.CheckedAt, 0)
	return entry, now.Sub(checkedAt) >= 0 && now.Sub(checkedAt) < checkInterval, nil
}

func writeCache(path string, entry cacheEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create update cache directory %s: %w", filepath.Dir(path), err)
	}
	content, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode update cache: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write update cache %s: %w", path, err)
	}
	return nil
}
