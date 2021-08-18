package serve

import (
	"pmanager/database"

	"pmanager/conf.new"
	"pmanager/util.new/mail"
)

func init() {
	database.Load(conf.String("database.uri"))
	mail.InitSmtp(
		conf.String("smtp.host"),
		conf.String("smtp.port"),
		conf.String("smtp.user"),
		conf.String("smtp.password"),
		conf.Bool("smtp.use_encryption"),
	)
}
