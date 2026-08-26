# format-search — Per-format Component Search

## ADDED Requirements

### Requirement: Format-first search commands
The CLI SHALL expose `nx maven search`, `nx npm search`, `nx pypi search`, `nx cargo search`, `nx go search`, and `nx docker search`. Each takes a format-specific query string and executes against the Nexus Search API (`GET /service/rest/v1/search`) with the corresponding `format` parameter, rendering results through the output-schema contract.

#### Scenario: Maven search by artifact coordinate
- **WHEN** user runs `nx maven search com.example:foo` against an instance indexing that artifact
- **THEN** structured YAML output begins with `schemaVersion: "1"` and lists matching components with their versions

#### Scenario: Npm search by free-text term
- **WHEN** user runs `nx npm search lodash`
- **THEN** the query is sent to the Search API as `format=npm&q=lodash` and matching components are rendered

### Requirement: Format-specific coordinate grammar
Each format adapter SHALL define and document its coordinate grammar for positional arguments. Minimum grammars:
- maven: `groupId:artifactId[:version]`
- npm / pypi / cargo / go: `name[@version]`
- docker: `image[:tag]`

Unparseable input MUST exit `10` with an error naming the offending token and the expected grammar. Grammar definitions are part of this spec; changing them requires a spec change.

#### Scenario: Malformed maven coordinate
- **WHEN** user runs `nx maven search com.example::1.0`
- **THEN** the command exits `10` naming the empty artifact segment

#### Scenario: Versioned npm name
- **WHEN** user runs `nx npm search lodash@4.17.21`
- **THEN** the Search API receives both the name constraint and version constraint

### Requirement: Search pagination
Search results SHALL follow Nexus `continuationToken` pagination until the result set is exhausted or the default limit is reached. A `--limit <n>` flag SHALL cap returned components; the default limit value is pinned in docs/schema.md.

#### Scenario: Limit caps output
- **WHEN** a search matches 500 components and user runs the command with `--limit 20`
- **THEN** exactly 20 component entries are returned and no additional HTTP pages beyond what is needed are fetched

#### Scenario: Exhausted result set
- **WHEN** a search matches fewer components than the limit
- **THEN** all matches are returned and the command exits `0`

### Requirement: Empty result is success
A search with zero matches SHALL exit `0` and emit valid schema output containing an empty results list — never an error.

#### Scenario: No matches
- **WHEN** user searches for a component that does not exist on the instance
- **THEN** the command exits `0` with an empty list in valid schema form

### Requirement: All search commands honor instance resolution
Every `<format> search` command SHALL accept `--instance <alias>` and otherwise follow the instance-auth resolution rules (default instance, env override).

#### Scenario: Instance flag on search
- **WHEN** instances `dev` (default) and `prod` exist and user runs `nx pypi search requests --instance prod`
- **THEN** the Search API call targets `prod`
