package util

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/xi2/xz"
)

func ReadXZ(r io.Reader) (*xz.Reader, error) { return xz.NewReader(r, 0) }

func ReadGZ(r io.Reader) (*gzip.Reader, error) { return gzip.NewReader(r) }

func ReadTAR(r io.Reader) *tar.Reader { return tar.NewReader(r) }
