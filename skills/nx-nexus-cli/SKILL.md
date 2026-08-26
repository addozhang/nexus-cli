---
name: nx-nexus-cli
description: Query Sonatype Nexus Repository from AI coding agents using the `nx` CLI. Use this skill whenever the user mentions Nexus Repository, artifact registries, Maven/npm/PyPI/Cargo/Go/Docker components, dependency versions, component checksums, or asks an agent to search components, list repository contents, or inspect asset details on a Nexus instance from a terminal.
---

# nx Nexus CLI

Use `nx` to query Sonatype Nexus Repository 3 instances from the terminal.

## Core Model

`nx` is a **read-only** query tool:

- Instances are identified by **alias**, not URL. Users register instances once with `nx auth add <alias>`; queries name an alias or fall back to the stored default.
- Formats are top-level resource namespaces (`nx maven ...`, `nx npm ...`, `nx docker ...`), like `kubectl <resource> get`.
- Output is stable and self-owned. Prefer `-o json` for agent parsing, YAML for humans. Every structured output starts with `schemaVersion: "1"`.
- There are no write or administration commands: no upload, delete, cleanup, user, role, or task management. Do not look for them.
- It works identically against OSS and Pro instances; no capability detection is needed.

## First Move

When Nexus work is requested:

1. Check whether `nx` is available with `nx version` or `nx --help`.
2. If the command shape is unclear, run `nx <format> --help` rather than guessing.
3. Ask which instance alias to use if the user has not provided one; run `nx auth list` to see registered instances.
4. Prefer `-o json` for any output you will parse or summarize.
5. Never paste URLs or hosts into queries — that is what aliases are for.

## Command Reference

```sh
nx auth add <alias>          # interactive: URL, username, token (verified before saving)
nx auth list                 # registered instances; tokens never printed
nx auth default <alias>      # set the default instance
nx auth remove <alias>

nx repo list [--format F] [--type hosted|proxy|group]
nx maven  search <query>  [--limit N]
nx npm    search <query>  [--limit N]
nx pypi   search <query>  [--limit N]
nx cargo  search <query>  [--limit N]
nx go     search <query>  [--limit N]
nx docker search <query>  [--limit N]
nx <format> versions <component>
nx <format> info <component-version>
```

Global flags on every query command:

```text
-o yaml | -o json     output format (yaml is default)
--instance <alias>    override the default instance
--insecure            disable TLS verification (last resort; prints a warning)
```

## Coordinate Grammars

Each format has its own positional grammar. Unparseable input exits `10`; fix the coordinate instead of retrying variants blindly.

| Format | Search / versions / info grammar |
|---|---|
| maven | `groupId:artifactId[:version]` |
| npm, pypi, cargo, go | `name[@version]` |
| docker | `image[:tag]` |

Free-form input without structural separators becomes a full-text query (`q`). Structured input maps onto exact constraints.

- `search`: version segment optional everywhere.
- `versions`: needs only the component identity (e.g. `com.example:foo`, `lodash`, `myteam/app`).
- `info`: **version required**. A missing version exits `10` and suggests running `versions` first — do not guess a version.

## Common Workflows

### Locate a Component

```sh
nx maven search com.example:foo -o json
nx npm search lodash -o json
```

Empty results exit `0` with an empty list — absence is a valid answer, not an error. Report "not found on this instance" rather than treating it as a failure.

### List Versions of a Component

```sh
nx pypi versions requests -o json
nx docker versions myteam/app -o json
```

Versions come back sorted newest-first with the hosting repository per version. Use this before `info` when the user has not pinned a version.

### Inspect a Specific Component Version

```sh
nx cargo info tokio@1.38.0 -o json
```

Returns one entry per repository occurrence, each with assets: path, size, checksums (sha1/sha256/md5 as reported), lastModified, lastDownloaded. A never-downloaded asset reports explicit `null` — do not interpret it as an error.

Unlike `search`/`versions`, an absent coordinate exits `10`. That distinction matters: empty search results mean "no matches"; an `info` failure means the named artifact does not exist as given.

### Discover Repositories

```sh
nx repo list -o json
nx repo list --format docker --type proxy -o json
```

Use this when the user asks "what repositories exist" or before searching an unusual format.

## Instance Resolution

Precedence order:

1. `--instance <alias>` flag
2. Complete environment override — all three of `NX_URL`, `NX_USERNAME`, `NX_TOKEN` (partial sets are ignored)
3. Stored default instance
4. Failure listing known aliases (exit `10`)

If a query fails with `no default instance configured`, run `nx auth list`; if the needed instance is missing, ask the user to run `nx auth add <alias>` in an interactive terminal — registration prompts for URL, username, and token interactively and verifies connectivity before saving. Do not try to script the interactive flow by piping secrets unless the user explicitly provides them in the environment.

## Credentials and TLS

Credentials live in `~/.config/nx/credentials` (TOML, mode `0600`). No command prints tokens.

Security rules:

- Do not ask users to paste tokens into chat if an interactive terminal path exists.
- Do not read or print the credentials file.
- Do not include tokens in command lines, logs, or summaries.
- For private CAs prefer `SSL_CERT_FILE=/path/to/ca.pem` env in front of the command.
- Use `--insecure` only as a last resort and say plainly that it disables certificate verification.

## Output Handling

Prefer `-o json` whenever you will parse results:

```sh
nx repo list -o json | jq '.repositories[].name'
nx npm versions lodash -o json | jq '.versions[0]'
```

Structured output begins with `schemaVersion: "1"`. If writing automation around `nx`, check the field reference in docs/schema.md and pin the schema version before relying on fields. Diagnostics always go to stderr, so stdout pipes stay clean even when nothing matched.

## Safety

Every query command is read-only and safe to run freely:

```sh
nx repo list ...
nx <format> search ... / versions ... / info ...
nx auth list ...
```

The only state-changing commands touch the local machine, not the instance:

```sh
nx auth add ...        # stores credentials locally after verifying connectivity
nx auth remove ...     # deletes stored credentials
nx auth default ...    # changes which alias flag-less queries hit
```

Confirm with the user before removing an instance's stored credentials or changing the default when they did not explicitly ask. If `nx` itself cannot answer something (uploading artifacts, managing users, cleanup policies), say so — do not attempt raw API calls against the instance on your own initiative.

## Error Triage

All nx-level failures exit `10` with a class prefix on stderr. Follow the class:

- `instance:` unknown alias or no default — run `nx auth list`, use `--instance <alias>` or set a default.
- `coordinate:` bad format grammar — fix the coordinate using the grammar table above.
- `auth:` rejected credentials — token may be revoked; ask the user to re-add the instance.
- `tls:` certificate verification failed — propose `SSL_CERT_FILE` pointing at the instance CA bundle before suggesting `--insecure`.
- `network:` connection failures — check the URL stored under the alias via `nx auth list`.
- `response:` unexpected HTTP status or malformed response — may be a proxy or permission issue; report the status code shown.
- `flag:` invalid `-o`, `--format`, or `--type` values — use exactly the supported values.

## Agent Response Pattern

When reporting findings back to the user:

1. State the instance alias and the command(s) run.
2. State what was found (or explicitly that nothing matched).
3. Surface the useful detail: latest version, hosting repository, checksums, size.
4. Suggest the smallest next action: another query, registering an instance, or fixing a coordinate.
5. Mention if inspection was limited by missing auth, unknown alias, or unavailable `nx`.
