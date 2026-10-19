# Immutable Blob Store — Engineering Design Document

**Version:** 0.1

## 1. Purpose

Accept bounded untrusted bytes, preserve the original bytes immutably, calculate a deterministic content identifier, verify integrity and provide a conservative delivery boundary.

> **Untrusted bytes stay bytes.**

## 2. Design constraints

1. Input bytes are always untrusted and are never parsed or executed.
2. The caller must provide an explicit positive maximum size.
3. The full input must not need to fit in memory.
4. Caller filenames and IDs must never become filesystem paths.
5. An existing immutable object must never be overwritten through the component API.
6. Integrity is SHA-256 of the exact original byte sequence.
7. Authorisation and business meaning remain upstream.
8. V0.1 delivery is conservative download-only, not inline rendering.

## 3. Runtime model

```text
untrusted io.Reader + MaxBytes
           |
           v
validate trusted store config
           |
           v
stream <= MaxBytes+1
into private temp file
while SHA-256 hashing
           |
      +----+------------------+
      |                       |
   too large / cancelled    complete
      |                       |
 remove temp                  v
                       sha256:<digest>
                              |
                              v
                    validated digest path
                              |
                              v
                    hard-link no-overwrite
                              |
                  +-----------+-----------+
                  |                       |
               created              already exists
                  |                       |
                  v                       v
              durable              verify existing
                  |                       |
                  +-----------+-----------+
                              |
                              v
                         Blob metadata
```

## 4. Blob identity

The only public identifier format is:

```text
sha256:<64 lowercase hexadecimal characters>
```

The digest is calculated from the exact byte stream before durable commit. IDs with another prefix, wrong length, uppercase/non-hex characters or path-like content are refused.

No original filename appears in the durable path.

## 5. Ingest and size boundary

`Store.Put` requires `MaxBytes > 0`.

The reader is wrapped in a `LimitReader(MaxBytes+1)`. Bytes are copied in a fixed 64 KiB buffer into both the temporary file and SHA-256 hasher. Reading one byte beyond the configured limit is enough to refuse the ingest.

Oversize or cancelled input returns without linking a durable object. Temporary material is removed on return.

## 6. Temporary storage

Temporary files live under `<root>/.tmp`, are created with the operating system's unique temporary naming and are explicitly set to mode `0600`.

The temporary file is synced and closed before commit.

The store root is trusted deployment configuration. V0.1 does not attempt to defend against a hostile administrator replacing directories with symlinks or modifying the store beneath the process.

## 7. Durable layout

```text
<root>/sha256/<first-two-hex>/<64-hex>.blob
```

Directories are created private (`0700`). The durable file is the same inode as the private temporary file after hard-link commit and therefore retains mode `0600`.

The layout is described in `FORMAT.md`.

## 8. Atomic no-overwrite commit

V0.1 targets Linux and uses `os.Link(temp, final)` within the same configured store filesystem.

This provides an atomic create-if-absent operation without rename-overwrite behaviour.

If the final object does not exist, the link succeeds and the containing directory is synced.

If it already exists, the component verifies the existing bytes against the digest and size. A valid match returns the same ID with `duplicate:true`; an integrity mismatch is refused. The existing object is never replaced.

## 9. Retrieval and verification

`Verify` validates the ID, opens the derived digest path, streams it through SHA-256 and compares the actual digest to the identifier.

`CopyVerified` performs a complete verification before reopening and copying the bytes to the caller's writer.

This intentionally prefers a simple integrity contract over single-pass delivery performance. A hostile/root administrator able to mutate data between verification and copy is outside V0.1's trust boundary.

## 10. Safe download boundary

`SafeDownloadHeaders` does not inspect or trust file contents. It returns:

```text
Content-Type: application/octet-stream
Content-Disposition: attachment; filename=<sanitised basename>
X-Content-Type-Options: nosniff
Cache-Control: no-store
```

Path components and control characters are removed from the supplied display filename. The original name is presentation metadata only and never influences storage.

V0.1 intentionally provides no generic inline-render decision. A consuming application that wants inline images/audio/video must own and test that narrower presentation policy separately.

## 11. Mutation model

There is no update or delete method.

The public operations are effectively:

```text
Put
Verify
CopyVerified
SafeDownloadHeaders
```

Retention and disposal are lifecycle concerns for the consuming application/storage operator and are not hidden inside this primitive.

## 12. CLI contract

```text
blobstore -version
blobstore put -store <dir> -file <path> -max-bytes <n>
blobstore verify -store <dir> -id <sha256:id>
blobstore get -store <dir> -id <sha256:id> -output <new-file>
```

`get` creates a new output file with `O_EXCL` and mode `0600`; it never silently overwrites an existing destination.

Exit codes:

```text
0 success
1 deterministic refusal: size / ID / not-found / integrity
2 invalid CLI usage
3 OS, file or output execution failure
```

## 13. State and dependencies

The directory-backed blob bytes are the component's state. It has no database, queue, background worker, index or metadata catalogue.

V0.1 targets Go 1.26 and uses the standard library only.

The standalone directory implementation is a reference component. A consuming monolith may refactor the same boundary into its chosen durable data tier rather than deploy this repository as a separate service.

## 14. Security boundary

Untrusted:

- uploaded/read input bytes;
- original/display filenames;
- requested blob IDs until strict validation;
- output destinations at the CLI edge.

Trusted deployment boundary:

- configured store root;
- process/host account;
- caller's authorisation decision before exposing component operations.

The component does not claim protection from a malicious root/host administrator, malware analysis, content safety classification or forensic authenticity.

## 15. Failure behaviour

- oversize input: deterministic refusal, no durable link;
- cancellation: return context error, no durable link;
- malformed ID: refusal before path derivation;
- missing ID: explicit not-found refusal;
- existing identical object: verified duplicate success;
- existing/tampered mismatch: integrity refusal, no overwrite;
- output copy failure: error returned to caller;
- CLI get failure: partial new output is removed before process exit.

## 16. Test/CI strategy

Package tests prove:

- exact-byte put/verify/copy;
- oversize cleanup;
- duplicate idempotence;
- concurrent duplicate commit;
- tamper detection;
- invalid/traversal-like ID refusal;
- hostile-looking bytes remain opaque;
- safe download headers;
- cancellation cleanup;
- private durable-file permissions.

CI adds Go 1.26 `gofmt`, vet, race tests, native build, exact version, Linux ARM64 cross-build and Bafflewick exact-byte round-trip acceptance.

## 17. Known limitations

- the reference durable backend is a local filesystem directory, not a database/object-store adapter;
- `os.Link` requires the temporary and final paths to be on the same filesystem and targets the Garage Linux runtime;
- verification before copy reads a blob twice;
- no file-type/inline-render policy beyond safe download-only headers;
- no retention/delete API;
- no malware scanning or quarantine classification;
- no protection from hostile root/storage administration;
- no constrained-hardware throughput benchmark in V0.1; ARM64 CI is compilation proof only.

## 18. Acceptance-to-evidence map

| Requirement | Evidence |
|---|---|
| exact binary round-trip + SHA-256 identity | `TestPutVerifyAndCopyExactBytes` |
| bounded ingest + no oversize commit | `TestOversizeLeavesNoDurableBlob` |
| duplicate idempotence | `TestDuplicateIsIdempotent` |
| concurrent no-overwrite commit | `TestConcurrentDuplicateCommit` |
| tamper evidence | `TestTamperDetected` |
| path/ID refusal | `TestInvalidIDAndTraversalRefused` |
| opaque hostile-looking bytes | `TestOpaqueHostileLookingBytesPreserved` |
| conservative download headers | `TestSafeDownloadHeaders` |
| cancellation cleanup | `TestCancelledIngestDoesNotCommit` |
| private durable permissions | `TestStoredBlobUsesPrivatePermissions` |
| format/vet/race/build/version/ARM64/Bafflewick | `.github/workflows/test.yml` |
