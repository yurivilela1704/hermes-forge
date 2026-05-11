package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type gitLabPayload struct {
	ObjectKind string `json:"object_kind"`
	Project    struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
	User struct {
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"user"`
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		SourceBranch string `json:"source_branch"`
		State        string `json:"state"`
		Action       string `json:"action"`
		URL          string `json:"url"`
		MergedAt     string `json:"merged_at"`
		UpdatedAt    string `json:"updated_at"`
		Labels       []struct {
			Title string `json:"title"`
		} `json:"labels"`
		LastCommit struct {
			Message string `json:"message"`
		} `json:"last_commit"`
		MergeCommitSHA string `json:"merge_commit_sha"`
	} `json:"object_attributes"`
}

// GitLabHandler parses MR merged events and emits normalized data.
func GitLabHandler(secret string, onEvent func(Event), onTrace func(payload []byte, respCode int, respBody any)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload, err := readPayload(r)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Closure to log trace after response is written
		var respCode int
		var respBody any
		defer func() {
			if onTrace != nil {
				onTrace(payload, respCode, respBody)
			}
		}()

		if err := validateGitLabToken(secret, r.Header.Get("X-Gitlab-Token")); err != nil {
			respCode = http.StatusUnauthorized
			respBody = "unauthorized"
			http.Error(w, "unauthorized", respCode)
			return
		}

		event, ignored, err := parseGitLabEvent(payload)
		if err != nil {
			respCode = http.StatusBadRequest
			respBody = "invalid gitlab payload"
			http.Error(w, "invalid gitlab payload", respCode)
			return
		}
		if ignored {
			respCode = http.StatusOK
			respBody = map[string]any{"ignored": true, "reason": "not a merged merge request"}
			writeJSON(w, respCode, respBody)
			return
		}

		if onEvent != nil {
			onEvent(event)
		}
		respCode = http.StatusAccepted
		respBody = map[string]any{"accepted": true, "provider": event.Provider, "number": event.Number, "title": event.Title}
		writeJSON(w, respCode, respBody)
	})
}

func parseGitLabEvent(payload []byte) (Event, bool, error) {
	var p gitLabPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return Event{}, false, err
	}
	if p.ObjectKind != "merge_request" {
		return Event{}, true, nil
	}
	if p.ObjectAttributes.State != "merged" && p.ObjectAttributes.Action != "merge" {
		return Event{}, true, nil
	}

	author := p.User.Username
	if author == "" {
		author = p.User.Name
	}

	var labels []string
	for _, l := range p.ObjectAttributes.Labels {
		labels = append(labels, l.Title)
	}

	event := Event{
		Provider:    "gitlab",
		Repository:  p.Project.PathWithNamespace,
		Title:       p.ObjectAttributes.Title,
		Author:      author,
		Branch:      p.ObjectAttributes.SourceBranch,
		Description: p.ObjectAttributes.Description,
		Number:      p.ObjectAttributes.IID,
		URL:         p.ObjectAttributes.URL,
		Labels:      strings.Join(labels, ", "),
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Commits: []Commit{{
			Message: p.ObjectAttributes.LastCommit.Message,
			Author:  author,
		}},
		MergedAt: time.Now().UTC(), // Default to now
	}

	if p.ObjectAttributes.MergedAt != "" {
		if t, err := parseFlexibleTime(p.ObjectAttributes.MergedAt); err == nil {
			event.MergedAt = t
		}
	} else if p.ObjectAttributes.UpdatedAt != "" {
		if t, err := parseFlexibleTime(p.ObjectAttributes.UpdatedAt); err == nil {
			event.MergedAt = t
		}
	}

	return event, false, nil
}

func parseFlexibleTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse time: %s", s)
}
