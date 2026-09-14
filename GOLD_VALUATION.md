# Gold reference valuation

Each gold account has **估值设置** in its existing gold detail page. Choose cost (default), manual reference price, or automatic feed. Manual price uses CNY/gram, an explicit reference time and a validity period of 1–10080 minutes. Save uses an independent optimistic version. Changing display settings never updates Account.Balance, investment cost/quantity, transactions, or realized P&L.

Account overview (desktop/mobile), home asset totals and institution groups replace the gold cost with a valid estimate once. Transaction entry, reconciliation, journal exports and financial posting continue using book balances. Missing, failed, stale, mismatched-position or mismatched-cost valuations have no numeric overlay: the UI explicitly labels cost fallback. Hidden amounts and existing excluded/hidden-account aggregation rules remain in place. Page refreshes request server estimates roughly once per minute while visible; these requests read the configured snapshot endpoint, not a new upstream collector.

## Operator configuration

In [user], set investment_quote_provider (cmb_platform or gold_json), investment_quote_name, investment_quote_url and investment_quote_max_age_minutes (1–10080; default30). Supply a Bearer credential using EBKCFP_USER_INVESTMENT_QUOTE_TOKEN pointing to a private read-only file. The deployment controls network access; browsers receive only source label/availability, never URL or token. Never expose credentials via browser configuration, use TLS validation, and scope the endpoint credential to quote reads. Redirects, query-string tokens and URL userinfo are refused. No arbitrary browser-provided URL fetching.

cmb_platform consumes the existing /v1/quotes/latest observation contract with CNY/g customer sell price, successful last attempt, timezone-aware fetched_at, raw SHA256, source cmb_spot and source series. MarketDay and QuotedPrice are preserved as flags; neither guarantees execution. Raw NowTime is not asserted to be last-price-update time.

gold_json permits other deployments to supply their own fixed endpoint without the NAS/CMB platform. It returns an object with source (nonempty display provenance), unit (exactly CNY/g), customerSell (positive decimal string), fetchedAt (RFC3339 with timezone). Example values belong in isolated tests; production must provide actual source data. Currency conversion and USD/oz conversion are not performed. Source-specific flags are optional and never described as CMB flags for another source.

## API and persistence

Authenticated GET investments/valuation-settings.json?id=positionId returns user-owned settings and provider availability. POST same endpoint saves positionId/version/mode/manualPrice/manualAsOf/maxAgeMinutes. GET investments/valuation.json?id=positionId returns one estimate; GET investments/valuations.json returns the user's estimates and shares a single upstream fetch within that request. HTTP transport failures and invalid quotes produce unavailable status and null marketValue, not prior quote fallback. Monetary results are decimal cent strings calculated with integer arithmetic.

One additive table investment_valuation_setting stores display settings. Old positions need no edits: missing row means cost. Database backups include settings; portable ledger export does not include this custom metadata. Existing data/schema are not rewritten. Rollback to earlier images does not erase the table, but UI loses valuation; restore must include consistent database/attachments/config/secret backups if needed. No changes to banking transfers, collection cadence or investment research.
