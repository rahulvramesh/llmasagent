package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
		// Fallback to a default model if not specified, or make it an error
		modelName = "gryphe/mythomax-l2-13b" // Or return an error
	}
	return &OpenRouterProvider{
		apiKey:    apiKey,
		modelName: modelName,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Sensible timeout
		},
	}, nil
}

// Define request and response structs for OpenRouter API
type OpenRouterRequest struct {
	Model    string          `json:"model"`
	Messages []Message       `json:"messages"`
	// Add other parameters like temperature, max_tokens if needed
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type APIError struct {
    Message string `json:"message"`
    Type    string `json:"type"`
    Param   string `json:"param"`
    Code    string `json:"code"`
}


func (p *OpenRouterProvider) GetResponse(prompt string) (string, error) {
	if prompt == "" {
		return "", errors.New("prompt cannot be empty")
	}

	requestPayload := OpenRouterRequest{
		Model: p.modelName,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}

	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenRouter request: %w", err)
	}

	req, err := http.NewRequest("POST", openRouterAPIURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenRouter request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	// OpenRouter documentation mentions HTTP-Referer, can be your site or a dummy value
	req.Header.Set("HTTP-Referer", "http://localhost/llmagent") // Replace with your actual site if applicable
	// req.Header.Set("X-Title", "LLMAgent") // Optional: Helpful for OpenRouter to identify your app


	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenRouter response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
        // Try to parse the error from OpenRouter
        var apiErr OpenRouterResponse
        if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Error != nil {
             return "", fmt.Errorf("OpenRouter API error (Status %d): %s (Type: %s)", resp.StatusCode, apiErr.Error.Message, apiErr.Error.Type)
        }
		return "", fmt.Errorf("OpenRouter API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(bodyBytes, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal OpenRouter response: %w. Body: %s", err, string(bodyBytes))
	}

	if len(openRouterResp.Choices) == 0 || openRouterResp.Choices[0].Message.Content == "" {
		if openRouterResp.Error != nil {
             return "", fmt.Errorf("OpenRouter API returned an error: %s", openRouterResp.Error.Message)
        }
		return "", errors.New("received empty response or choices from OpenRouter")
	}

	return openRouterResp.Choices[0].Message.Content, nil
}
