package database

import (
	"pmanager/log"
	"time"

	"gorm.io/gorm"
	"pmanager/conf.new"
)

type Git struct {
	gorm.Model
	Name       string
	Repository string
	Folder     string
}

type Flag struct {
	gorm.Model
	Name       string
	Version    string
	Arch       string
	Repository string
	Email      string
	Comment    string
	Flagged    bool
}

type Package struct {
	gorm.Model
	Repository    string
	Name          string
	Version       string
	Arch          string
	Description   string
	PackageSize   int64
	InstalledSize int64
	URL           string
	Licenses      SqlSlice `gorm:"type:blob"`
	Groups        SqlSlice `gorm:"type:blob"`
	BuildDate     time.Time
	Depends       SqlSlice `gorm:"type:blob"`
	MakeDepends   SqlSlice `gorm:"type:blob"`
	OptDepends    SqlSlice `gorm:"type:blob"`
	Files         SqlSlice `gorm:"type:blob"`
	Md5Sum        string
	Sha256Sum     string
	Filename      string
	FlagID        uint
	Flag          Flag
	GitID         uint
	Git           Git
}

type Repo struct {
	gorm.Model
	Name     string
	Sync     bool
	MirrorID uint
}

type Mirror struct {
	gorm.Model
	Name      string
	Online    bool
	Repos     []Repo
	CountryID uint
}

type Country struct {
	gorm.Model
	Name    string
	Mirrors []Mirror
}

func Load() {
	var err error
	if dbsingleton, err = newDb(load(conf.String("database.uri"))); err != nil {
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
