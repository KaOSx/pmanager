package util

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/klauspost/compress/zstd"
)

type FakeReadCloser struct {
	io.Reader
}

func (r FakeReadCloser) Close() error { return nil }

func ReadGZ(r io.Reader) (*gzip.Reader, error) { return gzip.NewReader(r) }

func ReadZST(r io.Reader) (io.ReadCloser, error) {
	rc, err := zstd.NewReader(r)
	return rc.IOReadCloser(), err
}

func ReadTAR(r io.Reader) *tar.Reader { return tar.NewReader(r) }
