# nx Output Schema

External output contract for `nx`. All structured output begins with
`schemaVersion: "1"`.

## Versioning policy

- **Breaking changes** to any `stable` field (rename, type change, removal,
  semantic change) bump `schemaVersion`.
- **Additive changes** (new optional field, new `experimental` field) do not
  bump it.
- New fields land tagged `experimental` and are promoted to `stable` no
  earlier than one minor release later.

## Output formats

| Flag | Default | Notes |
|---|---|---|
| `-o yaml` | yes | YAML document; first line is `schemaVersion: "1"` |
| `-o json` | no | Indented JSON; pipes into `jq` |

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Command succeeded (including empty result sets) |
| `10` | nx-level error: unknown alias, bad coordinates, auth rejection, network error, TLS verification failure, malformed Nexus response, invalid flags |

Diagnostics go to stderr only; stdout carries only rendered schema output.

## Instance resolution

Precedence order at query time:

1. `--instance <alias>` flag
2. Complete environment override — all three of `NX_URL`, `NX_USERNAME`,
   `NX_TOKEN`; partial sets are ignored
3. Stored default instance (`nx auth default <alias>`)
4. Failure listing known aliases (exit `10`)

Credentials live in `~/.config/nx/credentials` (TOML, mode `0600`). Tokens are
never printed by any command.

TLS: system roots by default; set `SSL_CERT_FILE` to a PEM bundle for
self-signed CAs; `--insecure` disables verification with a stderr warning.

## Search grammar and limits

Per-format coordinate grammars (part of the spec):

| Format | Grammar |
|---|---|
| maven | `groupId:artifactId[:version]` |
| npm / pypi / cargo / go | `name[@version]` |
| docker | `image[:tag]` (tag separator must follow the last `/`) |

Free-form input (no structural separators) is sent as a full-text query `q`.

- **Default search limit** (`stable`): `50` components; override with
  `--limit <n>`. Pagination follows Nexus `continuationToken` until the result
  set is exhausted or the limit is reached.

## Command fields

### `nx repo list`

Each entry of `repositories[]`:

| Field | Type | Tier | Notes |
|---|---|---|---|
| `name` | string | stable | repository name |
| `format` | string | stable | e.g. `maven`, `npm`, `docker` |
| `type` | string | stable | `hosted`, `proxy`, `group` |
| `url` | string | stable | repository base URL |
| `exposed` | bool \| null | stable | null when the instance does not report it |

### `nx <format> search`

Each entry of `results[]`:

| Field | Type | Tier | Notes |
|---|---|---|---|
| `id` | string | experimental | opaque Nexus component id |
| `group` | string | stable | empty for formats without grouping |
| `name` | string | stable | |
| `version` | string | stable | |
| `repository` | string | stable | hosting repository name |
| `format` | string | stable | |

### `nx <format> versions`

Top level: `group` (string), `name` (string). Each entry of `versions[]`,
sorted newest first:

| Field | Type | Tier | Notes |
|---|---|---|---|
| `version` | string | stable | |
| `repository` | string | stable | first-seen repository holding this version |
| `lastModified` | timestamp \| null | stable | newest asset modification time; null when no assets report one |

### `nx <format> info`

One entry per repository occurrence in `components[]`:

| Field | Type | Tier | Notes |
|---|---|---|---|
| `id` | string | experimental | opaque component id |
| `group` | string | stable | empty when not applicable |
| `name` | string | stable | |
| `version` | string | stable | |
| `format` | string | stable | |
| `repository` | string | stable | |
| `assets[]` | array | stable | see below |

Each asset:

| Field | Type | Tier | Notes |
|---|---|---|---|
| `path` | string | stable | repository-relative path |
| `size` | integer \| null | stable | bytes; null when not reported |
| `checksums` | map<string,string> | stable | algorithms as reported (sha1/sha256/md5/…) |
| `uploader` | string \| null | experimental | user that uploaded the asset; null when the instance does not report it |
| `uploaderIp` | string \| null | experimental | source IP of the upload; null when not reported |
| `blobCreated` | timestamp \| null | experimental | time the asset entered the repository |
| `lastModified` | timestamp \| null | stable | |
| `lastDownloaded` | timestamp \| null | stable | explicit null when never downloaded |

Timestamps render as RFC3339 UTC.

## Absence semantics

- `search` / `versions`: zero matches exit `0` with an empty list.
- `info`: absence of the requested component-version exits `10`
  (`response` class) because a specific artifact was named.
