# Gold investment assets — development candidate


## Deployed patch: 1.6.1-gold.2 (2026-09-14)

Commit `d4b54021` filters unset/nonpositive transaction IDs before investment ownership checks. A zero related ID previously matched unrelated investment postings and blocked ordinary cash transactions for users with positions. Regression first failed on old code, then passed after the fix; full Go suite and frontend build passed.

NAS PostgreSQL rehearsal verified ordinary income/expense/transfer create, read, modify and delete, with balances restored. Actual investment P/L modification/deletion and direct cost-account writes remain rejected. Deployed image ID: `sha256:85a0f1fe3b12d20f4c66efdff1e6fbcf8d072efc853607e7a3609df742266a26`. Existing 21 tables, attachments and secrets were preserved, native DeepSeek configuration retained. Post-upgrade encrypted backup was restored to an independent database/storage and 19 ledger-related table hashes, attachments and backed-up secrets matched. Real iPhone testing and an independent off-NAS backup remain outside this verified patch result.

Base: upstream v1.6.1 (`6ccd0c462100828c78e203792a5b2feb8d569039`). Branch: `feature/investment-assets`. Custom release `1.6.1-gold.1` was deployed to the authorized amd64 NAS after isolated rehearsal on 2026-09-14. It is not an official upstream release. Deployed code revision: `7645b870`.

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

## Deployment verification and remaining limitations

- The NAS PostgreSQL 17.11 backup was restored independently. All 16 pre-existing table fingerprints and original attachment files were unchanged by additive initialization. The pre-existing test account could log in and read its original transactions and receipt.
- The synthetic gold lifecycle passed on that restored NAS instance. Its 21 tables and attachment files were then restored again into an independent database/storage; table fingerprints, quantities, cost, realized gains, revision history and authenticated receipt bytes matched.
- Production was upgraded with the original database/storage mounts and application/database secrets. Validation ran on the isolated database network before restoring the original edge network. No real gold positions were initialized. Backup timer and registration policy were retained.
- With investment assets enabled, the old clear-all-data API returns a protection error before any deletion. This is an explicit temporary restriction, not a newly implemented coordinated clear workflow. With the feature disabled, existing managed positions still prevent legacy clearing.
- Runtime image: `nihplod/ezbookkeeping:1.6.1-gold.1`; image content ID `sha256:c7d96ab2c8c83e88df970adfb57df4495de14544ac7ef8d8beaa82fd42f66b82`. This is a locally built image ID, not a registry manifest digest. A Docker image archive was retained separately for recovery.
- Axios was updated to 1.20.0 for reported security fixes. Remaining npm findings involve build tooling (Sass/Immutable, PostCSS/Nanoid, Vite and other development chains); the runtime image contains the Go binary and compiled public assets, not a Node build server. This is not a claim that all dependencies have a clean security audit.

Portable versioned investment export/import, safe managed-account metadata edits, legacy transaction deep links and more detailed per-operation change previews remain follow-ups. Ordinary CSV is not a complete gold backup. Existing populated accounts are not converted; included-fee amounts must be normalized explicitly in the current form. The optional quote adapter is not yet connected to the live NAS data platform.

Real iPhone PWA session/background/network-retry checks still require user acceptance. No offline saving or unbounded exactly-once guarantee is claimed. This deployment does not implement the remaining convenience and portability items in the original design.

Full database + attachment + controlled secret backup remains the recovery foundation. Do not downgrade a live gold ledger to the official binary. Restore an explicitly selected compatible backup into an isolated target first; never erase the current ledger to hide upgrade problems.

## Asset definitions and unified entry (gold.7)

“资产定义” is a peer of Accounts/Categories/Tags/Templates/Schedules in desktop Basis Data and the corresponding mobile settings list. Definitions are per owner: stable ID, display name, rule kind, unit name/symbol and quantity precision (0–12). Current rule kind is gold only; stock-specific operations remain unsupported. Units describe the entered quantities and never perform conversions. Automatic CNY/g feeds apply only to gold with unit g; other configured units can use a matching manual per-unit reference price or cost display.

Existing gold positions resolve to the deterministic per-owner `gold-<uid>` definition (黄金、克/g、12-digit maximum precision) without rewriting any historical quantity, cost, transaction or valuation preference. The default definition is virtual until explicitly saved. Newly created holdings require a selected definition in the UI and receive a persistent binding. Old clients omitting definitionId retain the legacy gold definition. Name can change; unit name, unit symbol, rule kind and precision are locked once used by a holding (including an empty holding), under the same owner write lock as holding creation. No rebind or conversion of existing positions is offered. Closed/zero positions do not unlock history. The additive definition/binding tables are included in database backups; plain transaction CSV is not a complete export.

Add Transaction now offers “资产买卖” alongside the ordinary-entry choice. Desktop and mobile reuse GoldLedger in trade-only mode: choose holding, buy/sell, actual quantity, gross consideration, separately charged fee, cash account and categories; preview then confirm. Detail entry uses the same component and existing atomic service. No new ordinary TransactionType, duplicated cash write API, stock engine, price-to-fill shortcut or AI asset posting is introduced. Ordinary drafts are not translated into asset orders. Saved transactions are bank activities already performed by the user; this application does not execute bank trades. Existing investment-account categories such as Yu'e Bao remain ordinary monetary accounts unless explicitly bound to a quantity ledger.
