# Immutable Blob Store — PRD

**Version:** 0.1

## Problem

Applications often need to retain user-supplied documents, images, audio, video or other files that must be treated as untrusted. A storage helper that trusts filenames, guesses that content is safe, reads an entire file into memory, silently truncates oversize input or overwrites an existing object creates avoidable security and integrity failures.

## Product objective

Provide a small deterministic Go component that accepts a bounded untrusted byte stream, preserves the exact original bytes immutably, derives a SHA-256 content identifier, verifies integrity and exposes conservative download metadata.

> **Untrusted bytes stay bytes.**

## Users

Applications that need a small reusable binary-storage primitive without importing evidence, case-management, authorisation, media-processing or document-parsing semantics.

## In scope

- arbitrary binary input treated as opaque bytes;
- caller-supplied positive maximum ingest size;
- streaming ingest rather than whole-file buffering;
- SHA-256 over the exact original bytes;
- content-addressed immutable IDs;
- private temporary and durable files;
- atomic no-overwrite commit on the supported Linux runtime;
- idempotent duplicate detection for identical content;
- exact-byte verified retrieval;
- integrity verification against the content ID;
- strict ID validation with no caller-controlled storage path;
- context cancellation during streaming operations;
- conservative safe-download headers;
- stable process exit codes;
- Go 1.26, standard library only.

## Out of scope

- user authentication or authorisation;
- incidents, cases, roles or compartments;
- evidence meaning or forensic chain of custody;
- legal hold or retention policy;
- delete/update mutation API;
- antivirus, malware analysis or sandboxing;
- file-type trust or semantic validation;
- inline-render safety decisions;
- document parsing, OCR or transcription;
- image/audio/video transcoding or thumbnails;
- retrieval/vector/model ingestion;
- remote object-store/database adapters in V0.1.

## Acceptance criteria

1. arbitrary binary bytes round-trip exactly through put, verify and copy;
2. ingest streams through bounded memory and does not require the full blob in RAM;
3. content larger than the configured maximum is refused and leaves no durable blob;
4. the blob ID is `sha256:<64 lowercase hex>` derived from the exact stored bytes;
5. caller-supplied filenames never influence the durable storage path;
6. an invalid, malformed or traversal-like ID is refused before filesystem lookup;
7. a second put of identical bytes returns the same ID as an idempotent duplicate and does not overwrite the stored object;
8. concurrent identical puts result in one committed object and one duplicate outcome without corruption;
9. changing stored bytes after commit is detected by verification;
10. cancelled ingest leaves no durable blob;
11. durable blobs are created with mode `0600` on the supported Linux runtime;
12. hostile-looking HTML/SVG/executable-like bytes are stored and returned unchanged without parsing or execution;
13. safe download headers force attachment/octet-stream, disable browser sniffing and disable caching while stripping path components from the supplied filename;
14. deterministic refusals map to exit 1, CLI misuse to exit 2 and operating-system/execution failures to exit 3;
15. CI proves format, vet, race tests, exact version, Linux ARM64 build and Bafflewick round-trip acceptance.

## Definition of done

V0.1 is done when all criteria have executable evidence, the trust boundary is explicit, the Bafflewick fixture proves exact-byte treatment of active-looking content, code-derived EDD evidence is pinned to the implementation snapshot, and the design comparison contains no unexplained mismatch.
