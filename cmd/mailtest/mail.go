package mailtest

import (
	"fmt"
	"pmanager/conf"
	"pmanager/util/mail"
)

func SendMail() {
	mail.InitSmtp(
		conf.String("smtp.host"),
		conf.String("smtp.port"),
		conf.String("smtp.user"),
		conf.String("smtp.password"),
		conf.Bool("smtp.use_encryption"),
	)

	var m mail.Mail
	m.From(conf.String("smtp.send_from")).
		To(conf.String("smtp.send_to")).
		Header("Reply-To", conf.String("smtp.send_to")).
		Header("X-Mailer", "Packages").
		Header("MIME-Version", "1.0").
		Header("Content-Transfer-Encoding", "8bit").
		Header("Content-type", "text/plain; charset=utf-8").
		Subject("Test email from pmanager").
		Body("body of the mail")

	if err := mail.Send(m); err != nil {
		fmt.Println(err)
	}
}
