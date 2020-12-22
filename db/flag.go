package db

import (
	"strings"
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

func SearchFlagByName(e string) func(Flag) bool {
	e = strings.ToLower(e)
	return func(f Flag) bool {
		name := strings.ToLower(f.CompleteName())
		return strings.Contains(name, e)
	}
}

func SearchFlagByRepo(e string) func(Flag) bool {
	return func(f Flag) bool { return f.Repository == e }
}

func SearchFlagByEmail(e string) func(Flag) bool {
	e = strings.ToLower(e)
	return func(f Flag) bool {
		email := strings.ToLower(f.Email)
		return strings.Contains(email, e)
	}
}

func SearchFlagByDateFrom(e time.Time) func(Flag) bool {
	return func(f Flag) bool { return !f.Date.Before(e) }
}

func SearchFlagByDateTo(e time.Time) func(Flag) bool {
	return func(f Flag) bool { return !f.Date.After(e) }
}

func SearchFlagByFlagged(e bool) func(Flag) bool {
	return func(f Flag) bool { return f.Flagged == e }
}

func FlagFilter2MatchFunc(cb func(Flag) bool) MatchFunc {
	return func(e Data) bool {
		f, ok := e.(Flag)
		return ok && cb(f)
	}
}

func FlagCmp2CmpFunc(cb func(Flag, Flag) int) CmpFunc {
	return func(e1, e2 Data) int {
		f1, _ := e1.(Flag)
		f2, _ := e2.(Flag)
		return cb(f1, f2)
	}
}
