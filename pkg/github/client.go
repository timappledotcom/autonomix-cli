package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Asset struct {
	Name        string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
	Body    string  `json:"body"`
	HTMLURL string  `json:"html_url"`
}

// GetLatestRelease fetches the latest release info for a github repo url
// url format: https://github.com/owner/repo
func GetLatestRelease(repoURL string) (*Release, error) {
	parts := strings.Split(repoURL, "github.com/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid github url")
	}
	repoPath := strings.TrimSuffix(parts[1], "/")
	
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoPath)
	
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status: %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	return &rel, nil
}
