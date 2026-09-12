package installer

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// MaxDownloadRetries is the number of attempts for a stalled/network-failed
// download before giving up. HTTP errors and checksum mismatches are not
// retried. Source: download.ts:272 MAX_DOWNLOAD_RETRIES
const MaxDownloadRetries = 3

// downloadRetryDelay is the pause between retry attempts.
const downloadRetryDelay = time.Second

// assetName returns the release asset filename for a version/platform,
// matching goreleaser's default naming.
func assetName(version, goos, goarch string) string {
	name := fmt.Sprintf("gopher_%s_%s_%s", version, goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// releaseAssetURL builds the download URL for a release asset.
func releaseAssetURL(baseURL, version, filename string) string {
	return fmt.Sprintf("%s/%s/%s/releases/download/v%s/%s", baseURL, repoOwner, repoName, version, filename)
}

// fetchChecksum fetches checksums.txt for a release and returns the
// hex-encoded SHA-256 for filename, or an error if the entry is missing.
// Source: download.ts:424-435 (manifest.json lookup); here we use
// goreleaser's checksums.txt convention instead of a manifest.
func fetchChecksum(ctx context.Context, client *http.Client, baseURL, version, filename string) (string, error) {
	url := releaseAssetURL(baseURL, version, "checksums.txt")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksums for version %s: %w", version, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch checksums for version %s: status %d", version, resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}
		if fields[1] == filename {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("asset %s not found in checksums for version %s", filename, version)
}

// downloadVersion downloads and verifies the release asset for version onto
// dest, retrying on network/stall errors but not on HTTP errors or checksum
// mismatches. dest's parent directory must already exist.
// Source: download.ts:278-374 downloadAndVerifyBinary()
func downloadVersion(ctx context.Context, client *http.Client, baseURL, version, goos, goarch, dest string) error {
	if client == nil {
		client = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = DefaultDownloadBaseURL
	}

	filename := assetName(version, goos, goarch)

	expectedSum, err := fetchChecksum(ctx, client, baseURL, version, filename)
	if err != nil {
		return err
	}

	url := releaseAssetURL(baseURL, version, filename)

	var lastErr error
	for attempt := 0; attempt < MaxDownloadRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(downloadRetryDelay):
			}
		}

		retry, err := downloadOnce(ctx, client, url, dest, expectedSum)
		if err == nil {
			return os.Chmod(dest, 0o755)
		}
		lastErr = err
		if !retry {
			return err
		}
	}
	return fmt.Errorf("download failed after %d attempts: %w", MaxDownloadRetries, lastErr)
}

// downloadOnce performs a single download+verify attempt. The returned bool
// reports whether the error is retryable (network-level failure); HTTP
// status errors and checksum mismatches return false.
func downloadOnce(ctx context.Context, client *http.Client, url, dest, expectedSum string) (retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return true, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download failed: status %d for %s", resp.StatusCode, url)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return false, fmt.Errorf("failed to open %s: %w", dest, err)
	}

	hasher := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(f, hasher), resp.Body)
	closeErr := f.Close()

	if copyErr != nil {
		os.Remove(dest)
		return true, fmt.Errorf("download interrupted: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(dest)
		return false, closeErr
	}

	actualSum := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualSum, expectedSum) {
		os.Remove(dest)
		return false, fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSum, actualSum)
	}

	return false, nil
}
