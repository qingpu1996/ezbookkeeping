# Gold investment assets — development candidate

Base: upstream v1.6.1 (`6ccd0c462100828c78e203792a5b2feb8d569039`). Branch: `feature/investment-assets`. This is a local development candidate, not an official release or a production upgrade.

## Scope

Gold only, CNY settlement, grams with up to 12 decimal places, moving average cost (`MA_v1`). No trade execution, inferred holdings, investment AI, other asset workflows, or scheduled jobs. Existing cash balances remain integer minor currency units. Market value never overwrites ledger cost.

Bind a new empty CNY investment account to a position. Enter an opening quantity and cost without debiting cash, or record actual purchases/sales. Purchase costs include separately entered fees; sale proceeds subtract fees. The UI currently requires gross amounts excluding separately charged fees. Do not enter a fee twice. Existing populated investment accounts are not automatically converted.

## Enable in an isolated environment

Use the existing build and database initialization flow with a separate database, storage directory and generated application secret. Do not copy a production account into the preview.

```ini
[user]
enable_investment_assets = true
# Optional, operator-controlled data-platform endpoint; empty by default.
investment_quote_url =
```

The feature is off by default. Guards protecting managed accounts/postings remain active when the UI/API flag is off. Do not run an older official binary against a database containing investment records: it lacks these guards.

Optional quote credentials use the existing configuration-file override mechanism, `EBKCFP_USER_INVESTMENT_QUOTE_TOKEN`. Never embed tokens in a URL. The adapter consumes the existing CMB observation contract, rejects stale/failed/invalid responses and does not silently use an old quote. No provider is configured by default.

Desktop page: `/desktop#/investment/list`; mobile page: `/mobile#!/investment/list`. Both use the existing application login. Browser mobile mode is not real iPhone PWA validation.

## Storage and API

Five additive XORM models: InvestmentPosition, InvestmentRevision, InvestmentPosting, InvestmentCommand, InvestmentAttachment. Instrument metadata is currently on the position, rather than a separate catalogue. No new library or plugin framework is required.

Operations use an expected position version and a persistent `(uid, request_key)` unique command key. Corrections append revisions, replay the position, and regenerate only affected monetary postings in the same database transaction. Preview executes the same validation and rolls back. Historical overselling or invalid account/category references reject the entire command. User-row serialization coordinates ordinary financial writes and managed account binding. The current upstream datastore puts user and ledger tables in the same database; this is not a distributed/sharded locking design.

Authenticated endpoints, all below `/api/v1/investments/`:

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `list.json` | Owned positions |
| GET | `get.json?id=…` | Facts, revisions, current projection and receipt links |
| POST | `add.json` | Bind an empty cost account |
| POST | `preview.json` | Validate and preview without committing |
| POST | `save.json` | Buy, sell, opening, correction or cancellation |
| POST | `attachments/add.json` | Bind an owned uploaded receipt |
| GET | `pictures/:fileName` | Authenticated image bytes using Authorization header |
| GET | `valuation.json?id=…` | Optional reference valuation; no ledger mutation |

Amounts and IDs are JSON strings; timestamps are Unix seconds, `utcOffset` is minutes. Quantity is a decimal string. Login identity controls ownership; caller-supplied user IDs are not used. Ordinary API tokens retain upstream permissions: they are **not read-only tokens** and must not be supplied to a research model.

Managed financial records cannot be changed through ordinary transaction, bulk, import or account mutation services. The initial conservative guard also prevents renaming the managed account; safe metadata editing is still a follow-up. Original transaction pages currently report the protection error rather than linking directly to the operation.

## Verification on 2026-09-14

- Exact arithmetic tests: opening 10 g / CNY 9000, buying 5 g for 4800 + 5 fee, selling 3 g for 3000 − 5 fee yields 12 g, cost 11044, realized profit 234. These are synthetic fixtures.
- Lifecycle passed on SQLite, PostgreSQL 17.6 and MySQL 8.4.6: preview rollback, correction replay, failure after removing affected postings rolls back, same-key concurrency, distinct-key competing sales, oversell rejection, cancellation rejection, legacy write protection and cross-user receipt denial.
- Full offline upstream Go suite passed with `BUILD_PIPELINE=1`. A separate unrestricted run failed on the upstream Romania exchange-rate live network test; this external test is not claimed as passed.
- Frontend: 38,462 tests passed, TypeScript checking, targeted ESLint and production build passed.
- Actual desktop and mobile-route browser fixture entry/readback passed. Responsive viewport checked at 390 × 844. Real iPhone and production NAS remain untested for this branch.
- Uploaded a self-made TEST image; authenticated readback matched its SHA-256. Missing token was rejected with upstream HTTP 400 / `token is empty`, not image bytes.
- PostgreSQL dump/restore compared positions, balances, revisions and links in a separate database. Separately, the local SQLite preview was backed up with its attachment files and restored in an isolated container; HTTP readback matched records and actual image bytes. This does not claim a PostgreSQL + attachment + NAS end-to-end restore.

The Vite chunk dependency option now includes dependencies recursively. The old false setting produced a circular initialization runtime error despite a successful build; the true setting was verified in the browser. See [Rolldown's option contract](https://rolldown.rs/reference/OutputOptions.codeSplitting).

## Release gates still open

1. Portable versioned investment export/import and round-trip validation. Ordinary CSV is not a complete gold backup; do not import both cash CSV and investment operations and double-count them.
2. Versioned business migration/release packaging and image digest, NAS isolation upgrade/restore rehearsal, actual quote-source integration. Current migration only adds model tables via existing Sync2.
3. Direct links/protection hints from old transaction pages, safe account metadata edits, detailed per-operation preview differences, included-fee input convenience and broader old-client/CLI/import regression coverage.
4. The old clear-all API performs multiple independent service calls. The financial clear is transactionally guarded, but a concurrent new position bind can still race with earlier nonfinancial template clearing. A coordinated clear/bind operation must be addressed before production release.
5. Dependency audit: npm reported 12 inherited findings (3 moderate, 9 high); applicability has not been reviewed. No blind dependency upgrade was performed.
6. Real iPhone PWA login/session/background/retry behavior and user acceptance. No offline save guarantee.

Full database + attachment + controlled secret backup remains the recovery foundation. Do not downgrade a live gold ledger to the official binary. Restore an explicitly selected compatible backup into an isolated target first; never erase the current ledger to hide upgrade problems.
