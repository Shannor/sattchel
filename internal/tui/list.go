package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ListOption represents a single option in the selection list.
type ListOption struct {
	TitleStr       string
	DescriptionStr string
	ValueStr       string
}

// Title Implement the list.Item interface.
func (o ListOption) Title() string       { return o.TitleStr }
func (o ListOption) Description() string { return o.DescriptionStr }
func (o ListOption) FilterValue() string { return o.TitleStr + " " + o.DescriptionStr }

// ListSelectModel is the Bubble Tea model for selecting an item from a list.
type ListSelectModel struct {
	list     list.Model
	selected *ListOption
	quitting bool
}

// NewListSelect initializes and returns a ListSelectModel.
func NewListSelect(title string, options []ListOption) ListSelectModel {
	items := make([]list.Item, len(options))
	for i, o := range options {
		items[i] = o
	}

	// Create default delegate (handles multiline Title + Description rendering)
	delegate := list.NewDefaultDelegate()

	// Customize delegate styles using Sattchel TUI styles
	s := AutoStyles()

	// Active/Selected styles
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(s.Success.GetForeground()).
		BorderForeground(s.Success.GetForeground())
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(s.Info.GetForeground()).
		BorderForeground(s.Info.GetForeground())

	// Normal/Unselected styles
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(s.Text.GetForeground())
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(s.Muted.GetForeground())

	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = s.Title

	// Configure initial sizing
	l.SetSize(80, 20)

	return ListSelectModel{
		list: l,
	}
}

// Init initializes the bubble tea model.
func (m ListSelectModel) Init() tea.Cmd {
	return nil
}

// Update handles message updates.
func (m ListSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Handle window resizing dynamically
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyPressMsg:
		// Only trigger selection/exit keys if we're not currently typing in the search/filter box
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(ListOption); ok {
				m.selected = &item
			}
			m.quitting = true
			return m, tea.Quit

		case "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the list view.
func (m ListSelectModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	return tea.NewView(docStyle.Render(m.list.View()))
}

// Selected returns the selected ListOption, or nil if none was selected or user canceled.
func (m ListSelectModel) Selected() *ListOption {
	return m.selected
}

// docStyle defines the outer margins/padding of the list UI.
var docStyle = lipgloss.NewStyle().Margin(1, 2)
