# Grafana dashboard

`token-sdk.json` is a Grafana dashboard for the Token SDK metrics catalogued in
[../../metrics.md](../../metrics.md). It targets a Prometheus data source
scraping the Fabric Smart Client metrics endpoint exposed by the application.

## Import

1. In Grafana, **Dashboards → New → Import**.
2. Upload `token-sdk.json` (or paste its content).
3. When prompted, select the Prometheus data source that scrapes your
   FSC/Token-SDK metrics endpoint.

## Variables

The dashboard exposes four template variables, all multi-select with `All`
default:

- `network` / `channel` / `namespace` — filter to a specific TMS.
- `method` — filter the driver-service panels to a specific method
  (e.g. `Issue`, `Transfer`).

## Panels

| Row | Panels |
|-----|--------|
| Driver services (decorator layer) | Ops rate by service, error rate by service, per-method p95 duration |
| Transaction lifecycle (ttx) | Endorsed / audit-approved / accepted rate, phase duration p95 |
| Finality listener | Confirmed / deleted / hash-mismatch / retry-exhausted rate, `OnStatus` p50/p95 |
| Versioned envelope sessions (ttx) | Sent/received by type, errors by reason, body size p95 |
| Auditor service | Audit / append p95, lock conflicts, append errors, releases |
| Token selection (sherdlock) | Selection p95, outcomes, unspent-token fetches, immediate retries p95 |
| Certification + identity caches | Certified / errors / dropped rate, request p95, cache levels, pending buffer |
| Fabric-X finality queue | Pending events gauge, drops + errors rate, processing p95 |

## Caveats

- Histogram quantiles use a 5-minute sliding window; tune via panel queries
  if your scrape interval differs from the default.
- The dashboard only renders panels for which the corresponding metrics are
  emitted by the running application — services not enabled in your build
  will simply show "No data".
- Panels do not yet cover the metrics flagged as gaps in
  [../../metrics.md#coverage-gaps-and-recommended-additions](../../metrics.md#coverage-gaps-and-recommended-additions);
  add them once the underlying instrumentation lands.
