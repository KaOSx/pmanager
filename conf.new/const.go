package conf

import (
	"embed"
)

var (
	Debug    = false
	ConfDir  = "/etc/pmanager"
	ConfFile = "pmanager.conf"
	cnf      *configuration
)

//go:embed model/pmanager.conf
var model embed.FS
