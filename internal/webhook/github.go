package webhook

import (
	"encoding/json"
	"net/http"
	"time"
)

type gitHubPayload struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Number   int    `json:"number"`
		Title    string `json:"title"`
		Body     string `json:"body"`
		HTMLURL  string `json:"html_url"`
		Merged   bool   `json:"merged"`
		MergedAt string `json:"merged_at"`
		Head     struct {
			Ref string `json:"ref"`
		} `json:"head"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"pull_request"`
}

// GitHubHandler parses PR merged events and emits normalized data.
func GitHubHandler(secret string, onEvent func(Event), onTrace func(payload []byte, respCode int, respBody any)) http.Handler {
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

		if err := validateGitHubSignature(secret, payload, r.Header.Get("X-Hub-Signature-256")); err != nil {
			respCode = http.StatusUnauthorized
			respBody = "unauthorized"
			http.Error(w, "unauthorized", respCode)
			return
		}

		event, ignored, err := parseGitHubEvent(payload)
		if err != nil {
			respCode = http.StatusBadRequest
			respBody = "invalid github payload"
			http.Error(w, "invalid github payload", respCode)
			return
		}
		if ignored {
			respCode = http.StatusOK
			respBody = map[string]any{"ignored": true, "reason": "not a merged pull request"}
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

func parseGitHubEvent(payload []byte) (Event, bool, error) {
	var p gitHubPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return Event{}, false, err
	}
	if p.Action != "closed" || !p.PullRequest.Merged {
		return Event{}, true, nil
	}

	event := Event{
		Provider:    "github",
		Repository:  p.Repository.FullName,
		Title:       p.PullRequest.Title,
		Author:      p.PullRequest.User.Login,
		Branch:      p.PullRequest.Head.Ref,
		Description: p.PullRequest.Body,
		Number:      p.PullRequest.Number,
		URL:         p.PullRequest.HTMLURL,
	}
	if p.PullRequest.MergedAt != "" {
		if mergedAt, err := time.Parse(time.RFC3339, p.PullRequest.MergedAt); err == nil {
			event.MergedAt = mergedAt
		}
	}

	return event, false, nil
}
