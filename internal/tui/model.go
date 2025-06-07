package tui

import (
	"fmt"
	"llmasagent/internal/llm" // Adjust import path
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss" // For styling
)

// llmStreamChunkMsg is sent for each piece of content from the LLM stream.
type llmStreamChunkMsg struct {
	chunk llm.Message // Contains Content, IsLast, Error
}

// streamDoneMsg is a sentinel message to indicate the LLM stream goroutine has finished.
type streamDoneMsg struct{}

type model struct {
	viewport           viewport.Model
	messages           []string
	textInput          textinput.Model
	llmProvider        llm.LLMProvider
	err                error
	ready              bool // To handle viewport initialization
	senderStyle        lipgloss.Style
	botStyle           lipgloss.Style
	llmStreamChannel   chan llm.Message // Channel for receiving messages from LLM
	streamingInProgress bool             // True if LLM response is currently streaming
	currentLLMResponse string           // Accumulates LLM response chunks
}

func NewTUIModel(provider llm.LLMProvider) model {
	ti := textinput.New()
	ti.Placeholder = "Ask something..."
	ti.Focus()
	ti.CharLimit = 250
	ti.Width = 50 // Initial width, will be adjusted

	// Viewport will be initialized in Init/Update once dimensions are known
	return model{
		textInput:   ti,
		messages:    []string{},
		llmProvider: provider,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // Purple for user
		botStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),  // Cyan for bot
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink // Start the text input blinking
}

// type llmResponseMsg string // Deleted: No longer used
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// listenForLLMStreamCmd is a helper command to listen for messages on the LLM stream channel.
func listenForLLMStreamCmd(ch chan llm.Message) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-ch:
			if !ok { // Channel has been closed
				return streamDoneMsg{}
			}
			return llmStreamChunkMsg{chunk: msg}
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			userInput := strings.TrimSpace(m.textInput.Value())
			// The original if userInput == "" check is now covered by the combined condition below
			if userInput == "" || m.streamingInProgress { // Prevent new requests if already streaming
				return m, nil
			}

			m.messages = append(m.messages, m.senderStyle.Render("You: ")+userInput)
			m.messages = append(m.messages, m.botStyle.Render("LLM: ")) // Add placeholder
			m.currentLLMResponse = ""
			m.streamingInProgress = true
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()
			m.textInput.SetValue("")

			m.llmStreamChannel = make(chan llm.Message) // Create the channel

			// Command to start the goroutine that calls the LLM provider
			initCmd := func() tea.Msg {
				// This goroutine will block until GetResponseStream is done.
				// GetResponseStream is responsible for sending messages (data, errors, IsLast)
				// to m.llmStreamChannel and then closing its side of the channel when done.
				// The provider's implementation of GetResponseStream should handle closing the channel.
				go m.llmProvider.GetResponseStream(userInput, m.llmStreamChannel)
				return nil // This command itself sends no message
			}

			return m, tea.Batch(tiCmd, vpCmd, initCmd, listenForLLMStreamCmd(m.llmStreamChannel))
		}

	case llmStreamChunkMsg:
		chunk := msg.chunk

		if chunk.Error != nil {
			// Replace the last message (LLM placeholder) with the error
			if len(m.messages) > 0 {
				m.messages[len(m.messages)-1] = m.botStyle.Render("LLM: Error: ") + chunk.Error.Error()
			}
			m.streamingInProgress = false
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()
			// Ensure channel is drained and closed if error occurs, though provider should close it.
			// We might close our reference to it or rely on streamDoneMsg.
			// For now, just stop listening.
			return m, vpCmd // No further listening
		}

		m.currentLLMResponse += chunk.Content
		if len(m.messages) > 0 {
			m.messages[len(m.messages)-1] = m.botStyle.Render("LLM: ") + m.currentLLMResponse
		}

		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()

		if chunk.IsLast {
			m.streamingInProgress = false
			return m, vpCmd // Stop listening, stream is finished
		}
		// Continue listening for more chunks
		return m, tea.Batch(vpCmd, listenForLLMStreamCmd(m.llmStreamChannel))

	case streamDoneMsg:
		m.streamingInProgress = false
		// This message indicates the llmStreamChannel was closed by the sender (provider).
		// Final UI update might be needed if currentLLMResponse is empty but placeholder was shown.
		if m.currentLLMResponse == "" && len(m.messages) > 0 {
			// Check if the last message is still the placeholder
			if strings.HasSuffix(m.messages[len(m.messages)-1], m.botStyle.Render("LLM: ")) && len(m.messages[len(m.messages)-1]) == len(m.botStyle.Render("LLM: ")) {
				m.messages[len(m.messages)-1] = m.botStyle.Render("LLM: ") + "[No response or stream ended abruptly]"
				m.viewport.SetContent(strings.Join(m.messages, "\n"))
				m.viewport.GotoBottom()
			}
		}
		return m, vpCmd // No further commands

	case errMsg:
		m.err = msg
		// Display general errors not related to streaming directly, or if streaming setup failed in a way not caught by llmStreamChunkMsg
		m.messages = append(m.messages, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: "+msg.Error()))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		m.streamingInProgress = false // Ensure streaming is marked as stopped
		return m, tea.Batch(tiCmd, vpCmd)

	case tea.WindowSizeMsg:
		headerHeight := 0 // Adjust if you add a header
		footerHeight := lipgloss.Height(m.inputView())
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			// Initialize viewport now that we have the size.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.YPosition = headerHeight
			m.messages = append(m.messages, "Welcome to LLMAgent Chat! Type your message and press Enter.")
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}
		// Adjust text input width
		m.textInput.Width = msg.Width - 2 // Small padding
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return fmt.Sprintf(
		"%s\n%s",
		m.viewport.View(),
		m.inputView(),
	)
}

func (m model) inputView() string {
	return m.textInput.View()
}
