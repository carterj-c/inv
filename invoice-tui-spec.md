# Invoice TUI - Design Spec

## Overview
A terminal-based CRUD app for managing and generating invoices. Invoked globally as `invoice`. Clean, minimal TUI with vim-style controls. Exports to PDF.

## Core Principles
- Simple and clean UI (Ghostty-like aesthetic — minimal, intentional)
- Vim keybindings (hjkl navigation, etc.)
- PDF export (read-only output for sending to clients)
- Updates overwrite the source file (no versioning — git handles history)
- Config-driven defaults to minimize repetitive input
- Sent invoices are locked (immutable for audit integrity)
- All monetary values stored as integers (cents) to avoid floating point issues

## Data Model

### Invoice Fields
| Field | Behavior |
|---|---|
| **Company Name** | Your business/personal name. Stored in global config. |
| **Your Address** | Your address. Stored in global config. |
| **Client Name** | Per-client config. |
| **Client Address** | Per-client config. |
| **Invoice Date** | Defaults to today, editable. |
| **Invoice Number** | Auto-generated as `YYYY-NNN` (e.g., `2026-001`). Sequential per year across all clients. Burned on delete (never reused). |
| **Line Items** | List of `(description, quantity, rate)` tuples. See below. |
| **Tax** | Optional. List of named tax entries. Only shown on PDF if configured. |
| **Payment Terms** | Per-client config default (e.g., "Net 30"). Editable per invoice. |
| **Status** | `draft` or `sent`. Sent invoices are locked and cannot be edited. |
| **Currency** | Per-client config (e.g., `CAD`, `USD`, `EUR`). Determines symbol on PDF. |

### Line Items
Each invoice has one or more line items:

| Field | Type | Example |
|---|---|---|
| Description | string | "Software development" |
| Quantity | number | 5 |
| Rate | integer (cents) | 15000 (displayed as $150.00) |

- **Total per line** = quantity × rate (computed, not stored)
- **Subtotal** = sum of all line totals (computed)
- **Tax lines** = subtotal × each tax rate (computed, only if tax configured)
- **Invoice total** = subtotal + tax (computed)
- No explicit "unit" field — description handles context
- Line items pre-filled from previous invoice for the same client
- Long descriptions: `Ctrl+e` opens `$EDITOR` for multi-line editing (writes to tmpfile, reads back on close — standard Unix pattern)

### Tax (Optional)
Tax is entirely optional. If no tax entries exist in the client config, the invoice shows a flat total with no subtotal/tax breakdown.

When configured, tax is a list of named entries per client:
```toml
[[tax]]
name = "GST"
rate = 0.05

[[tax]]
name = "QST"
rate = 0.09975
```

This supports any jurisdiction (Canadian GST/QST/HST/PST, US sales tax, EU VAT, or a single flat rate). The PDF renders each tax line separately above the total.

### Currency
Configured per client. Determines the symbol displayed in the TUI and on the PDF.

```toml
currency = "CAD"  # Renders as CA$ or $ depending on context
```

Supported currencies are mapped to symbols at the application level (e.g., CAD → $, USD → $, EUR → €, GBP → £). Defaults to CAD if unset.

## Invoice Lifecycle
```
 [new] → draft → [edit, export preview as needed] → [mark sent] → locked
                                                          ↓
                                                   immutable forever
```

- **Draft**: Editable. Can export PDF for preview.
- **Sent**: Locked. Cannot be edited or deleted. PDF can be re-exported but content is frozen.
- Deleted draft invoice numbers are burned (never reused).

## Config & State

### Directory Structure
```
~/.config/invoice/
├── .git/                 # Git repo for backup/portability
├── config.toml           # Global config (your info, export path)
├── clients/
│   ├── acme-corp.toml    # Per-client defaults
│   └── initech.toml
└── data/
    └── invoices.json     # Invoice records (source of truth)
```

### config.toml
```toml
[user]
name = "Jane Doe"
address = "123 Main St, Montreal, QC H2X 1Y4"

[settings]
export_dir = "~/invoices"    # Where PDFs land. Default: ~/invoices
active_client = "acme-corp"  # Last-used client, auto-set
default_currency = "CAD"     # Fallback if client has no currency set
```

### Per-Client Config (e.g., acme-corp.toml)
```toml
name = "Acme Corporation"
address = "456 Business Ave, Suite 200, Toronto, ON M5V 2H1"
payment_terms = "Net 30"
currency = "CAD"

# Optional tax entries — omit entirely for no tax
[[tax]]
name = "GST"
rate = 0.05

[[tax]]
name = "QST"
rate = 0.09975

# Auto-populated from last invoice's line items
[[last_line_items]]
description = "Software development"
quantity = 5
rate = 15000

[[last_line_items]]
description = "Code review"
quantity = 2
rate = 12000
```

## Git Backup & Portability

### On First Run (Init)
1. `git init` the `~/.config/invoice/` directory
2. If `gh` CLI is detected: offer to create a private GitHub repo and set it as remote
3. If `gh` not available: inform user they can manually add a remote (`git remote add origin ...`) and tell them the path to the config directory

### Ongoing Sync
- **Auto-commit on every write operation** (save, send, delete) with a descriptive message (e.g., `"create invoice 2026-003 for acme-corp"`, `"mark 2026-002 as sent"`)
- **Auto-pull on launch** to sync from remote (if remote is configured). Fail silently if offline.
- **Auto-push after commit** (if remote is configured). Fail silently if offline.
- No daemon, no systemd — git ops are baked into the save path.
- Conflicts: if pull encounters a conflict, warn the user and let them resolve manually. This should be extremely rare since it's single-user.

## First Run Experience
1. App detects no config exists
2. Guided setup (in TUI):
   - "What is your name/business name?"
   - "What is your address?"
   - "Where should PDFs be exported?" (default: ~/invoices)
   - "Default currency?" (default: CAD)
3. Git init + optional GitHub repo setup
4. Prompts to create first client profile (name, address, payment terms, currency, tax entries)
5. Drops into main view

## TUI Views

### 1. Invoice List (Main View)
Default view on launch. Shows all invoices for the active client.

```
 Invoice                                   [client: Acme Corp | CAD]

 #         Status  Date         Description              Total
 ────────────────────────────────────────────────────────────────
 2026-003  ✓ sent  2026-03-15   Web app development      $4,500.00
 2026-002  ✓ sent  2026-02-12   API integration work     $3,200.00
 2026-001  draft   2026-01-20   Initial consulting       $1,500.00

 [n]ew  [e]dit  [d]elete  [p]df  [s]end  [c]lient  [?]help
```

- "Description" column shows first line item description (truncated)
- "Total" includes tax if configured
- Sent invoices visually distinct (dimmed or marked)

### 2. Invoice Editor
Opened via `n` (new) or `e` (edit on selected draft).

```
 New Invoice                                   [client: Acme Corp]

 Invoice #:      2026-004 (auto)
 Date:           2026-03-24
 Payment Terms:  Net 30
 ────────────────────────────────────────────────────────────────

 Line Items:
  #  Description                    Qty     Rate       Total
  1  Software development           5       $150.00    $750.00
  2  Code review                    2       $120.00    $240.00

                                             Subtotal: $990.00
                                             GST (5%): $49.50
                                           QST (9.975%): $98.75
                                               Total: $1,138.25

 [a]dd line  [e]dit line  [dd] remove line  [Ctrl+e] open $EDITOR
 [Tab] next field  [Enter] save  [Esc] cancel
```

- Line items pre-filled from client's last invoice
- Tax lines only shown if client has tax configured
- On save: updates client's `last_line_items` in config
- Attempting to edit a `sent` invoice shows: "This invoice is locked."

### 3. Client Switcher
Triggered via `c` from list view.

```
 Select Client

 > Acme Corporation
   Initech
   + Add new client...

 [j/k] navigate  [Enter] select  [Esc] back
```

### 4. Send Confirmation
Triggered via `s` on a draft invoice.

```
 ┌──────────────────────────────────────────┐
 │  Mark invoice 2026-003 as sent?          │
 │                                          │
 │  This will lock the invoice permanently. │
 │  It cannot be edited or deleted after.   │
 │                                          │
 │  [y] Yes, mark as sent    [n] Cancel     │
 └──────────────────────────────────────────┘
```

## Keybindings

### List View
| Key | Action |
|---|---|
| `j` / `k` | Navigate up/down |
| `n` | New invoice |
| `e` / `Enter` | Edit selected (drafts only) |
| `d` | Delete selected (drafts only, with confirmation) |
| `p` | Export selected to PDF |
| `s` | Mark selected as sent (with confirmation) |
| `c` | Switch client |
| `?` | Help overlay |
| `q` | Quit |

### Editor View
| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Next/prev field |
| `a` | Add new line item |
| `e` | Edit selected line item |
| `dd` | Remove selected line item |
| `j` / `k` | Navigate line items |
| `Ctrl+e` | Open `$EDITOR` for current description field |
| `Enter` | Save and return to list |
| `Esc` | Cancel and return to list |

## PDF Layout
- Filename: `{invoice_number}_{client-slug}_{date}.pdf`
  - e.g., `2026-003_acme-corp_2026-03-15.pdf`
- Exported to `export_dir` from config
- Clean, professional, minimal — no color, print-friendly
- Layout:
  - Your name/address (top left)
  - Client name/address (top right)
  - Invoice number, date, payment terms
  - Line items table: description, qty, rate, line total
  - If tax configured: subtotal → each tax line → total
  - If no tax: just total
  - Currency symbol throughout

## Technical Notes
- Language: Implementer's choice (Go or Rust recommended for single-binary distribution)
- Installable globally so `invoice` works from any directory
- All state lives in `~/.config/invoice/` — app never writes to CWD
- Invoice numbers globally sequential per year, burned on delete
- All monetary values stored as integers (cents), displayed with proper formatting
- `$EDITOR` fallback chain: `$EDITOR` → `$VISUAL` → `vi`
- Git operations should never block the UI or crash the app — always fail silently with a warning
