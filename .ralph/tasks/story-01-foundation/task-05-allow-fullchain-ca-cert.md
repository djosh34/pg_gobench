## Task: 05 Allow Fullchain Certificates In PostgreSQL CA Config <status>done</status> <passes>true</passes>

<blocked_by>.ralph/tasks/story-01-foundation/task-04-sslmode-config-param.md</blocked_by>

<description>
Must use tdd skill to complete

**Goal:** Allow `source.tls.ca_cert` to point at a PEM fullchain file, not only at a single standalone CA certificate. Operators commonly mount `tls.crt` from certificate tooling, and in this deployment that file may contain the leaf certificate followed by one or more intermediate/root CA certificates concatenated together. When such a fullchain file is configured as `source.tls.ca_cert`, pg_gobench must extract and trust the CA certificates from the chain so PostgreSQL TLS verification succeeds instead of failing with an `unknown certificate authority` error.

The implementation must continue to support a traditional CA bundle containing one or more CA certificates. It must also support a fullchain layout containing the server/leaf certificate plus multiple CA certificates in a single PEM file. The loader/database TLS setup must not assume the first certificate in the PEM file is directly usable as a trust anchor. It must parse all PEM certificate blocks, identify the certificates that can be used as trust anchors, and build the `RootCAs` pool from those certificates. If no usable CA/trust-anchor certificate exists in the configured file, fail fast with a useful error naming `source.tls.ca_cert`; do not silently continue with system roots, an empty pool, or the leaf certificate alone.

This task is limited to PostgreSQL TLS configuration for `source.tls.ca_cert`. Keep `source.tls.cert` and `source.tls.key` behavior unchanged. Do not add backwards compatibility paths or alternate config names. Do not swallow certificate parsing errors; malformed PEM, invalid certificates, unreadable files, and fullchain files without a usable CA must all surface clear errors.
</description>

<acceptance_criteria>
- [x] TDD red/green coverage proves `source.tls.ca_cert` accepts a traditional CA PEM bundle containing one or more CA certificates.
- [x] TDD red/green coverage proves `source.tls.ca_cert` accepts a fullchain PEM containing a leaf certificate followed by one or more CA/intermediate certificates, and uses the CA certificates as trust anchors.
- [x] TDD red/green coverage proves a fullchain-style PEM containing only a leaf/non-CA certificate is rejected with an error that names `source.tls.ca_cert`.
- [x] TDD red/green coverage proves malformed PEM, invalid certificate bytes, and unreadable `source.tls.ca_cert` files fail with useful errors instead of being ignored.
- [x] TLS client certificate behavior for `source.tls.cert` and `source.tls.key` remains covered and unchanged.
- [x] The implementation does not fall back to system roots or an empty root pool when `source.tls.ca_cert` is configured but unusable.
- [x] `make check` — passes cleanly
- [x] `make test` — passes cleanly (default suite; excludes only ultra-long tests moved to `make test-long`)
- [x] `make lint` — passes cleanly
- [ ] If this task impacts ultra-long tests (or their selection): `make test-long` — passes cleanly (ultra-long-only)
</acceptance_criteria>

<plan>.ralph/tasks/story-01-foundation/task-05-allow-fullchain-ca-cert_plans/2026-05-01-fullchain-ca-cert-plan.md</plan>
