package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"budgie/internal/db"
	"budgie/internal/ui"
)

func main() {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	p := tea.NewProgram(ui.New(database), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
