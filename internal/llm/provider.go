package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Provider is the common abstraction for all LLM backends.
type Provider interface {
	Generate(ctx context.Context, prompt string) (Generation, error)
	Model() string
}

// Usage represents provider-reported token usage.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Generation is a generated text response with usage metadata.
type Generation struct {
	Text  string `json:"text"`
	Usage Usage  `json:"usage"`
}

// Config stores provider-specific runtime configuration.
type Config struct {
	Provider string

	OpenAIAPIKey       string
	OpenAIModel        string
	OpenAIOrganization string
	OpenAIBaseURI      string

	AzureEndpoint   string
	AzureAPIKey     string
	AzureDeployment string
	AnthropicAPIKey string
	AnthropicModel  string

	GeminiAPIKey string
	GeminiModel  string
}

var (
	// ErrUnsupportedProvider indicates an unknown LLM provider.
	ErrUnsupportedProvider = errors.New("unsupported llm provider")
	// ErrMissingConfig indicates the required environment configuration is missing.
	ErrMissingConfig = errors.New("missing llm configuration")
)

func NewFromEnv() (Provider, error) {
	cfg := Config{
		Provider:           strings.TrimSpace(os.Getenv("LLM_PROVIDER")),
		OpenAIAPIKey:       strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIModel:        strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
		OpenAIOrganization: strings.TrimSpace(os.Getenv("OPENAI_ORGANIZATION")),
		OpenAIBaseURI:      strings.TrimSpace(os.Getenv("OPENAI_BASE_URI")),
		AzureEndpoint:      strings.TrimSpace(os.Getenv("AZURE_OPENAI_ENDPOINT")),
		AzureAPIKey:        strings.TrimSpace(os.Getenv("AZURE_OPENAI_API_KEY")),
		AzureDeployment:    strings.TrimSpace(os.Getenv("AZURE_OPENAI_DEPLOYMENT")),
		AnthropicAPIKey:    strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		AnthropicModel:     strings.TrimSpace(os.Getenv("ANTHROPIC_MODEL")),
		GeminiAPIKey:       strings.TrimSpace(os.Getenv("GOOGLE_GEMINI_API_KEY")),
		GeminiModel:        strings.TrimSpace(os.Getenv("GEMINI_MODEL")),
	}
	return New(cfg, defaultHTTPClient())
}

func New(cfg Config, client *http.Client) (Provider, error) {
	if client == nil {
		client = defaultHTTPClient()
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch provider {
	case "mock":
		return &MockProvider{}, nil
	case "", "openai":
		if cfg.OpenAIAPIKey == "" {
			return nil, fmt.Errorf("%w: OPENAI_API_KEY", ErrMissingConfig)
		}
		model := cfg.OpenAIModel
		if model == "" {
			model = "gpt-4o"
		}
		return &OpenAIProvider{
			apiKey:       cfg.OpenAIAPIKey,
			model:        model,
			organization: cfg.OpenAIOrganization,
			baseURI:      cfg.OpenAIBaseURI,
			client:       client,
		}, nil
	case "azure":
		if cfg.AzureEndpoint == "" || cfg.AzureAPIKey == "" || cfg.AzureDeployment == "" {
			return nil, fmt.Errorf("%w: AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_KEY and AZURE_OPENAI_DEPLOYMENT are required", ErrMissingConfig)
		}
		return &AzureProvider{
			endpoint:   strings.TrimSuffix(cfg.AzureEndpoint, "/"),
			apiKey:     cfg.AzureAPIKey,
			deployment: cfg.AzureDeployment,
			client:     client,
		}, nil
	case "anthropic":
		if cfg.AnthropicAPIKey == "" {
			return nil, fmt.Errorf("%w: ANTHROPIC_API_KEY", ErrMissingConfig)
		}
		model := cfg.AnthropicModel
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		return &AnthropicProvider{
			apiKey: cfg.AnthropicAPIKey,
			model:  model,
			client: client,
		}, nil
	case "gemini", "google":
		if cfg.GeminiAPIKey == "" {
			return nil, fmt.Errorf("%w: GOOGLE_GEMINI_API_KEY", ErrMissingConfig)
		}
		model := cfg.GeminiModel
		if model == "" {
			model = "gemini-3-flash-preview"
		}
		return &GeminiProvider{
			apiKey: cfg.GeminiAPIKey,
			model:  model,
			client: client,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, provider)
	}
}

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 45 * time.Second}
}
