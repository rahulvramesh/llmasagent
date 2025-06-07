package tui

import (
	"fmt"
	"llmasagent/internal/llm" // Adjust import path
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func StartTUI(provider llm.LLMProvider) {
	model := NewTUIModel(provider)
	p := tea.NewProgram(model, tea.WithAltScreen()) // Use AltScreen for better TUI experience

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}
	fmt.Println("TUI finished.")
}
