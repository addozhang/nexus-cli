# Tasks: init-nx-nexus-cli

## 1. Project Scaffolding

- [x] 1.1 Initialize Go module (`go.mod`, module path), `cmd/nx/main.go` wiring-only entry, Makefile (build/test/lint/fmt targets mirroring jk)
- [x] 1.2 Add dependencies: `spf13/cobra`, `sigs.k8s.io/yaml`, `BurntSushi/toml`; configure `.golangci.yml` strict preset
- [x] 1.3 Create `internal/` package skeletons: cli, nexus, auth, format, schema, output, errors
- [x] 1.4 Set up GitHub Actions CI: test (-race, coverage ≥70% on core packages) + lint gates

## 2. Errors and Output

- [x] 2.1 Implement `internal/errors`: error classes (URL/auth/network/TLS/response/coordinate/alias/flag), translation layer, exit-code mapping to `10`
- [x] 2.2 Write table-driven unit tests for `internal/errors` class mapping
- [x] 2.3 Implement `internal/output`: YAML (default) and JSON renderers injecting `schemaVersion: "1"`; reject unknown `-o` values with exit `10`
- [x] 2.4 Unit tests for renderers (preamble first line in YAML, field equality in JSON, stderr/stdout separation)

## 3. Instance Registry (instance-auth)

- [x] 3.1 Implement credentials store: TOML at `~/.config/nx/credentials`, mode `0600`, dir `0700`; alias CRUD; default marker semantics (remove default clears marker)
- [x] 3.2 Unit tests for store round-trip, permissions, default-marker lifecycle
- [x] 3.3 Implement resolution chain: `--instance` flag → stored default → env override (`NX_URL`/`NX_USERNAME`/`NX_TOKEN`, all-or-nothing) → error listing aliases
- [x] 3.4 Unit tests for resolution precedence matrix
- [x] 3.5 Implement TLS configuration: system roots, `SSL_CERT_FILE`, global `--insecure` with stderr warning
- [x] 3.6 Implement `nx auth add <alias>` (mandatory alias, interactive hidden token input, `/service/rest/v1/status` verification before save), `nx auth list` (never prints tokens), `nx auth remove`, `nx auth default`
- [x] 3.7 Integration tests (httptest.Server): add success/bad-creds/unreachable, list hides tokens, remove-default scenario

## 4. Nexus Client

- [x] 4.1 Implement `internal/nexus` HTTP client: base URL joining, basic auth, JSON decode into raw structs, error translation by status code (401/403/404/5xx/timeouts)
- [x] 4.2 Implement repositories listing endpoint wrapper
- [x] 4.3 Implement search endpoint wrapper with `continuationToken` pagination and limit cap
- [x] 4.4 Implement components detail endpoint wrapper (`/service/rest/v1/components/{id}`)
- [x] 4.5 Integration tests with recorded fixtures: pagination exhaustion, limit capping, 401/404/500 paths, malformed response

## 5. Format Adapters

- [x] 5.1 Define `FormatAdapter` interface and registry; wire format names to subcommands via a shared command constructor
- [x] 5.2 Implement maven adapter grammar (`groupId:artifactId[:version]`) with parse-error tests naming offending segment
- [x] 5.3 Implement npm/pypi/cargo/go adapter grammar (`name[@version]`) with parse-error tests
- [x] 5.4 Implement docker adapter grammar (`image[:tag]`) with parse-error tests

## 6. Query Commands (repo-query, format-search, format-versions, format-info)

- [x] 6.1 Implement `nx repo list` with `--format`/`--type` client-side validation and stable-field schema mapping (name/format/type/url/exposed)
- [x] 6.2 Implement `nx <format> search` across six formats: coordinate parsing, Search API call, pagination, `--limit` flag, empty-result-is-success contract
- [x] 6.3 Implement `nx <format> versions`: component-constrained search, stable fields (version/repository/lastModified), unknown-component-empty-list behavior, identifier-required validation
- [x] 6.4 Implement `nx <format> info`: version-required validation, component + assets detail mapping (path/size/checksums/lastDownloaded with explicit null), multi-repo occurrence listing, absent-coordinate exit `10`
- [x] 6.5 Integration tests per spec scenario for repo-query, format-search, format-versions, format-info against httptest fixtures

## 7. Schema Contract Docs

- [x] 7.1 Write `docs/schema.md`: field reference for all commands, stability tiers, versioning policy, exit-code table, default search limit value — before merging query commands
- [x] 7.2 Verify every rendered field in output matches docs/schema.md exactly (docs-first audit task)

## 8. Quality Gates

- [x] 8.1 Raise core-package coverage to ≥70%: internal/auth, internal/format, internal/schema, internal/output, internal/errors, internal/nexusurl-free resolution logic
- [x] 8.2 `make lint` zero warnings; `make test` green with no race detections
- [x] 8.3 Manual smoke pass superseded by the automated e2e suite (group 9)

## 9. E2E Against Real Nexus (docker, -tags=e2e)

- [x] 9.1 test/e2e/up.sh: provision sonatype/nexus3 container (wait for readiness, rotate admin password, accept CE EULA, disable anonymous access)
- [x] 9.2 Seed six hosted repositories + two maven component versions; idempotent re-runs
- [x] 9.3 e2e suite covering repo list (+filters), maven search/versions/info against real data, per-format search compatibility, limit/pagination, auth lifecycle, bad-credentials exit code, env override
- [x] 9.4 Makefile targets e2e-up / e2e-down / e2e-test
- [x] 9.5 Nightly CI workflow (.github/workflows/e2e.yml); PR gate unchanged
- [x] 9.6 Wire-format corrections found by e2e: search param & responses use "maven2" (normalized to "maven"), assets carry "checksum"/"fileSize", sort=desc rejected (client-side sorting), env-override credential propagation bug fixed
