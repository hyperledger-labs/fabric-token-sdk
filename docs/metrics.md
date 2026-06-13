# Token SDK Metrics Reference

This document catalogs every metric emitted by the Token SDK and, in the final
section, identifies areas of the codebase that are not yet instrumented.

Metrics are produced through the metrics provider supplied by the
[Fabric Smart Client](https://github.com/hyperledger-labs/fabric-smart-client)
(Prometheus-compatible). See [development/monitoring.md](development/monitoring.md)
for the high-level monitoring setup, and [drivers/metrics.md](drivers/metrics.md)
for the design of the driver-service decorator layer summarized in
[Driver services](#driver-services) below.

A ready-to-import Grafana dashboard covering all the metrics in this document
is available at [monitoring/grafana/token-sdk.json](monitoring/grafana/token-sdk.json)
(see [monitoring/grafana/README.md](monitoring/grafana/README.md) for import
instructions and the panel layout).

Metric types follow Prometheus conventions:

- **Counter** — monotonically increasing total.
- **Gauge** — value that can go up or down.
- **Histogram** — distribution of observed values (durations, sizes).

---

## Driver services

Defined in `token/core/common/metrics/`. Each core driver service is wrapped by a
decorator that records, per method invocation, a call counter, a duration
histogram, and an error counter. All carry the labels `network`, `channel`,
`namespace`, `method`, so metrics can be filtered per TMS and per operation
(e.g. `Issue`, `Transfer`). The duration histograms are native (sparse) Prometheus
histograms.

| Metric | Type | Description |
|--------|------|-------------|
| `issue_service_operations_total` | Counter | Total IssueService method invocations |
| `issue_service_duration_seconds` | Histogram | Duration of IssueService method calls |
| `issue_service_errors_total` | Counter | Total IssueService method errors |
| `transfer_service_operations_total` | Counter | Total TransferService method invocations |
| `transfer_service_duration_seconds` | Histogram | Duration of TransferService method calls |
| `transfer_service_errors_total` | Counter | Total TransferService method errors |
| `tokens_service_operations_total` | Counter | Total TokensService method invocations |
| `tokens_service_duration_seconds` | Histogram | Duration of TokensService method calls |
| `tokens_service_errors_total` | Counter | Total TokensService method errors |
| `auditor_service_operations_total` | Counter | Total AuditorService method invocations |
| `auditor_service_duration_seconds` | Histogram | Duration of AuditorService method calls |
| `auditor_service_errors_total` | Counter | Total AuditorService method errors |
| `tokens_upgrade_service_operations_total` | Counter | Total TokensUpgradeService method invocations |
| `tokens_upgrade_service_duration_seconds` | Histogram | Duration of TokensUpgradeService method calls |
| `tokens_upgrade_service_errors_total` | Counter | Total TokensUpgradeService method errors |

Labels (all): `network`, `channel`, `namespace`, `method`.

## Transaction lifecycle (ttx)

Defined in `token/services/ttx/metrics.go`. Counts and times the main phases of a
token transaction. Labels (all): `network`, `channel`, `namespace`.

| Metric | Type | Description |
|--------|------|-------------|
| `endorsed_transactions` | Counter | Number of endorsed transactions |
| `audit_approved_transactions` | Counter | Number of transactions approved by the auditor |
| `accepted_transactions` | Counter | Number of accepted transactions |
| `endorsement_duration_seconds` | Histogram | Duration of the full endorsement collection phase (signatures, audit, chaincode approval) |
| `audit_approval_duration_seconds` | Histogram | Duration of the auditor approval phase (validation, append, signing) |
| `ordering_duration_seconds` | Histogram | Duration of the broadcast to the ordering service |

## Finality listener (ttx)

Defined in `token/services/ttx/finality/metrics.go`. Tracks the listener that
reacts to on-ledger finality. No labels.

| Metric | Type | Description |
|--------|------|-------------|
| `finality_listener_confirmed_total` | Counter | Transactions confirmed on the ledger and committed to local storage |
| `finality_listener_deleted_total` | Counter | Transactions marked deleted due to an invalid ledger status or token-request hash mismatch |
| `finality_listener_hash_mismatch_total` | Counter | Transactions rejected because the committed token-request hash did not match the local one |
| `finality_listener_retry_exhausted_total` | Counter | Transactions abandoned after all finality-processing retries were exhausted |
| `finality_listener_on_status_duration_seconds` | Histogram | Total OnStatus processing time per transaction, including retries |

## Versioned envelope sessions (ttx)

Defined in `token/services/utils/json/session/metrics.go`. Instruments the
versioned envelope protocol used by interactive ttx views.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ttx_envelope_sent_total` | Counter | `version`, `type` | Versioned envelopes sent |
| `ttx_envelope_received_total` | Counter | `version`, `type` | Versioned envelopes received |
| `ttx_envelope_errors_total` | Counter | `error` | Envelope validation errors |
| `ttx_envelope_body_bytes` | Histogram | `type` | Size of the envelope body in bytes |

## Auditor service

Defined in `token/services/auditor/metrics.go`. Instruments the auditor's
`Audit`/`Append`/`Release` path (distinct from the driver-level
`auditor_service_*` metrics above). No labels.

| Metric | Type | Description |
|--------|------|-------------|
| `auditor_audit_duration_seconds` | Histogram | Audit() processing time per transaction, including lock acquisition |
| `auditor_audit_lock_conflicts_total` | Counter | Audit() calls that failed to acquire enrollment-ID locks |
| `auditor_append_duration_seconds` | Histogram | Append() processing time per transaction |
| `auditor_append_errors_total` | Counter | Append() calls that failed to write to the audit database |
| `auditor_releases_total` | Counter | Release() calls (explicit and deferred) |

## Token selection (sherdlock)

Defined in `token/services/selector/sherdlock/metrics.go`.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `unspent_tokens_invocations` | Counter | `fetcher_type` | Number of unspent-token fetch invocations |
| `selection_duration_seconds` | Histogram | — | Duration of a token selection call |
| `selection_outcome_total` | Counter | `outcome` | Token selection outcomes by result type |
| `selection_immediate_retries` | Histogram | — | Distribution of immediate retry counts per selection call |

## Certification (interactive)

Defined in `token/services/certifier/interactive/metrics.go`.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `certified_tokens` | Counter | `network`, `channel`, `namespace` | Number of tokens certified |
| `certification_request_duration_seconds` | Histogram | `channel`, `namespace` | Certification batch request durations |
| `certification_errors_total` | Counter | `channel`, `namespace` | Failed certification request attempts |
| `certification_pending_tokens` | Gauge | `channel`, `namespace` | Tokens waiting in the certification input buffer |
| `certification_dropped_tokens_total` | Counter | `channel`, `namespace` | Tokens dropped because the certification buffer was full |

## Identity caches

| Metric | Type | Labels | Source | Description |
|--------|------|--------|--------|-------------|
| `cache_level` | Gauge | `network`, `channel`, `namespace` | `token/services/identity/idemix/cache/metrics.go` | Fill level of the Idemix credential cache |
| `recipient_data_cache_level` | Gauge | `network`, `channel`, `namespace` | `token/services/identity/role/metrics.go` | Fill level of the wallet recipient-data cache |

## Fabric-X finality queue

Defined in `token/services/network/fabricx/finality/queue/metrics.go`. No labels.

| Metric | Type | Description |
|--------|------|-------------|
| `finality_queue_pending_events` | Gauge | Finality events currently waiting in the queue buffer |
| `finality_queue_enqueue_drops_total` | Counter | Finality events dropped because the queue was full |
| `finality_queue_processing_errors_total` | Counter | Errors returned by `event.Process` in worker goroutines |
| `finality_queue_processing_duration_seconds` | Histogram | Successful event processing time in worker goroutines |

---

## Coverage gaps and recommended additions

The metrics above concentrate on transaction processing (driver services, the ttx
lifecycle, finality, auditing, selection). Several layers are currently
uninstrumented. The items below are ordered by impact.

> Note: the Fabric Smart Client already instruments view execution (per-view
> count and duration) at the platform layer. The suggestions below intentionally
> stay domain-specific (per-store / per-operation / per-phase counters) so they
> add semantics the FSC view instrumentation cannot infer, rather than
> duplicating it.

### 1. Storage / persistence layer (highest priority)

`token/services/storage/...` (token DB, transaction DB, identity/wallet DB,
keystore) emits **no metrics**. Reads, writes, deletes, query latency, and
backend errors are invisible, even though this is where every token and
transaction is ultimately persisted, and the backend (sqlite vs postgres) is a
common performance and failure boundary.

Recommended:

- `storage_operations_total{store, operation, backend}` — counter (store =
  tokendb/ttxdb/identitydb/keystore; operation = read/write/delete/query).
- `storage_operation_duration_seconds{store, operation, backend}` — histogram.
- `storage_errors_total{store, operation, backend}` — counter.

These complement, rather than duplicate, what the database backend already
exposes (e.g. `pg_stat_statements` via `postgres_exporter`). The SDK-level
metrics carry semantic labels the DB layer cannot infer without per-query
parsing — `store=tokendb, operation=write` instead of an opaque
`INSERT INTO tokens` row count — and they are the only source of metrics when
the backend is sqlite or another embedded store with no exporter. Write-rate
counters here also provide the natural signal for detecting anomalous
token-insertion activity, which application-level metrics cannot currently see.

### 2. Distributed lock manager

The pluggable auditor `Locker` (`token/services/storage/auditdb/locker/`,
memory and postgres) has no metrics. The auditor surfaces only
`auditor_audit_lock_conflicts_total` at the call site; the lock manager itself is
opaque.

Recommended:

- `auditor_lock_acquisitions_total{result}` — counter (result =
  acquired/contended/failed).
- `auditor_lock_wait_duration_seconds` — histogram.
- `auditor_lock_leases_renewed_total` and `auditor_lock_lost_total` — counters
  (the postgres locker already detects lease loss via `ErrLockLost`).
- `auditor_locks_held` — gauge of currently held anchors.

### 3. Endorser / approval path (standard Fabric network)

Only the Fabric-X finality queue is instrumented. The standard Fabric endorser
path — `EndorserService` in `token/services/network/fabric/endorsement/provider.go`
(`ReceiveTx`, `Endorse`, `CollectEndorsements`) and `RequestApprovalView` in
`token/services/network/fabric/endorsement/fsc/initiator.go` — emits no
domain-specific metrics. The ttx layer only times ordering via
`ordering_duration_seconds`.

Recommended (per-operation semantic counters, not generic view timing — FSC
already covers that):

- `endorser_requests_total{operation, result}` — counter (operation =
  receive_tx / endorse / collect_endorsements; result = success / failure).
- `endorser_operation_duration_seconds{operation}` — histogram.
- `approval_requests_total{result}` and `approval_request_duration_seconds`
  for `RequestApprovalView`, labelled by `network`/`channel`/`namespace` like
  the other lifecycle counters.

Note: double-spend is enforced at commit time by Fabric / the token chaincode,
not by the SDK validators, so it does not belong on this list as an SDK-level
metric.

### 4. Transaction-level failure counter

The lifecycle exposes `endorsed`, `audit_approved`, and `accepted` counters but
no symmetric failure counter; failures are only visible indirectly through
per-service `*_errors_total` and `finality_listener_deleted_total`.

Recommended: `transactions_failed_total{phase, reason}` with the same
`network`/`channel`/`namespace` labels as the other lifecycle counters, so success
and failure rates can be compared directly.

### 5. Wallet / identity resolution

Only cache fill levels are gauged (`cache_level`, `recipient_data_cache_level`).
There are no counters for wallet lookups or signer/verifier resolution, so cache
hit/miss ratios cannot be derived.

Recommended: `wallet_lookups_total{result}` (hit/miss) and a resolution duration
histogram.
