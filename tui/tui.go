package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sah-Abhishek/mockSmith/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type view int

const (
	listView view = iota
	addView
)

type Model struct {
	config     *config.Config
	updateChan chan<- *config.Config
	view       view
	cursor     int
	inputs     []textinput.Model
	focusIndex int
	err        string
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1)

	listStyle = lipgloss.NewStyle().
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1)
)

func NewModel(cfg *config.Config, updateChan chan<- *config.Config) Model {
	m := Model{
		config:     cfg,
		updateChan: updateChan,
		view:       listView,
		inputs:     make([]textinput.Model, 4),
	}

	// Setup input fields
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "GET"
	m.inputs[0].Focus()
	m.inputs[0].CharLimit = 10
	m.inputs[0].Width = 20

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "/api/users"
	m.inputs[1].CharLimit = 100
	m.inputs[1].Width = 40

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "200"
	m.inputs[2].CharLimit = 3
	m.inputs[2].Width = 10

	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = `{"message": "success"}`
	m.inputs[3].CharLimit = 1000
	m.inputs[3].Width = 60

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.view == listView {
				return m, tea.Quit
			}
			// In add view, 'q' goes back to list
			m.view = listView
			m.err = ""
			return m, nil

		case "a":
			if m.view == listView {
				m.view = addView
				m.focusIndex = 0
				m.err = ""
				// Reset inputs
				for i := range m.inputs {
					m.inputs[i].SetValue("")
				}
				m.inputs[0].Focus()
				return m, nil
			}

		case "d":
			if m.view == listView && len(m.config.GetEndpoints()) > 0 {
				endpoints := m.config.GetEndpoints()
				if m.cursor < len(endpoints) {
					m.config.RemoveEndpoint(endpoints[m.cursor].ID)
					if m.cursor >= len(m.config.GetEndpoints()) && m.cursor > 0 {
						m.cursor--
					}
					m.updateChan <- m.config
				}
				return m, nil
			}

		case "up", "k":
			if m.view == listView && m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.view == listView && m.cursor < len(m.config.GetEndpoints())-1 {
				m.cursor++
			}

		case "tab", "shift+tab", "enter":
			if m.view == addView {
				s := msg.String()

				if s == "enter" {
					if m.focusIndex == len(m.inputs) {
						// Submit form
						if err := m.addEndpoint(); err != nil {
							m.err = err.Error()
							return m, nil
						}
						m.view = listView
						m.err = ""
						return m, nil
					}
					m.focusIndex++
				}

				if s == "shift+tab" {
					m.focusIndex--
				} else if s == "tab" {
					m.focusIndex++
				}

				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}

				cmds := make([]tea.Cmd, len(m.inputs))
				for i := 0; i < len(m.inputs); i++ {
					if i == m.focusIndex {
						cmds[i] = m.inputs[i].Focus()
					} else {
						m.inputs[i].Blur()
					}
				}
				return m, tea.Batch(cmds...)
			}
		}
	}

	// Update inputs if in add view
	if m.view == addView {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m *Model) addEndpoint() error {
	method := strings.ToUpper(m.inputs[0].Value())
	if method == "" {
		method = "GET"
	}

	path := m.inputs[1].Value()
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	statusCode := 200
	if m.inputs[2].Value() != "" {
		fmt.Sscanf(m.inputs[2].Value(), "%d", &statusCode)
	}

	response := m.inputs[3].Value()
	if response == "" {
		response = `{"message": "success"}`
	}

	// Validate JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(response), &js); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	endpoint := config.Endpoint{
		ID:         uuid.New().String(),
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Response:   js,
	}

	if err := m.config.AddEndpoint(endpoint); err != nil {
		return err
	}

	m.updateChan <- m.config
	return nil
}

func (m Model) View() string {
	if m.view == addView {
		return m.addView()
	}
	return m.listView()
}

func (m Model) listView() string {
	s := titleStyle.Render("🚀 Mock API Server")
	s += "\n\n"

	endpoints := m.config.GetEndpoints()
	if len(endpoints) == 0 {
		s += listStyle.Render("No endpoints yet. Press 'a' to add one!")
	} else {
		s += listStyle.Render("Endpoints:")
		for i, e := range endpoints {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			line := fmt.Sprintf("%s [%s] %s -> %d",
				cursor, e.Method, e.Path, e.StatusCode)

			if m.cursor == i {
				line = selectedStyle.Render(line)
			}
			s += "\n" + line
		}
	}

	s += "\n\n" + helpStyle.Render(
		"a: add • d: delete • ↑/↓: navigate • q: quit")

	return s
}

func (m Model) addView() string {
	s := titleStyle.Render("Add New Endpoint")
	s += "\n\n"

	labels := []string{"Method:", "Path:", "Status Code:", "Response JSON:"}
	for i := range m.inputs {
		s += labels[i] + "\n"
		s += m.inputs[i].View() + "\n\n"
	}

	button := "[ Submit ]"
	if m.focusIndex == len(m.inputs) {
		button = selectedStyle.Render(button)
	}
	s += button + "\n\n"

	if m.err != "" {
		s += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render("Error: "+m.err) + "\n\n"
	}

	s += helpStyle.Render("tab: next • enter: submit • q: cancel")
	return s
}
