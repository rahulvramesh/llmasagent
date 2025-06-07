package llm

// Message represents a piece of a streamed response or a control signal.
type Message struct {
	Content string // Text content of the chunk
	IsLast  bool   // True if this is the last message in the stream
	Error   error  // If an error occurred during streaming for this chunk
}

type LLMProvider interface {
	// GetResponseStream sends response chunks to the provided streamChan.
	// It should send a Message with IsLast=true to indicate the end of the stream.
	// If an error occurs that stops the stream, it can be sent as a Message with Error set.
	// The function itself returns an error for setup issues (e.g., bad API key).
	GetResponseStream(prompt string, streamChan chan<- Message) error
}
