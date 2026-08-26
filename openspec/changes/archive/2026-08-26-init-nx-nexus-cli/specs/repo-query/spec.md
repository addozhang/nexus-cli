# repo-query — Cross-format Repository Listing

## ADDED Requirements

### Requirement: Repo list command
The `nx repo list` command SHALL list all repositories visible to the authenticated user on the resolved instance, sourced from `GET /service/rest/v1/repositories`, rendered through the output-schema contract.

#### Scenario: List all repositories
- **WHEN** user runs `nx repo list` against an instance hosting three repositories
- **THEN** structured YAML output begins with `schemaVersion: "1"` and contains exactly three repository entries

#### Scenario: Empty instance
- **WHEN** the resolved instance hosts zero repositories
- **THEN** the command exits `0` and emits valid schema output with an empty repository list (not an error)

### Requirement: Filter by format
`nx repo list --format <format>` SHALL return only repositories whose format matches the given value (`maven`, `npm`, `pypi`, `cargo`, `go`, `docker`, or other Nexus formats). An unknown format value MUST exit `10`.

#### Scenario: Format filter narrows results
- **WHEN** the instance hosts two maven repos and one npm repo, and user runs `nx repo list --format npm`
- **THEN** exactly the npm repository is listed

#### Scenario: Invalid format value
- **WHEN** user runs `nx repo list --format rpm2`
- **THEN** the command exits `10` naming the invalid format (validation is client-side; arbitrary Nexus formats beyond the six core ones remain passable — see requirement below)

### Requirement: Filter by type
`nx repo list --type <type>` SHALL filter by repository type: `hosted`, `proxy`, or `group`. Any other value exits `10`.

#### Scenario: Type filter
- **WHEN** the instance hosts one hosted and two proxy docker repos, and user runs `nx repo list --format docker --type proxy`
- **THEN** only the two proxy docker repositories appear

### Requirement: Stable repository fields
Each repository entry SHALL include at minimum: `name`, `format`, `type`, `url`, and `exposed`. These fields are `stable` from `schemaVersion: "1"`. Additional fields MAY appear tagged `experimental`.

#### Scenario: Field presence
- **WHEN** `nx repo list -o json` runs against a live instance
- **THEN** every entry carries `name`, `format`, `type`, `url`, and `exposed`
