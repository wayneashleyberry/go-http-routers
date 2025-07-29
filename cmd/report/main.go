package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// Shared struct from previous script
type RepoData struct {
	URL           string     `json:"url"`
	Exists        bool       `json:"exists"`
	Archived      bool       `json:"archived,omitempty"`
	Stars         int        `json:"stars,omitempty"`
	HasRelease    bool       `json:"has_release"`
	LastReleaseAt *time.Time `json:"last_release_at,omitempty"`
}

func isQualified(repo RepoData) bool {
	if !repo.Exists || !repo.HasRelease || repo.LastReleaseAt == nil {
		return false
	}
	return time.Since(*repo.LastReleaseAt).Hours() <= 365*24
}

func printMarkdownTable(title string, criteria string, repos []RepoData) {
	fmt.Printf("## %s (%d)\n\n", title, len(repos))
	fmt.Println(criteria)
	fmt.Println()
	fmt.Println("| Repo | Stars | Archived | Last Release |")
	fmt.Println("|------|-------|----------|---------------|")
	for _, r := range repos {
		archived := "No"
		if r.Archived {
			archived = "Yes"
		}
		release := "-"
		if r.LastReleaseAt != nil {
			release = r.LastReleaseAt.Format("2006-01-02")
		}
		fmt.Printf("| [%s](%s) | %d | %s | %s |\n",
			repoNameFromURL(r.URL), r.URL, r.Stars, archived, release)
	}
	fmt.Println()
}

func repoNameFromURL(url string) string {
	parts := len("https://github.com/")
	return url[parts:]
}

func main() {
	file, err := os.Open("data.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var repos []RepoData
	if err := json.NewDecoder(file).Decode(&repos); err != nil {
		panic(err)
	}

	var qualified, unqualified []RepoData

	for _, repo := range repos {
		if isQualified(repo) {
			qualified = append(qualified, repo)
		} else {
			unqualified = append(unqualified, repo)
		}
	}

	// Sort qualified by stars descending
	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].Stars > qualified[j].Stars
	})

	// Sort unqualified by last release date descending
	sort.Slice(unqualified, func(i, j int) bool {
		a, b := unqualified[i], unqualified[j]
		// Put those with no release at the bottom
		if a.LastReleaseAt == nil && b.LastReleaseAt == nil {
			return false
		}
		if a.LastReleaseAt == nil {
			return false
		}
		if b.LastReleaseAt == nil {
			return true
		}
		return a.LastReleaseAt.After(*b.LastReleaseAt)
	})

	fmt.Println("# Go HTTP Router Repositories")

	printMarkdownTable(
		"Qualified Routers",
		"These repositories **exist**, have at least one **GitHub release**, and their **latest release is within the past 365 days**. Ordered by star count (descending).",
		qualified,
	)

	printMarkdownTable(
		"Unqualified Routers",
		"These repositories **do not meet all qualification criteria** — they may no longer exist, lack any releases, or have not had a release in over a year. Ordered by most recent release (if any).",
		unqualified,
	)

}
