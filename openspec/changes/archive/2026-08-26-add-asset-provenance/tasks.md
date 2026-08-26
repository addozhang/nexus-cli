# Tasks: add-asset-provenance

## 1. Mapping

- [x] 1.1 Add `Uploader` / `UploaderIp` / `BlobCreated` decoding to internal/nexus RawAsset (TimeValue for blobCreated)
- [x] 1.2 Add `Uploader` / `UploaderIp` / `BlobCreated` fields to schema.AssetInfo and map them in MapInfoResults

## 2. Tests

- [x] 2.1 Unit: MapInfoResults maps provenance fields; absent fields render explicit nulls
- [x] 2.2 Integration fixture: include uploader/uploaderIp/blobCreated in the seeded asset; assert output
- [x] 2.3 E2E: real instance exposes non-null uploader and blobCreated for the seeded maven asset

## 3. Docs

- [x] 3.1 docs/schema.md: add experimental rows for uploader/uploaderIp/blobCreated under info assets
