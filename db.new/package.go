package db

import (
	"time"
)

type Package struct {
	Repository    string
	Name          string
	Version       string
	Arch          string
	Description   string
	PackageSize   int64
	InstalledSize int64
	URL           string
	Licenses      []string
	Groups        []string
	BuildDate     time.Time
	Depends       []string
	MakeDepends   []string
	OptDepends    []string
	Files         []string
	Md5Sum        string
	Sha256Sum     string
	Filename      string
}

func (p Package) CompleteName() string {
	return completeName(p.Name, p.Version)
}

func (p Package) RepoName() string {
	return repoName(p.Name, p.Version, p.Repository)
}

func (p Package) FileName() string {
	if p.Filename == "" {
		return fileName(p.Name, p.Version, p.Arch)
	}
	return p.Filename
}

func (p Package) IsFlagged() bool {
	cb := func(e Data) bool {
		f := e.(Flag)
		return f.Flagged && f.RepoName() == p.RepoName()
	}
	return Contains("flag", cb)
}

func (p Package) GetGit(git *Git) (ok bool) {
	cb := func(e Data) bool { return e.(Git).Name == p.Name }
	return Find("git", git, cb)
}
