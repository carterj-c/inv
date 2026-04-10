# inv

`inv` is a terminal invoice manager written in Go. It uses Bubble Tea for the TUI, stores invoice data locally under your config directory, exports invoices to PDF, and keeps a lightweight git history of invoice changes.

The app is designed around a simple workflow:

- create or switch a client
- draft invoices with one or more line items
- export PDFs for review or sending
- mark invoices as sent to lock them from further edits

## Features

- terminal-first invoice workflow with vim-style navigation
- first-run setup flow for your business info and first client
- per-client defaults for address, payment terms, currency, and last-used line items
- sequential invoice numbering in `YYYY-NNN` format
- PDF export with subtotal, tax, and total rendering
- draft vs sent invoice states
- sent invoices are immutable
- local persistence in JSON and TOML
- automatic git initialization and best-effort pull/commit/push for the invoice config directory
- optional GitHub remote creation on first run when `gh` is installed and authenticated

## Requirements

- Go 1.26 or newer
- `git` on your `PATH` if you want the built-in history/sync behavior
- a terminal that supports interactive TUI apps

Optional but useful:

- `xdg-open` to auto-open exported PDFs on Linux
- `$EDITOR` or `$VISUAL` for multi-line line item descriptions
- `gh` if you want the app to auto-create a private GitHub backup repo on first run

## Setup

Clone the repository and build the binary:

```bash
git clone https://github.com/carterj-c/inv
cd inv
go build .
```

That produces an `inv` binary in the project root. You can also install it into your Go bin directory:

```bash
go install .
```

If you use `go install`, make sure your Go bin directory is on `PATH`. On many systems that is:

```bash
export PATH="$HOME/go/bin:$PATH"
```

## Run

From the repository:

```bash
go run .
```

Or, if you built the binary:

```bash
./inv
```

Or, after `go install`:

```bash
inv
```

Manual PDF sync:

```bash
inv sync
```

## First Run

On first launch, `inv` creates its data directory and walks you through:

1. your name or business name
2. your address
3. the PDF export directory
4. your default currency
5. your first client

Default values baked into the app:

- export directory: `~/invoices`
- default currency: `CAD`
- default payment terms for new clients: `Net 30`

After setup, the app:

- writes the global config
- writes the first client config
- initializes a git repository in the invoice config directory if one does not already exist
- if `gh` is installed, authenticated, and no `origin` remote exists, it attempts to create a private GitHub repo named `invoice-config` and push the initial state

## Storage Layout

The app stores everything under:

- `$XDG_CONFIG_HOME/invoice` if `XDG_CONFIG_HOME` is set
- otherwise `~/.config/invoice`

Directory layout:

```text
~/.config/invoice/
├── .git/
├── config.toml
├── clients/
│   └── <client-slug>.toml
└── data/
    ├── invoices.json
    └── pdfs/
```

Files are used as follows:

- `config.toml`: your business info, export directory, active client, default currency
- `clients/*.toml`: per-client details and the last line items used for that client
- `data/invoices.json`: invoice records and next invoice number counters
- `data/pdfs/`: tracked archive copies of exported PDFs

## Configuration

Global config example:

```toml
[user]
name = "Jane Doe"
address = "123 Main St, Toronto, ON M5V 2H1"

[settings]
export_dir = "~/invoices"
active_client = "acme-corp"
default_currency = "CAD"
```

Client config example:

```toml
name = "Acme Corp"
address = "456 Business Ave, Suite 200, Montreal, QC H2X 1Y4"
payment_terms = "Net 30"
currency = "CAD"

[[tax]]
name = "GST"
rate = 0.05

[[last_line_items]]
description = "Software development"
quantity = 5
rate = 15000
```

Note: the data model supports `tax` entries, but the current TUI does not provide screens for creating or editing tax rules. To use tax lines in totals and PDFs, add them directly to the client TOML file.

## Usage

Main list view:

- `j` / `k`: move selection
- `n`: new invoice
- `e` or `Enter`: edit selected draft
- `d`: delete selected draft
- `p`: export selected invoice to PDF
- `s`: mark selected invoice as sent
- `c`: switch client
- `?`: toggle help
- `q`: quit

Editor view:

- `Tab` / `Shift+Tab`: move between fields
- `a`: add a line item
- `e`: edit the selected field or line item
- `d`: remove the selected line item
- `j` / `k`: move between line items
- `Ctrl+e`: open the selected description in `$EDITOR`
- `Enter`: save
- `Esc`: cancel

Behavior worth knowing:

- new invoice numbers are allocated per year, for example `2026-001`
- invoice numbers are intentionally not reused
- new invoices are prefilled from the client's `last_line_items` when available
- sent invoices cannot be edited or deleted
- PDFs are written to the configured export directory, which is created automatically if needed
- every exported PDF is also copied into the tracked archive at `data/pdfs/` inside the config repo
- on launch, the app performs a non-destructive PDF sync between the export directory and `data/pdfs/`
- when the same PDF exists in both places with different contents, the newer file wins
- sync does not delete PDFs from either side automatically

## Git Behavior

The app keeps invoice data in a git repo inside the config directory:

- first run initializes the repo
- launch performs a best-effort `git pull --rebase --quiet` if `origin` exists
- launch then syncs PDFs between the user-facing export directory and the tracked archive
- saving, deleting, or marking an invoice as sent stages all changes and commits them
- exporting a PDF also updates the tracked archive copy and commits it
- `inv sync` runs the same PDF sync from the command line and commits archive changes
- after commit, the app attempts a best-effort `git push --quiet` if `origin` exists

If `gh` is not installed or not authenticated, the app leaves the repo local-only. In that case, add a remote yourself:

```bash
cd ~/.config/invoice
git remote add origin <your-remote-url>
git push -u origin main
```

You may need to create the initial branch name that matches your git config.

## Development

Build:

```bash
go build ./...
```

Test:

```bash
go test ./...
```

There are currently no Go test files in the repository, but the package set should build cleanly.

## Project Structure

- [`main.go`](/home/ptable/Dev/inv/main.go): entrypoint
- [`internal/tui`](/home/ptable/Dev/inv/internal/tui): Bubble Tea application and views
- [`internal/config`](/home/ptable/Dev/inv/internal/config/config.go): config paths and TOML persistence
- [`internal/store`](/home/ptable/Dev/inv/internal/store/store.go): JSON invoice storage
- [`internal/pdf`](/home/ptable/Dev/inv/internal/pdf/pdf.go): PDF generation
- [`internal/git`](/home/ptable/Dev/inv/internal/git/git.go): git helpers for backup/sync
- [`internal/model`](/home/ptable/Dev/inv/internal/model/model.go): core data model
