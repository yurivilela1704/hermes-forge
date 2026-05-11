package webhook

import "testing"

func TestParseGitHubEventMerged(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"action":"closed",
		"repository":{"full_name":"acme/hermes"},
		"pull_request":{
			"number":7,
			"title":"Fix health route",
			"body":"Add missing checks",
			"html_url":"https://github.com/acme/hermes/pull/7",
			"merged":true,
			"merged_at":"2026-04-16T10:30:00Z",
			"head":{"ref":"fix/health"},
			"user":{"login":"yuri"}
		}
	}`)

	event, ignored, err := parseGitHubEvent(payload)
	if err != nil {
		t.Fatalf("parseGitHubEvent returned error: %v", err)
	}
	if ignored {
		t.Fatal("expected merged event not to be ignored")
	}
	if event.Provider != "github" || event.Number != 7 || event.Repository != "acme/hermes" {
		t.Fatalf("unexpected normalized event: %+v", event)
	}
}

func TestParseGitHubEventIgnored(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"action":"opened",
		"pull_request":{"merged":false}
	}`)

	_, ignored, err := parseGitHubEvent(payload)
	if err != nil {
		t.Fatalf("parseGitHubEvent returned error: %v", err)
	}
	if !ignored {
		t.Fatal("expected non-merged event to be ignored")
	}
}
