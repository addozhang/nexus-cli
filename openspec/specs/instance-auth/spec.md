# instance-auth Specification

## Purpose
TBD - created by archiving change init-nx-nexus-cli. Update Purpose after archive.
## Requirements
### Requirement: Auth add registers a named instance
The `nx auth add <alias>` command SHALL require a non-empty alias as its first positional argument and SHALL interactively prompt for the instance URL (scheme, host, port), username, and token (hidden input). It MUST NOT accept an empty alias.

#### Scenario: Successful registration
- **WHEN** user runs `nx auth add prod` and provides `https://nexus.example.com:8443`, a username, and a token
- **THEN** the credential entry is stored under key `prod` in the credentials file with mode `0600` and the command exits `0`

#### Scenario: Missing alias
- **WHEN** user runs `nx auth add` without an alias
- **THEN** the command fails with usage help and does not write anything to the credentials file

### Requirement: Connectivity verification before save
The `nx auth add` command SHALL call the Nexus `/service/rest/v1/status` endpoint with the provided credentials before persisting them. If verification fails (unreachable, auth rejected, TLS failure), the credentials MUST NOT be saved.

#### Scenario: Unreachable instance rejected
- **WHEN** user provides a URL that cannot be reached
- **THEN** `nx auth add` exits `10`, prints a diagnostic naming the failure class, and stores nothing

#### Scenario: Bad credentials rejected
- **WHEN** the status endpoint returns HTTP 401 for the provided username/token
- **THEN** `nx auth add` exits `10` and stores nothing

### Requirement: Tokens are never printed
No `nx` command SHALL print a stored token to stdout or stderr under any flag combination.

#### Scenario: Listing instances hides tokens
- **WHEN** user runs `nx auth list`
- **THEN** each registered alias is shown with its URL and username, and no token value appears anywhere in the output

### Requirement: Default instance selection
Users SHALL be able to mark exactly one stored instance as default via `nx auth default <alias>`. Any query command run without `--instance <alias>` SHALL resolve to the default instance; if none is set, the command MUST fail with exit code `10` and list available aliases.

#### Scenario: Query falls back to default
- **WHEN** instances `dev` (default) and `prod` are stored, and user runs `nx repo list` with no flags
- **THEN** the query executes against `dev`

#### Scenario: No default configured
- **WHEN** instances exist but no default is set, and user runs `nx maven search foo` without `--instance`
- **THEN** the command exits `10` with a message listing the available aliases and how to set a default

#### Scenario: Explicit instance wins
- **WHEN** `dev` is default and user runs `nx repo list --instance prod`
- **THEN** the query executes against `prod`

### Requirement: Unknown or ambiguous alias resolution errors
A query referencing an alias that is not stored MUST fail with exit code `10` and name the offending alias plus the known aliases.

#### Scenario: Unknown alias
- **WHEN** user runs `nx repo list --instance staging` and `staging` is not stored
- **THEN** the command exits `10` with `unknown instance "staging"` and lists known aliases

### Requirement: Credential file location and permissions
Credentials SHALL be stored at `~/.config/nx/credentials` in TOML format with file mode `0600`. The directory SHALL be created with mode `0700` if absent.

#### Scenario: Fresh machine first add
- **WHEN** `~/.config/nx/` does not exist and user runs `nx auth add dev`
- **THEN** the directory is created `0700`, the file created `0600`, and the entry persisted

### Requirement: Environment-variable override path
When `NX_URL`, `NX_USERNAME`, and `NX_TOKEN` are all set, query commands SHALL use them directly against that URL even when no instance is stored or default. If only some of the three are set, the command MUST ignore the partial override and follow normal resolution.

#### Scenario: Full env override on empty store
- **WHEN** no instances are stored and `NX_URL`, `NX_USERNAME`, `NX_TOKEN` are set, and user runs `nx repo list`
- **THEN** the query executes against the environment-provided URL and exits per output-schema rules

### Requirement: Self-signed TLS support
`nx` SHALL trust system root CAs by default and SHALL honor `SSL_CERT_FILE` pointing at a PEM bundle. The global `--insecure` flag SHALL disable certificate verification entirely and print a warning to stderr when used.

#### Scenario: Custom CA bundle
- **WHEN** `SSL_CERT_FILE` points at the PEM bundle of the Nexus instance's CA and user queries it
- **THEN** the query succeeds over TLS without any flag

#### Scenario: Insecure escape hatch
- **WHEN** user runs a query with `--insecure` against a self-signed instance without `SSL_CERT_FILE`
- **THEN** the query succeeds and a warning about disabled certificate verification appears on stderr

### Requirement: Auth lifecycle commands
`nx auth list`, `nx auth remove <alias>`, and `nx auth default <alias>` SHALL manage the registry. Removing the default instance MUST clear the default marker; removing a non-existent alias exits `10`.

#### Scenario: Remove default clears marker
- **WHEN** `dev` is default and user runs `nx auth remove dev`
- **THEN** `dev` is removed, no instance is default, and subsequent flag-less queries exit `10`

