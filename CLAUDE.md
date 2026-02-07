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
go test ./...            # Run tests
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

### Style & Conventions

#### DB layer (`internal/db/`)
- All DB functions take `*sql.DB` as the first argument.
- Always wrap errors with context using `fmt.Errorf("doing thing: %w", err)` — never return bare errors from DB operations.
- Use `fmt.Errorf` for all error creation, including simple validation messages. Reserve `errors.New` only for package-level sentinel errors (e.g. `ErrAccountHasTransactions`).
- User-facing error messages should be friendly — detect constraint violations (e.g. `UNIQUE constraint failed`) and return readable messages instead of raw SQLite errors.
- Pre-release: no migration logic. Schema changes mean deleting and recreating the database.

#### UI layer (`internal/ui/`)
- Each form/view is a separate file with its own unexported bubbletea model (e.g. `accountFormModel`, `transactionFormModel`).
- All message types are unexported with a `Msg` suffix: `accountInsertedMsg`, `accountFormCancelledMsg`, `transactionInsertErrMsg`.
- Form field enums use a short prefix matching the form: `acctFieldName`, `acctFieldType` for account form; `txFieldAccount`, `txFieldDate` for transaction form.
- Shared styles (`formLabelStyle`, `formActiveLabel`, `formErrorStyle`, `formButtonStyle`, `formActiveButton`) are defined at package level in `account_form.go` without hardcoded widths. Each form sets its own label width explicitly.
- `ui.go` owns view routing. The top-level `Model.Update` handles cross-cutting messages (insert success, form cancelled) and delegates to per-view update methods.
- `ctrl+c` quits from any view. `q` only quits from the dashboard (not forms where it could be typed).
- DB operations from forms are always async via `tea.Cmd`. On success, return to dashboard and reinitialise `summaryModel` to refresh data.

#### Testing (`internal/db/`)
- Tests use in-memory SQLite (`:memory:?_foreign_keys=on`) via the `testDB(t)` helper.
- Test helpers (`insertTestAccount`, `insertTestTransaction`) insert data directly via SQL, bypassing the functions under test.
- Test function names follow `TestFunctionName_Scenario` convention (e.g. `TestInsertAccount_DuplicateName`).

## Dependencies

Requires Go 1.24+ and CGo enabled (for sqlite3 driver). Key dependencies:
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/mattn/go-sqlite3` - SQLite driver
