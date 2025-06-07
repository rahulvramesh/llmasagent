package server

import (
	"fmt"
	"llmasagent/internal/llm" // Adjust import path
	"log"
	"net/http"
)

func StartServer(provider llm.LLMProvider, port string) {
	http.HandleFunc("/mcp", mcpHandler(provider)) // Pass the provider to the handler

	addr := ":" + port
	// fmt.Printf("Starting MCP server on port %s...\n", port) // Old message
	fmt.Printf("Starting MCP server on http://localhost%s/mcp \n", addr) // More user-friendly
	log.Printf("MCP Server listening on %s", addr) // For structured logging
	log.Fatal(http.ListenAndServe(addr, nil))
}
