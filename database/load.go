package database

import (
	"pmanager/log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	dbsingleton *database
)

func load(uri string) gorm.Dialector {
	return sqlite.Open(uri)
}

// Load loads the SQlite database file.
func Load(uri string) {
	var err error
	if dbsingleton, err = newDb(load(uri)); err != nil {
		log.Fatalf("Failed to load the database: %s\n", err)
	}

	err = dbsingleton.AutoMigrate(
		&Git{},
		&Flag{},
		&Package{},
		&Repo{},
		&Mirror{},
		&Country{},
	)

	if err != nil {
		log.Fatalf("Failed to update the schema database: %s\n", err)
	}
}
