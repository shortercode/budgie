package ui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"budgie/internal/db"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	sectionTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	tableHeader = lipgloss.NewStyle().
			Bold(true).
			Underline(true)

	totalRow = lipgloss.NewStyle().
			Bold(true)

	incomeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	expenseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	netPositive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10"))

	netNegative = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9"))
)

// summaryData holds everything the dashboard needs to render.
type summaryData struct {
	accounts     []db.AccountBalance
	transactions []db.RecentTx
	monthly      db.MonthlySummary
}

// summaryDataMsg delivers fetched data to the model.
type summaryDataMsg struct {
	data summaryData
	err  error
}

// summaryModel is the dashboard view.
type summaryModel struct {
	database *sql.DB
	data     summaryData
	loaded   bool
	err      error
}

func newSummaryModel(database *sql.DB) summaryModel {
	return summaryModel{database: database}
}

func (m summaryModel) Init() tea.Cmd {
	return m.fetchData()
}

func (m summaryModel) fetchData() tea.Cmd {
	return func() tea.Msg {
		var d summaryData
		var err error

		d.accounts, err = db.AccountBalances(m.database)
		if err != nil {
			return summaryDataMsg{err: err}
		}

		d.transactions, err = db.RecentTransactions(m.database, 10)
		if err != nil {
			return summaryDataMsg{err: err}
		}

		now := time.Now()
		d.monthly, err = db.MonthSummary(m.database, now.Year(), now.Month())
		if err != nil {
			return summaryDataMsg{err: err}
		}

		return summaryDataMsg{data: d}
	}
}

func (m summaryModel) Update(msg tea.Msg) (summaryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case summaryDataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.data = msg.data
			m.loaded = true
		}
	}
	return m, nil
}

func (m summaryModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading data: %v", m.err)
	}
	if !m.loaded {
		return "Loading..."
	}

	var b strings.Builder

	// Accounts section
	b.WriteString(sectionTitle.Render("Accounts"))
	b.WriteString("\n\n")
	b.WriteString(m.renderAccounts())
	b.WriteString("\n\n")

	// This Month section
	b.WriteString(m.renderMonthly())
	b.WriteString("\n\n")

	// Recent Transactions section
	b.WriteString(sectionTitle.Render("Recent Transactions"))
	b.WriteString("\n\n")
	b.WriteString(m.renderTransactions())

	return b.String()
}

func (m summaryModel) renderAccounts() string {
	nameW := len("Name")
	typeW := len("Type")
	balW := len("Balance")

	for _, a := range m.data.accounts {
		if len(a.Name) > nameW {
			nameW = len(a.Name)
		}
		dt := a.Type
		if len(dt) > typeW {
			typeW = len(dt)
		}
		bs := formatPence(a.Balance)
		if len(bs) > balW {
			balW = len(bs)
		}
	}

	// Compute total for width calculation
	var total int64
	for _, a := range m.data.accounts {
		total += a.Balance
	}
	ts := formatPence(total)
	if len(ts) > balW {
		balW = len(ts)
	}
	if len("Total") > nameW {
		nameW = len("Total")
	}

	fmtRow := fmt.Sprintf("  %%-%ds  %%-%ds  %%%ds", nameW, typeW, balW)

	var b strings.Builder

	header := fmt.Sprintf(fmtRow, "Name", "Type", "Balance")
	b.WriteString(tableHeader.Render(header))
	b.WriteString("\n")

	for _, a := range m.data.accounts {
		row := fmt.Sprintf(fmtRow, a.Name, a.Type, formatPence(a.Balance))
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Separator
	lineLen := 2 + nameW + 2 + typeW + 2 + balW
	b.WriteString("  " + strings.Repeat("─", lineLen-2))
	b.WriteString("\n")

	totalLabel := fmt.Sprintf("  %%-%ds  %%-%ds  %%%ds", nameW, typeW, balW)
	b.WriteString(totalRow.Render(fmt.Sprintf(totalLabel, "Total", "", formatPence(total))))

	return b.String()
}

func (m summaryModel) renderMonthly() string {
	ms := m.data.monthly
	title := fmt.Sprintf("This Month (%s %d)", ms.Month.String(), ms.Year)

	var b strings.Builder
	b.WriteString(sectionTitle.Render(title))
	b.WriteString("\n\n")

	income := incomeStyle.Render(formatPence(ms.Income))
	expenses := expenseStyle.Render(formatPence(-ms.Expenses)) // Expenses stored as negative, display as positive
	net := ms.Income + ms.Expenses

	b.WriteString(fmt.Sprintf("  Income:   %s    Expenses: %s\n", income, expenses))

	netStr := formatSignedPence(net)
	if net >= 0 {
		b.WriteString(fmt.Sprintf("  Net:      %s", netPositive.Render(netStr)))
	} else {
		b.WriteString(fmt.Sprintf("  Net:      %s", netNegative.Render(netStr)))
	}

	return b.String()
}

func (m summaryModel) renderTransactions() string {
	if len(m.data.transactions) == 0 {
		return "  No transactions yet."
	}

	dateW := len("Date")
	acctW := len("Account")
	descW := len("Description")
	amtW := len("Amount")

	for _, tx := range m.data.transactions {
		ds := formatDate(tx.Date)
		if len(ds) > dateW {
			dateW = len(ds)
		}
		if len(tx.Account) > acctW {
			acctW = len(tx.Account)
		}
		if len(tx.Description) > descW {
			descW = len(tx.Description)
		}
		as := formatSignedPence(tx.Amount)
		if len(as) > amtW {
			amtW = len(as)
		}
	}

	fmtRow := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%%ds", dateW, acctW, descW, amtW)

	var b strings.Builder

	header := fmt.Sprintf(fmtRow, "Date", "Account", "Description", "Amount")
	b.WriteString(tableHeader.Render(header))
	b.WriteString("\n")

	for _, tx := range m.data.transactions {
		row := fmt.Sprintf(fmtRow, formatDate(tx.Date), tx.Account, tx.Description, formatSignedPence(tx.Amount))
		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}
