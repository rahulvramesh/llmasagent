package cli

import (
	"fmt"
	"llmasagent/internal/llm" // Make sure this import path matches your go module name
	"strings"
)

func HandleSingleProblem(problemDesc string, provider llm.LLMProvider) {
	if problemDesc == "" {
		fmt.Println("Error: Problem description cannot be empty.")
		return
	}

	fmt.Printf("Processing problem: %s\n", problemDesc) // Use \n for actual newline in subtask

	streamChan := make(chan llm.Message)
	var fullResponse strings.Builder
	var streamErr error

	go func() {
		// The provider's GetResponseStream is responsible for closing the streamChan.
		// No need to close it here in this goroutine.
		err := provider.GetResponseStream(problemDesc, streamChan)
		if err != nil {
			// This error is for setup issues, actual stream errors come via channel
			// We'll send it via the channel to simplify error handling in the main loop.
			// However, it's also useful to log it here if the channel send blocks or fails.
			fmt.Printf("Error setting up stream: %v\n", err)
			// Attempt to send the setup error through the channel if possible.
			// This might not always work if the channel is not yet being listened to,
			// or if the goroutine scheduling doesn't allow it.
			select {
			case streamChan <- llm.Message{Error: err, IsLast: true}:
			default:
				// Fallback if channel send fails (e.g., not listened to yet)
				// Store it to be checked by the main thread.
				streamErr = err
			}
		}
	}()

	for msg := range streamChan {
		if msg.Error != nil {
			fmt.Printf("\nError during streaming: %v\n", msg.Error)
			streamErr = msg.Error // Capture the first error encountered
			break
		}
		if msg.Content != "" {
			fmt.Print(msg.Content) // Print content as it arrives for CLI
			fullResponse.WriteString(msg.Content)
		}
		if msg.IsLast {
			break
		}
	}
	fmt.Println() // Ensure a newline after streaming is done or if it was interrupted.

	// Check for setup error that might not have come through the channel
	if streamErr != nil && fullResponse.Len() == 0 { // Prioritize errors from stream if content was received
		fmt.Printf("\nError getting response from LLM: %v\n", streamErr)
		return
	}

	// If we received content, it's often useful to see it even if an error occurred later in the stream.
	// The error is already printed. We can decide if we want to print the partial response or not.
	// For now, we will print the aggregated response if any content was received.

	if fullResponse.Len() > 0 {
		fmt.Println("\nLLM Response (fully aggregated):")
		fmt.Println(fullResponse.String())
	} else if streamErr == nil {
		// No content and no error, could mean an empty stream or logic error.
		fmt.Println("\nLLM response was empty.")
	}
}
