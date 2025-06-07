package llm

import "fmt"

type MockLLMProvider struct{}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{}
}

func (m *MockLLMProvider) GetResponse(prompt string) (string, error) {
	// For now, just echo the prompt or return a fixed response.
	response := fmt.Sprintf("Mock response for prompt: '%s'", prompt)
	return response, nil
}
