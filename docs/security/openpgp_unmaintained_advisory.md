# GO-2026-5932: the `openpgp` "unmaintained package" advisory

[GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) says that
`golang.org/x/crypto/openpgp` is **unmaintained and unsafe by design**. This is
not a bug in one version: it is introduced at version `0` with no fixed version,
the Go vulnerability database's way of saying "the whole package is deprecated;
do not use it." No upgrade of `golang.org/x/crypto` ever clears it.

It is **not a Panurus vulnerability**. `govulncheck` only raises it against code
that actually imports the package, and Panurus imports neither
`golang.org/x/crypto/openpgp` nor any of its subpackages (`packet`, `armor`,
`clearsign`, `errors`, `elgamal`, `s2k`), in any module. The risk the advisory
warns about lands on whoever *chooses* to use OpenPGP — and Panurus has not made
that choice. If OpenPGP were ever needed, the advisory's own advice is to use the
maintained drop-in fork
[`github.com/ProtonMail/go-crypto/openpgp`](https://github.com/ProtonMail/go-crypto).

## What Panurus does about it

- **The linter forbids OpenPGP.** A `depguard` rule in `.golangci.yml` denies
  importing `golang.org/x/crypto/openpgp` and
  `github.com/ProtonMail/go-crypto/openpgp` (and their subpackages), so openpgp
  cannot be introduced through first-party code without a failing lint.
- **The advisory ID is allow-listed** in
  [`ci/govulncheck-allowlist.txt`](../../ci/govulncheck-allowlist.txt) as a
  standing, pre-approved disposition (today it reports nothing). Recording it in
  advance means that if a future transitive dependency ever pulled `openpgp` into
  the build, CI would fail loudly for a maintainer to re-review rather than the
  finding appearing out of nowhere. The correct response then is to move the code
  onto `github.com/ProtonMail/go-crypto/openpgp` and drop the entry, not to lean
  on the allow-list.

## See also

- [govulncheck and the allow-list](../development/tools.md#govulncheck)
- [`ci/govulncheck-allowlist.txt`](../../ci/govulncheck-allowlist.txt)
- [Security Policy](../../SECURITY.md) — how to report a real vulnerability.
