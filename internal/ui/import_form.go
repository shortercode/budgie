package ui

import (
	"database/sql"
	"fmt"
	"strings"

	"budgie/internal/csvimport"
	"budgie/internal/db"
	"budgie/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// impFormField tracks which field is focused in the import form.
type impFormField int

const (
	impFieldPath impFormField = iota
	impFieldAccount
	impFieldSubmit
	impFieldCount // sentinel for wrapping
)

// impState tracks the current phase of the import flow.
type impState int

const (
	impStateForm impState = iota
	impStatePreview
	impStateResult
)

// importAccountsLoadedMsg delivers accounts to the import form.
type importAccountsLoadedMsg struct {
	accounts []model.Account
	err      error
}

// importCSVParsedMsg delivers parsed CSV data and existing external IDs.
type importCSVParsedMsg struct {
	transactions []csvimport.MonzoTransaction
	existing     map[string]bool
	err          error
}

// importCompletedMsg delivers the result of a bulk insert.
type importCompletedMsg struct {
	imported int
	skipped  int
	err      error
}

// importFormCancelledMsg is sent when the user cancels the import form.
type importFormCancelledMsg struct{}

// importDoneMsg is sent when the user dismisses the result screen.
type importDoneMsg struct{}

type importFormModel struct {
	database     *sql.DB
	accounts     []model.Account
	accountIndex int
	pathInput    textinput.Model
	focused      impFormField
	err          string
	loading      bool
	state        impState
	parsed       []csvimport.MonzoTransaction
	toImport     []db.ImportTransaction
	skippedCount int
	importedCount int
}

func newImportFormModel(database *sql.DB) importFormModel {
	pathIn := textinput.New()
	pathIn.Placeholder = "~/Downloads/monzo.csv"
	pathIn.CharLimit = 256
	pathIn.Focus()

	return importFormModel{
		database: database,
		pathInput: pathIn,
		focused:  impFieldPath,
		loading:  true,
		state:    impStateForm,
	}
}

func (m importFormModel) Init() tea.Cmd {
	database := m.database
	return func() tea.Msg {
		accounts, err := db.ListAccounts(database)
		return importAccountsLoadedMsg{accounts: accounts, err: err}
	}
}

func (m importFormModel) Update(msg tea.Msg) (importFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case importAccountsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if len(msg.accounts) == 0 {
			m.err = "No accounts exist. Create an account first."
			return m, nil
		}
		m.accounts = msg.accounts
		return m, nil

	case importCSVParsedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.state = impStateForm
			return m, nil
		}
		m.parsed = msg.transactions
		accountID := m.accounts[m.accountIndex].ID

		// Filter duplicates
		var toImport []db.ImportTransaction
		skipped := 0
		for _, tx := range msg.transactions {
			if msg.existing[tx.ExternalID] {
				skipped++
				continue
			}
			toImport = append(toImport, db.ImportTransaction{
				AccountID:     accountID,
				Date:          tx.Date,
				Description:   tx.Description,
				Amount:        tx.Amount,
				Category:      tx.Category,
				ExternalID:    tx.ExternalID,
				Notes:         tx.Notes,
				Type:          tx.Type,
				LocalAmount:   tx.LocalAmount,
				LocalCurrency: tx.LocalCurrency,
			})
		}
		m.toImport = toImport
		m.skippedCount = skipped
		m.state = impStatePreview
		return m, nil

	case importCompletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.state = impStatePreview
			return m, nil
		}
		m.importedCount = msg.imported
		m.state = impStateResult
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case impStateForm:
			return m.updateForm(msg)
		case impStatePreview:
			return m.updatePreview(msg)
		case impStateResult:
			return m.updateResult(msg)
		}
	}

	// Pass through non-key messages to text input
	if m.state == impStateForm && m.focused == impFieldPath {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m importFormModel) updateForm(msg tea.KeyMsg) (importFormModel, tea.Cmd) {
	m.err = ""

	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return importFormCancelledMsg{} }

	case "tab", "down":
		return m.focusNext(), nil

	case "shift+tab", "up":
		return m.focusPrev(), nil

	case "enter":
		if m.focused == impFieldSubmit {
			return m.submitForm()
		}
		return m.focusNext(), nil

	case "left":
		if m.focused == impFieldAccount && len(m.accounts) > 0 {
			m.accountIndex = (m.accountIndex - 1 + len(m.accounts)) % len(m.accounts)
			return m, nil
		}

	case "right":
		if m.focused == impFieldAccount && len(m.accounts) > 0 {
			m.accountIndex = (m.accountIndex + 1) % len(m.accounts)
			return m, nil
		}
	}

	// Update the focused text input
	if m.focused == impFieldPath {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m importFormModel) updatePreview(msg tea.KeyMsg) (importFormModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = impStateForm
		m.err = ""
		return m, nil

	case "enter":
		if len(m.toImport) == 0 {
			return m, func() tea.Msg { return importDoneMsg{} }
		}
		m.loading = true
		database := m.database
		toImport := m.toImport
		skipped := m.skippedCount
		return m, func() tea.Msg {
			count, err := db.BulkInsertTransactions(database, toImport)
			return importCompletedMsg{imported: count, skipped: skipped, err: err}
		}
	}
	return m, nil
}

func (m importFormModel) updateResult(msg tea.KeyMsg) (importFormModel, tea.Cmd) {
	return m, func() tea.Msg { return importDoneMsg{} }
}

func (m importFormModel) focusField(field impFormField) importFormModel {
	m.pathInput.Blur()
	m.focused = field
	if field == impFieldPath {
		m.pathInput.Focus()
	}
	return m
}

func (m importFormModel) focusNext() importFormModel {
	next := (m.focused + 1) % impFieldCount
	return m.focusField(next)
}

func (m importFormModel) focusPrev() importFormModel {
	prev := (m.focused - 1 + impFieldCount) % impFieldCount
	return m.focusField(prev)
}

func (m importFormModel) submitForm() (importFormModel, tea.Cmd) {
	path := strings.TrimSpace(m.pathInput.Value())
	if path == "" {
		m.err = "File path is required"
		return m, nil
	}
	if len(m.accounts) == 0 {
		m.err = "No accounts exist. Create an account first."
		return m, nil
	}

	m.loading = true
	database := m.database
	return m, func() tea.Msg {
		txns, err := csvimport.ParseMonzoCSV(path)
		if err != nil {
			return importCSVParsedMsg{err: err}
		}
		existing, err := db.ExistingExternalIDs(database)
		if err != nil {
			return importCSVParsedMsg{err: err}
		}
		return importCSVParsedMsg{transactions: txns, existing: existing}
	}
}

func (m importFormModel) View() string {
	switch m.state {
	case impStatePreview:
		return m.viewPreview()
	case impStateResult:
		return m.viewResult()
	default:
		return m.viewForm()
	}
}

func (m importFormModel) viewForm() string {
	var b strings.Builder

	b.WriteString(sectionTitle.Render("Import Transactions"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("  Loading accounts...")
		return b.String()
	}

	labelW := 12

	// Path input
	label := m.impLabelStyle(impFieldPath, labelW)
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("CSV File:"), m.pathInput.View()))

	// Account picker
	label = m.impLabelStyle(impFieldAccount, labelW)
	if len(m.accounts) > 0 {
		b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Account:"), m.renderAccountPicker()))
	} else {
		b.WriteString(fmt.Sprintf("  %s (none)\n", label.Render("Account:")))
	}

	// Error
	if m.err != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", formErrorStyle.Render(m.err)))
	}

	// Submit button
	b.WriteString("\n")
	if m.focused == impFieldSubmit {
		b.WriteString(fmt.Sprintf("  %s", formActiveButton.Render("Import")))
	} else {
		b.WriteString(fmt.Sprintf("  %s", formButtonStyle.Render("Import")))
	}
	b.WriteString("\n")

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  tab/shift+tab: navigate  enter: submit  esc: cancel"))

	return b.String()
}

func (m importFormModel) viewPreview() string {
	var b strings.Builder

	b.WriteString(sectionTitle.Render("Import Preview"))
	b.WriteString("\n\n")

	totalParsed := len(m.parsed)
	willImport := len(m.toImport)

	b.WriteString(fmt.Sprintf("  Found %d transactions", totalParsed))
	if m.skippedCount > 0 {
		b.WriteString(fmt.Sprintf(", skipping %d duplicates", m.skippedCount))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Will import %d new transactions\n", willImport))

	if willImport > 0 {
		b.WriteString("\n")

		// Determine column widths
		dateW := len("Date")
		descW := len("Description")
		catW := len("Category")
		amtW := len("Amount")

		limit := willImport
		if limit > 10 {
			limit = 10
		}

		for _, tx := range m.toImport[:limit] {
			ds := formatDate(tx.Date)
			if len(ds) > dateW {
				dateW = len(ds)
			}
			if len(tx.Description) > descW {
				descW = len(tx.Description)
			}
			if len(tx.Category) > catW {
				catW = len(tx.Category)
			}
			as := formatSignedPence(tx.Amount)
			if len(as) > amtW {
				amtW = len(as)
			}
		}

		// Cap description width
		if descW > 30 {
			descW = 30
		}

		fmtRow := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%%ds", dateW, descW, catW, amtW)

		header := fmt.Sprintf(fmtRow, "Date", "Description", "Category", "Amount")
		b.WriteString(tableHeader.Render(header))
		b.WriteString("\n")

		for _, tx := range m.toImport[:limit] {
			desc := tx.Description
			if len(desc) > descW {
				desc = desc[:descW-1] + "…"
			}
			row := fmt.Sprintf(fmtRow, formatDate(tx.Date), desc, tx.Category, formatSignedPence(tx.Amount))
			b.WriteString(row)
			b.WriteString("\n")
		}

		if willImport > 10 {
			b.WriteString(fmt.Sprintf("\n  ... and %d more\n", willImport-10))
		}
	}

	// Error
	if m.err != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", formErrorStyle.Render(m.err)))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  enter: confirm  esc: back"))

	return b.String()
}

func (m importFormModel) viewResult() string {
	var b strings.Builder

	b.WriteString(sectionTitle.Render("Import Complete"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Imported %d transactions", m.importedCount))
	if m.skippedCount > 0 {
		b.WriteString(fmt.Sprintf(", skipped %d duplicates", m.skippedCount))
	}
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  press any key to continue"))

	return b.String()
}

func (m importFormModel) impLabelStyle(field impFormField, width int) lipgloss.Style {
	if m.focused == field {
		return formActiveLabel.Width(width)
	}
	return formLabelStyle.Width(width)
}

func (m importFormModel) renderAccountPicker() string {
	var parts []string
	for i, a := range m.accounts {
		s := a.Name
		if i == m.accountIndex {
			if m.focused == impFieldAccount {
				s = formActiveButton.Render(s)
			} else {
				s = lipgloss.NewStyle().Bold(true).Render("[" + s + "]")
			}
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "  ")
}
