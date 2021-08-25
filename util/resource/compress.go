package resource

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type FakeReadCloser struct {
	io.Reader
}

func (r FakeReadCloser) Close() error { return nil }

func ReadGZ(r io.Reader) (io.ReadCloser, error) { return gzip.NewReader(r) }

func ReadZST(r io.Reader) (io.ReadCloser, error) {
	rc, err := zstd.NewReader(r)
	return rc.IOReadCloser(), err
}

func ReadTAR(r io.Reader) *tar.Reader { return tar.NewReader(r) }

func Decompress(uri string) (rc io.ReadCloser, err error) {
	var dec func(io.Reader) (io.ReadCloser, error)
	switch {
	case strings.HasSuffix(uri, ".gz"):
		dec = ReadGZ

	case strings.HasSuffix(uri, "zst"):
		dec = ReadZST
	default:
		err = errors.New("Unsupported compression format")
		return
	}
	var r io.Reader
	if r, err = Open(uri); err != nil {
		return
	}
	return dec(r)
}

func OpenTAR(uri string) (tr *tar.Reader, err error) {
	var r io.Reader
	if r, err = Open(uri); err == nil {
		tr = ReadTAR(r)
	}
	return
}

func OpenArchive(uri string) (tr *tar.Reader, err error) {
	var rc io.ReadCloser
	if rc, err = Decompress(uri); err != nil {
		return
	}
	defer rc.Close()
	r := new(bytes.Buffer)
	if _, err = io.Copy(r, rc); err == nil {
		tr = ReadTAR(r)
	}
	return
}
