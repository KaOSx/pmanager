package database

import (
	"fmt"
	"pmanager/log"
	"time"

	"gorm.io/gorm"
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
}

func (f Flag) RepoName() string {
	return fmt.Sprintf("%s/%s", f.Repository, f.Name)
}

func (f Flag) VersionName() string {
	return fmt.Sprintf("%s-%s", f.Name, f.Version)
}

func (f Flag) FullName() string {
	return fmt.Sprintf("%s/%s-%s", f.Repository, f.Name, f.Version)
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
	BuildVersion  *Package `gorm:"-"`
}

func (p Package) RepoName() string {
	return fmt.Sprintf("%s/%s", p.Repository, p.Name)
}

func (p Package) VersionName() string {
	return fmt.Sprintf("%s-%s", p.Name, p.Version)
}

func (p Package) FullName() string {
	return fmt.Sprintf("%s/%s-%s", p.Repository, p.Name, p.Version)
}

func (p1 Package) Equal(p2 Package) bool {
	return p1.ID == p2.ID &&
		p1.Repository == p2.Repository &&
		p1.Name == p2.Name &&
		p1.Version == p2.Version &&
		p1.Arch == p2.Arch &&
		p1.Description == p2.Description &&
		p1.PackageSize == p2.PackageSize &&
		p1.InstalledSize == p2.InstalledSize &&
		p1.URL == p2.URL &&
		p1.Licenses.Equal(p2.Licenses) &&
		p1.Groups.Equal(p2.Groups) &&
		p1.BuildDate.Equal(p2.BuildDate) &&
		p1.Depends.Equal(p2.Depends) &&
		p1.Files.Equal(p2.Files) &&
		p1.Md5Sum == p2.Md5Sum &&
		p1.Sha256Sum == p2.Sha256Sum &&
		p1.Filename == p2.Filename &&
		p1.FlagID == p2.FlagID &&
		p1.GitID == p2.GitID
}

type Repo struct {
	gorm.Model
	Name     string
	Sync     bool
	MirrorID uint
	md5      string `gorm:"-"`
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
