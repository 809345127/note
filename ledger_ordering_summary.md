1. `AUTO_INCREMENT` + auto `create_time` in MySQL cannot guarantee commit order under concurrency.
2. If you need strict order, use a serialized mechanism (for example a sequence lock), or a separate `commit_seq` column as the true order key.
3. This strict ordering reduces insert concurrency because sequence allocation becomes a bottleneck.
4. In your code (`ledger.go`), `BalanceSn` is generated with `balance.BalanceSn + 1` under `SELECT ... FOR UPDATE`, then persisted in the same transaction.
5. That design is correct for strict ordering **within one `balance_id` stream** (your business ledger requirement).
6. Recommended hardening: add DB constraint `UNIQUE(balance_id, balance_sn)`.
7. `create_time` is useful but not a strict ordering source; use `balance_sn` as the authoritative order.
8. Bigger `balance_sn` usually means not-earlier `created_at`, but not 100% guaranteed in extreme cases (clock changes, bypass writes, precision ties).
9. For statements: filter by the chosen time axis (`created_at` / `posted_at` / `occurred_at` / `effective_date`) and always `ORDER BY balance_sn` (plus tie-breaker like `id`).
10. Redesign suggestions: keep `balance_sn`, add `posted_at`, `occurred_at`, `effective_date`, grouping/idempotency/reversal fields for stronger auditability.
11. `posted_at` is not always “most important”; depends on purpose:
- ledger booking time: `posted_at`
- business event time: `occurred_at`
- accounting period: `effective_date`
- strict per-balance order: `balance_sn`
12. For “when this money income is recognized in ledger,” `posted_at` is the right primary field, with `balance_sn` for exact sequence.
