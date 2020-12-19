package db

type Repo struct {
	Name string
	Sync bool
}

type Mirror struct {
	Name   string
	Online bool
	Repos  []Repo
}

type Country struct {
	Name    string
	Mirrors []Mirror
}
