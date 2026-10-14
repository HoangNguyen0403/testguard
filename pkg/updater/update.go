package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ReleaseAsset represents an asset attached to a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubRelease represents the release object returned by GitHub API.
type GitHubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

// CleanVersion strips leading 'v' from a version string.
func CleanVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// IsNewer checks if the latest version is greater than the current version using semantic versioning.
func IsNewer(current, latest string) bool {
	c := strings.Split(CleanVersion(current), ".")
	l := strings.Split(CleanVersion(latest), ".")

	for i := 0; i < len(c) && i < len(l); i++ {
		cNum, err1 := strconv.Atoi(c[i])
		lNum, err2 := strconv.Atoi(l[i])
		if err1 != nil || err2 != nil {
			continue
		}
		if lNum > cNum {
			return true
		}
		if lNum < cNum {
			return false
		}
	}
	return len(l) > len(c)
}

// FetchLatestRelease queries the GitHub API for the latest release metadata.
func FetchLatestRelease(repo string) (*GitHubRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "testguard-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No releases published yet
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// UpdateExecutable replaces the current running binary with a newly downloaded binary.
func UpdateExecutable(currentVersion, repo string) error {
	release, err := FetchLatestRelease(repo)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if release == nil {
		fmt.Printf("ℹ️ No published releases found yet on GitHub. You are on the latest build (%s).\n", currentVersion)
		return nil
	}

	if !IsNewer(currentVersion, release.TagName) {
		fmt.Printf("✨ TestGuard is already up to date (%s).\n", currentVersion)
		return nil
	}

	fmt.Printf("🚀 Found newer version: %s (current: %s). Updating...\n", release.TagName, currentVersion)

	expectedAssetName := fmt.Sprintf("testguard-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == expectedAssetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no matching binary asset found for %s-%s in release %s", runtime.GOOS, runtime.GOARCH, release.TagName)
	}

	// 1. Download asset
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decompress archive: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var newBinaryBytes []byte

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Name == "testguard" || header.Name == "testguard.exe" {
			newBinaryBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return err
			}
			break
		}
	}

	if len(newBinaryBytes) == 0 {
		return fmt.Errorf("testguard executable not found inside downloaded archive")
	}

	// 2. Locate current running binary
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to determine current executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("unable to resolve executable symlinks: %w", err)
	}

	// 3. Write to temporary file in same directory for atomic replace
	tempFile := execPath + ".tmp"
	if err := os.WriteFile(tempFile, newBinaryBytes, 0755); err != nil {
		return fmt.Errorf("failed to write update: %w", err)
	}

	if err := os.Rename(tempFile, execPath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	fmt.Printf("✅ TestGuard successfully updated to %s!\n", release.TagName)
	return nil
}
