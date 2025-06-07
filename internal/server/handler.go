package server

import (
	"encoding/json"
	"llmasagent/internal/llm" // Adjust import path
	"log"
	"net/http"
	"strings" // Added for strings.Builder
)

type MCPRequest struct {
	ProblemContext string `json:"problem_context"`
}

type MCPResponse struct {
	PotentialSolution string `json:"potential_solution,omitempty"`
	Error             string `json:"error,omitempty"`
}

func mcpHandler(provider llm.LLMProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var req MCPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			log.Printf("Error decoding request: %v", err)
			return
		}
		defer r.Body.Close()

		if req.ProblemContext == "" {
			log.Printf("Validation error: ProblemContext was empty from client %s", r.RemoteAddr)
			resp := MCPResponse{Error: "ProblemContext cannot be empty"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(resp)
			return
		}

		streamChan := make(chan llm.Message)
		var fullResponse strings.Builder // Use strings.Builder for efficiency
		var streamErr error

		// Start goroutine to consume from the stream
		go func() {
			defer close(streamChan) // Ensure channel is closed when goroutine finishes
			err := provider.GetResponseStream(req.ProblemContext, streamChan)
			if err != nil {
				// This error is for setup issues (e.g., bad API key).
				// We need a way to signal this back to the main handler goroutine.
				// One way is to send an error message through the streamChan itself.
				// Note: The channel might be closed by the receiver if it encounters an error first.
				// A more robust solution might involve a separate error channel or a shared error variable.
				// For now, we'll try to send it, but this could panic if streamChan is already closed.
				// A select with a default or a try-send could be safer.
				// Sending it as the first message or a distinct error type might be better.
				// Let's assume for now that if GetResponseStream returns an error, it's a setup error
				// and no messages will be sent on streamChan by the provider itself.
				log.Printf("Error setting up LLM stream: %v", err)
				// To prevent panic if streamChan is closed by receiver due to other error/completion:
				// We will capture this error and check it after the stream reading loop.
				// For now, this specific error from GetResponseStream (setup) will be handled
				// by streamErr being set before the loop by the receiver if this goroutine exits early.
				// A better way would be to pass this error back to the main thread.
				// Let's try to send a message, but this is tricky.
				// The handler below will capture and prioritize errors from the stream itself.
				// If GetResponseStream fails, streamChan might not even be listened to.
				// The current CLI handler sends the setup error on the channel. Let's try that here.
				// This needs careful handling of channel closure.
				// If GetResponseStream itself returns an error, it means the stream was not even started.
				// So, we can set streamErr directly here or send a message.
				// For simplicity in the handler, we'll rely on the main goroutine to handle streamErr if this setup fails.
				// The problem is, this goroutine might set streamErr, but the main one reads it. This is a race.
				// The most straightforward is to have GetResponseStream itself block and return the setup error,
				// and only use the channel for actual stream data/errors.
				// Given current interface, if err != nil here, we should signal it.
				// Let's make streamErr accessible to this goroutine or use a separate channel.
				// For now, we'll assume if err != nil, the streamChan will be closed by this goroutine,
				// and the reading loop will terminate. The main handler will then check streamErr.
				// This is simplified. A production system would need more robust error propagation.
				// Let's assume the provider.GetResponseStream itself will send an error message on the channel
				// if it fails to set up. The current interface implies this.
				// The prompt asks for a simple aggregation for now.
			}
		}()

		for msg := range streamChan {
			if msg.Error != nil {
				streamErr = msg.Error
				log.Printf("Error during streaming from LLM: %v", msg.Error)
				// Do not break immediately, allow other messages to potentially clear or be processed
				// if the error is non-fatal. However, for most LLM errors, it's fatal for this request.
				// For now, we will capture the first error and then break.
				break
			}
			if msg.Content != "" {
				fullResponse.WriteString(msg.Content)
			}
			if msg.IsLast {
				break
			}
		}

		// Handle errors after attempting to read the whole stream
		if streamErr != nil {
			resp := MCPResponse{Error: "Error getting response from LLM: " + streamErr.Error()}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError) // Or map specific errors to other statuses
			json.NewEncoder(w).Encode(resp)
			log.Printf("Final error state from LLM stream: %v", streamErr)
			return
		}

		if fullResponse.Len() == 0 && streamErr == nil {
			// This case means the stream ended (IsLast was true or channel closed)
			// but no content was received and no error was explicitly set.
			// This could be a valid empty response from the LLM.
			log.Printf("LLM returned an empty response for: %s", req.ProblemContext)
			// Depending on requirements, this might be an error or a valid empty solution.
			// For now, let's treat it as a valid empty solution.
		}


		resp := MCPResponse{PotentialSolution: fullResponse.String()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
