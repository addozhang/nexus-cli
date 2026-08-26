# format-info — Delta: Asset Provenance Fields

## MODIFIED Requirements

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
