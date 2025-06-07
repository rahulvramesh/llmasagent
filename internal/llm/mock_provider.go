package llm

import (
	"fmt"
	"time"
)

// MockLLMProvider is a mock implementation of the LLMProvider interface.
type MockLLMProvider struct {
	// MockResponses can be used to queue up specific responses for testing.
	MockResponses []Message
	// ResponseDelay simulates network latency.
	ResponseDelay time.Duration
}

// NewMockLLMProvider creates a new MockLLMProvider.
func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		ResponseDelay: 100 * time.Millisecond, // Default delay
	}
}

// SetMockResponses allows setting a predefined sequence of messages for the stream.
func (m *MockLLMProvider) SetMockResponses(messages []Message) {
	m.MockResponses = messages
}

// GetResponseStream simulates streaming by sending predefined or default messages.
func (m *MockLLMProvider) GetResponseStream(prompt string, streamChan chan<- Message) error {
	go func() {
		defer close(streamChan)

		if len(m.MockResponses) > 0 {
			// Send predefined responses
			for _, msg := range m.MockResponses {
				streamChan <- msg
				if m.ResponseDelay > 0 {
					time.Sleep(m.ResponseDelay)
				}
				if msg.IsLast {
					return
				}
			}
			// Ensure IsLast is sent if the loop finishes without it
			streamChan <- Message{IsLast: true}
			return
		}

		// Default mock behavior if no specific responses are set
		defaultMessages := []Message{
			{Content: fmt.Sprintf("Mock response part 1 for prompt: '%s'\n", prompt)},
			{Content: "Mock response part 2: some details.\n"},
			{Content: "Mock response part 3: concluding thoughts.\n", IsLast: true},
		}

		for _, msg := range defaultMessages {
			streamChan <- msg
			if m.ResponseDelay > 0 {
				time.Sleep(m.ResponseDelay)
			}
			if msg.IsLast {
				return
			}
		}
		// Fallback, though defaultMessages should end with IsLast: true
		streamChan <- Message{IsLast: true}
	}()
	return nil
}
