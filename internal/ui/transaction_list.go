package ui

import (
	"database/sql"
	"fmt"

	"budgie/internal/db"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const pageSize = 20

type transactionListDataMsg struct {
	rows  []db.TransactionRow
	total int
	err   error
}

type transactionListCancelledMsg struct{}

type transactionListModel struct {
	database *sql.DB
	table    table.Model
	loaded   bool
	err      error
	page     int
	total    int
}

func newTransactionListModel(database *sql.DB) transactionListModel {
	cols := []table.Column{
		{Title: "Date", Width: 10},
		{Title: "Account", Width: 16},
		{Title: "Description", Width: 24},
		{Title: "Category", Width: 14},
		{Title: "Amount", Width: 12},
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(pageSize),
	)
	t.SetStyles(s)

	return transactionListModel{
		database: database,
		table:    t,
	}
}

func (m transactionListModel) Init() tea.Cmd {
	return m.fetchPage()
}

func (m transactionListModel) fetchPage() tea.Cmd {
	offset := m.page * pageSize
	return func() tea.Msg {
		rows, total, err := db.AllTransactions(m.database, offset, pageSize)
		return transactionListDataMsg{rows: rows, total: total, err: err}
	}
}

func (m transactionListModel) totalPages() int {
	if m.total == 0 {
		return 1
	}
	return (m.total + pageSize - 1) / pageSize
}

func (m transactionListModel) Update(msg tea.Msg) (transactionListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case transactionListDataMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.total = msg.total
		rows := make([]table.Row, len(msg.rows))
		for i, tx := range msg.rows {
			rows[i] = table.Row{
				formatDate(tx.Date),
				tx.Account,
				tx.Description,
				tx.Category,
				formatSignedPence(tx.Amount),
			}
		}
		m.table.SetRows(rows)
		m.loaded = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return transactionListCancelledMsg{} }
		case "right", "n":
			if m.page < m.totalPages()-1 {
				m.page++
				m.loaded = false
				return m, m.fetchPage()
			}
			return m, nil
		case "left", "p":
			if m.page > 0 {
				m.page--
				m.loaded = false
				return m, m.fetchPage()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m transactionListModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading transactions: %v", m.err)
	}
	if !m.loaded {
		return "Loading..."
	}

	pager := fmt.Sprintf("Page %d of %d (%d transactions)", m.page+1, m.totalPages(), m.total)

	return sectionTitle.Render("All Transactions") + "\n\n" +
		m.table.View() + "\n\n" +
		helpStyle.Render(pager)
}
