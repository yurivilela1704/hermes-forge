package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"release-notes-gen/internal/webhook"
)

// Client is a minimal GitLab API client for listing merged MRs.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type mrResponse struct {
	IID            int    `json:"iid"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	SourceBranch   string `json:"source_branch"`
	WebURL         string `json:"web_url"`
	MergedAt       string `json:"merged_at"`
	MergeCommitSHA string `json:"merge_commit_sha"`
	Author         struct {
		Username string `json:"username"`
	} `json:"author"`
	Labels []string `json:"labels"`
}

// ListMergedMRs returns merged MRs for a project between startDate and endDate (YYYY-MM-DD).
func (c *Client) ListMergedMRs(project, startDate, endDate string) ([]webhook.Event, error) {
	// GitLab expects project path URL-encoded
	encodedProject := url.PathEscape(project)

	// Filter: updated_after = startDate 00:00:00, updated_before = endDate 23:59:59
	updatedAfter := startDate + "T00:00:00Z"
	updatedBefore := endDate + "T23:59:59Z"

	apiURL := fmt.Sprintf(
		"%s/api/v4/projects/%s/merge_requests?state=merged&updated_after=%s&updated_before=%s&per_page=100",
		c.BaseURL,
		encodedProject,
		url.QueryEscape(updatedAfter),
		url.QueryEscape(updatedBefore),
	)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab api returned status %d", resp.StatusCode)
	}

	var mrs []mrResponse
	if err := json.NewDecoder(resp.Body).Decode(&mrs); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	events := make([]webhook.Event, 0, len(mrs))
	for _, mr := range mrs {
		// Strict filtering by merged date range
		if mr.MergedAt == "" {
			continue
		}

		// Check if MergedAt is within [startDate, endDate]
		// Since MergedAt is ISO8601, we can do string comparison if we are careful,
		// or parse and compare.
		mrMergedTime, err := time.Parse(time.RFC3339, mr.MergedAt)
		if err != nil {
			continue
		}

		start, _ := time.Parse("2006-01-02", startDate)
		end, _ := time.Parse("2006-01-02", endDate)
		end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		if mrMergedTime.Before(start) || mrMergedTime.After(end) {
			continue
		}

		// Fetch real commits for this MR
		commits, err := c.fetchMRCommits(project, mr.IID)
		if err != nil {
			// Log error but continue with placeholder if fetching commits fails
			fmt.Printf("warning: failed to fetch commits for project=%s MR!%d: %v\n", project, mr.IID, err)
			commits = []webhook.Commit{{
				Message: "Merged via GitLab Sync (fallback)",
				Author:  mr.Author.Username,
			}}
		}

		events = append(events, webhook.Event{
			Provider:    "gitlab",
			Repository:  project,
			Title:       mr.Title,
			Author:      mr.Author.Username,
			Branch:      mr.SourceBranch,
			Description: mr.Description,
			Number:      mr.IID,
			URL:         mr.WebURL,
			Labels:      strings.Join(mr.Labels, ", "),
			MergedAt:    mrMergedTime,
			GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
			Commits:     commits,
		})
	}
	return events, nil
}

// fetchMRCommits returns all commits for a specific merge request.
func (c *Client) fetchMRCommits(project string, mrIID int) ([]webhook.Commit, error) {
	encodedProject := url.PathEscape(project)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/commits?per_page=100", c.BaseURL, encodedProject, mrIID)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab api commits returned status %d", resp.StatusCode)
	}

	var commits []struct {
		Message    string `json:"message"`
		AuthorName string `json:"author_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}

	res := make([]webhook.Commit, 0, len(commits))
	for _, cm := range commits {
		res = append(res, webhook.Commit{
			Message: strings.TrimSpace(cm.Message),
			Author:  cm.AuthorName,
		})
	}
	return res, nil
}

func nextDay(date string) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02") + "T00:00:00Z", nil
}
