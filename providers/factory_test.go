package providers

import (
	"testing"

	"github.com/smallnest/goclaw/config"
)

func TestDetermineProviderStripsPrefixes(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProvider ProviderType
		wantModel    string
	}{
		{
			name:         "openrouter prefix",
			model:        "openrouter:anthropic/claude-opus-4-5",
			wantProvider: ProviderTypeOpenRouter,
			wantModel:    "anthropic/claude-opus-4-5",
		},
		{
			name:         "anthropic prefix",
			model:        "anthropic:claude-3-5-sonnet",
			wantProvider: ProviderTypeAnthropic,
			wantModel:    "claude-3-5-sonnet",
		},
		{
			name:         "qianfan prefix",
			model:        "qianfan:deepseek-v3.2",
			wantProvider: ProviderTypeQianfan,
			wantModel:    "deepseek-v3.2",
		},
		{
			name:         "openai prefix",
			model:        "openai:gpt-4o-mini",
			wantProvider: ProviderTypeOpenAI,
			wantModel:    "gpt-4o-mini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Agents: config.AgentsConfig{
					Defaults: config.AgentDefaults{
						Model: tt.model,
					},
				},
			}

			gotProvider, gotModel, err := determineProvider(cfg)
			if err != nil {
				t.Fatalf("determineProvider returned error: %v", err)
			}
			if gotProvider != tt.wantProvider {
				t.Fatalf("provider = %q, want %q", gotProvider, tt.wantProvider)
			}
			if gotModel != tt.wantModel {
				t.Fatalf("model = %q, want %q", gotModel, tt.wantModel)
			}
		})
	}
}

func TestDetermineProviderFallsBackToConfiguredProvider(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Model: "deepseek-v3.2",
			},
		},
		Providers: config.ProvidersConfig{
			OpenAI: config.OpenAIProviderConfig{
				APIKey: "test-key",
			},
		},
	}

	gotProvider, gotModel, err := determineProvider(cfg)
	if err != nil {
		t.Fatalf("determineProvider returned error: %v", err)
	}
	if gotProvider != ProviderTypeOpenAI {
		t.Fatalf("provider = %q, want %q", gotProvider, ProviderTypeOpenAI)
	}
	if gotModel != "deepseek-v3.2" {
		t.Fatalf("model = %q, want %q", gotModel, "deepseek-v3.2")
	}
}

func TestDetermineProviderFallsBackToConfiguredQianfan(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Model: "deepseek-v3.2",
			},
		},
		Providers: config.ProvidersConfig{
			Qianfan: config.OpenAIProviderConfig{
				APIKey: "test-key",
			},
		},
	}

	gotProvider, gotModel, err := determineProvider(cfg)
	if err != nil {
		t.Fatalf("determineProvider returned error: %v", err)
	}
	if gotProvider != ProviderTypeQianfan {
		t.Fatalf("provider = %q, want %q", gotProvider, ProviderTypeQianfan)
	}
	if gotModel != "deepseek-v3.2" {
		t.Fatalf("model = %q, want %q", gotModel, "deepseek-v3.2")
	}
}

func TestNormalizeModelForProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerType ProviderType
		model        string
		want         string
	}{
		{
			name:         "strip openai prefix",
			providerType: ProviderTypeOpenAI,
			model:        "openai:gpt-4o",
			want:         "gpt-4o",
		},
		{
			name:         "strip anthropic prefix",
			providerType: ProviderTypeAnthropic,
			model:        "anthropic:claude-sonnet-4",
			want:         "claude-sonnet-4",
		},
		{
			name:         "strip qianfan prefix",
			providerType: ProviderTypeQianfan,
			model:        "qianfan:deepseek-v3.2",
			want:         "deepseek-v3.2",
		},
		{
			name:         "strip openrouter prefix",
			providerType: ProviderTypeOpenRouter,
			model:        "openrouter:anthropic/claude-opus-4-5",
			want:         "anthropic/claude-opus-4-5",
		},
		{
			name:         "leave plain model untouched",
			providerType: ProviderTypeOpenAI,
			model:        "deepseek-v3.2",
			want:         "deepseek-v3.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeModelForProvider(tt.providerType, tt.model)
			if got != tt.want {
				t.Fatalf("normalizeModelForProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}
