# format-info Specification

## Purpose
TBD - created by archiving change init-nx-nexus-cli. Update Purpose after archive.
## Requirements
### Requirement: Info command per format
The CLI SHALL expose `nx <format> info <coordinate>` for maven, npm, pypi, cargo, go, and docker. The coordinate MUST identify one component at a specific version (per format grammar). The command SHALL render component detail plus its asset list through the output-schema contract, sourcing data from the Search API detail endpoints (`/service/rest/v1/search`, `/service/rest/v1/components/{id}`).

#### Scenario: Maven asset detail
- **WHEN** user runs `nx maven info com.example:foo:1.2.0`
- **THEN** output begins with `schemaVersion: "1"`, carries the component fields, and lists archived assets (jar, sources, pom) with their metadata

#### Scenario: Ambiguous multi-repo component
- **WHEN** the same component+version exists in two repositories of the instance
- **THEN** the command returns entries for each repository occurrence rather than guessing one

### Requirement: Detail fields
Component info SHALL include at minimum: `group` (where applicable to the format), `name`, `version`, `format`, `repositories`, and per-asset: `path`, `size`, `checksums` (sha1/sha256/md5 as reported), and `lastDownloaded`. Fields are `stable` from `schemaVersion: "1"`; fields Jenkins-style proxies may omit (e.g., `lastDownloaded` when never downloaded) MAY be null but MUST be present or explicitly null — never silently missing from stable tier. Additionally, each asset SHALL include three `experimental` provenance fields: `uploader` (string | null), `uploaderIp` (string | null), and `blobCreated` (timestamp | null). These experimental fields do not bump `schemaVersion`.

#### Scenario: Checksum exposure
- **WHEN** `nx cargo info tokio:1.38.0 -o json` runs against an indexed instance
- **THEN** each asset entry includes a checksums object with the algorithms Nexus reports

#### Scenario: Never-downloaded asset
- **WHEN** an asset has never been downloaded
- **THEN** `lastDownloaded` appears as explicit `null`, and the command still exits `0`

#### Scenario: Provenance exposure
- **WHEN** `nx maven info com.example:e2e:1.0.0 -o json` runs against a real instance where the asset was uploaded by user `admin`
- **THEN** each asset entry carries `uploader` with the uploading username, `uploaderIp` with the source IP or null, and `blobCreated` with a parseable timestamp or null

#### Scenario: Instance without provenance support
- **WHEN** an instance's response omits `uploader`/`uploaderIp`/`blobCreated`
- **THEN** the fields render as explicit `null`, the key is present, and the command exits `0`

### Requirement: Version required
The `info` coordinate MUST include a version segment. A coordinate without a version MUST exit `10` with a message directing the user to `versions` first.

#### Scenario: Missing version rejected
- **WHEN** user runs `nx npm info lodash` (no version)
- **THEN** the command exits `10` and suggests `nx npm versions lodash`

### Requirement: Nonexistent component-version
Querying info for a component-version absent from the instance SHALL exit `10` with `not found: <coordinate>` — unlike `search`/`versions`, absence here is an error because the user named a specific artifact.

#### Scenario: Absent coordinate errors
- **WHEN** user runs `nx maven info com.example:ghost:9.9.9`
- **THEN** the command exits `10` naming the coordinate

