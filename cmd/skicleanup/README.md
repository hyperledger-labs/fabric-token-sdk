# skicleanup

`skicleanup` is a diagnostic command-line tool for inspecting orphaned signer entries in a Panurus token database.

An **orphaned signer** is an entry in the `Signers` identity table whose identity is not referenced by any token in the `Tokens` table (neither spent nor unspent). These entries may accumulate over time and represent identities for which the corresponding cryptographic keys can safely be removed from the keystore.

## Build

```bash
make skicleanup
```

The binary is installed to `$GOPATH/bin/skicleanup`.

## Commands

### `config example`

Prints a fully-annotated YAML configuration file to stdout. Use this to bootstrap a new configuration:

```bash
skicleanup config example > config.yaml
```

No flags required.

### `signers`

Iterates all entries in the `Signers` table in batches, unmarshals each identity, and checks whether it appears in the `Tokens` table. For every orphaned signer (not found in `Tokens`), the command prints the identity hash and the derived Subject Key Identifiers (SKIs) to stdout.

This is a **read-only** operation. No data is modified or deleted.

```bash
skicleanup signers --config <path-to-config.yaml> [--batch-size <n>]
```

**Flags:**

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--config` | Yes | — | Path to the YAML configuration file |
| `--batch-size` | No | `1000` | Number of signer rows read per database query |

**Output format** (one line per orphaned signer):

```
<identity-hash>: [<ski1>, <ski2>, ...]
```

## Configuration

The tool reads a YAML file that describes how to connect to the target database.
Generate a starter file with:

```bash
skicleanup config example > config.yaml
```

### SQLite example

```yaml
driver: sqlite
dataSource: /var/lib/panurus/node/data.db
tablePrefix: ""
```

### PostgreSQL example

```yaml
driver: postgres
dataSource: "host=db.example.com port=5432 user=panurus password=secret dbname=panurus sslmode=require"
tablePrefix: "prod_"
```

## SKI Extraction

SKIs are derived from each orphaned identity using the same extraction logic as the runtime cleanup service:

| Identity type | Extraction method |
|---------------|-------------------|
| `idemix` | Extracts SKI from the Idemix NymPublicKey |
| `idemixnym` | Looks up signer info in the identity store, then extracts SKI from the embedded Idemix signature |
| `x509` | Returns no SKIs (X.509 key material is not stored in the Panurus keystore) |
| Any other type | SHA-256 hash of the raw identity bytes (fallback) |

## Environment variables

Configuration values can be overridden with environment variables prefixed `CORE_`, using `_` in place of `.`:

```bash
CORE_DATASOURCE="postgres://..." skicleanup signers --config config.yaml
```
