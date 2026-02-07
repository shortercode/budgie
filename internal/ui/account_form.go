package ui

import (
	"database/sql"
	"fmt"
	"strings"

	"budgie/internal/db"
	"budgie/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var accountTypes = []model.AccountType{
	model.AccountTypeCurrent,
	model.AccountTypeSavings,
	model.AccountTypeCredit,
	model.AccountTypeCash,
}

// accountFormField tracks which field is focused.
type accountFormField int

const (
	fieldName accountFormField = iota
	fieldType
	fieldSubmit
)

// accountInsertedMsg is sent when the account was successfully created.
type accountInsertedMsg struct{}

// accountInsertErrMsg is sent when the DB insert fails.
type accountInsertErrMsg struct{ err error }

// AccountFormCancelled is sent when the user presses Esc.
type AccountFormCancelled struct{}

var (
	formLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Width(10)

	formActiveLabel = lipgloss.NewStyle().
			Bold(true).
			Width(10).
			Foreground(lipgloss.Color("12"))

	formErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	formButtonStyle = lipgloss.NewStyle().
			Padding(0, 2)

	formActiveButton = lipgloss.NewStyle().
				Padding(0, 2).
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("12"))
)

// accountFormModel is the bubbletea model for the add-account form.
type accountFormModel struct {
	database  *sql.DB
	nameInput textinput.Model
	typeIndex int
	focused   accountFormField
	err       string
}

func newAccountFormModel(database *sql.DB) accountFormModel {
	ti := textinput.New()
	ti.Placeholder = "Account name"
	ti.CharLimit = 50
	ti.Focus()

	return accountFormModel{
		database:  database,
		nameInput: ti,
		focused:   fieldName,
	}
}

func (m accountFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m accountFormModel) Update(msg tea.Msg) (accountFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case accountInsertErrMsg:
		m.err = msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		// Clear error on any keypress.
		m.err = ""

		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return AccountFormCancelled{} }

		case "tab", "down":
			return m.focusNext(), nil

		case "shift+tab", "up":
			return m.focusPrev(), nil

		case "enter":
			if m.focused == fieldSubmit {
				return m.submit()
			}
			return m.focusNext(), nil

		case "left":
			if m.focused == fieldType {
				m.typeIndex = (m.typeIndex - 1 + len(accountTypes)) % len(accountTypes)
				return m, nil
			}

		case "right":
			if m.focused == fieldType {
				m.typeIndex = (m.typeIndex + 1) % len(accountTypes)
				return m, nil
			}
		}
	}

	if m.focused == fieldName {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m accountFormModel) focusNext() accountFormModel {
	switch m.focused {
	case fieldName:
		m.focused = fieldType
		m.nameInput.Blur()
	case fieldType:
		m.focused = fieldSubmit
	case fieldSubmit:
		// wrap around
		m.focused = fieldName
		m.nameInput.Focus()
	}
	return m
}

func (m accountFormModel) focusPrev() accountFormModel {
	switch m.focused {
	case fieldName:
		// wrap around
		m.focused = fieldSubmit
		m.nameInput.Blur()
	case fieldType:
		m.focused = fieldName
		m.nameInput.Focus()
	case fieldSubmit:
		m.focused = fieldType
	}
	return m
}

func (m accountFormModel) submit() (accountFormModel, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		m.err = "Account name is required"
		return m, nil
	}
	acctType := accountTypes[m.typeIndex]
	database := m.database
	cmd := func() tea.Msg {
		_, err := db.InsertAccount(database, name, acctType)
		if err != nil {
			return accountInsertErrMsg{err: err}
		}
		return accountInsertedMsg{}
	}
	return m, cmd
}

func (m accountFormModel) View() string {
	var b strings.Builder

	b.WriteString(sectionTitle.Render("Add Account"))
	b.WriteString("\n\n")

	// Name field
	label := formLabelStyle
	if m.focused == fieldName {
		label = formActiveLabel
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Name:"), m.nameInput.View()))

	// Type field
	label = formLabelStyle
	if m.focused == fieldType {
		label = formActiveLabel
	}
	typeStr := m.renderTypeSelector()
	b.WriteString(fmt.Sprintf("  %s %s\n", label.Render("Type:"), typeStr))

	// Error
	if m.err != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", formErrorStyle.Render(m.err)))
	}

	// Submit button
	b.WriteString("\n")
	if m.focused == fieldSubmit {
		b.WriteString(fmt.Sprintf("  %s", formActiveButton.Render("Create Account")))
	} else {
		b.WriteString(fmt.Sprintf("  %s", formButtonStyle.Render("Create Account")))
	}
	b.WriteString("\n")

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  tab/shift+tab: navigate  enter: submit  esc: cancel"))

	return b.String()
}

func (m accountFormModel) renderTypeSelector() string {
	var parts []string
	for i, t := range accountTypes {
		s := string(t)
		if i == m.typeIndex {
			if m.focused == fieldType {
				s = formActiveButton.Render(s)
			} else {
				s = lipgloss.NewStyle().Bold(true).Render("[" + s + "]")
			}
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "  ")
}
