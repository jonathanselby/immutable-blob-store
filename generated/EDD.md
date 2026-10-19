# Immutable Blob Store — Generated Engineering Design Document

**Evidence commit:** `f6810e93d21c4f0c9574cb9af970b72f484e51b1`  
**Method:** code-derived review using the `codebase-to-edd` evidence discipline  
**Generated:** 2026-10-20

This document describes the implementation at the evidence commit. The hand-written PRD/EDD are comparison targets, not runtime evidence.

## 1. Executive description

**KNOWN** — `immutable-blob-store` is a small Go component with one material package, `internal/blobstore`, and a thin CLI.

**KNOWN** — it accepts arbitrary untrusted bytes as an `io.Reader`, requires a caller-supplied maximum size, streams the bytes into a private temporary file while calculating SHA-256, and commits them under a digest-derived immutable path.

**KNOWN** — the public identifier is `sha256:<64 lowercase hexadecimal characters>` and the durable path contains only the validated digest, never the original filename.

## 2. Evidence inspected

- `internal/blobstore/blobstore.go` — ingest, hashing, ID validation, commit, verification, retrieval and safe headers;
- `internal/blobstore/blobstore_test.go` — behavioural evidence;
- `cmd/blobstore/main.go` — process and exit contract;
- `.github/workflows/test.yml` — engineering gate;
- `examples/bafflewick/` — hostile-looking offline acceptance fixture;
- `go.mod`, `VERSION`, `CHANGELOG.md` — language/release boundary.

## 3. Runtime structure

```text
untrusted reader + explicit MaxBytes
              |
              v
      private temporary file
      + streaming SHA-256
              |
         +----+----+
         |         |
     refusal     complete
         |         |
    temp removed   v
             sha256:<digest>
                   |
                   v
          validated digest path
                   |
                   v
          atomic hard-link commit
              +----+----+
              |         |
           created    exists
              |         |
              |         v
              |     verify existing
              +----+----+
                   |
                   v
              Blob metadata
```

## 4. Ingest contract

**KNOWN** — `Store.Put` refuses an empty root, non-positive `MaxBytes` or nil reader.

**KNOWN** — the source is wrapped in `io.LimitReader(MaxBytes+1)`, so the component can detect oversize input without reading an unbounded stream.

**KNOWN** — `copyWithContext` uses a fixed 64 KiB buffer. The full blob is not buffered in memory.

**KNOWN** — input is simultaneously written to the temporary file and SHA-256 hasher through `io.MultiWriter`.

**KNOWN** — when the copied byte count is greater than `MaxBytes`, the operation returns `ErrTooLarge`; deferred cleanup removes the temporary file and no durable link has been created.

## 5. Blob identity and path contract

**KNOWN** — the digest is hex-encoded lowercase SHA-256 of the exact input bytes and the ID is `sha256:` plus that digest.

**KNOWN** — `parseID` requires that exact prefix, exactly 64 digest characters, and only lowercase hexadecimal characters.

**KNOWN** — the durable path is derived as `<root>/sha256/<first-two>/<digest>.blob` only after strict ID/digest construction or validation.

**KNOWN** — original filenames and MIME declarations do not participate in durable path construction.

## 6. Temporary and durable permissions

**KNOWN** — store and digest directories are requested as `0700` and the temporary file is explicitly chmodded to `0600`.

**KNOWN** — durable commit uses a hard link to the private temporary inode, so the durable file retains the private mode on the supported Linux runtime.

**KNOWN** — `TestStoredBlobUsesPrivatePermissions` checks the durable mode is `0600`.

## 7. Commit / duplicate contract

**KNOWN** — after writing, the temporary file is synced and closed.

**KNOWN** — durable commit is `os.Link(temp, final)`, which creates the final digest path if absent rather than using rename-overwrite semantics.

**KNOWN** — if the digest path already exists, the component verifies the existing object. A valid digest and matching byte size returns success with `Duplicate=true`.

**KNOWN** — if existing content does not verify, the duplicate path is refused as an integrity failure. The existing file is not overwritten by the component.

**KNOWN** — `TestConcurrentDuplicateCommit` exercises two simultaneous puts of the same bytes and expects one creator plus one verified duplicate.

## 8. Verification contract

**KNOWN** — `Verify` validates the requested ID before filesystem lookup, opens the digest-derived path, re-hashes the complete stored bytes and requires equality with the ID digest.

**KNOWN** — changing durable bytes after commit is detected by `TestTamperDetected`.

**KNOWN** — invalid/traversal-like identifiers are rejected before path derivation by `TestInvalidIDAndTraversalRefused`.

## 9. Retrieval contract

**KNOWN** — `CopyVerified` first performs a complete `Verify`, then reopens the digest-derived file and copies exact bytes into the caller's writer.

**INFERRED** — this makes retrieval intentionally two-pass. It is simpler to reason about than streaming unverified bytes to the caller while verification is still in progress.

**KNOWN** — the design does not protect against a hostile privileged host administrator mutating the file between verification and the second read; that actor is outside V0.1's trust boundary.

## 10. Content treatment

**KNOWN** — no parser, MIME sniffer, antivirus, decompressor, image library, document library, media library, OCR library or model call exists in the implementation.

**KNOWN** — `TestOpaqueHostileLookingBytesPreserved` uses HTML/script/SVG/executable-looking bytes and proves they round-trip unchanged.

**KNOWN** — the Bafflewick acceptance file is active-looking HTML but is only read as bytes.

## 11. Safe browser delivery metadata

**KNOWN** — `SafeDownloadHeaders` sets:

```text
Content-Type: application/octet-stream
Content-Disposition: attachment; filename=<sanitised basename>
X-Content-Type-Options: nosniff
Cache-Control: no-store
```

**KNOWN** — display filenames are reduced to a basename and control characters are removed.

**KNOWN** — no generic inline-render helper exists in V0.1.

## 12. Mutation model

**KNOWN** — the reusable package exports no update or delete method.

**KNOWN** — the only storage mutation exposed by the package is `Put`; an existing digest is treated as immutable.

**KNOWN** — retention, legal hold and business deletion policy are absent from the component.

## 13. Cancellation and cleanup

**KNOWN** — streaming checks the supplied context between reads.

**KNOWN** — cancellation or other pre-commit failure returns through deferred temporary-file cleanup.

**KNOWN** — `TestCancelledIngestDoesNotCommit` proves a cancelled ingest leaves no durable blob.

## 14. CLI contract

**KNOWN** — the executable supports:

```text
blobstore -version
blobstore put -store <dir> -file <path> -max-bytes <n>
blobstore verify -store <dir> -id <sha256:id>
blobstore get -store <dir> -id <sha256:id> -output <new-file>
```

**KNOWN** — `-version` prints `blobstore 0.1.0`.

**KNOWN** — `get` opens its destination with create/exclusive/write and mode `0600`, so it does not overwrite an existing caller file.

**KNOWN** — on a failed `CopyVerified`, the CLI explicitly closes/removes the partial output before calling the exit path because `os.Exit` does not run deferred cleanup.

**KNOWN** — exit mapping is:

```text
0 success
1 deterministic blob refusal
2 usage error
3 operating-system/file/output execution failure
```

## 15. State and dependencies

**KNOWN** — the reference implementation's state is the directory-backed blob bytes only. It has no database, queue, metadata catalogue or background worker.

**KNOWN** — `go.mod` targets Go 1.26 and declares no third-party modules.

**KNOWN** — the repository is a reusable reference component, not a required separately deployed service.

## 16. Security boundary

**KNOWN** — input bytes and display filenames are untrusted.

**KNOWN** — requested IDs are untrusted until strict parsing succeeds.

**KNOWN** — the configured root and process/host account are trusted deployment inputs.

**KNOWN** — the component performs no caller authorisation; the consuming application must authorise before exposing put/read operations.

**KNOWN** — V0.1 makes no claim to protect plaintext from hostile root/storage administration.

## 17. Executable evidence

Tests include:

- `TestPutVerifyAndCopyExactBytes`;
- `TestOversizeLeavesNoDurableBlob`;
- `TestDuplicateIsIdempotent`;
- `TestConcurrentDuplicateCommit`;
- `TestTamperDetected`;
- `TestInvalidIDAndTraversalRefused`;
- `TestOpaqueHostileLookingBytesPreserved`;
- `TestSafeDownloadHeaders`;
- `TestCancelledIngestDoesNotCommit`;
- `TestStoredBlobUsesPrivatePermissions`.

CI adds Go 1.26 formatting, vet, race tests, native build, exact version, Linux ARM64 cross-build, Bafflewick exact-byte round trip and the named blob-boundary suite.

## 18. Known limits / unknowns

**KNOWN** — the reference backend is a local same-filesystem directory and depends on hard-link semantics available on the Garage Linux target.

**KNOWN** — verified copy reads content twice.

**KNOWN** — there is no generic inline-media policy, remote storage adapter, retention API, malware scan, content parser or constrained-hardware throughput benchmark.

**UNKNOWN** — measured throughput and filesystem behaviour on the eventual constrained-hardware acceptance rig beyond normal Go/Linux semantics.

## 19. Evidence conclusion

**KNOWN** — the source implements the one-job immutable-blob boundary: bounded opaque ingest, exact-byte content identity, no-overwrite commit, verified duplicate handling, tamper detection and conservative download metadata without importing application semantics.
