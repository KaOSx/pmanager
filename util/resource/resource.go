package resource

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	TimeoutInSeconds time.Duration = 20
)

func IsURL(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

func IsPath(uri string) bool {
	return !IsURL(uri)
}

func isAvailableURL(uri string) bool {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	resp, err := client.Head(uri)

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

func Open(uri string) (data io.Reader, err error) {
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
		var buf bytes.Buffer
		if _, err = io.Copy(&buf, f); err == nil {
			data = &buf
		}
	}

	return
}

func IsPortOpen(host, port string) bool {
	timeout := 5 * time.Second
	target := host + ":" + port
	conn, err := net.DialTimeout("tcp", target, timeout)

	if conn == nil || err != nil {
		return false
	}
	defer conn.Close()

	return true
}
