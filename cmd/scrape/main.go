package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type RoutersJSON struct {
	Routers []string `json:"routers"`
}

type RepoData struct {
	URL              string     `json:"url"`
	Exists           bool       `json:"exists"`
	Archived         bool       `json:"archived,omitempty"`
	Stars            int        `json:"stars,omitempty"`
	HasRelease       bool       `json:"has_release"`
	LastReleaseAt    *time.Time `json:"last_release_at,omitempty"`
	OpenIssues       int        `json:"open_issues"`
	OpenPullRequests int        `json:"open_pull_requests"`
}

func runGHAPI(path string) ([]byte, error) {
	cmd := exec.Command("gh", "api", path)
	return cmd.Output()
}

func getRepoInfo(owner, repo string) (*RepoData, error) {
	repoPath := fmt.Sprintf("repos/%s/%s", owner, repo)
	data := &RepoData{
		URL: fmt.Sprintf("https://github.com/%s/%s", owner, repo),
	}

	output, err := runGHAPI(repoPath)
	if err != nil {
		data.Exists = false
		return data, nil
	}

	var repoResp struct {
		Archived        bool `json:"archived"`
		StargazersCount int  `json:"stargazers_count"`
	}
	if err := json.Unmarshal(output, &repoResp); err != nil {
		return nil, err
	}
	data.Exists = true
	data.Archived = repoResp.Archived
	data.Stars = repoResp.StargazersCount

	// Attempt to get latest release
	releasePath := fmt.Sprintf("repos/%s/%s/releases/latest", owner, repo)
	output, err = runGHAPI(releasePath)
	if err == nil {
		var releaseResp struct {
			PublishedAt time.Time `json:"published_at"`
		}
		if err := json.Unmarshal(output, &releaseResp); err == nil {
			data.HasRelease = true
			data.LastReleaseAt = &releaseResp.PublishedAt
		}
	}

	// Get open issues count
	issuesPath := fmt.Sprintf("repos/%s/%s/issues?state=open&per_page=1", owner, repo)
	output, err = runGHAPI(issuesPath)
	if err == nil {
		var issues []interface{}
		if err := json.Unmarshal(output, &issues); err == nil {
			// GitHub API paginates, but we can get total from the Link header or just count returned items for small repos
			// For simplicity, use the total_count from the search API
			searchIssuesPath := fmt.Sprintf("search/issues?q=repo:%s/%s+type:issue+state:open", owner, repo)
			output, err = runGHAPI(searchIssuesPath)
			if err == nil {
				var searchResp struct {
					TotalCount int `json:"total_count"`
				}
				if err := json.Unmarshal(output, &searchResp); err == nil {
					data.OpenIssues = searchResp.TotalCount
				}
			}
		}
	}

	// Get open pull requests count
	prsPath := fmt.Sprintf("search/issues?q=repo:%s/%s+type:pr+state:open", owner, repo)
	output, err = runGHAPI(prsPath)
	if err == nil {
		var searchResp struct {
			TotalCount int `json:"total_count"`
		}
		if err := json.Unmarshal(output, &searchResp); err == nil {
			data.OpenPullRequests = searchResp.TotalCount
		}
	}

	return data, nil
}

func main() {
	f, err := os.Open("routers.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var routers RoutersJSON
	if err := json.NewDecoder(f).Decode(&routers); err != nil {
		panic(err)
	}

	var results []RepoData

	for _, url := range routers.Routers {
		parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Skipping malformed URL: %s\n", url)
			continue
		}
		owner, repo := parts[0], parts[1]

		info, err := getRepoInfo(owner, repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching info for %s: %v\n", url, err)
			continue
		}
		results = append(results, *info)
	}

	out, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(out))
}
