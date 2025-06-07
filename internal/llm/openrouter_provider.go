package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"

type OpenRouterProvider struct {
	apiKey     string
	modelName  string
	httpClient *http.Client
}

func NewOpenRouterProvider(apiKey, modelName string) (*OpenRouterProvider, error) {
	if apiKey == "" {
		return nil, errors.New("OpenRouter API key cannot be empty")
	}
	if modelName == "" {
		modelName = "gryphe/mythomax-l2-13b" // Default model
	}
	return &OpenRouterProvider{
		apiKey:    apiKey,
		modelName: modelName,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for streaming
		},
	}, nil
}

// Renamed from Message to avoid conflict with new llm.Message
type RequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterRequest struct {
	Model    string           `json:"model"`
	Messages []RequestMessage `json:"messages"`
	Stream   bool             `json:"stream,omitempty"`
}

// OpenRouterStreamChoice represents a single choice in a streamed SSE event.
type OpenRouterStreamChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"` // Pointer to distinguish between null and empty string
}

// OpenRouterStreamResponse represents an SSE event from OpenRouter.
type OpenRouterStreamResponse struct {
	ID      string                   `json:"id"`
	Choices []OpenRouterStreamChoice `json:"choices"`
	Error   *APIError                `json:"error,omitempty"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

func (p *OpenRouterProvider) GetResponseStream(prompt string, streamChan chan<- Message) error {
	if prompt == "" {
		// Send error to channel for consistency, though this is a setup error
		// streamChan <- Message{Error: errors.New("prompt cannot be empty"), IsLast: true}
		return errors.New("prompt cannot be empty")
	}

	requestPayload := OpenRouterRequest{
		Model: p.modelName,
		Messages: []RequestMessage{
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal OpenRouter request: %w", err)
	}

	req, err := http.NewRequest("POST", openRouterAPIURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create OpenRouter request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost/llmagent")
	// req.Header.Set("X-Title", "LLMAgent")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to OpenRouter: %w", err)
	}
	// No defer resp.Body.Close() here, as we need the body open for the go routine.
	// It will be closed in the goroutine.

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close() // Close body if there's an early error return
		var apiErrResp OpenRouterStreamResponse // Use stream response for error structure
		if json.Unmarshal(bodyBytes, &apiErrResp) == nil && apiErrResp.Error != nil {
			return fmt.Errorf("OpenRouter API error (Status %d): %s (Type: %s)", resp.StatusCode, apiErrResp.Error.Message, apiErrResp.Error.Type)
		}
		return fmt.Errorf("OpenRouter API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Process the stream in a goroutine to allow the main function to return
	go func() {
		defer resp.Body.Close()
		defer func() {
			// Ensure a final message is sent if not already by finish reason
			// This handles cases like EOF or loop break without explicit IsLast
			// recover from panic if channel is closed
			 _ = recover()
			// streamChan <- Message{IsLast: true} // This might send a duplicate IsLast
		}()


		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					streamChan <- Message{IsLast: true} // EOF means stream ended
				} else {
					streamChan <- Message{Error: fmt.Errorf("failed to read stream from OpenRouter: %w", err), IsLast: true}
				}
				return // Exit goroutine
			}

			trimmedLine := strings.TrimSpace(string(line))
			if strings.HasPrefix(trimmedLine, "data: ") {
				jsonData := strings.TrimPrefix(trimmedLine, "data: ")
				if jsonData == "[DONE]" {
					streamChan <- Message{IsLast: true}
					return // Exit goroutine
				}

				var streamResp OpenRouterStreamResponse
				if err := json.Unmarshal([]byte(jsonData), &streamResp); err != nil {
					streamChan <- Message{Error: fmt.Errorf("failed to unmarshal stream data from OpenRouter '%s': %w", jsonData, err), IsLast: true}
					return // Exit goroutine
				}

				if streamResp.Error != nil {
					streamChan <- Message{Error: fmt.Errorf("OpenRouter stream error: %s (Type: %s)", streamResp.Error.Message, streamResp.Error.Type), IsLast: true}
					return // Exit goroutine
				}

				for _, choice := range streamResp.Choices {
					if choice.Delta.Content != "" {
						streamChan <- Message{Content: choice.Delta.Content}
					}
					if choice.FinishReason != nil && *choice.FinishReason != "" {
						streamChan <- Message{IsLast: true}
						return // Exit goroutine, as stream is finished
					}
				}
			}
		}
	}()

	return nil
}
