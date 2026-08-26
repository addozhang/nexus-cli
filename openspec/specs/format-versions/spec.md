# format-versions Specification

## Purpose
TBD - created by archiving change init-nx-nexus-cli. Update Purpose after archive.
## Requirements
### Requirement: Versions command per format
The CLI SHALL expose `nx <format> versions <component-coordinate>` for maven, npm, pypi, cargo, go, and docker. Each resolves the component's identity via its format adapter and SHALL return the distinct list of versions available on the resolved instance, sourced from the Search API constrained to that component.

#### Scenario: Maven version listing
- **WHEN** user runs `nx maven versions com.example:foo` and the artifact has three versions on the instance
- **THEN** output begins with `schemaVersion: "1"` and contains exactly three version entries for that group/artifact pair

#### Scenario: Docker tag listing
- **WHEN** user runs `nx docker versions myteam/app` against an instance where the image has four tags
- **THEN** four tag entries are listed for that image name

### Requirement: Version entry fields
Each version entry SHALL include at minimum: `version`, `repository` (the repo holding it), and `lastModified`. Fields are `stable` from `schemaVersion: "1"`.

#### Scenario: Field presence
- **WHEN** `nx npm versions lodash -o json` runs successfully
- **THEN** every entry carries `version`, `repository`, and `lastModified`

### Requirement: Unknown component behavior
Querying versions for a component absent from the instance SHALL exit `0` with an empty version list (consistent with format-search's empty-result contract), not an error.

#### Scenario: Component not found
- **WHEN** user runs `nx pypi versions nonexistent-pkg`
- **THEN** the command exits `0` with an empty list

### Requirement: Coordinate grammar shared with search
The `versions` command SHALL accept the same coordinate grammar as its format's `search` command, requiring at least the component-identifying portion (maven `group:artifact`; others `name`; docker image name). A bare version without a component identifier MUST exit `10`.

#### Scenario: Identifier required
- **WHEN** user runs `nx maven versions :foo` (missing group)
- **THEN** the command exits `10` naming the missing group segment

