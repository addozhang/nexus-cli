# nx

A read-only query CLI for Sonatype Nexus Repository 3.

`nx` is for the day-to-day developer: list repositories, search components,
list versions of a dependency, inspect asset details — across Maven, npm,
PyPI, Cargo, Go, and Docker — all from the terminal, without opening a browser.

**What nx is**

- Instance aliases: register each Nexus instance once (`nx auth add <alias>`),
  mark a default, and query by name — no URLs in daily commands.
- Read-only: every command is a pure query. `nx` never mutates your instances.
- Format-first: formats are top-level resource namespaces (`nx maven ...`,
  `nx npm ...`), like `kubectl <resource> get`.
- Stable, self-owned schema: `yaml` (default) or `json`. Structured output
  begins with `schemaVersion: "1"`; breaking changes bump the version.
- Scripting-friendly: empty result sets exit `0`; any nx-level failure exits
  `10`; diagnostics go to stderr so stdout pipes stay clean.
- Works against OSS and Pro alike — both expose the same query APIs.
- Agent-ready: ships an optional skill that teaches AI coding agents how to
  use `nx` safely from the terminal ([skills/nx-nexus-cli](./skills/nx-nexus-cli)).

**What nx is not**

- Not an administration tool — no users, roles, tasks, cleanup policies, or
  staging management.
- Not a publisher — no artifact upload, deletion, or tagging.
- Not a proxy client — talk to your build tools; let them resolve from Nexus.

## Disclaimer

> [!WARNING]
> This project started as a hands-on study of the Nexus Repository 3 RESTful API and as a testbed for driving such a CLI from AI coding agents. Every command is a read-only query, but the credentials `nx` holds may hold more power than `nx` ever uses — in production, scope the account to read-only and manage permissions tightly.

## Install

### Download a pre-built binary

Grab `nx_<version>_<os>_<arch>.tar.gz` from the
[Releases page](https://github.com/addozhang/nexus-cli/releases), extract,
and move `nx` onto your `PATH`.

### go install

```sh
go install github.com/addozhang/nexus-cli/cmd/nx@latest
```

Requires Go 1.22+. For a private fork, set `GOPRIVATE=github.com/addozhang/*`
first.

### Homebrew

```sh
brew install addozhang/tap/nx
```

## Quick start

```sh
# 1. Register an instance (prompts for URL, username, token; verifies
#    connectivity before saving)
nx auth add prod

# The first registered instance becomes the default automatically.
# Change it anytime:
nx auth default dev

# 2. List repositories on the default instance
nx repo list

# 3. Search components
nx maven search com.example:foo
nx npm search lodash --limit 20

# 4. List versions of a component (newest first)
nx pypi versions requests
nx docker versions myteam/app

# 5. Inspect a specific version: assets, sizes, checksums
nx cargo info tokio@1.38.0 -o json

# 6. Query a non-default instance explicitly
nx repo list --instance staging
```

## Instance management

```sh
nx auth add <alias>       # interactive registration + connectivity check
nx auth list              # registered instances (tokens are never printed)
nx auth default <alias>   # set the instance used when --instance is absent
nx auth remove <alias>
```

Credentials live in `~/.config/nx/credentials` (TOML, mode `0600`). Tokens are
never printed by any command.

### OS keyring storage

By default tokens are written to the credentials file. To keep the token in
the operating system keyring instead (macOS Keychain, Windows Credential
Manager, Linux Secret Service):

```sh
nx auth add prod --secure-storage
# or opt in per shell: export NX_SECURE_STORAGE=1
```

The token is then stored under the instance alias in the OS keyring and never
touchs the credentials file — the file only records `secure = true` for that
alias, and `nx auth list` marks secure instances with `(keyring)`. Re-running
`nx auth add <alias>` without `--secure-storage` moves the token back to the
file and deletes the keyring entry; `nx auth remove <alias>` deletes both.

Resolution order at query time:

1. `--instance <alias>` flag
2. Environment override — all three of `NX_URL`, `NX_USERNAME`, `NX_TOKEN`
   must be set; partial sets are ignored
3. Stored default instance
4. Failure listing known aliases (exit `10`)

### Self-signed certificates

Point `SSL_CERT_FILE` at a PEM bundle of your instance's CA:

```sh
export SSL_CERT_FILE=/etc/ssl/my-ca-bundle.pem
nx repo list
```

Pass `--insecure` only as a last resort; it disables all certificate
verification and prints a warning.

## Coordinate grammars

Each format has its own positional grammar:

| Format | Grammar |
|---|---|
| maven | `groupId:artifactId[:version]` |
| npm / pypi / cargo / go | `name[@version]` |
| docker | `image[:tag]` |

Free-form input (no separators) becomes a full-text query.

- `search` accepts anything, version optional.
- `versions` needs only the component identity.
- `info` **requires a version** — absence exits `10` and suggests running
  `versions` first.

## Scripting with nx

### JSON output

```sh
nx repo list -o json | jq '.repositories[].name'
nx maven versions com.example:foo -o json | jq '.versions[0].version'
nx cargo info tokio@1.38.0 -o json | jq '.components[0].assets[].checksums'
```

### Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success — including empty result sets |
| `10` | nx-level error: unknown alias, bad coordinates, auth rejection, network/TLS failure, malformed response, invalid flags |

stderr carries the failure class (`auth:` / `coordinate:` / `instance:` /
`network:` / `response:` / `tls:` / `flag:`); stdout carries only rendered
schema output.

### Schema pinning

All structured output begins with `schemaVersion: "1"`. Breaking changes bump
the version; additive changes do not. See
[`docs/schema.md`](./docs/schema.md) for the full field reference, stability
tiers, and versioning policy.

## Develop

```sh
make build              # build ./bin/nx
make test               # unit + integration with -race
make e2e-up             # provision a local Nexus container (docker)
make e2e-test           # end-to-end suite against that container
make lint               # golangci-lint v2, zero warnings required
make release-snapshot   # local cross-platform snapshot via GoReleaser
make help               # full target list
```

Requires Go 1.22+, golangci-lint v2, and (for e2e) Docker.

## Docs

- [`docs/schema.md`](./docs/schema.md) — output schema contract (field
  reference, stability tiers, versioning policy).
- [`openspec/`](./openspec/) — change management; behavior changes go through
  OpenSpec.
- [`skills/nx-nexus-cli/`](./skills/nx-nexus-cli/) — operating guide for AI
  coding agents using `nx`.
