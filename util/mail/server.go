package mail

import (
	"bufio"
	"crypto/tls"
	"net"
	"net/smtp"
)

var (
	srv server
)

type server struct {
	host     string
	port     string
	tls      bool
	user     string
	password string
}

func (s server) name() string {
	return s.host + ":" + s.port
}

func (s server) auth() smtp.Auth {
	return smtp.PlainAuth("", s.user, s.password, s.host)
}

func (s server) dial() (net.Conn, error) {
	if s.tls {
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         s.host,
		}

		return tls.Dial("tcp", s.name(), tlsconfig)
	}

	return net.Dial("tcp", s.name())
}

func (s server) client() (*smtp.Client, error) {
	dial, err := s.dial()
	if err != nil {
		return nil, err
	}

	c, err := smtp.NewClient(dial, s.host)
	if err == nil {
		err = c.Auth(s.auth())
	}

	return c, err
}

func (s server) send(m Mail) error {
	c, err := s.client()
	if err != nil {
		return err
	}

	if err := c.Mail(m.from); err != nil {
		return err
	}

	for _, t := range m.to {
		if err := c.Rcpt(t); err != nil {
			return err
		}
	}

	wc, err := c.Data()
	if err != nil {
		return err
	}
	defer wc.Close()

	_, err = bufio.NewReader(m.Reader()).WriteTo(wc)
	if err != nil {
		return err
	}

	return c.Quit()
}

func InitSmtp(host, port, user, password string, tls bool) {
	srv = server{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		tls:      tls,
	}
}

func Send(mail Mail) error {
	return srv.send(mail)
}
