package main

import (
	"testing"
)

func TestGetCredential(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		envValue  string
		expected  string
	}{
		{
			name:      "flag takes precedence",
			flagValue: "flag-value",
			envValue:  "env-value",
			expected:  "flag-value",
		},
		{
			name:      "env value when flag empty",
			flagValue: "",
			envValue:  "env-value",
			expected:  "env-value",
		},
		{
			name:      "empty when both empty",
			flagValue: "",
			envValue:  "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock environment variable
			if tt.envValue != "" {
				t.Setenv("TEST_ENV_VAR", tt.envValue)
			}

			result := getCredential(tt.flagValue, "TEST_ENV_VAR")
			if result != tt.expected {
				t.Errorf("getCredential() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		PushGatewayURL: "http://pushgateway:9091",
		PushGatewayJob: "azure-carbon-exporter",
	}

	if config.PushGatewayURL != "http://pushgateway:9091" {
		t.Errorf("Expected PushGatewayURL to be http://pushgateway:9091, got %s", config.PushGatewayURL)
	}

	if config.PushGatewayJob != "azure-carbon-exporter" {
		t.Errorf("Expected PushGatewayJob to be azure-carbon-exporter, got %s", config.PushGatewayJob)
	}
}
