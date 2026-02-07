package ui

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"budgie/internal/db"
	"budgie/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// txFormField tracks which field is focused.
type txFormField int

const (
	txFieldAccount txFormField = iota
	txFieldDate
	txFieldDescription
	txFieldAmount
	txFieldCategory
	txFieldDirection
	txFieldSubmit
	txFieldCount // sentinel for wrapping
)

// transactionInsertedMsg is sent when the transaction was successfully created.
type transactionInsertedMsg struct{}

// transactionInsertErrMsg is sent when the DB insert fails.
type transactionInsertErrMsg struct{ err error }

// transactionFormCancelledMsg is sent when the user presses Esc.
type transactionFormCancelledMsg struct{}

// accountsLoadedMsg delivers the account list to the form.
type accountsLoadedMsg struct {
	accounts []model.Account
	err      error
}

type txDirection int

const (
	txExpense txDirection = iota
	txIncome
)

// transactionUpdateErrMsg is sent when the DB update fails.
type transactionUpdateErrMsg struct{ err error }

// transactionFormModel is the bubbletea model for the add/edit-transaction form.
type transactionFormModel struct {
	database      *sql.DB
	accounts      []model.Account
	accountIndex  int
	editID        int64  // 0 = add mode, >0 = edit mode
	editAccountID int64  // target account ID for edit mode (resolved after accounts load)
	dateInput     textinput.Model
	descInput     textinput.Model
	amountInput   textinput.Model
	catInput      textinput.Model
	direction     txDirection
	focused       txFormField
	err           string
	loading       bool
}

func newTransactionFormModel(database *sql.DB) transactionFormModel {
	dateIn := textinput.New()
	dateIn.Placeholder = "DD/MM/YYYY"
	dateIn.CharLimit = 10
	dateIn.SetValue(time.Now().Format("02/01/2006"))

	descIn := textinput.New()
	descIn.Placeholder = "Description"
	descIn.CharLimit = 100

	amountIn := textinput.New()
	amountIn.Placeholder = "0.00"
	amountIn.CharLimit = 15

	catIn := textinput.New()
	catIn.Placeholder = "Category"
	catIn.CharLimit = 50

	return transactionFormModel{
		database:    database,
		dateInput:   dateIn,
		descInput:   descIn,
		amountInput: amountIn,
		catInput:    catIn,
		direction:   txExpense,
		focused:     txFieldAccount,
		loading:     true,
	}
}

func newTransactionFormModelForEdit(database *sql.DB, tx db.TransactionRow) transactionFormModel {
	dateIn := textinput.New()
	dateIn.Placeholder = "DD/MM/YYYY"
	dateIn.CharLimit = 10
	dateIn.SetValue(tx.Date.Format("02/01/2006"))

	descIn := textinput.New()
	descIn.Placeholder = "Description"
	descIn.CharLimit = 100
	descIn.SetValue(tx.Description)

	// Display absolute value for the amount
	absAmount := tx.Amount
	if absAmount < 0 {
		absAmount = -absAmount
	}
	amountIn := textinput.New()
	amountIn.Placeholder = "0.00"
	amountIn.CharLimit = 15
	amountIn.SetValue(fmt.Sprintf("%.2f", float64(absAmount)/100))

	catIn := textinput.New()
	catIn.Placeholder = "Category"
	catIn.CharLimit = 50
	catIn.SetValue(tx.Category)

	dir := txExpense
	if tx.Amount > 0 {
		dir = txIncome
	}

	return transactionFormModel{
		database:      database,
		editID:        tx.ID,
		editAccountID: tx.AccountID,
		dateInput:     dateIn,
		descInput:     descIn,
		amountInput:   amountIn,
		catInput:      catIn,
		direction:     dir,
		focused:       txFieldAccount,
		loading:       true,
	}
}

func (m transactionFormModel) Init() tea.Cmd {
	database := m.database
	return func() tea.Msg {
		accounts, err := db.ListAccounts(database)
		return accountsLoadedMsg{accounts: accounts, err: err}
	}
}

func (m transactionFormModel) Update(msg tea.Msg) (transactionFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case accountsLoadedMsg:
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
		// In edit mode, select the account matching editAccountID
		if m.editID > 0 {
			for i, a := range m.accounts {
				if a.ID == m.editAccountID {
					m.accountIndex = i
					break
				}
			}
		}
		return m, nil

	case transactionUpdateErrMsg:
		m.err = msg.err.Error()
		return m, nil

	case transactionInsertErrMsg:
		m.err = msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		m.err = ""

		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return transactionFormCancelledMsg{} }

		case "tab", "down":
			return m.focusNext(), nil

		case "shift+tab", "up":
			return m.focusPrev(), nil

		case "enter":
			if m.focused == txFieldSubmit {
				return m.submit()
			}
			return m.focusNext(), nil

		case "left":
			if m.focused == txFieldAccount && len(m.accounts) > 0 {
				m.accountIndex = (m.accountIndex - 1 + len(m.accounts)) % len(m.accounts)
				return m, nil
			}
			if m.focused == txFieldDirection {
				m.direction = txExpense
				return m, nil
			}

		case "right":
			if m.focused == txFieldAccount && len(m.accounts) > 0 {
				m.accountIndex = (m.accountIndex + 1) % len(m.accounts)
				return m, nil
			}
			if m.focused == txFieldDirection {
				m.direction = txIncome
				return m, nil
			}
		}
	}

	// Update the focused text input.
	var cmd tea.Cmd
	switch m.focused {
	case txFieldDate:
		m.dateInput, cmd = m.dateInput.Update(msg)
	case txFieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	case txFieldAmount:
		m.amountInput, cmd = m.amountInput.Update(msg)
	case txFieldCategory:
		m.catInput, cmd = m.catInput.Update(msg)
	}
	return m, cmd
}

func (m transactionFormModel) focusField(field txFormField) transactionFormModel {
	m.dateInput.Blur()
	m.descInput.Blur()
	m.amountInput.Blur()
	m.catInput.Blur()

	m.focused = field
	switch field {
	case txFieldDate:
		m.dateInput.Focus()
	case txFieldDescription:
		m.descInput.Focus()
	case txFieldAmount:
		m.amountInput.Focus()
	case txFieldCategory:
		m.catInput.Focus()
	}
	return m
}

func (m transactionFormModel) focusNext() transactionFormModel {
	next := (m.focused + 1) % txFieldCount
	return m.focusField(next)
}

func (m transactionFormModel) focusPrev() transactionFormModel {
	prev := (m.focused - 1 + txFieldCount) % txFieldCount
	return m.focusField(prev)
}

func (m transactionFormModel) submit() (transactionFormModel, tea.Cmd) {
	if len(m.accounts) == 0 {
		m.err = "No accounts exist. Create an account first."
		return m, nil
	}

	// Parse date
	dateStr := strings.TrimSpace(m.dateInput.Value())
	date, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		m.err = "Invalid date. Use DD/MM/YYYY format."
		return m, nil
	}

	// Parse amount
	amountStr := strings.TrimSpace(m.amountInput.Value())
	amountStr = strings.TrimPrefix(amountStr, "£")
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	if amountStr == "" {
		m.err = "Amount is required"
		return m, nil
	}
	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amountFloat < 0 {
		m.err = "Invalid amount. Enter a positive number like 12.50"
		return m, nil
	}
	if amountFloat == 0 {
		m.err = "Amount must be greater than zero"
		return m, nil
	}
	amountPence := int64(amountFloat*100 + 0.5) // round to nearest pence
	if m.direction == txExpense {
		amountPence = -amountPence
	}

	accountID := m.accounts[m.accountIndex].ID
	description := strings.TrimSpace(m.descInput.Value())
	category := strings.TrimSpace(m.catInput.Value())
	database := m.database

	if m.editID > 0 {
		editID := m.editID
		cmd := func() tea.Msg {
			err := db.UpdateTransaction(database, editID, accountID, date, description, amountPence, category)
			if err != nil {
				return transactionUpdateErrMsg{err: err}
			}
			return transactionUpdatedMsg{}
		}
		return m, cmd
	}

	cmd := func() tea.Msg {
		_, err := db.InsertTransaction(database, accountID, date, description, amountPence, category)
		if err != nil {
			return transactionInsertErrMsg{err: err}
		}
		return transactionInsertedMsg{}
	}
	return m, cmd
}

func (m transactionFormModel) View() string {
	var b strings.Builder

	if m.editID > 0 {
		b.WriteString(sectionTitle.Render("Edit Transaction"))
	} else {
		b.WriteString(sectionTitle.Render("Add Transaction"))
	}
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("  Loading accounts...")
		return b.String()
	}

	labelW := 14

	// Account picker
	label := m.labelStyle(txFieldAccount, labelW)
	if len(m.accounts) > 0 {
		b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Account:"), m.renderAccountPicker()))
	} else {
		b.WriteString(fmt.Sprintf("  %s (none)\n", label.Render("Account:")))
	}

	// Date
	label = m.labelStyle(txFieldDate, labelW)
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Date:"), m.dateInput.View()))

	// Description
	label = m.labelStyle(txFieldDescription, labelW)
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Description:"), m.descInput.View()))

	// Amount
	label = m.labelStyle(txFieldAmount, labelW)
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Amount (£):"), m.amountInput.View()))

	// Category
	label = m.labelStyle(txFieldCategory, labelW)
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Category:"), m.catInput.View()))

	// Direction toggle
	label = m.labelStyle(txFieldDirection, labelW)
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Type:"), m.renderDirectionToggle()))

	// Error
	if m.err != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", formErrorStyle.Render(m.err)))
	}

	// Submit
	b.WriteString("\n")
	submitLabel := "Add Transaction"
	if m.editID > 0 {
		submitLabel = "Save"
	}
	if m.focused == txFieldSubmit {
		b.WriteString(fmt.Sprintf("  %s", formActiveButton.Render(submitLabel)))
	} else {
		b.WriteString(fmt.Sprintf("  %s", formButtonStyle.Render(submitLabel)))
	}
	b.WriteString("\n")

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  tab/shift+tab: navigate  enter: submit  esc: cancel"))

	return b.String()
}

func (m transactionFormModel) labelStyle(field txFormField, width int) lipgloss.Style {
	if m.focused == field {
		return formActiveLabel.Width(width)
	}
	return formLabelStyle.Width(width)
}

func (m transactionFormModel) renderAccountPicker() string {
	var parts []string
	for i, a := range m.accounts {
		s := a.Name
		if i == m.accountIndex {
			if m.focused == txFieldAccount {
				s = formActiveButton.Render(s)
			} else {
				s = lipgloss.NewStyle().Bold(true).Render("[" + s + "]")
			}
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "  ")
}

func (m transactionFormModel) renderDirectionToggle() string {
	expense := "Expense"
	income := "Income"

	if m.direction == txExpense {
		if m.focused == txFieldDirection {
			expense = formActiveButton.Render(expense)
		} else {
			expense = lipgloss.NewStyle().Bold(true).Render("[" + expense + "]")
		}
	}
	if m.direction == txIncome {
		if m.focused == txFieldDirection {
			income = formActiveButton.Render(income)
		} else {
			income = lipgloss.NewStyle().Bold(true).Render("[" + income + "]")
		}
	}

	return expense + "  " + income
}
