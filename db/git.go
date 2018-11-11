package db

import (
	"sort"
)

type Git struct {
	Name       string
	Repository string
	Folder     string
}

type Gitlist []*Git

func (gl Gitlist) Len() int { return len(gl) }

func (gl Gitlist) Get(pkgname string) *Git {
	for _, g := range gl {
		if g.Name == pkgname {
			return g
		}
	}
	return nil
}

func (gl *Gitlist) Add(gits ...*Git) *Gitlist {
	*gl = append(*gl, gits...)
	return gl
}

func (gl *Gitlist) Remove(gits ...*Git) *Gitlist {
	out := make(Gitlist, 0, len(*gl))
	mg := make(map[string]bool)
	for _, g := range gits {
		mg[g.Name] = true
	}
	for _, g := range *gl {
		if !mg[g.Name] {
			out = append(out, g)
		}
	}
	*gl = out
	return gl
}

func (gl *Gitlist) Filter(args ...func(*Git) bool) *Gitlist {
	out := make(Gitlist, 0, len(*gl))
	filter := func(g *Git) bool {
		for _, op := range args {
			if !op(g) {
				return false
			}
		}
		return true
	}
	for _, g := range *gl {
		if filter(g) {
			out = append(out, g)
		}
	}
	return &out
}

func (gl *Gitlist) Sort(args ...func(*Git, *Git) int) *Gitlist {
	less := func(g1, g2 *Git) bool {
		for _, op := range args {
			c := op(g1, g2)
			if c != 0 {
				return c < 0
			}
		}
		return true
	}
	out := make(Gitlist, len(*gl))
	copy(out, *gl)
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return &out
}

func (gl *Gitlist) LimitOffset(l, o int) *Gitlist {
	if o < 0 || o >= len(*gl) || l <= 0 {
		return nil
	}
	var out Gitlist
	if len(*gl) < o+l {
		out = (*gl)[o:]
	} else {
		out = (*gl)[o : o+l]
	}
	return &out
}
