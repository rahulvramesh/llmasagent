package llm

type LLMProvider interface {
	GetResponse(prompt string) (string, error)
}
