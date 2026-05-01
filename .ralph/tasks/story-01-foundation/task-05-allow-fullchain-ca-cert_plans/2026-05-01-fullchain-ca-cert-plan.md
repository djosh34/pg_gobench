# Fullchain CA Cert Plan

## Scope

Implement PostgreSQL CA-file loading so `source.tls.ca_cert` can point either to a traditional CA bundle or to a fullchain PEM whose first certificate is a leaf and whose later certificates provide the usable trust anchors.

This task should:

- keep the YAML config shape unchanged
- keep `internal/config` responsible only for config-shape and cross-field validation
- keep runtime file I/O and PEM/x509 parsing in `internal/database`
- stop trusting every PEM certificate block blindly
- build `tls.Config.RootCAs` only from certificates that can act as trust anchors
- fail fast when `source.tls.ca_cert` is unreadable, malformed, contains invalid certificates, or contains no usable CA certificate
- leave `source.tls.cert` and `source.tls.key` behavior unchanged

This turn is planning-only. Execution belongs to the next turn unless the design proves incomplete and must be reopened.

## Public Interface

- `internal/config` stays unchanged at the public shape level:
  - `config.Source`
  - `config.TLS`
- `internal/database` continues exposing:
  - `Open(source config.Source) (*sql.DB, error)`
  - `CheckReadiness(ctx context.Context, db pinger) error`
- The behavioral contract for `source.tls.ca_cert` changes:
  - accept a CA bundle containing one or more CA certificates
  - accept a fullchain PEM containing leaf plus one or more CA certificates
  - reject a PEM that contains only non-CA certificates

No new config fields or backwards-compatibility branches should be added.

## Boundary Decision

Keep certificate-file parsing private to `internal/database`, but reduce that boundary so the database layer owns one coherent responsibility: converting validated runtime TLS file paths into a `tls.Config`.

This is the concrete `improve-code-boundaries` cleanup for this task:

- replace the current `AppendCertsFromPEM` trust-all behavior with one private helper that parses PEM blocks, decodes certificates, filters to CA-capable trust anchors, and returns exactly one `*x509.CertPool`
- keep config validation in `internal/config`; do not move file readability or certificate parsing into config
- keep `buildTLSConfig` focused on composing `tls.Config`, while the new helper owns CA-pool construction from `source.tls.ca_cert`
- remove any stringly “no certificates found” branch that loses the distinction between malformed PEM, invalid cert bytes, and no usable CA certificates

That addresses the `mixed-responsibilities` smell in `buildTLSConfig` and avoids a bad `validation-outside-config` move by keeping YAML validation where it already belongs.

## Design Notes

Planned runtime behavior for `source.tls.ca_cert`:

- read the configured file once
- iterate over PEM blocks in file order
- reject non-certificate PEM leftovers or malformed PEM input with an error that names `source.tls.ca_cert`
- parse every `CERTIFICATE` block with `x509.ParseCertificate`
- if any certificate block cannot be parsed, fail immediately with an error that names `source.tls.ca_cert`
- add only certificates that can be used as CA trust anchors to the root pool
- do not add the leaf/non-CA certificate from a fullchain file to `RootCAs`
- if no CA-capable certificate exists after parsing, fail with an error that names `source.tls.ca_cert`
- when `source.tls.ca_cert` is configured, always set `tls.Config.RootCAs` from the extracted pool and keep `connConfig.Fallbacks = nil`; never fall back to system roots or an empty pool

Planned trust-anchor rule:

- treat a certificate as usable for `RootCAs` only when it is CA-capable according to parsed x509 metadata
- leaf certificates without CA authority must never become trust anchors, even if they are the first or only PEM block

Planned unchanged behavior:

- `source.tls.cert` and `source.tls.key` still load through `tls.LoadX509KeyPair`
- `source.sslmode` continues to come from validated config
- `internal/config` does not gain file-system or x509 logic

## TDD Slices

Use vertical red-green slices only.

- [x] Slice 1: failing database test proving `source.tls.ca_cert` accepts a traditional CA PEM bundle and produces a usable connection config
- [x] Slice 2: failing database test proving `source.tls.ca_cert` accepts a fullchain PEM with leaf plus CA certificates and uses only the CA certificates as trust anchors
- [x] Slice 3: failing database test proving a PEM containing only a leaf/non-CA certificate is rejected with an error that mentions `source.tls.ca_cert`
- [x] Slice 4: failing database tests proving malformed PEM, invalid certificate bytes, and unreadable CA files all fail with useful `source.tls.ca_cert` errors
- [x] Slice 5: confirm existing client-certificate tests remain green and unchanged for `source.tls.cert` and `source.tls.key`
- [x] Slice 6: refactor after green to keep CA-pool parsing in one private helper and simplify `buildTLSConfig`

## File Plan

- `internal/database/database.go`
- `internal/database/database_test.go`
- `internal/database/database_internal_test.go`

Possible cleanup during execution if it improves boundaries:

- replace `hasTLS` if the final code reads more clearly with a helper that expresses “custom TLS material configured”
- extract shared test certificate builders so the tests can generate leaf/intermediate/root chains without duplicating PEM-writing logic
- keep all new helpers private to `internal/database`

## Quality Gates

- `make check`
- `make lint`
- `make test`
- no `make test-long` unless execution shows this task explicitly affects the long lane

If execution reveals that CA selection needs config-level semantics rather than runtime PEM parsing, switch this plan back to `TO BE VERIFIED` immediately.

NOW EXECUTE
