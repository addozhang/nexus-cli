# Design: add-asset-provenance

## Context

Real Nexus `/service/rest/v1/search` asset payloads carry provenance fields the current mapping ignores:

```json
{
  "uploader": "admin",
  "uploaderIp": "192.168.215.1",
  "blobCreated": "2026-08-26T05:11:44.647+00:00"
}
```

These were observed during the e2e bring-up and are stable across OSS/Pro query APIs.

## Goals / Non-Goals

**Goals:** surface uploader, uploaderIp, blobCreated per asset in `<format> info` output as `experimental`.

**Non-Goals:** component-level creator (Nexus has no such concept); exposing provenance in `search`/`versions` output (asset-level detail belongs to `info`).

## Decisions

### D1: Additive, experimental tier

New fields land as `experimental` per the schema versioning policy: no `schemaVersion` bump, promotion to `stable` after one minor release. Nulls are explicit when an instance omits the fields (older versions may not send them).

*Why*: matches the established stability-tier workflow; consumers pinning `"1"` keep working.

### D2: Reuse TimeValue for blobCreated

`blobCreated` uses the same wire timestamp layouts as lastModified/lastDownloaded; decode via the existing `nexus.TimeValue`.

## Risks / Trade-offs

- [Older instances omit the fields] → explicit nulls, never missing keys; documented in schema.md.
- [uploaderIp is mildly sensitive] → read-only CLI already requires credentials that can see this in the UI; acceptable.

## Open Questions

_None._
