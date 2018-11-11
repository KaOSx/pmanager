package db

import (
	"sort"
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

type Flaglist []*Flag

func (f *Flag) Flag() *Flag {
	f.Flagged = true
	return f
}

func (f *Flag) Unflag() *Flag {
	f.Flagged = false
	return f
}

func (f *Flag) CompleteName() string {
	return completeName(f.Name, f.Version)
}

func (f *Flag) RepoName() string {
	return repoName(f.Name, f.Version, f.Repository)
}

func (f *Flag) FileName() string {
	return fileName(f.Name, f.Version, f.Arch)
}

//TODO func(f *Flag)GetPackage() *Package

func (fl Flaglist) Len() int { return len(fl) }

func (fl Flaglist) Get(pkgname, pkgver, repo string) *Flag {
	for _, f := range fl {
		if f.Name == pkgname && f.Version == pkgver && f.Repository == repo {
			return f
		}
	}
	return nil
}

func (fl *Flaglist) Add(flags ...*Flag) *Flaglist {
	*fl = append(*fl, flags...)
	return fl
}

func (fl *Flaglist) Remove(flags ...*Flag) *Flaglist {
	out := make(Flaglist, 0, len(*fl))
	mf := make(map[string]bool)
	for _, f := range flags {
		mf[f.RepoName()] = true
	}
	for _, f := range *fl {
		if !mf[f.RepoName()] {
			out = append(out, f)
		}
	}
	*fl = out
	return fl
}

func (fl *Flaglist) Flag(flags ...*Flag) *Flaglist {
	mf := make(map[string]bool)
	for _, f := range flags {
		mf[f.RepoName()] = true
	}
	for _, f := range *fl {
		if mf[f.RepoName()] {
			f.Flag()
		}
	}
	return fl
}

func (fl *Flaglist) Unflag(flags ...*Flag) *Flaglist {
	mf := make(map[string]bool)
	for _, f := range flags {
		mf[f.RepoName()] = true
	}
	for _, f := range *fl {
		if mf[f.RepoName()] {
			f.Unflag()
		}
	}
	return fl
}

func (fl *Flaglist) Filter(args ...func(*Flag) bool) *Flaglist {
	out := make(Flaglist, 0, len(*fl))
	filter := func(f *Flag) bool {
		for _, op := range args {
			if !op(f) {
				return false
			}
		}
		return true
	}
	for _, f := range *fl {
		if filter(f) {
			out = append(out, f)
		}
	}
	return &out
}

func (fl *Flaglist) Sort(args ...func(*Flag, *Flag) int) *Flaglist {
	less := func(f1, f2 *Flag) bool {
		for _, op := range args {
			c := op(f1, f2)
			if c != 0 {
				return c < 0
			}
		}
		return true
	}
	out := make(Flaglist, len(*fl))
	copy(out, *fl)
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return &out
}

func (fl *Flaglist) LimitOffset(l, o int64) *Flaglist {
	c := int64(len(*fl))
	if o < 0 || o >= c || l <= 0 {
		return new(Flaglist)
	}
	var out Flaglist
	if c < o+l {
		out = (*fl)[o:]
	} else {
		out = (*fl)[o : o+l]
	}
	return &out
}
