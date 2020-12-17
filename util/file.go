package util

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/xi2/xz"
)

type FakeReadCloser struct {
	io.Reader
}

func (r FakeReadCloser) Close() error { return nil }

func ReadXZ(r io.Reader) (rc FakeReadCloser, err error) {
	rc.Reader, err = xz.NewReader(r, 0)
	return rc, err
}

func ReadGZ(r io.Reader) (*gzip.Reader, error) { return gzip.NewReader(r) }

func ReadZST(r io.Reader) (io.ReadCloser, error) {
	rc, err := zstd.NewReader(r)
	return rc.IOReadCloser(), err
}

func ReadTAR(r io.Reader) *tar.Reader { return tar.NewReader(r) }
