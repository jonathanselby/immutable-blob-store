package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jonathanselby/immutable-blob-store/internal/blobstore"
)

const version = "0.1.0"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-version" {
		fmt.Println("blobstore " + version)
		return
	}
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "put":
		put(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	case "get":
		get(os.Args[2:])
	default:
		usage()
	}
}

func put(args []string) {
	fs := flag.NewFlagSet("put", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var root, input string
	var max int64
	fs.StringVar(&root, "store", "", "store root directory")
	fs.StringVar(&input, "file", "", "input file")
	fs.Int64Var(&max, "max-bytes", 0, "maximum accepted bytes")
	if fs.Parse(args) != nil || fs.NArg() != 0 || root == "" || input == "" || max <= 0 {
		usage()
	}
	f, err := os.Open(input)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	b, err := (blobstore.Store{Root: root, MaxBytes: max}).Put(context.Background(), f)
	if err != nil {
		handle(err)
	}
	writeJSON(b)
}

func verify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var root, id string
	fs.StringVar(&root, "store", "", "store root directory")
	fs.StringVar(&id, "id", "", "blob id")
	if fs.Parse(args) != nil || fs.NArg() != 0 || root == "" || id == "" {
		usage()
	}
	b, err := (blobstore.Store{Root: root, MaxBytes: 1}).Verify(context.Background(), id)
	if err != nil {
		handle(err)
	}
	writeJSON(b)
}

func get(args []string) {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var root, id, output string
	fs.StringVar(&root, "store", "", "store root directory")
	fs.StringVar(&id, "id", "", "blob id")
	fs.StringVar(&output, "output", "", "new output file")
	if fs.Parse(args) != nil || fs.NArg() != 0 || root == "" || id == "" || output == "" {
		usage()
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		fatal(err)
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(output)
		}
	}()
	b, err := (blobstore.Store{Root: root, MaxBytes: 1}).CopyVerified(context.Background(), id, f)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(output)
		handle(err)
	}
	if err := f.Sync(); err != nil {
		fatal(err)
	}
	if err := f.Close(); err != nil {
		fatal(err)
	}
	ok = true
	writeJSON(b)
}

func writeJSON(v any) {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		fatal(err)
	}
}

func handle(err error) {
	if errors.Is(err, blobstore.ErrTooLarge) || errors.Is(err, blobstore.ErrInvalidID) || errors.Is(err, blobstore.ErrNotFound) || errors.Is(err, blobstore.ErrIntegrity) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fatal(err)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(3)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  blobstore -version")
	fmt.Fprintln(os.Stderr, "  blobstore put -store <dir> -file <path> -max-bytes <n>")
	fmt.Fprintln(os.Stderr, "  blobstore verify -store <dir> -id <sha256:id>")
	fmt.Fprintln(os.Stderr, "  blobstore get -store <dir> -id <sha256:id> -output <new-file>")
	os.Exit(2)
}
