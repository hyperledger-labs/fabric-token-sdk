# Dependency License Dispositions

## Intro

This page records the disposition of third-party dependency licenses that an
automated license scan flagged as `FORBIDDEN`, `CAUTION`, or `UNKNOWN`, together
with the licenses it failed to detect at all. Each entry states the verified
SPDX identifier, the upstream `LICENSE` reference, and the decision taken.

The scope is the flags raised against the dependency graph as of issue
[#2461](https://github.com/LFDT-Panurus/panurus/issues/2461). A flag here means
the scanner's label, not necessarily a real policy problem — several entries are
scanner artefacts recorded so they are not re-investigated from scratch next
time.

License facts below were confirmed by reading the `LICENSE` file shipped in each
package (Go: the module cache under `$(go env GOMODCACHE)`; Python: the package's
PyPI distribution), not taken from the scanner's output.

## Go dependencies

### `github.com/hashicorp/go-uuid` — REPLACED

- **Scanner flag:** MPL-2.0, undetected.
- **Verified license:** MPL-2.0 (SPDX: `MPL-2.0`).
  Upstream: <https://github.com/hashicorp/go-uuid/blob/master/LICENSE>.
- **Disposition:** Replaced with `github.com/google/uuid` (BSD-3-Clause, SPDX:
  `BSD-3-Clause`; upstream
  <https://github.com/google/uuid/blob/master/LICENSE>), which was already in
  the dependency graph as an indirect dependency, so the swap introduces no new
  module.

The two call sites that used `go-uuid` were:

- `token/services/storage/db/sql/common/transactions.go` (3 calls)
- `token/services/storage/services/recovery/manager.go` (1 call)

`uuid.GenerateUUID()` (returning a `string`) was replaced with
`uuid.NewRandom()` followed by `.String()`. Both produce an RFC-4122
version-4 (random) UUID in the canonical 36-character hyphenated lowercase
form, so persisted identifier values are unchanged in format. The explicit
error handling on the underlying `crypto/rand` read is preserved — `NewRandom`
returns an error rather than panicking, unlike `uuid.NewString()`.

Removing the direct dependency and running `go mod tidy` dropped `go-uuid`
from every module that carried it (10 `go.mod`/`go.sum` files across the
repository).

### License undetected — recorded as a scanner detection gap

The scanner failed to detect a license for roughly 60 Go modules
(`go.uber.org/*`, `golang.org/x/*`, `google.golang.org/*`, `fabric-smart-client`,
`jackc/*`, OpenTelemetry, and others). This is a detection gap, not a licensing
problem: these modules carry standard permissive licenses.

A representative sample was spot-checked by reading each module's `LICENSE`:

| Module | SPDX | Upstream `LICENSE` |
| --- | --- | --- |
| `go.uber.org/zap` | `MIT` | <https://github.com/uber-go/zap/blob/master/LICENSE> |
| `golang.org/x/sync` | `BSD-3-Clause` | <https://github.com/golang/sync/blob/master/LICENSE> |
| `google.golang.org/protobuf` | `BSD-3-Clause` | <https://github.com/protocolbuffers/protobuf-go/blob/master/LICENSE> |
| `github.com/jackc/pgxlisten` | `MIT` | <https://github.com/jackc/pgxlisten/blob/master/LICENSE> |

**Disposition:** Recorded as a class — the undetected Go modules are
Apache-2.0 / BSD / MIT. No per-package action is required. If the scanner is
replaced or reconfigured, re-run detection rather than maintaining this list by
hand.

## Python dependencies

These packages belong to the documentation toolchain (`mkdocs` and
`mkdocs-material`, pinned in `requirements.in`) and the benchmark tooling under
`cmd/benchmarking/`. None are shipped as part of Panurus itself.

### `pandas` — recorded as a scanner artefact

- **Scanner flag:** `FORBIDDEN`.
- **Verified license:** BSD-3-Clause (SPDX: `BSD-3-Clause`).
  Upstream: <https://github.com/pandas-dev/pandas/blob/main/LICENSE>.
- **Cause of the flag:** pandas bundles a `LICENSES/` directory containing the
  licenses of vendored components; read as a single blob, that directory trips a
  `FORBIDDEN` match even though pandas itself is BSD-3-Clause.
- **Disposition:** Scanner artefact, not a policy violation. pandas is benchmark
  tooling only (`cmd/benchmarking/`), declared in `requirements.in`, and is not
  shipped. No change.

### `certifi`, `pathspec` — MPL-2.0, retained

- **Scanner flag:** `CAUTION`, MPL-2.0.
- **Verified licenses:**
  - `certifi` — MPL-2.0 (SPDX: `MPL-2.0`); upstream
    <https://github.com/certifi/python-certifi/blob/2026.10.18/LICENSE>.
    Pulled in via `requests`.
  - `pathspec` — MPL-2.0 (SPDX: `MPL-2.0`); upstream
    <https://github.com/cpburnz/python-pathspec/blob/v1.1.1/LICENSE>.
    Pulled in via `mkdocs`.

  Both URLs are pinned to the exact version in use. `pathspec` relicensed to
  dual MIT/MPL-2.0 on its `master` branch after `v1.1.1`, so a `master`-branch
  link no longer resolves to the single `LICENSE` file this version ships.
- **Disposition:** Both are unmodified docs-toolchain dependencies used as-is
  and not shipped. MPL-2.0's copyleft is file-level and triggers only on
  modification of the licensed files, which does not happen here. Retained. No
  change.

### `python-dateutil` — dual license recorded

- **Scanner flag:** `UNKNOWN` ("Dual License").
- **Verified license:** Apache-2.0 OR BSD-3-Clause (SPDX:
  `Apache-2.0 OR BSD-3-Clause`).
  Upstream: <https://github.com/dateutil/dateutil/blob/master/LICENSE>.
- **Disposition:** The dual license is recorded; either disjunct is acceptable.
  Pulled in via `ghp-import` and `pandas`. No change.

## Summary of changes

- **Code/dependency change:** `go-uuid` → `google/uuid` (4 call sites, 2 files;
  dependency removed from 10 `go.mod`/`go.sum` files).
- **Recorded, no change:** `pandas` (scanner artefact), `certifi` and `pathspec`
  (MPL-2.0, docs toolchain), `python-dateutil` (dual license), and the ~60
  undetected Go modules (permissive — detection gap).
