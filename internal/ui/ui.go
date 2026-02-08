package ui

import (
	"database/sql"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	viewSummary view = iota
	viewAccountForm
	viewTransactionForm
	viewTransactionList
	viewChart
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type Model struct {
	db              *sql.DB
	width           int
	height          int
	active          view
	summary         summaryModel
	accountForm     accountFormModel
	transactionForm transactionFormModel
	transactionList transactionListModel
	chart           chartModel
}

func New(db *sql.DB) Model {
	return Model{
		db:      db,
		active:  viewSummary,
		summary: newSummaryModel(db),
	}
}

func (m Model) Init() tea.Cmd {
	return m.summary.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case accountInsertedMsg, transactionInsertedMsg:
		m.active = viewSummary
		m.summary = newSummaryModel(m.db)
		return m, m.summary.Init()

	case chartClosedMsg:
		m.active = viewSummary
		m.summary = newSummaryModel(m.db)
		return m, m.summary.Init()

	case accountFormCancelledMsg, transactionFormCancelledMsg, transactionListCancelledMsg:
		m.active = viewSummary
		return m, nil
	}

	switch m.active {
	case viewSummary:
		return m.updateSummary(msg)
	case viewAccountForm:
		return m.updateAccountForm(msg)
	case viewTransactionForm:
		return m.updateTransactionForm(msg)
	case viewTransactionList:
		return m.updateTransactionList(msg)
	case viewChart:
		return m.updateChart(msg)
	}

	return m, nil
}

func (m Model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "a":
			m.active = viewAccountForm
			m.accountForm = newAccountFormModel(m.db)
			return m, m.accountForm.Init()
		case "t":
			m.active = viewTransactionForm
			m.transactionForm = newTransactionFormModel(m.db)
			return m, m.transactionForm.Init()
		case "l":
			m.active = viewTransactionList
			m.transactionList = newTransactionListModel(m.db)
			return m, m.transactionList.Init()
		case "c":
			m.active = viewChart
			m.chart = newChartModel(m.db, m.width, m.height)
			return m, m.chart.Init()
		}
	}

	var cmd tea.Cmd
	m.summary, cmd = m.summary.Update(msg)
	return m, cmd
}

func (m Model) updateAccountForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.accountForm, cmd = m.accountForm.Update(msg)
	return m, cmd
}

func (m Model) updateTransactionForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.transactionForm, cmd = m.transactionForm.Update(msg)
	return m, cmd
}

func (m Model) updateTransactionList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.transactionList, cmd = m.transactionList.Update(msg)
	return m, cmd
}

func (m Model) updateChart(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.chart, cmd = m.chart.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := titleStyle.Render("budgie")

	var content string
	var help string

	switch m.active {
	case viewSummary:
		content = m.summary.View()
		help = helpStyle.Render("a: add account  t: add transaction  l: transactions  c: chart  q: quit")
	case viewAccountForm:
		content = m.accountForm.View()
		help = ""
	case viewTransactionForm:
		content = m.transactionForm.View()
		help = ""
	case viewTransactionList:
		content = m.transactionList.View()
		help = helpStyle.Render("esc: back  ↑/↓: scroll  ←/→: page")
	case viewChart:
		content = m.chart.View()
		help = helpStyle.Render("esc: back")
	}

	return title + "\n\n" + content + "\n\n" + help + "\n"
}
