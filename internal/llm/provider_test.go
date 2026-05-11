package llm

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewUnsupportedProvider(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Provider: "nope"}, &http.Client{})
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
	if !errors.Is(err, ErrUnsupportedProvider) {
		t.Fatalf("expected ErrUnsupportedProvider, got: %v", err)
	}
}

func TestNewOpenAIDefaultModel(t *testing.T) {
	t.Parallel()

	p, err := New(Config{
		Provider:     "openai",
		OpenAIAPIKey: "x",
	}, &http.Client{})
	if err != nil {
		t.Fatalf("expected provider, got error: %v", err)
	}
	op, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected OpenAIProvider type, got: %T", p)
	}
	if op.model != "gpt-4o" {
		t.Fatalf("expected default model gpt-4o, got: %s", op.model)
	}
}

func TestNewMockProvider(t *testing.T) {
	t.Parallel()

	p, err := New(Config{Provider: "mock"}, &http.Client{})
	if err != nil {
		t.Fatalf("expected mock provider, got error: %v", err)
	}
	if _, ok := p.(*MockProvider); !ok {
		t.Fatalf("expected MockProvider type, got: %T", p)
	}
}
