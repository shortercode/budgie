# budgie

A personal finance tracker for the terminal, built with Go.

## Overview

budgie is a terminal UI application for tracking income, expenses, and account balances. It runs entirely in the terminal using an interactive full-screen interface and stores all data locally in a SQLite database. No network access, no cloud sync, no accounts to create -- just a fast, private tool for keeping tabs on your money.

## Current status

**Early development.** The foundation is in place and the app launches with a working dashboard, but there is no way to add or manage data through the UI yet.

### What works

- **Summary dashboard** -- the startup screen shows three sections:
  - Account balances with a total row
  - Monthly income vs expenses with a net figure
  - Last 10 transactions with date, account, description, and amount
- **SQLite database** -- schema for accounts (current, savings, credit, cash) and transactions with automatic migration on first run
- **UK localisation** -- all currency displayed in GBP (£), dates formatted as DD/MM/YYYY, "checking" account type displayed as "current"

### What doesn't exist yet

- Adding, editing, or deleting accounts
- Adding, editing, or deleting transactions
- Navigating between views
- Filtering or searching transactions
- Category management and reporting
- Import/export (CSV, OFX, etc.)

## Design choices

### Terminal UI (Bubbletea)

The interface is built with the [Charmbracelet](https://charm.sh/) ecosystem:

- **bubbletea** -- Elm-architecture framework for terminal apps
- **lipgloss** -- styling (colours, bold, underline)
- **bubbles** -- pre-built components (available but not yet used)

This was chosen over a web UI or desktop GUI to keep the tool lightweight, fast to launch, and usable over SSH.

### SQLite

All data lives in a single file at `~/.local/share/budgie/budgie.db`. SQLite was chosen because:

- Zero configuration, no server process
- Single-file database, easy to back up
- WAL journal mode enabled for safe concurrent reads
- Foreign key constraints enforced

### Integer arithmetic for money

Amounts are stored as integers in pence (1/100th of a pound) rather than floating-point numbers. This avoids rounding errors that plague float-based currency calculations. The formatting layer converts pence to display strings like `£1,200.00`.

### UK-first localisation

The app uses British English and GBP throughout:

| Convention | Example |
|---|---|
| Currency | £1,200.00 |
| Negative currency | -£45.00 |
| Date format | 28/01/2026 |
| Account type display | "current" (stored as "checking" for DB compatibility) |

## Project structure

```
budgie/
├── cmd/budgie/main.go        # Entry point
├── internal/
│   ├── db/
│   │   ├── db.go             # Database connection, migrations
│   │   └── queries.go        # Dashboard query functions
│   ├── model/
│   │   └── model.go          # Domain types (Account, Transaction)
│   └── ui/
│       ├── ui.go             # Top-level Bubbletea model
│       ├── summary.go        # Dashboard view
│       └── format.go         # GBP/date formatting helpers
├── go.mod
└── go.sum
```

## Requirements

- Go 1.24+
- CGo enabled (required by the SQLite driver)

## Building and running

```
go build ./cmd/budgie
./budgie
```

Or directly:

```
go run ./cmd/budgie
```

Press `q` or `Ctrl+C` to quit.

## Future plans

Roughly in priority order:

1. **Account management** -- add, rename, and delete accounts from within the UI
2. **Transaction entry** -- form for adding transactions with date, description, amount, and category
3. **Transaction list view** -- scrollable, filterable list of all transactions per account
4. **Edit and delete** -- modify or remove existing transactions
5. **Category reporting** -- breakdown of spending by category, monthly trends
6. **CSV import** -- bulk-load transactions from bank exports
7. **Budget targets** -- set monthly spending limits per category and track against them
8. **Multi-month views** -- compare spending across months

## Data location

The database is stored at:

```
~/.local/share/budgie/budgie.db
```

To back up your data, copy this file. To start fresh, delete it -- the schema will be recreated on next launch.
