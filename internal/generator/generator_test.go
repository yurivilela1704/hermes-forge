package generator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"release-notes-gen/internal/llm"
	"release-notes-gen/internal/webhook"
)

type fakeProvider struct {
	result string
	err    error
	prompt string
}

func (f *fakeProvider) Generate(_ context.Context, prompt string) (llm.Generation, error) {
	f.prompt = prompt
	if f.err != nil {
		return llm.Generation{}, f.err
	}
	return llm.Generation{
		Text: f.result,
		Usage: llm.Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	}, nil
}

func TestGenerateReleaseNoteSuccess(t *testing.T) {
	t.Parallel()

	fp := &fakeProvider{result: "### Summary\nEverything ok"}
	gen := New("default", fp)

	event := webhook.Event{
		Title:       "Add auth",
		Author:      "yuri",
		Branch:      "feature/auth",
		Description: "Adds auth middleware",
		Labels:      "auth, feature",
		GeneratedAt: "2024-05-20 10:00:00",
		Commits: []webhook.Commit{
			{Message: "feat: add middleware", Author: "yuri"},
		},
	}

	result, err := gen.GenerateReleaseNote(context.Background(), event)
	if err != nil {
		t.Fatalf("GenerateReleaseNote returned error: %v", err)
	}
	if !strings.Contains(result.Note, "Summary") {
		t.Fatalf("expected summary in note, got: %s", result.Note)
	}
	if result.Usage.InputTokens == 0 || result.Usage.OutputTokens == 0 {
		t.Fatalf("expected usage values in result, got: %+v", result.Usage)
	}
	if !strings.Contains(fp.prompt, "Title: Add auth") {
		t.Fatalf("expected rendered prompt to include event data, got: %s", fp.prompt)
	}
}

func TestGenerateReleaseNoteProviderError(t *testing.T) {
	t.Parallel()

	fp := &fakeProvider{err: errors.New("provider down")}
	gen := New("default", fp)

	event := webhook.Event{Title: "x"}
	_, err := gen.GenerateReleaseNote(context.Background(), event)
	if err == nil {
		t.Fatal("expected error from provider")
	}
	if !strings.Contains(err.Error(), "generate release note") {
		t.Fatalf("expected wrapped error message, got: %v", err)
	}
}

func TestGeneratorBuildPrompt(t *testing.T) {
	t.Parallel()

	gen := New("default", nil)
	event := webhook.Event{Title: "Test MR"}
	prompt, err := gen.BuildPrompt(event)
	if err != nil {
		t.Fatalf("BuildPrompt returned error: %v", err)
	}
	if !strings.Contains(prompt, "Title: Test MR") {
		t.Fatalf("expected title in prompt, got: %s", prompt)
	}
}

func TestGeneratorBuildDailyPrompt(t *testing.T) {
	t.Parallel()

	gen := New("default", nil)
	events := []webhook.Event{{Title: "MR 1"}, {Title: "MR 2"}}
	prompt, err := gen.BuildDailyPrompt(events, "daily")
	if err != nil {
		t.Fatalf("BuildDailyPrompt returned error: %v", err)
	}
	if !strings.Contains(prompt, "MR #0: MR 1") {
		t.Fatalf("expected MR 1 in prompt, got: %s", prompt)
	}
}
