package db

import (
	"time"
)

type Flag struct {
	Name       string
	Version    string
	Arch       string
	Repository string
	Email      string
	Comment    string
	Flagged    bool
	Date       time.Time
}

func (f Flag) Flag() Flag {
	f.Flagged = true
	return f
}

func (f Flag) Unflag() Flag {
	f.Flagged = false
	return f
}

func (f Flag) CompleteName() string {
	return completeName(f.Name, f.Version)
}

func (f Flag) RepoName() string {
	return repoName(f.Name, f.Version, f.Repository)
}

func (f Flag) FileName() string {
	return fileName(f.Name, f.Version, f.Arch)
}
