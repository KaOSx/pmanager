package update

import (
	"pmanager/conf"
	database "pmanager/database2"
	"pmanager/util/resource"
)

var (
	serverOpen bool
	upd        = map[string]func() map[string]int{
		"mirror": func() map[string]int {
			return database.UpdateMirrors(
				conf.String("mirror.pacmanconf"),
				conf.String("mirror.mirrorlist"),
				conf.String("mirror.main_mirror"),
			)
		},
		"repo": func() map[string]int {
			return database.UpdatePackages(
				conf.String("repository.basedir"),
				conf.String("repository.extension"),
				conf.Slice("repository.include"),
				conf.Slice("repository.exclude"),
			)
		},
		"all": func() map[string]int {
			return database.UpdateAll(
				conf.String("mirror.pacmanconf"),
				conf.String("mirror.mirrorlist"),
				conf.String("mirror.main_mirror"),
				conf.String("repository.basedir"),
				conf.String("repository.extension"),
				conf.Slice("repository.include"),
				conf.Slice("repository.exclude"),
			)
		},
	}
)

func init() {
	port := conf.String("api.port")
	if serverOpen = resource.IsPortOpen("localhost", port); !serverOpen {
		database.Load(conf.String("database.uri"))
	}
}
