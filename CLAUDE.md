# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Budgie is a personal finance tracker built as a terminal UI application in Go. It uses the Charmbracelet ecosystem (bubbletea, lipgloss, bubbles) for the TUI and SQLite for data storage. Data is stored in `~/.local/share/budgie/budgie.db`.

## Build & Run Commands

```bash
go build ./cmd/budgie    # Build binary to ./budgie
go run ./cmd/budgie      # Run directly without building
go fmt ./...             # Format code
go vet ./...             # Check for issues
go test ./...            # Run tests (none exist yet)
```

## Architecture

The codebase follows the Elm-style Model-View-Update architecture via bubbletea:

- **cmd/budgie/** - Entry point, initializes database and UI
- **internal/model/** - Pure domain types (Account, Transaction, AccountType)
- **internal/db/** - Database connection, migrations, and query functions
- **internal/ui/** - Bubbletea models and views (summary dashboard)

### Key Patterns

- **Integer arithmetic for currency**: All amounts are stored as int64 in pence (not pounds) to avoid floating-point errors. Conversion to display format happens only in `internal/ui/format.go`.
- **Async data loading**: Dashboard fetches data via Init() commands and displays "Loading..." until data arrives. No blocking DB calls in the UI thread.
- **UK localization**: Currency as GBP with thousands separators, dates as DD/MM/YYYY, account type "current" used throughout.

### Database Schema

Two tables: `accounts` (id, name, type) and `transactions` (id, account_id, date, description, amount, category). Foreign key from transactions to accounts. Amount stored in pence as INTEGER.

## Dependencies

Requires Go 1.24+ and CGo enabled (for sqlite3 driver). Key dependencies:
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/mattn/go-sqlite3` - SQLite driver
