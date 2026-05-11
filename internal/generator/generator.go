package generator

import (
	"context"
	"fmt"

	"release-notes-gen/internal/llm"
	"release-notes-gen/internal/webhook"
)

// Generator orchestrates prompt building and LLM completion.
type Generator struct {
	templateName string
	provider     llm.Provider
}

type Result struct {
	Note  string
	Usage llm.Usage
}

func New(templateName string, provider llm.Provider) *Generator {
	return &Generator{
		templateName: templateName,
		provider:     provider,
	}
}

func (g *Generator) GenerateReleaseNote(ctx context.Context, event webhook.Event) (Result, error) {
	if g.provider == nil {
		return Result{}, fmt.Errorf("llm provider is required")
	}

	prompt, err := g.BuildPrompt(event)
	if err != nil {
		return Result{}, err
	}

	gen, err := g.provider.Generate(ctx, prompt)
	if err != nil {
		return Result{}, fmt.Errorf("generate release note: %w", err)
	}
	return Result{
		Note:  gen.Text,
		Usage: gen.Usage,
	}, nil
}

func (g *Generator) GenerateDailyReleaseNote(ctx context.Context, events []webhook.Event, templateName string) (Result, error) {
	if g.provider == nil {
		return Result{}, fmt.Errorf("llm provider is required")
	}

	prompt, err := g.BuildDailyPrompt(events, templateName)
	if err != nil {
		return Result{}, err
	}

	gen, err := g.provider.Generate(ctx, prompt)
	if err != nil {
		return Result{}, fmt.Errorf("generate daily release note: %w", err)
	}
	return Result{
		Note:  gen.Text,
		Usage: gen.Usage,
	}, nil
}

func (g *Generator) BuildPrompt(event webhook.Event) (string, error) {
	return BuildPrompt(g.templateName, event)
}

func (g *Generator) BuildDailyPrompt(events []webhook.Event, templateName string) (string, error) {
	if templateName == "" {
		templateName = "daily"
	}
	return BuildDailyPrompt(templateName, events)
}

func (g *Generator) Model() string {
	if g.provider == nil {
		return "none"
	}
	return g.provider.Model()
}
