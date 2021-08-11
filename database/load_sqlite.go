package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func load(uri string) gorm.Dialector {
	return sqlite.Open(uri)
}
