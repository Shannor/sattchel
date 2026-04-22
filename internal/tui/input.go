package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type InputModel struct {
	textInput textinput.Model
	err       error
	quitting  bool
	header    string
	footer    string
}

func (m InputModel) Value() string {
	return m.textInput.Value()
}

type InputConfig struct {
	Placeholder string
	CharLimit   int
	header      string
	footer      string
}

func NewTextPrompt(c InputConfig) InputModel {

	ti := textinput.New()
	ti.SetVirtualCursor(false)
	ti.SetWidth(20)
	ti.Focus()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 300

	if c.Placeholder != "" {
		ti.Placeholder = c.Placeholder
	}

	if c.CharLimit != 0 {
		ti.CharLimit = c.CharLimit
	}

	return InputModel{
		textInput: ti,
	}
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m InputModel) View() tea.View {
	var c *tea.Cursor
	if !m.textInput.VirtualCursor() {
		c = m.textInput.Cursor()
		if c != nil {
			c.Y += lipgloss.Height(m.headerView())
		}
	}

	str := lipgloss.JoinVertical(lipgloss.Top, m.headerView(), m.textInput.View(), m.footerView())
	if m.quitting {
		str += "\n"
	}

	v := tea.NewView(str)
	v.Cursor = c
	return v
}

func (m InputModel) headerView() string {
	if m.header != "" {
		return m.header
	}
	return "Please provide a value.\n"
}

func (m InputModel) footerView() string {
	if m.footer != "" {
		return m.footer
	}
	return "\n(esc to quit) (enter to save)"
}
