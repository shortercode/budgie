package ui

import (
	"database/sql"
	"fmt"
	"strings"

	"budgie/internal/db"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
)

type chartDataMsg struct {
	data []db.MonthlyFinances
	err  error
}

type chartClosedMsg struct{}

type chartModel struct {
	database *sql.DB
	data     []db.MonthlyFinances
	loaded   bool
	err      error
	width    int
	height   int
}

func newChartModel(database *sql.DB, width, height int) chartModel {
	return chartModel{database: database, width: width, height: height}
}

func (m chartModel) Init() tea.Cmd {
	return func() tea.Msg {
		data, err := db.MonthlyFinancesByMonth(m.database)
		return chartDataMsg{data: data, err: err}
	}
}

func (m chartModel) Update(msg tea.Msg) (chartModel, tea.Cmd) {
	switch msg := msg.(type) {
	case chartDataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.data = msg.data
			m.loaded = true
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return chartClosedMsg{} }
		}
	}
	return m, nil
}

var (
	chartLegendCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	chartLegendGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	chartLegendRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func (m chartModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading chart data: %v", m.err)
	}
	if !m.loaded {
		return "Loading..."
	}
	if len(m.data) == 0 {
		return "No transaction data yet."
	}

	var b strings.Builder
	b.WriteString(sectionTitle.Render("Net Worth Over Time"))
	b.WriteString("\n\n")

	// Build three series: net worth, income, expenses (converted to pounds)
	netWorth := make([]float64, len(m.data))
	income := make([]float64, len(m.data))
	expenses := make([]float64, len(m.data))

	for i, mf := range m.data {
		netWorth[i] = float64(mf.NetWorth) / 100
		income[i] = float64(mf.Income) / 100
		expenses[i] = float64(-mf.Expenses) / 100 // display as positive
	}

	// Chart dimensions
	chartWidth := m.width - 14 // room for Y-axis labels and padding
	if chartWidth < 20 {
		chartWidth = 20
	}
	chartHeight := m.height - 12 // room for title, legend, labels, help
	if chartHeight < 5 {
		chartHeight = 5
	}

	graph := asciigraph.PlotMany(
		[][]float64{netWorth, income, expenses},
		asciigraph.Height(chartHeight),
		asciigraph.Width(chartWidth),
		asciigraph.SeriesColors(asciigraph.Cyan, asciigraph.Green, asciigraph.Red),
		asciigraph.Precision(0),
	)
	b.WriteString(graph)
	b.WriteString("\n\n")

	// Month labels
	b.WriteString("  ")
	for i, mf := range m.data {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(fmt.Sprintf("%s %d", mf.Month.String()[:3], mf.Year))
	}
	b.WriteString("\n\n")

	// Legend
	b.WriteString("  ")
	b.WriteString(chartLegendCyan.Render("--- Net Worth"))
	b.WriteString("  ")
	b.WriteString(chartLegendGreen.Render("--- Income"))
	b.WriteString("  ")
	b.WriteString(chartLegendRed.Render("--- Expenses"))

	return b.String()
}
