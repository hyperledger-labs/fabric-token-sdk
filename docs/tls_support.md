/*
TLS Support Documentation
----------------------

The Fabric Token SDK now includes TLS (Transport Layer Security) support for the PostgreSQL storage driver. This documentation explains how to configure and use TLS in your token services.

## Configuration
Add a `tls` block under the persistence configuration in your `fsc.yaml` (or any configuration source used by the SDK).
```yaml
fsc:
  persistences:
    mydb:
      driver: postgres
      opts:
        dsn: "host=localhost port=5432 user=postgres dbname=token sslmode=disable"
        tls:
          enabled: true
          ssl_mode: "require" # Options: disable, require, verify-full, verify-ca
          # Optional paths – provide absolute or relative to the working directory
          root_cert_path: "/path/to/ca.pem"   # CA certificate for server verification
          cert_path: "/path/to/client.crt"    # Client certificate (for mutual TLS)
          key_path: "/path/to/client.key"     # Private key matching the client cert
          server_name: "db.mycompany.com"    # Override server name for verification (optional)
```

### Field Descriptions
- `enabled` (bool): Enables TLS handling. If `false` or omitted, the driver behaves as before.
- `ssl_mode` (string): Same values as the PostgreSQL `sslmode` parameter.
  - `disable` – No TLS.
  - `require` – TLS without server verification (`InsecureSkipVerify`).
  - `verify-full` – Verify server cert and hostname.
  - `verify-ca` – Verify server certificate against provided CA, but skip hostname verification.
- `root_cert_path` – Path to a PEM‑encoded CA certificate file. Required for `verify-full` and `verify-ca`.
- `cert_path` & `key_path` – Paths to client TLS certificate and private key for mutual authentication. Optional; required only when a client certificate is needed.
- `server_name` – Custom server name for hostname verification. Overrides the host part of the DSN.

## How It Works
1. The `tlsConfigProvider` decorates the original PostgreSQL config provider.
2. When a persistence’s `tls.enabled` flag is true, the provider loads the TLS configuration based on the fields above.
3. A unique connection name is generated (e.g., `pgx_config_<hash>`), and the TLS config is registered with `pgx/v5/stdlib`.
4. The driver uses this registered connection name, ensuring all connections use the configured TLS settings.

## Testing
The `tls_test.go` file includes unit tests that:
- Verify each `ssl_mode` behaves as expected.
- Test error cases such as missing certificates or unsupported modes.
- Confirm provider fallback to default TLS settings when a specific persistence does not define TLS.

Run the tests with:
```bash
go test -v ./token/services/storage/db/sql/postgres/...
```

## Compatibility
- Existing configurations without a `tls` block continue to work unchanged.
- The TLS implementation uses `pgx/v5` which is already a dependency of the SDK.
- Ensure your PostgreSQL server is configured for TLS if you enable `require`, `verify-full`, or `verify-ca`.

## References
- PostgreSQL documentation on SSL: https://www.postgresql.org/docs/current/ssl-tcp.html
- pgx/v5 TLS handling: https://github.com/jackc/pgx/tree/v5

---
*This documentation was added in PR #1746 (branch `chore/tls-support`).*
