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

type model struct {
	viewport      viewport.Model
	messages      []string
	textInput     textinput.Model
	llmProvider   llm.LLMProvider
	err           error
	ready         bool // To handle viewport initialization
	senderStyle   lipgloss.Style
	botStyle      lipgloss.Style
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

type llmResponseMsg string
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }


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
			if userInput == "" {
				return m, nil
			}

			m.messages = append(m.messages, m.senderStyle.Render("You: ")+userInput)
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()

			// Create a command to get LLM response
			cmd := func() tea.Msg {
				resp, err := m.llmProvider.GetResponse(userInput)
				if err != nil {
					return errMsg{err}
				}
				return llmResponseMsg(resp)
			}

			m.textInput.SetValue("") // Clear input field
			return m, tea.Batch(tiCmd, vpCmd, cmd)
		}

	case llmResponseMsg:
		m.messages = append(m.messages, m.botStyle.Render("LLM: ")+string(msg))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
        return m, tea.Batch(tiCmd, vpCmd)

    case errMsg:
        m.err = msg
        m.messages = append(m.messages, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: "+msg.Error()))
        m.viewport.SetContent(strings.Join(m.messages, "\n"))
        m.viewport.GotoBottom()
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
