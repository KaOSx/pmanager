package mail

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/smtp"
	"strings"
)

type Server struct {
	Host     string
	Port     string
	TLS      bool
	User     string
	Password string
}

func (s Server) ServerName() string {
	return s.Host + ":" + s.Port
}

func (s Server) Auth() smtp.Auth {
	return smtp.PlainAuth("", s.User, s.Password, s.Host)
}

func (s Server) Dial() (net.Conn, error) {
	if s.TLS {
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         s.Host,
		}
		return tls.Dial("tcp", s.ServerName(), tlsconfig)
	}
	return net.Dial("tcp", s.ServerName())
}

func (s Server) Client() (*smtp.Client, error) {
	dial, err := s.Dial()
	if err != nil {
		return nil, err
	}
	c, err := smtp.NewClient(dial, s.Host)
	if err == nil {
		err = c.Auth(s.Auth())
	}
	return c, err
}

type Header struct {
	Key   string
	Value string
}

func (h Header) String() string {
	return h.Key + ": " + h.Value
}

type Mail struct {
	Server  Server
	From    string
	To      []string
	Headers []Header
	Subject string
	Body    string
}

func New(s Server, from, to, subject, body string) *Mail {
	return &Mail{
		Server:  s,
		From:    from,
		To:      []string{to},
		Subject: subject,
		Body:    body,
	}
}

func (m *Mail) AddRcpt(rcpt ...string) *Mail {
	m.To = append(m.To, rcpt...)
	return m
}

func (m *Mail) AddHeader(key, value string) *Mail {
	m.Headers = append(m.Headers, Header{
		Key:   key,
		Value: value,
	})
	return m
}

func (m *Mail) ContainsHeader(key string) bool {
	for _, h := range m.Headers {
		if h.Key == key {
			return true
		}
	}
	return false
}

func (m *Mail) BuildMessage() io.Reader {
	var buf bytes.Buffer
	if !m.ContainsHeader("From") {
		buf.WriteString("From: " + m.From + "\r\n")
	}
	if !m.ContainsHeader("To") {
		buf.WriteString("To: " + strings.Join(m.To, ";") + "\r\n")
	}
	for _, h := range m.Headers {
		buf.WriteString(h.String() + "\r\n")
	}
	if !m.ContainsHeader("Subject") {
		buf.WriteString("Subject: " + m.Subject + "\r\n")
	}
	if !m.ContainsHeader("Content-Type") {
		buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	}
	buf.WriteString("\r\n" + m.Body + "\r\n")
	return &buf
}

func (m *Mail) Send() error {
	c, err := m.Server.Client()
	if err != nil {
		return err
	}
	if err := c.Mail(m.From); err != nil {
		return err
	}
	for _, t := range m.To {
		if err := c.Rcpt(t); err != nil {
			return err
		}
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	defer wc.Close()
	_, err = bufio.NewReader(m.BuildMessage()).WriteTo(wc)
	if err != nil {
		return err
	}
	return c.Quit()
}
