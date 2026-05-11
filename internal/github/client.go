package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"release-notes-gen/internal/webhook"
)

// Client is a minimal GitHub API client for listing merged PRs.
type Client struct {
	Token      string
	HTTPClient *http.Client
}

func New(token string) *Client {
	return &Client{
		Token:      token,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type searchResponse struct {
	Items []struct {
		Number   int    `json:"number"`
		Title    string `json:"title"`
		Body     string `json:"body"`
		HTMLURL  string `json:"html_url"`
		MergedAt string `json:"merged_at"`
		User     struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		// Repository is not directly in search items if we search within a repo, 
		// but we know it from the query.
	} `json:"items"`
}

// ListMergedPRs returns merged PRs for a repository between startDate and endDate (YYYY-MM-DD).
func (c *Client) ListMergedPRs(repo, startDate, endDate string) ([]webhook.Event, error) {
	query := fmt.Sprintf("repo:%s is:pr is:merged merged:%s..%s", repo, startDate, endDate)
	apiURL := fmt.Sprintf("https://api.github.com/search/issues?q=%s&per_page=100", url.QueryEscape(query))

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	events := make([]webhook.Event, 0, len(sr.Items))
	for _, item := range sr.Items {
		mergedAt, _ := time.Parse(time.RFC3339, item.MergedAt)

		labels := make([]string, 0, len(item.Labels))
		for _, l := range item.Labels {
			labels = append(labels, l.Name)
		}

		// Fetch commits for this PR
		commits, err := c.fetchPRCommits(repo, item.Number)
		if err != nil {
			fmt.Printf("warning: failed to fetch commits for repo=%s PR#%d: %v\n", repo, item.Number, err)
			commits = []webhook.Commit{{
				Message: "Merged via GitHub Sync (fallback)",
				Author:  item.User.Login,
			}}
		}

		events = append(events, webhook.Event{
			Provider:    "github",
			Repository:  repo,
			Title:       item.Title,
			Author:      item.User.Login,
			Branch:      item.Head.Ref,
			Description: item.Body,
			Number:      item.Number,
			URL:         item.HTMLURL,
			Labels:      strings.Join(labels, ", "),
			MergedAt:    mergedAt,
			GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
			Commits:     commits,
		})
	}
	return events, nil
}

func (c *Client) fetchPRCommits(repo string, prNumber int) ([]webhook.Commit, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d/commits?per_page=100", repo, prNumber)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api commits returned status %d", resp.StatusCode)
	}

	var commits []struct {
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}

	res := make([]webhook.Commit, 0, len(commits))
	for _, cm := range commits {
		res = append(res, webhook.Commit{
			Message: strings.TrimSpace(cm.Commit.Message),
			Author:  cm.Commit.Author.Name,
		})
	}
	return res, nil
}
