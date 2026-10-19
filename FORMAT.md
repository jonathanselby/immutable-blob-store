# Immutable Blob Store — V0.1 Format

## Identifier

```text
sha256:<64 lowercase hexadecimal characters>
```

Example:

```text
sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
```

The hexadecimal portion is SHA-256 of the exact original bytes.

## Directory layout

For digest `abcdef...`:

```text
<root>/sha256/ab/abcdef....blob
```

Only the validated digest determines this path. Original filenames, MIME types and application metadata are never used in path construction.

Temporary ingest material is kept under:

```text
<root>/.tmp/
```

and is removed after success or refusal.

## Immutability contract

V0.1 exposes no update/delete method. Durable commit uses a same-filesystem hard link so an existing digest path is not silently replaced.

If an identical ID already exists, the existing object is re-hashed before the operation is treated as an idempotent duplicate.

## Metadata ownership

The component returns only storage/integrity metadata:

```json
{
  "id": "sha256:<digest>",
  "sha256": "<digest>",
  "size": 1234,
  "duplicate": false
}
```

Original filename, declared MIME type, uploader, business meaning, access scope, retention and other domain metadata belong to the consuming application.
