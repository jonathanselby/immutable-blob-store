# Release checklist

## Contract
- [ ] `VERSION`, CLI version and tag agree.
- [ ] Go 1.26 baseline is current.
- [ ] untrusted bytes remain opaque; no parser/preview path has entered the core.
- [ ] `MaxBytes` is mandatory and oversize input leaves no durable blob.
- [ ] blob identity remains exact-byte SHA-256 with strict ID validation.
- [ ] commit remains no-overwrite and duplicate success requires verification.
- [ ] safe download headers remain attachment/octet-stream/nosniff/no-store.
- [ ] no auth/evidence/retention/media-processing policy has leaked into V0.1.

## Engineering gate
- [ ] `gofmt` clean.
- [ ] `go vet ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] native/version and Linux ARM64 builds pass.
- [ ] Bafflewick exact-byte round-trip acceptance passes.

## Evidence
- [ ] PRD acceptance criteria map to executable evidence.
- [ ] generated EDD pins the exact source snapshot.
- [ ] generated comparison explains any correction or mismatch.
