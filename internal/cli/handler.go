package cli

import (
	"fmt"
	"llmasagent/internal/llm" // Make sure this import path matches your go module name
)

func HandleSingleProblem(problemDesc string, provider llm.LLMProvider) {
	if problemDesc == "" {
		fmt.Println("Error: Problem description cannot be empty.")
		return
	}

	fmt.Printf("Processing problem: %s\n", problemDesc)
	response, err := provider.GetResponse(problemDesc)
	if err != nil {
		fmt.Printf("Error getting response from LLM: %v\n", err)
		return
	}
	fmt.Println("LLM Response:")
	fmt.Println(response)
}
