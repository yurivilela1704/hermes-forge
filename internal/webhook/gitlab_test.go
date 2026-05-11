package webhook

import (
	"testing"
	"time"
)

func TestParseGitLabEventMerged(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"object_kind":"merge_request",
		"project":{"path_with_namespace":"acme/hermes"},
		"user":{"username":"yuri"},
		"object_attributes":{
			"iid":42,
			"title":"Add webhook parser",
			"description":"Parses merge events",
			"source_branch":"feature/webhooks",
			"state":"merged",
			"action":"merge",
			"url":"https://gitlab.example/acme/hermes/-/merge_requests/42",
			"merged_at":"2026-04-16T10:30:00Z",
			"last_commit":{"message":"feat: add parser"}
		}
	}`)

	event, ignored, err := parseGitLabEvent(payload)
	if err != nil {
		t.Fatalf("parseGitLabEvent returned error: %v", err)
	}
	if ignored {
		t.Fatal("expected merged event not to be ignored")
	}
	if event.Provider != "gitlab" || event.Number != 42 || event.Repository != "acme/hermes" {
		t.Fatalf("unexpected normalized event: %+v", event)
	}

	expectedMergedAt := "2026-04-16T10:30:00Z"
	if event.MergedAt.Format(time.RFC3339) != expectedMergedAt {
		t.Errorf("expected merged_at %s, got %s", expectedMergedAt, event.MergedAt.Format(time.RFC3339))
	}
}

func TestParseGitLabEventMerged_UpdatedAtFallback(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"object_kind":"merge_request",
		"project":{"path_with_namespace":"acme/hermes"},
		"user":{"username":"yuri"},
		"object_attributes":{
			"iid":42,
			"title":"Add webhook parser",
			"updated_at":"2026-04-17T11:45:00Z",
			"state":"merged",
			"action":"merge",
			"last_commit":{"message":"feat: add parser"}
		}
	}`)

	event, ignored, err := parseGitLabEvent(payload)
	if err != nil {
		t.Fatalf("parseGitLabEvent returned error: %v", err)
	}
	if ignored {
		t.Fatal("expected merged event not to be ignored")
	}

	expectedMergedAt := "2026-04-17T11:45:00Z"
	if event.MergedAt.Format(time.RFC3339) != expectedMergedAt {
		t.Errorf("expected merged_at %s, got %s", expectedMergedAt, event.MergedAt.Format(time.RFC3339))
	}
}

func TestParseGitLabEventIgnored(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"object_kind":"merge_request",
		"object_attributes":{"state":"opened","action":"open"}
	}`)

	_, ignored, err := parseGitLabEvent(payload)
	if err != nil {
		t.Fatalf("parseGitLabEvent returned error: %v", err)
	}
	if !ignored {
		t.Fatal("expected non-merged event to be ignored")
	}
}
