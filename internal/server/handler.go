package server

import (
	"encoding/json"
	"llmasagent/internal/llm" // Adjust import path
	"log"
	"net/http"
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

		llmResponse, err := provider.GetResponse(req.ProblemContext)
		if err != nil {
			resp := MCPResponse{Error: "Error getting response from LLM: " + err.Error()}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			log.Printf("Error from LLM provider: %v", err)
			return
		}

		resp := MCPResponse{PotentialSolution: llmResponse}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
