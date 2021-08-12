package resource

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
)

func IsURL(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

func IsPath(uri string) bool {
	return !IsURL(uri)
}

func isAvailableURL(uri string) bool {
	resp, err := http.Head(uri)
	return err == nil && resp.StatusCode == http.StatusOK
}

func isAvailablePath(uri string) bool {
	_, err := os.Stat(uri)
	return err != nil
}

func Exists(uri string) bool {
	if IsURL(uri) {
		return isAvailableURL(uri)
	}
	return isAvailablePath(uri)
}

func IsFile(uri string) bool {
	if IsPath(uri) {
		if st, err := os.Stat(uri); err == nil {
			return !st.IsDir()
		}
	}
	return false
}

func IsDir(uri string) bool {
	if IsPath(uri) {
		if st, err := os.Stat(uri); err == nil {
			return st.IsDir()
		}
	}
	return false
}

func Open(uri string) (data *bytes.Buffer, err error) {
	var f io.ReadCloser
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		var resp *http.Response
		if resp, err = http.Get(uri); err == nil {
			f = resp.Body
		}
	} else {
		f, err = os.Open(uri)
	}
	if err == nil {
		defer f.Close()
		data = new(bytes.Buffer)
		_, err = io.Copy(data, f)
	}
	return
}
