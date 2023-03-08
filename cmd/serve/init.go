package serve

import (
	"pmanager/conf"
	database "pmanager/database2"
	"pmanager/util/mail"
)

var (
	port              string
	defaultPagination int64
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
	port = conf.String("api.port")
	defaultPagination = conf.Int("api.pagination")
}
