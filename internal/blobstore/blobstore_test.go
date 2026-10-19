package blobstore

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPutVerifyAndCopyExactBytes(t *testing.T) {
	s := Store{Root: t.TempDir(), MaxBytes: 1024}
	want := []byte("Bafflewick incident photograph bytes\x00\x01")
	b, err := s.Put(context.Background(), bytes.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	if b.Size != int64(len(want)) || !strings.HasPrefix(b.ID, "sha256:") || b.Duplicate {
		t.Fatalf("unexpected metadata: %+v", b)
	}
	verified, err := s.Verify(context.Background(), b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if verified.SHA256 != b.SHA256 || verified.Size != b.Size {
		t.Fatalf("verify mismatch: %+v != %+v", verified, b)
	}
	var got bytes.Buffer
	if _, err := s.CopyVerified(context.Background(), b.ID, &got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("bytes changed: %q", got.Bytes())
	}
}

func TestOversizeLeavesNoDurableBlob(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root, MaxBytes: 4}
	_, err := s.Put(context.Background(), bytes.NewReader([]byte("12345")))
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "sha256", "*", "*.blob"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("oversize ingest left durable blob: %v", matches)
	}
}

func TestDuplicateIsIdempotent(t *testing.T) {
	s := Store{Root: t.TempDir(), MaxBytes: 1024}
	content := []byte("same bytes")
	first, err := s.Put(context.Background(), bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Put(context.Background(), bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || !second.Duplicate {
		t.Fatalf("duplicate contract failed: first=%+v second=%+v", first, second)
	}
}

func TestConcurrentDuplicateCommit(t *testing.T) {
	s := Store{Root: t.TempDir(), MaxBytes: 1024}
	content := []byte("concurrent identical content")
	var wg sync.WaitGroup
	results := make(chan Blob, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := s.Put(context.Background(), bytes.NewReader(content))
			if err != nil {
				errs <- err
				return
			}
			results <- b
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var xs []Blob
	for b := range results {
		xs = append(xs, b)
	}
	if len(xs) != 2 || xs[0].ID != xs[1].ID {
		t.Fatalf("concurrent result mismatch: %+v", xs)
	}
	if xs[0].Duplicate == xs[1].Duplicate {
		t.Fatalf("expected one creator and one duplicate: %+v", xs)
	}
}

func TestTamperDetected(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root, MaxBytes: 1024}
	b, err := s.Put(context.Background(), bytes.NewReader([]byte("original")))
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.TrimPrefix(b.ID, "sha256:")
	p := filepath.Join(root, "sha256", digest[:2], digest+".blob")
	if err := os.WriteFile(p, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = s.Verify(context.Background(), b.ID)
	if !errors.Is(err, ErrIntegrity) {
		t.Fatalf("expected integrity failure, got %v", err)
	}
}

func TestInvalidIDAndTraversalRefused(t *testing.T) {
	s := Store{Root: t.TempDir(), MaxBytes: 1024}
	for _, id := range []string{"", "../../etc/passwd", "sha256:../bad", "sha256:ABCDEF", "md5:0011"} {
		if _, err := s.Verify(context.Background(), id); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("%q: expected ErrInvalidID, got %v", id, err)
		}
	}
}

func TestOpaqueHostileLookingBytesPreserved(t *testing.T) {
	s := Store{Root: t.TempDir(), MaxBytes: 4096}
	payload := []byte("<html><script>location='https://example.invalid'</script><svg onload='x()'></svg>\x00MZ")
	b, err := s.Put(context.Background(), bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if _, err := s.CopyVerified(context.Background(), b.ID, &got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), payload) {
		t.Fatal("opaque payload was changed")
	}
}

func TestSafeDownloadHeaders(t *testing.T) {
	h := SafeDownloadHeaders("../../dangerous.svg")
	if got := h.Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("content type = %q", got)
	}
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("nosniff = %q", got)
	}
	if got := h.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache control = %q", got)
	}
	cd := h.Get("Content-Disposition")
	if !strings.HasPrefix(cd, "attachment;") || strings.Contains(cd, "..") || strings.Contains(cd, "/") {
		t.Fatalf("unsafe disposition: %q", cd)
	}
}

func TestCancelledIngestDoesNotCommit(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root, MaxBytes: 1 << 20}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Put(ctx, io.LimitReader(strings.NewReader(strings.Repeat("x", 1024)), 1024))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "sha256", "*", "*.blob"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("cancelled ingest left durable blob: %v", matches)
	}
}

func TestStoredBlobUsesPrivatePermissions(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root, MaxBytes: 1024}
	b, err := s.Put(context.Background(), bytes.NewReader([]byte("private")))
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.TrimPrefix(b.ID, "sha256:")
	info, err := os.Stat(filepath.Join(root, "sha256", digest[:2], digest+".blob"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("blob mode = %o, want 600", got)
	}
}
