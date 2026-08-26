# Design: init-nx-nexus-cli

## Context

`nx` is a greenfield Go CLI for read-only queries against Sonatype Nexus Repository 3 instances (OSS and Pro). It borrows the engineering constitution of the `jk` Jenkins CLI: standard-library-first stack, alias/credential isolation from business logic, a self-owned stable output schema, and strict test/lint gates. Unlike `jk`, identity is **instance-alias based**, not URL based: users register instances once (`nx auth add <alias>`) and every subsequent query names an alias or falls back to a default instance.

Nexus exposes a versioned REST API under `/service/rest/v1/`. Relevant read endpoints:

| Endpoint | Used for |
|---|---|
| `GET /service/rest/v1/repositories` | repo list |
| `GET /service/rest/v1/status` | connectivity/whoami checks (`auth add` verification) |
| `GET /service/rest/v1/security/users` (or `/status`) | identity check during `auth add` |
| `GET /service/rest/v1/search?format=...&q=...&group=...&name=...&version=...` | component search + version lists |
| `GET /service/rest/v1/search/assets?...` | asset-level search (checksums, sizes) |
| `GET /service/rest/v1/components/{id}` | component detail |
| `GET /service/rest/v1/assets?repository=...` | asset browse fallback |

The same API surface exists in OSS and Pro for these endpoints; `nx` performs no capability detection in MVP.

## Goals / Non-Goals

**Goals:**

- Read-only querying of six formats (maven, npm, pypi, cargo, go, docker) plus cross-format repository listing.
- Alias-based multi-instance support with a default instance.
- Stable, scriptable output: YAML default, JSON optional, `schemaVersion: "1"` first line.
- Works against any host:port with username/token auth and self-signed TLS certificates.
- Engineering parity with `jk`: cobra wiring, `internal/` business packages, table-driven tests, httptest integration tests, ≥70% coverage on core packages, zero-lint gate.

**Non-Goals:**

- Any write operation (upload, delete, cleanup, tag, staging).
- Nexus administration (users, roles, tasks, firewall/IQ data).
- Capability detection or Pro-specific features.
- Interactive TUI, daemon, watch mode.

## Decisions

### D1: Tech stack mirrors `jk`

Go 1.22+; `spf13/cobra` for CLI; `gopkg.in/yaml.v3` for YAML; `BurntSushi/toml` for the credentials file; stdlib `net/http`; stdlib `testing` + `httptest`; `golang.org/x/term` for hidden token entry.

*Why*: proven combination in `jk`, with one deliberate deviation — `jk` uses `sigs.k8s.io/yaml`, but it marshals through JSON and sorts keys alphabetically, which cannot guarantee the `schemaVersion`-first-line contract. `yaml.v3` preserves struct field order, so every type keeps `SchemaVersion` as its first field. Avoids third-party Nexus clients whose response shapes would leak into our schema and couple us to plugin/version drift. *Alternative considered*: a Nexus client library — rejected for schema-isolation reasons identical to `jk`'s rationale.

### D2: Instance registry replaces URL-as-identity

Credentials live at `~/.config/nx/credentials` (TOML, mode `0600`), keyed by mandatory alias:

```toml
[instances.prod]
url = "https://nexus.example.com:8443"
username = "deployer"
token = "..."
default = false

[instances.dev]
url = "http://localhost:8081"
username = "admin"
token = "..."
default = true
```

- `nx auth add <alias>` prompts interactively for URL, username, token (hidden input); verifies connectivity via `/service/rest/v1/status` before saving.
- `nx auth list`, `nx auth remove <alias>`, `nx auth default <alias>` complete the registry. Tokens are never printed by any command.
- Resolution order at query time: `--instance <alias>` flag → stored default instance → error listing available aliases.
- Environment overrides (`NX_URL`, `NX_USERNAME`, `NX_TOKEN`) bypass the store entirely for ephemeral use; when set they take precedence over flags-less resolution only if no explicit flag is given. (Exact precedence pinned in the spec.)

*Why alias over URL*: users think in instance names, not URLs; it also removes jk's entire context-path/prefix-matching complexity, which is unnecessary when the user names the instance explicitly. *Alternative*: URL-as-identity like jk — rejected after interview; users will not paste URLs for routine queries.

### D3: Format-first commands with a shared adapter layer

Formats are top-level resource namespaces:

```
nx <format> search <query> [--version v] ...
nx <format> versions <component-coordinate>
nx <format> info <component-or-asset-coordinate>
nx repo list [--format F] [--type hosted|proxy|group]
```

Each supported format implements a small internal adapter interface:

```go
type FormatAdapter interface {
    ParseSearch(query string) (SearchParams, error)
    ParseComponent(coord string) (ComponentRef, error)
    ComponentKey(ref ComponentRef) string // canonical key for output
}
```

Adapters translate user-facing coordinate syntax onto the common Search API parameters (`group`, `name`, `version`, `q`, `format`). Coordinate syntaxes differ per format (e.g., maven `group:artifact[:version]`, npm/pypi/cargo/go `name[@version]`); each adapter owns its grammar and its parse errors.

Docker is special-cased internally but keeps the same command shape: search resolves image names via the Search API where the instance indexes docker components; `versions` maps to tag listing (Search API `name=` + version facets; falls back documented in spec if an instance returns empty for docker).

*Why adapters*: keeps `internal/cli` thin and makes adding a seventh format a single package addition. *Alternative*: one generic `nx search -f maven` — rejected; interview confirmed format-first reads more naturally to the target users.

### D4: Self-owned output schema with stability tiers

All structured output starts with `schemaVersion: "1"`. Schema types live in `internal/schema` with mappers from raw Nexus JSON; renderers live in `internal/output` (YAML default, `-o json`). Field reference and stability tiers (`stable` / `experimental`) are documented in `docs/schema.md` **before implementation**, following `jk`'s docs-first rule. Breaking changes bump `schemaVersion`; additive changes do not.

Exit codes: `0` success; `10` nx-level error (bad coordinates, unknown instance, auth failure, network/TLS failure, malformed Nexus response). stderr messages go through `internal/errors` translation.

### D5: TLS posture identical to `jk`

System roots are the default; `SSL_CERT_FILE` pointing at a PEM bundle enables self-signed CA trust without flags. `--insecure` (global flag) disables verification and prints a warning. No per-instance TLS config in MVP.

### D6: Package layout

```
cmd/nx/main.go              # wiring only
internal/cli/               # cobra commands: auth.go, repo.go, format.go
internal/nexus/             # HTTP client, CSRF-free read calls, pagination
internal/auth/              # credentials IO, alias resolution
internal/format/            # FormatAdapter implementations (maven.go, npm.go, ...)
internal/schema/            # schema types + mappers
internal/output/            # YAML/JSON renderers
internal/errors/            # NxError translation layer
test/integration/           # httptest.Server fixture tests
docs/schema.md              # output contract
```

## Risks / Trade-offs

- [Docker search coverage varies by Nexus version/plugin] → Spec defines behavior for empty result sets explicitly ("no results" is a valid answer, not an error); integration fixtures cover both indexed and non-indexed responses.
- [Search API pagination defaults may truncate large result sets] → `nexus` client follows `continuationToken` until exhaustion or a `--limit` cap; default limit pinned in schema doc.
- [Interactive token entry complicates automation] → Env-var override path (D2) covers CI without weakening the interactive flow.
- [Per-format coordinate grammars invite ambiguity, e.g. pypi names containing `-`] → Each adapter documents its grammar in command help; unparseable input exits `10` with the offending token named; grammar pinned in specs so changes require a change proposal.
- [Plaintext token storage] → Same accepted threat model as `jk` and `~/.aws/credentials`: trusted developer machine, mode `0600`. Keychain integration deferred until requested.

## Migration Plan

Greenfield; nothing to migrate. Rollback = delete the binary and `~/.config/nx/`.

## Open Questions

_None blocking_. Deferred by design: OS-keychain credential storage (revisit post-first-release); additional formats beyond the initial six (adapter interface makes them additive).
