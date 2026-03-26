package config

import (
	"testing"
	"time"
)

func TestValidatorValidConfig(t *testing.T) {
	validator := NewValidator(true)

	cfg := &Config{
		Workspace: WorkspaceConfig{
			Path: "/tmp/test-workspace",
		},
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model:         "qianfan:test-model",
				MaxIterations: 11,
				Temperature:   1.7,
				MaxTokens:     2048,
			},
		},
		Models: ModelsConfig{
			Mode: "merge",
			Providers: map[string]ModelProviderConfig{
				"qianfan": {
					BaseURL: "https://qianfan.baidubce.com/v2",
					APIKey:  "test-valid-api-key-12345",
					API:     ModelAPIOpenAICompletions,
					Models: []ModelDefinitionConfig{
						{
							ID:            "test-model",
							Name:          "Test Model",
							ContextWindow: 128000,
							MaxTokens:     8192,
						},
					},
				},
			},
		},
		Gateway: GatewayConfig{
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			WebSocket: WebSocketConfig{
				Host:         "localhost",
				Port:         8081,
				PingInterval: 30 * time.Second,
				PongTimeout:  30 * time.Second,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
		},
		Tools: ToolsConfig{
			Web: WebToolConfig{
				Timeout: 11,
			},
		},
		Memory: MemoryConfig{
			Backend: "builtin",
		},
	}

	if err := validator.Validate(cfg); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidatorInvalidModel(t *testing.T) {
	validator := NewValidator(true)

	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model: "", // Invalid
			},
		},
		Memory: MemoryConfig{
			Backend: "builtin",
		},
	}

	err := validator.Validate(cfg)
	if err == nil {
		t.Error("expected error for empty model")
	}
}

func TestValidatorInvalidTemperature(t *testing.T) {
	validator := NewValidator(true)

	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model:       "qianfan:test-model",
				Temperature: 3.0, // Invalid > 1
			},
		},
		Models: ModelsConfig{
			Mode: "merge",
			Providers: map[string]ModelProviderConfig{
				"qianfan": {
					BaseURL: "https://qianfan.baidubce.com/v2",
					APIKey:  "test-valid-api-key-12345",
					API:     ModelAPIOpenAICompletions,
					Models: []ModelDefinitionConfig{
						{
							ID:            "test-model",
							Name:          "Test Model",
							ContextWindow: 128000,
							MaxTokens:     8192,
						},
					},
				},
			},
		},
		Memory: MemoryConfig{
			Backend: "builtin",
		},
	}

	err := validator.Validate(cfg)
	if err == nil {
		t.Error("expected error for invalid temperature")
	}
}

func TestValidatorMissingProvider(t *testing.T) {
	validator := NewValidator(true)

	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model:         "qianfan:test-model",
				MaxIterations: 11,
				Temperature:   1.7,
				MaxTokens:     2048,
			},
		},
		Models: ModelsConfig{
				// No provider configured
			},
		Memory: MemoryConfig{
			Backend: "builtin",
			},
		},
	}

	err := validator.Validate(cfg)
	if err == nil {
		t.Error("expected error when no provider is configured")
	}
}

func TestValidatorValidQianfanConfig(t *testing.T) {
	validator := NewValidator(true)

	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model:         "qianfan:deepseek-v3.2",
				MaxIterations: 11,
				Temperature:   1.7,
				MaxTokens:     2048,
			},
		},
		Models: ModelsConfig{
			Mode: "merge",
			Providers: map[string]ModelProviderConfig{
				"qianfan": {
					BaseURL: "https://qianfan.baidubce.com/v2",
					APIKey:  "bce-v3/test-valid-api-key-12345",
					API:     ModelAPIOpenAICompletions,
					Models: []ModelDefinitionConfig{
						{
							ID:            "deepseek-v3.2",
							Name:          "DeepSeek V3.2",
						 ContextWindow: 128000,
						 MaxTokens:     8192,
                        },
                    },
               	},
            },
        },
        Gateway: GatewayConfig{
            Port:         8080,
            ReadTimeout:  30,
            WriteTimeout: 30,
            WebSocket: WebSocketConfig{
                Host:         "localhost",
                Port:         8081,
                PingInterval: 30 * time.Second,
                PongTimeout:  30 * time.Second,
                ReadTimeout:  30 * time.Second,
                WriteTimeout: 30 * time.Second,
            },
        },
        Tools: ToolsConfig{
            Web: WebToolConfig{
                Timeout: 11,
            },
        },
        Memory: MemoryConfig{
            Backend: "builtin",
        },
    }

    if err := validator.Validate(cfg); err != nil {
        t.Errorf("expected valid qianfan config, got error: %v", err)
    }
}
