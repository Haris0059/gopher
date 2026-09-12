package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// DefaultAPIBaseURL is the GitHub API base used to resolve "latest".
const DefaultAPIBaseURL = "https://api.github.com"

// DefaultDownloadBaseURL is the GitHub host used to download release assets.
const DefaultDownloadBaseURL = "https://github.com"

// repoOwner and repoName identify the release source.
const (
	repoOwner = "Haris0059"
	repoName  = "gopher"
)

var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-\S+)?$`)

// gitHubRelease is the subset of the GitHub releases API response we need.
type gitHubRelease struct {
	TagName string `json:"tag_name"`
}

// ResolveVersion turns a version/channel string into a concrete version
// (without a leading "v"). "", "latest", and "stable" all resolve against
// the GitHub releases API; anything shaped like a semver is used verbatim.
// Source: download.ts:108-145 getLatestVersion()
func ResolveVersion(ctx context.Context, client *http.Client, apiBaseURL, channelOrVersion string) (string, error) {
	if semverRe.MatchString(channelOrVersion) {
		return strings.TrimPrefix(channelOrVersion, "v"), nil
	}
	if channelOrVersion != "" && channelOrVersion != "latest" && channelOrVersion != "stable" {
		return "", fmt.Errorf("invalid channel: %s (use 'stable' or 'latest', or a version like 1.2.3)", channelOrVersion)
	}

	if client == nil {
		client = http.DefaultClient
	}
	if apiBaseURL == "" {
		apiBaseURL = DefaultAPIBaseURL
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBaseURL, repoOwner, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode release response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("release response had no tag_name")
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}
