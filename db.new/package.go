package db

import (
	"strings"
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

func SearchPackageByName(search string, exact bool) func(Package) bool {
	if !exact {
		search = strings.ToLower(search)
	}
	return func(p Package) bool {
		if exact {
			return p.Name == search
		}
		return strings.Contains(strings.ToLower(p.CompleteName()), search)
	}
}

func SearchPackageByDateFrom(e time.Time) func(Package) bool {
	return func(p Package) bool { return !p.BuildDate.Before(e) }
}

func SearchPackageByDateTo(e time.Time) func(Package) bool {
	return func(p Package) bool { return !p.BuildDate.After(e) }
}

func SearchPackageByFlagged(e bool) func(Package) bool {
	return func(p Package) bool { return p.IsFlagged() == e }
}

func PackageFilter2MatchFunc(cb func(Package) bool) MatchFunc {
	return func(e Data) bool {
		p, ok := e.(Package)
		return ok && cb(p)
	}
}
