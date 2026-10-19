# Immutable Blob Store

[![test](https://github.com/jonathanselby/immutable-blob-store/actions/workflows/test.yml/badge.svg)](https://github.com/jonathanselby/immutable-blob-store/actions/workflows/test.yml)

**Untrusted bytes stay bytes.**

`immutable-blob-store` is a deliberately small Go component for accepting bounded untrusted binary content, preserving the exact original bytes immutably, calculating a content hash and returning verified bytes through a conservative download boundary. It is usable independently of any larger application.

## The job

Given an untrusted byte stream, an explicit maximum size and a trusted store root:

1. stream the input into a private temporary file while calculating SHA-256;
2. refuse content that exceeds the configured maximum before durable commit;
3. derive the immutable ID from the exact bytes;
4. atomically commit the blob without overwriting an existing object;
5. treat an identical existing digest as an idempotent duplicate only after verification;
6. verify stored bytes against the content-addressed ID before copying them out;
7. provide conservative download headers that do not trust the original filename or MIME type.

## Storage boundary

V0.1 is a standalone directory-backed reference implementation:

```text
<root>/
  .tmp/
  sha256/
    ab/
      abcdef...<64 hex>....blob
```

The public blob ID is:

```text
sha256:<64 lowercase hexadecimal characters>
```

The stored filename is derived only from that validated digest. Caller-supplied filenames never participate in the storage path.

## What this does not do

It does not authorise callers, interpret evidence, own business metadata, delete/update blobs, scan for malware, run antivirus, parse documents, trust MIME declarations, generate thumbnails, transcode media, perform OCR/transcription, apply retention/legal hold, ingest content into retrieval systems or decide whether content is safe to render inline.

The V0.1 delivery helper is deliberately **download-only**: `application/octet-stream`, `Content-Disposition: attachment`, `X-Content-Type-Options: nosniff` and `Cache-Control: no-store`.

## Install

Requires Go 1.26.

```bash
go install github.com/jonathanselby/immutable-blob-store/cmd/blobstore@main
blobstore -version
```

Expected version: `blobstore 0.1.0`.

## Run

```bash
store=$(mktemp -d)
blobstore put \
  -store "$store" \
  -file examples/bafflewick/evidence.html \
  -max-bytes 1048576
```

Example result:

```json
{"id":"sha256:<digest>","sha256":"<digest>","size":173}
```

Then:

```bash
blobstore verify -store "$store" -id 'sha256:<digest>'
blobstore get -store "$store" -id 'sha256:<digest>' -output /tmp/bafflewick-evidence.bin
cmp examples/bafflewick/evidence.html /tmp/bafflewick-evidence.bin
```

The Bafflewick fixture deliberately contains active-looking HTML. The store preserves it as opaque bytes and never interprets it as application content.

## Stable exit codes

| Code | Meaning |
|---:|---|
| `0` | operation succeeded |
| `1` | deterministic refusal: too large, invalid/not-found ID, or integrity failure |
| `2` | invalid CLI usage |
| `3` | operating-system/file/encoding execution failure |

## Security boundary

The byte stream is untrusted. The configured store root and operating-system account are trusted deployment inputs.

The component prevents caller-controlled filenames and IDs becoming storage paths, never parses uploaded content, commits with private file permissions, refuses overwrite through the public API, and detects content tampering by re-hashing against the immutable ID.

It does **not** protect plaintext from a hostile host/root administrator. Caller authorisation must happen before `Put`, `Verify`, `CopyVerified` or download delivery is exposed through an application.

## Engineering artefacts

- `PRD.md` — product boundary and testable acceptance criteria;
- `EDD.md` — hand-written design and operational boundary;
- `FORMAT.md` — content-addressed identifier and storage format;
- `architecture.txt` — plain-text runtime view;
- `internal/blobstore` — reusable core;
- `cmd/blobstore` — thin CLI edge;
- `examples/bafflewick` — fictional hostile-looking acceptance fixture;
- `.github/workflows/test.yml` — Go 1.26 format/vet/race/build/version/ARM64/acceptance gate;
- `generated/EDD.md` — code-derived design evidence pinned to a source commit;
- `generated/COMPARISON.md` — intended design versus recovered implementation;
- `SKILL.md`, `RELEASE.md`, `CHANGELOG.md`, `VERSION` — adoption/release discipline.

## Licence

MIT. See `LICENSE`.
