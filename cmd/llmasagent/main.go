package main

import (
	"flag"
	"fmt"
	"llmasagent/internal/cli"
	"llmasagent/internal/llm"
	"llmasagent/internal/server"
	"llmasagent/internal/tui"
	"llmasagent/pkg/config" // Import the new config package
	"log"                   // For logging errors
	"os"
)

func main() {
	appConfig := config.LoadConfig()

	problemDesc := flag.String("problem", "", "Describe the problem for the LLM to solve.")
	chatMode := flag.Bool("chat", false, "Enter interactive chat mode.")
	serverMode := flag.Bool("server", false, "Start in MCP server mode.")

	flag.Parse()

	var llmService llm.LLMProvider
	var err error // Declare error variable

	switch appConfig.LLMProviderType {
	case "openrouter":
		if appConfig.OpenRouterAPIKey == "" {
			log.Fatalf("Error: LLM provider type is 'openrouter' but LLMAGENT_OPENROUTER_API_KEY is not set.")
		}
		llmService, err = llm.NewOpenRouterProvider(appConfig.OpenRouterAPIKey, appConfig.OpenRouterModel)
		if err != nil {
			log.Fatalf("Error initializing OpenRouter provider: %v", err)
		}
		fmt.Println("Using OpenRouter LLM provider.")
	case "mock":
		llmService = llm.NewMockLLMProvider()
		fmt.Println("Using Mock LLM provider.")
	default:
		log.Fatalf("Error: Unknown LLM provider type '%s'. Supported types are 'mock', 'openrouter'.", appConfig.LLMProviderType)
	}

	if *chatMode {
		tui.StartTUI(llmService)
	} else if *serverMode {
		server.StartServer(llmService, appConfig.MCPServerPort)
	} else if *problemDesc != "" {
		cli.HandleSingleProblem(*problemDesc, llmService)
	} else {
		fmt.Println("Usage: llmasagent [options]")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\nConfiguration (via environment variables):")
		fmt.Printf("  LLMAGENT_LLM_PROVIDER_TYPE (current: %s, options: 'mock', 'openrouter')\n", appConfig.LLMProviderType)
		fmt.Printf("  LLMAGENT_OPENROUTER_API_KEY (required if provider is 'openrouter')\n")
		fmt.Printf("  LLMAGENT_OPENROUTER_MODEL (current: %s, used if provider is 'openrouter')\n", appConfig.OpenRouterModel)
		fmt.Printf("  LLMAGENT_MCP_SERVER_PORT (current: %s)\n", appConfig.MCPServerPort)
		os.Exit(1)
	}
}
