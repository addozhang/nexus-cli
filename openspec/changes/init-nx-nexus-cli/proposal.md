# Proposal: init-nx-nexus-cli

## Why

Developers working with multiple Sonatype Nexus Repository 3 instances (both OSS and Pro deployments) currently have to open a browser to answer everyday questions: which repositories exist, where is a component, what versions are available, and what are the details of a specific asset. There is no terminal-native, scriptable way to query Nexus across formats (Maven, npm, PyPI, Cargo, Go, Docker). This change introduces `nx`, a read-only query CLI modeled on the engineering practices of the `jk` Jenkins CLI.

## What Changes

- New Go CLI binary `nx` (Go 1.22+, cobra, standard-library-first stack) — **greenfield; no existing code is modified**.
- Instance management via alias: `nx auth add <alias>` (alias mandatory) interactively stores host/port/username/token in a local credentials file (`0600`); one instance can be marked as default; queries without `--instance <alias>` use the default.
- Read-only query commands organized format-first (format as top-level resource namespace, like `kubectl <resource> get`):
  - `nx repo list` — list repositories on an instance, filterable by format and type (hosted/proxy/group).
  - `nx maven|npm|pypi|cargo|go|docker search <query>` — component search per format.
  - `nx maven|npm|pypi|cargo|go|docker versions <component>` — version list for a component.
  - `nx maven|npm|pypi|cargo|go|docker info <asset-or-component>` — asset/component detail (size, checksums, last-downloaded, repository location).
- Stable self-owned output schema: YAML by default, `-o json` optional, every structured output begins with `schemaVersion: "1"`; breaking changes bump the version.
- Connectivity support for any host:port and self-signed TLS certificates (`SSL_CERT_FILE` honored, `--insecure` last resort); works identically against OSS and Pro instances (no Pro-specific API handling in MVP).
- Scripting-friendly exit codes: `0` success, `10` nx-level error.

## Capabilities

### New Capabilities
- `instance-auth`: Alias-based instance registry and credential storage — add/list/remove/select-default instances, secure token storage, credential resolution at command time, TLS trust configuration.
- `repo-query`: Cross-format repository listing (`nx repo list`) with format/type filters and stable output schema.
- `format-search`: Per-format component search commands (`search`) for maven/npm/pypi/cargo/go/docker, mapping unified query input onto each format's coordinates and the Nexus Search API.
- `format-versions`: Per-format component version listing (`versions`) for maven/npm/pypi/cargo/go/docker.
- `format-info`: Per-format asset/component detail output (`info`) for maven/npm/pypi/cargo/go/docker.
- `output-schema`: The external output contract — `schemaVersion`, field reference, stability tiers (stable/experimental), YAML/JSON rendering, exit-code contract.

### Modified Capabilities
- None. Greenfield project; no existing specs under `openspec/specs/`.

## Impact

- New repository layout: `cmd/nx/` (wiring only), `internal/{cli,nexus,nexusurl-free auth,schema,output,errors}` business packages, `test/integration/` (httptest-based), `docs/schema.md`.
- New dependencies (all justified in design.md): `spf13/cobra`, `sigs.k8s.io/yaml`, `BurntSushi/toml`. HTTP via `net/http`; tests via stdlib `testing` + `httptest`.
- No impact on existing systems: this is a new standalone CLI; it only reads from Nexus REST APIs (`/service/rest/v1/...`) and never mutates state.
