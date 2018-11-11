package db

import (
	"sort"
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
}

type Packagelist []*Package

func (p *Package) CompleteName() string {
	return completeName(p.Name, p.Version)
}

func (p *Package) RepoName() string {
	return repoName(p.Name, p.Version, p.Repository)
}

func (p *Package) FileName() string {
	return fileName(p.Name, p.Version, p.Arch)
}

func (p *Package) IsFlagged() bool {
	fl := LoadFlags()
	filter := func(f *Flag) bool {
		return f.Flagged && f.RepoName() == p.RepoName()
	}
	if fl.Filter(filter).Len() > 0 {
		return true
	}
	return false
}

func (p *Package) GetGit() *Git { return LoadGits().Get(p.Name) }

func (pl Packagelist) Len() int { return len(pl) }

func (pl Packagelist) Get(pkgname, pkgver, repo string) *Package {
	for _, p := range pl {
		if p.Name == pkgname && p.Version == pkgver && p.Repository == repo {
			return p
		}
	}
	return nil
}

func (pl *Packagelist) Add(packages ...*Package) *Packagelist {
	*pl = append(*pl, packages...)
	return pl
}

func (pl *Packagelist) Remove(packages ...*Package) *Packagelist {
	out := make(Packagelist, 0, len(*pl))
	mp := make(map[string]bool)
	for _, p := range packages {
		mp[p.RepoName()] = true
	}
	for _, p := range *pl {
		if !mp[p.RepoName()] {
			out = append(out, p)
		}
	}
	*pl = out
	return pl
}

func (pl *Packagelist) Filter(args ...func(*Package) bool) *Packagelist {
	out := make(Packagelist, 0, len(*pl))
	filter := func(p *Package) bool {
		for _, op := range args {
			if !op(p) {
				return false
			}
		}
		return true
	}
	for _, p := range *pl {
		if filter(p) {
			out = append(out, p)
		}
	}
	return &out
}

func (pl *Packagelist) Sort(args ...func(*Package, *Package) int) *Packagelist {
	less := func(p1, p2 *Package) bool {
		for _, op := range args {
			c := op(p1, p2)
			if c != 0 {
				return c < 0
			}
		}
		return true
	}
	out := make(Packagelist, len(*pl))
	copy(out, *pl)
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return &out
}

func (pl *Packagelist) LimitOffset(l, o int64) *Packagelist {
	c := int64(len(*pl))
	if o < 0 || o >= c || l <= 0 {
		return new(Packagelist)
	}
	var out Packagelist
	if c < o+l {
		out = (*pl)[o:]
	} else {
		out = (*pl)[o : o+l]
	}
	return &out
}
