package blobstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	ErrTooLarge  = errors.New("blob exceeds configured maximum")
	ErrInvalidID = errors.New("invalid blob id")
	ErrNotFound  = errors.New("blob not found")
	ErrIntegrity = errors.New("blob integrity verification failed")
)

type Blob struct {
	ID        string `json:"id"`
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

type Store struct {
	Root     string
	MaxBytes int64
}

func (s Store) Put(ctx context.Context, r io.Reader) (Blob, error) {
	if err := s.validate(); err != nil {
		return Blob{}, err
	}
	if r == nil {
		return Blob{}, errors.New("reader is required")
	}
	if err := os.MkdirAll(s.Root, 0700); err != nil {
		return Blob{}, err
	}
	tmpDir := filepath.Join(s.Root, ".tmp")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return Blob{}, err
	}
	tmp, err := os.CreateTemp(tmpDir, "ingest-*")
	if err != nil {
		return Blob{}, err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	defer cleanup()
	if err := tmp.Chmod(0600); err != nil {
		return Blob{}, err
	}

	h := sha256.New()
	limited := io.LimitReader(r, s.MaxBytes+1)
	n, err := copyWithContext(ctx, io.MultiWriter(tmp, h), limited)
	if err != nil {
		return Blob{}, err
	}
	if n > s.MaxBytes {
		return Blob{}, ErrTooLarge
	}
	if err := tmp.Sync(); err != nil {
		return Blob{}, err
	}
	if err := tmp.Close(); err != nil {
		return Blob{}, err
	}

	digest := hex.EncodeToString(h.Sum(nil))
	id := "sha256:" + digest
	finalDir := filepath.Join(s.Root, "sha256", digest[:2])
	if err := os.MkdirAll(finalDir, 0700); err != nil {
		return Blob{}, err
	}
	finalPath := filepath.Join(finalDir, digest+".blob")

	if err := os.Link(tmpName, finalPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			b, verifyErr := s.Verify(ctx, id)
			if verifyErr != nil {
				return Blob{}, verifyErr
			}
			if b.Size != n {
				return Blob{}, ErrIntegrity
			}
			b.Duplicate = true
			return b, nil
		}
		return Blob{}, err
	}
	if err := syncDir(finalDir); err != nil {
		return Blob{}, err
	}

	return Blob{ID: id, SHA256: digest, Size: n}, nil
}

func (s Store) Verify(ctx context.Context, id string) (Blob, error) {
	if err := s.validateRoot(); err != nil {
		return Blob{}, err
	}
	digest, err := parseID(id)
	if err != nil {
		return Blob{}, err
	}
	f, err := os.Open(s.pathForDigest(digest))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Blob{}, ErrNotFound
		}
		return Blob{}, err
	}
	defer f.Close()

	h := sha256.New()
	n, err := copyWithContext(ctx, h, f)
	if err != nil {
		return Blob{}, err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != digest {
		return Blob{}, ErrIntegrity
	}
	return Blob{ID: id, SHA256: digest, Size: n}, nil
}

func (s Store) CopyVerified(ctx context.Context, id string, w io.Writer) (Blob, error) {
	if w == nil {
		return Blob{}, errors.New("writer is required")
	}
	b, err := s.Verify(ctx, id)
	if err != nil {
		return Blob{}, err
	}
	digest, _ := parseID(id)
	f, err := os.Open(s.pathForDigest(digest))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Blob{}, ErrNotFound
		}
		return Blob{}, err
	}
	defer f.Close()
	if _, err := copyWithContext(ctx, w, f); err != nil {
		return Blob{}, err
	}
	return b, nil
}

func SafeDownloadHeaders(filename string) http.Header {
	h := make(http.Header)
	h.Set("Content-Type", "application/octet-stream")
	h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeFilename(filename)}))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-store")
	return h
}

func (s Store) validate() error {
	if err := s.validateRoot(); err != nil {
		return err
	}
	if s.MaxBytes <= 0 {
		return errors.New("max bytes must be greater than zero")
	}
	return nil
}

func (s Store) validateRoot() error {
	if strings.TrimSpace(s.Root) == "" {
		return errors.New("store root is required")
	}
	return nil
}

func (s Store) pathForDigest(digest string) string {
	return filepath.Join(s.Root, "sha256", digest[:2], digest+".blob")
}

func parseID(id string) (string, error) {
	if !strings.HasPrefix(id, "sha256:") {
		return "", ErrInvalidID
	}
	digest := strings.TrimPrefix(id, "sha256:")
	if len(digest) != 64 {
		return "", ErrInvalidID
	}
	for _, r := range digest {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return "", ErrInvalidID
		}
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", ErrInvalidID
	}
	return digest, nil
}

func safeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" {
		return "download.bin"
	}
	return name
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 64*1024)
	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			wn, werr := dst.Write(buf[:n])
			total += int64(wn)
			if werr != nil {
				return total, werr
			}
			if wn != n {
				return total, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return total, nil
			}
			return total, rerr
		}
	}
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync directory: %w", err)
	}
	return nil
}
