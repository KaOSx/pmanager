package mailtest

import (
	"fmt"
	"pmanager/conf"
	"pmanager/util/mail"
)

func SendMail([]string) {
	server := mail.Server{
		Host:     conf.Read("smtp.host"),
		Port:     conf.Read("smtp.port"),
		TLS:      conf.ReadBool("smtp.use_encryption"),
		User:     conf.Read("smtp.user"),
		Password: conf.Read("smtp.password"),
	}
	subject := "Test email from pmanager"
	body := "body of the mail"
	m := mail.New(server, conf.Read("smtp.send_from"), conf.Read("smtp.send_to"), subject, body)
	m.AddHeader("Reply-To", conf.Read("smtp.send_to")).
		AddHeader("X-Mailer", "Packages").
		AddHeader("MIME-Version", "1.0").
		AddHeader("Content-Transfer-Encoding", "8bit").
		AddHeader("Content-type", "text/plain; charset=utf-8")
	if err := m.Send(); err != nil {
		fmt.Println(err)
	}
}
