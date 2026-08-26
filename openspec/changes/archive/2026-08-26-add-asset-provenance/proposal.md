# Proposal: add-asset-provenance

## Why

When investigating "who put this artifact here and when", users currently have to open the Nexus UI. The data already exists on every asset (`uploader`, `uploaderIp`, `blobCreated`) but `nx` does not surface it, so provenance questions require a browser round-trip.

## What Changes

- Add three `experimental` fields to the per-asset output of `nx <format> info`:
  - `uploader` (string | null) — user that uploaded the asset
  - `uploaderIp` (string | null) — source IP of the upload
  - `blobCreated` (timestamp | null) — time the asset entered the repository
- Update integration fixtures and e2e assertions to cover the new fields.
- Document the fields in docs/schema.md as `experimental` (additive change; no `schemaVersion` bump).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `format-info`: asset detail output gains three experimental provenance fields; existing stable fields unchanged.

## Impact

- `internal/nexus` RawAsset: two new decoded fields (`uploader`, `uploaderIp`; `blobCreated` reuses TimeValue decoding).
- `internal/schema` AssetInfo: three new fields mapped through.
- `docs/schema.md`: field reference rows tagged experimental.
- No breaking changes: additive only, `schemaVersion` stays `"1"`.
