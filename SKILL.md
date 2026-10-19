# Immutable Blob Store — Skill

## Use this when

Use this component when an application needs to preserve arbitrary user-supplied bytes immutably with an explicit size bound and verifiable content identity, while keeping business/evidence semantics upstream.

## Core rule

> **Untrusted bytes stay bytes.**

## Change discipline

1. Never parse or execute content inside the storage primitive.
2. Keep maximum ingest size explicit and bounded; do not silently truncate.
3. Derive durable identity only from the exact bytes.
4. Never let caller filenames or IDs become filesystem paths.
5. Do not introduce update/overwrite behaviour under an immutable ID.
6. Verify an existing digest before treating a duplicate as success.
7. Keep download delivery conservative: attachment, octet-stream, nosniff, no-store.
8. Keep authorisation, evidence meaning, retention and legal hold upstream.
9. Do not add malware scanning, OCR, transcoding or retrieval ingestion to this one-job component.
10. Run format, vet, race, native, ARM64 and Bafflewick gates after changes.
11. Regenerate code-derived evidence before release.
