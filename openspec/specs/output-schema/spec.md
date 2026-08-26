# output-schema Specification

## Purpose
TBD - created by archiving change init-nx-nexus-cli. Update Purpose after archive.
## Requirements
### Requirement: schemaVersion preamble
Every structured output (YAML or JSON) SHALL begin with `schemaVersion: "1"` as its first field. The value SHALL change only on breaking schema changes per the versioning policy below.

#### Scenario: YAML output preamble
- **WHEN** any successful query command renders default YAML output
- **THEN** the first line is `schemaVersion: "1"`

#### Scenario: JSON output preamble
- **WHEN** any successful query command renders `-o json` output
- **THEN** the deserialized object's `schemaVersion` field equals `"1"`

### Requirement: Output formats
Query commands SHALL support `-o yaml` (default) and `-o json`. An unknown format value MUST exit `10`. Raw passthrough (`-o raw`) is NOT provided in MVP.

#### Scenario: JSON pipes into jq
- **WHEN** user runs `nx repo list -o json | jq '.repositories[0].name'`
- **THEN** jq parses the stream and prints the first repository name

#### Scenario: Invalid format flag
- **WHEN** user runs `nx repo list -o table`
- **THEN** the command exits `10` naming the supported formats

### Requirement: Stability tiers and versioning policy
Fields documented in docs/schema.md are tagged `stable` or `experimental`. Breaking changes to `stable` fields (rename, type change, removal, semantic change) SHALL bump `schemaVersion`. Additive changes (new optional field, new `experimental` field) SHALL NOT bump it. New fields land `experimental` before promotion to `stable`.

#### Scenario: Additive field keeps version
- **WHEN** a minor release adds an `experimental` field to repository output
- **THEN** `schemaVersion` remains `"1"` and scripts pinning `"1"` continue to work

### Requirement: Exit code contract
All `nx` commands SHALL exit `0` on success and `10` on any nx-level failure (unknown alias, bad coordinates, auth rejection, network error, TLS verification failure, malformed Nexus response, invalid flags). stderr messages SHALL identify the failure class. No nx-level failure SHALL exit between 1–9.

#### Scenario: Auth failure exit code
- **WHEN** a stored token is revoked and user runs any query
- **THEN** the command exits `10` with a message identifying authentication failure

#### Scenario: Scripted branch on exit code
- **WHEN** a shell script checks `$?` after `nx maven search foo`
- **THEN** `0` reliably means "command ran, results follow (possibly empty)" and anything else means nx-level failure

### Requirement: Errors bypass stdout
All diagnostics SHALL be written to stderr. stdout SHALL carry only rendered schema output, so stdout can be piped safely even on partial failures.

#### Scenario: Pipe safety on failure
- **WHEN** a query fails mid-run due to network loss
- **THEN** stdout contains no partial schema output and stderr names the failure class

