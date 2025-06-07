package config

import (
	"os"
	// "strconv" // For port conversion if needed, though string is fine for ListenAndServe
)

type Config struct {
	LLMProviderType  string
	OpenRouterAPIKey string
	OpenRouterModel  string
	MCPServerPort    string
}

func LoadConfig() Config {
	return Config{
		LLMProviderType:  getEnv("LLMAGENT_LLM_PROVIDER_TYPE", "mock"),
		OpenRouterAPIKey: getEnv("LLMAGENT_OPENROUTER_API_KEY", ""),
		OpenRouterModel:  getEnv("LLMAGENT_OPENROUTER_MODEL", "gryphe/mythomax-l2-13b"), // Default model
		MCPServerPort:    getEnv("LLMAGENT_MCP_SERVER_PORT", "8080"),
	}
}

// Helper function to get environment variables or return a default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
