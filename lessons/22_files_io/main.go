// Lesson 22: files and the io interfaces.
//
// io.Reader and io.Writer are the two most important interfaces in Go. Learn
// to think in them and files, network sockets, buffers, compressors and HTTP
// bodies all become interchangeable.
//
// Run: go run ./lessons/22_files_io
package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Work in a temp dir that cleans itself up.
	dir, err := os.MkdirTemp("", "lesson22")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	fmt.Println("working in a temp dir")

	path := filepath.Join(dir, "notes.txt")

	// --- Whole-file write and read: simplest, fine for small files ---
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		panic(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	fmt.Printf("read %d bytes\n", len(data))

	// --- Open + defer Close: the pattern for everything streaming ---
	f, err := os.Open(path) // read-only
	if err != nil {
		panic(err)
	}
	defer f.Close() // ALWAYS defer Close immediately after a successful Open

	// --- bufio.Scanner: read line by line without loading the whole file ---
	sc := bufio.NewScanner(f)
	// Default max token is 64KB; raise it for long lines:
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		fmt.Printf("  %d: %s\n", lineNo, sc.Text())
	}
	if err := sc.Err(); err != nil { // NEVER forget this check
		panic(err)
	}

	// --- Create / truncate, and buffered writing ---
	outPath := filepath.Join(dir, "out.txt")
	out, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(out) // batches small writes into one syscall
	for i := range 3 {
		fmt.Fprintf(w, "row %d\n", i)
	}
	if err := w.Flush(); err != nil { // buffered data is lost without Flush
		panic(err)
	}
	out.Close()
	fmt.Println("buffered write ok:", fileSize(outPath), "bytes")

	// --- Append mode: flags are OR'd together ---
	ap, err := os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(ap, "appended line")
	ap.Close()
	fmt.Println("after append:", fileSize(outPath), "bytes")

	// --- io.Copy: stream from any Reader to any Writer, constant memory ---
	src, _ := os.Open(outPath)
	defer src.Close()
	dstPath := filepath.Join(dir, "copy.txt")
	dst, _ := os.Create(dstPath)
	n, err := io.Copy(dst, src)
	dst.Close()
	fmt.Println("copied", n, "bytes, err:", err)

	// --- Readers and writers are composable ---
	// strings.Reader, bytes.Buffer, os.Stdout: all the same interfaces.
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, strings.NewReader("from a string reader"))
	fmt.Println("buffer:", buf.String())

	// io.MultiWriter: write once, land in several places (file + stdout = tee).
	var log bytes.Buffer
	tee := io.MultiWriter(&log, io.Discard)
	fmt.Fprintln(tee, "tee'd output")
	fmt.Print("captured: " + log.String())

	// io.LimitReader: refuse to read more than N bytes (protects against
	// hostile or runaway input - use it on HTTP request bodies).
	limited, _ := io.ReadAll(io.LimitReader(strings.NewReader("abcdefghij"), 4))
	fmt.Println("limited to 4:", string(limited))

	// io.TeeReader: read and copy in one pass (e.g. hash while uploading).
	var sink bytes.Buffer
	_, _ = io.ReadAll(io.TeeReader(strings.NewReader("hash me"), &sink))
	fmt.Println("tee reader captured:", sink.String())

	// --- Checking errors properly: use errors.Is, not string matching ---
	if _, err := os.Open(filepath.Join(dir, "missing.txt")); err != nil {
		fmt.Println("is not exist:", errors.Is(err, os.ErrNotExist))
		var pe *os.PathError
		if errors.As(err, &pe) {
			fmt.Println("failed op:", pe.Op)
		}
	}

	// io.EOF is a normal, expected sentinel - not a failure.
	r := strings.NewReader("ab")
	b := make([]byte, 4)
	for {
		nn, err := r.Read(b)
		if nn > 0 {
			fmt.Println("read chunk:", string(b[:nn]))
		}
		if err == io.EOF { // handle data BEFORE checking the error
			fmt.Println("reached EOF")
			break
		}
		if err != nil {
			panic(err)
		}
	}

	// --- File metadata and directories ---
	info, _ := os.Stat(outPath)
	fmt.Printf("name=%s size=%d mode=%v dir=%t\n", info.Name(), info.Size(), info.Mode(), info.IsDir())

	sub := filepath.Join(dir, "a", "b")
	_ = os.MkdirAll(sub, 0o755) // like mkdir -p
	_ = os.WriteFile(filepath.Join(sub, "deep.txt"), []byte("x"), 0o644)

	entries, _ := os.ReadDir(dir)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	fmt.Println("dir entries:", names)

	// Walk a tree.
	count := 0
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err // propagate, or return nil/fs.SkipDir to continue
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	fmt.Println("files found by walk:", count)

	// Path helpers - use filepath (OS-aware), not string concatenation.
	fmt.Println("base/dir/ext:", filepath.Base(outPath), filepath.Dir(outPath), filepath.Ext(outPath))

	// Rename and remove.
	renamed := filepath.Join(dir, "renamed.txt")
	_ = os.Rename(dstPath, renamed)
	_ = os.Remove(renamed)
	fmt.Println("removed:", !exists(renamed))

	// --- CSV, as a taste of a format built on io.Reader/Writer ---
	var csvBuf bytes.Buffer
	cw := csv.NewWriter(&csvBuf)
	_ = cw.WriteAll([][]string{{"name", "age"}, {"Alice", "30"}, {"Bob", "25"}})
	cw.Flush()
	rows, _ := csv.NewReader(strings.NewReader(csvBuf.String())).ReadAll()
	fmt.Println("csv rows:", rows)

	// --- Atomic writes: write to a temp file, then rename.
	//     Rename is atomic on POSIX, so readers never see a half-written file. ---
	if err := atomicWrite(filepath.Join(dir, "config.json"), []byte(`{"ok":true}`)); err != nil {
		panic(err)
	}
	fmt.Println("atomic write done")

	// --- Stdin/stdout/stderr are just *os.File ---
	fmt.Fprintln(os.Stderr, "this goes to stderr")
	fmt.Println("env PATH set:", os.Getenv("PATH") != "")
	fmt.Println("args:", os.Args[0] != "")
}

func fileSize(p string) int64 {
	info, err := os.Stat(p)
	if err != nil {
		return -1
	}
	return info.Size()
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func atomicWrite(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil { // force to disk before swapping
		tmp.Close()
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	return os.Rename(tmp.Name(), path)
}
