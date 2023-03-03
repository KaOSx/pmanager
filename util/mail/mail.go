package mail

import (
	"bytes"
	"io"
	"strings"
)

type header struct {
	key   string
	value string
}

func (h header) String() string {
	return h.key + ": " + h.value
}

type Mail struct {
	from    string
	to      []string
	headers []header
	subject string
	body    string
}

func (m *Mail) From(from string) *Mail {
	m.from = from

	return m
}

func (m *Mail) To(to ...string) *Mail {
	m.to = append(m.to, to...)

	return m
}

func (m *Mail) Header(key, value string) *Mail {
	m.headers = append(m.headers, header{
		key:   key,
		value: value,
	})

	return m
}

func (m *Mail) Subject(subject string) *Mail {
	m.subject = subject

	return m
}

func (m *Mail) Body(body string) *Mail {
	m.body = body

	return m
}

func (m Mail) contains(key string) bool {
	for _, h := range m.headers {
		if h.key == key {
			return true
		}
	}

	return false
}

func (m Mail) Reader() io.Reader {
	var buf bytes.Buffer

	if !m.contains("From") {
		buf.WriteString("From: " + m.from + "\r\n")
	}

	if !m.contains("To") {
		buf.WriteString("To: " + strings.Join(m.to, ";") + "\r\n")
	}

	for _, h := range m.headers {
		buf.WriteString(h.String() + "\r\n")
	}

	if !m.contains("Subject") {
		buf.WriteString("Subject: " + m.subject + "\r\n")
	}

	if !m.contains("Content-Type") {
		buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	}

	buf.WriteString("\r\n" + m.body + "\r\n")

	return &buf
}
