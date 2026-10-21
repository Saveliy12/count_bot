package env

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	llmAPIKeyEnvName  = "GROQ_API_KEY"
	llmModelEnvName   = "LLM_MODEL"
	llmBaseURLEnvName = "LLM_BASE_URL"
	llmTimeoutEnvName = "LLM_TIMEOUT"

	defaultLLMModel   = "openai/gpt-oss-120b"
	defaultLLMBaseURL = "https://api.groq.com/openai/v1"
	defaultLLMTimeout = 20 * time.Second
)

type llmConfig struct {
	apiKey  string
	model   string
	baseURL string
	timeout time.Duration
}

func NewLLMConfig() (*llmConfig, error) {
	model := strings.TrimSpace(os.Getenv(llmModelEnvName))
	if model == "" {
		model = defaultLLMModel
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(llmBaseURLEnvName)), "/")
	if baseURL == "" {
		baseURL = defaultLLMBaseURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("LLM_BASE_URL must be an absolute URL")
	}

	timeout := defaultLLMTimeout
	if raw := strings.TrimSpace(os.Getenv(llmTimeoutEnvName)); raw != "" {
		timeout, err = time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return nil, errors.New("LLM_TIMEOUT must be a positive duration")
		}
	}

	return &llmConfig{
		apiKey:  strings.TrimSpace(os.Getenv(llmAPIKeyEnvName)),
		model:   model,
		baseURL: baseURL,
		timeout: timeout,
	}, nil
}

func (cfg *llmConfig) APIKey() string {
	return cfg.apiKey
}

func (cfg *llmConfig) Model() string {
	return cfg.model
}

func (cfg *llmConfig) BaseURL() string {
	return cfg.baseURL
}

func (cfg *llmConfig) Timeout() time.Duration {
	return cfg.timeout
}

func (cfg *llmConfig) Enabled() bool {
	return cfg.apiKey != ""
}
