package db

type Repo struct {
	Name string
	Sync bool
}

type Mirror struct {
	Name   string
	Online bool
	Repos  []*Repo
}

type Country struct {
	Name    string
	Mirrors []*Mirror
}

type CountryList []*Country

func (cl *CountryList) GetMirror(mirror string) *Mirror {
	for _, c := range *cl {
		for _, m := range c.Mirrors {
			if m.Name == mirror {
				return m
			}
		}
	}
	return nil
}
