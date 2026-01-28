package ui

import (
	"database/sql"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type Model struct {
	db      *sql.DB
	width   int
	height  int
	summary summaryModel
}

func New(db *sql.DB) Model {
	return Model{
		db:      db,
		summary: newSummaryModel(db),
	}
}

func (m Model) Init() tea.Cmd {
	return m.summary.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	var cmd tea.Cmd
	m.summary, cmd = m.summary.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := titleStyle.Render("budgie")
	help := helpStyle.Render("q: quit")
	return title + "\n\n" + m.summary.View() + "\n\n" + help + "\n"
}
