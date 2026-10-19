# Immutable Blob Store — Design / Implementation Comparison

**Implementation evidence:** `f6810e93d21c4f0c9574cb9af970b72f484e51b1`  
**Generated:** 2026-10-20

| Design area | Intended contract | Recovered implementation | Assessment |
|---|---|---|---|
| one-job boundary | bounded immutable storage/integrity of untrusted bytes | one core package plus thin CLI; no case/evidence/auth/media subsystem | MATCH |
| opaque content | never parse or execute input | byte stream only; no content parser or media/document dependency | MATCH |
| bounded ingest | explicit limit without whole-file buffering | `LimitReader(MaxBytes+1)` + fixed 64 KiB buffer | MATCH |
| exact identity | SHA-256 of exact original bytes | simultaneous hash during streamed write | MATCH |
| storage path | digest only; no filename/path trust | strict lowercase digest path under `sha256/<prefix>/` | MATCH |
| immutability | no silent overwrite | same-filesystem hard-link create-if-absent | MATCH |
| duplicate handling | identical content may be idempotent only after verification | existing digest is fully re-hashed and size checked | MATCH |
| concurrency | simultaneous identical writes do not corrupt/overwrite | named race/concurrent duplicate test | MATCH |
| tamper evidence | changed bytes detected | `Verify` re-hashes and compares to ID | MATCH |
| retrieval | exact verified bytes | full verify then exact-byte copy | MATCH |
| hostile content | preserve as data | hostile-looking HTML/SVG/executable bytes round-trip unchanged | MATCH |
| browser delivery | conservative safe download | attachment + octet-stream + nosniff + no-store | MATCH |
| cancellation | pre-commit failure leaves no durable object | context-aware copy + deferred temp cleanup | MATCH |
| private files | durable blob private on Linux target | temp chmod `0600`, hard-link retains inode mode, tested | MATCH |
| CLI output safety | do not overwrite output/leave failed partial file | `O_EXCL`, explicit remove before `os.Exit` error path | MATCH |
| dependencies | stdlib-first | Go 1.26, no third-party modules | MATCH |
| CI | format/vet/race/version/ARM64/acceptance | encoded in `test.yml` | MATCH |
| application independence | no incidents/roles/retention/evidence semantics | none present in reusable package | MATCH |

## Material corrections made before publication

The first CLI draft relied on a deferred cleanup around `get`, but the shared error handler exits with `os.Exit`, which skips deferred functions. The published source explicitly closes and removes a partially created output file before entering that exit path.

The initial application discussion considered a generic inline-media delivery class. The published V0.1 component is deliberately narrower: generic browser delivery is download-only with octet-stream, attachment, `nosniff` and `no-store`. Any future inline image/audio/video policy remains an application presentation decision rather than weakening the storage primitive.

## Result

**MATCH after correction.** The published source is aligned with the hand-written PRD/EDD and preserves the one-job component boundary. No unexplained mismatch remains at the evidence commit.
