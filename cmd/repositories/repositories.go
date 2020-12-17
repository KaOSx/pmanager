package repositories

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"pmanager/conf"
	"pmanager/db"
	"pmanager/util"
	"strings"
)

func basedir() string               { return conf.Read("repository.basedir") }
func ext() string                   { return "." + conf.Read("repository.extension") }
func rfilespath(repo string) string { return path.Join(basedir(), repo, repo+ext()) }
func excludes() map[string]bool {
	excl := conf.ReadArray("repository.exclude")
	out := make(map[string]bool)
	for _, e := range excl {
		out[e] = true
	}
	return out
}

func scanDesc(sc *bufio.Scanner, p *db.Package) {
	var section string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		l := len(line)
		if l == 0 {
			continue
		}
		if line[0] == '%' && line[l-1] == '%' {
			section = line[1 : l-1]
			continue
		}
		switch section {
		case "NAME":
			p.Name = line
		case "VERSION":
			p.Version = line
		case "ARCH":
			p.Arch = line
		case "DESC":
			p.Description = line
		case "CSIZE":
			p.PackageSize = conf.String2Int(line)
		case "ISIZE":
			p.InstalledSize = conf.String2Int(line)
		case "URL":
			p.URL = line
		case "LICENSE":
			p.Licenses = append(p.Licenses, line)
		case "GROUPS":
			p.Groups = append(p.Groups, line)
		case "BUILDDATE":
			unix := conf.String2Int(line)
			p.BuildDate = conf.Int2Date(unix)
		case "DEPENDS":
			p.Depends = append(p.Depends, line)
		case "MAKEDEPENDS":
			p.MakeDepends = append(p.MakeDepends, line)
		case "OPTDEPENDS":
			p.OptDepends = append(p.OptDepends, line)
		case "MD5SUM":
			p.Md5Sum = line
		case "SHA256SUM":
			p.Sha256Sum = line
		case "FILENAME":
			p.Filename = line
		}
	}
}

func scanFiles(sc *bufio.Scanner, p *db.Package) {
	var section string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		l := len(line)
		if l == 0 {
			continue
		}
		if line[0] == '%' && line[l-1] == '%' {
			section = line[1 : l-1]
			continue
		}
		if section == "FILES" {
			p.Files = append(p.Files, line)
		}
	}
}

func getRepo(name string) (repo db.Packagelist, err error) {
	rpath := rfilespath(name)
	if conf.Debug() {
		util.Printf("Extracting %s\n", rpath)
	}
	file, err := os.Open(rpath)
	if err != nil {
		if conf.Debug() {
			util.Printf("\033[1;31mFailed to extract %s\033[m\n", rpath)
		}
		return
	}
	defer file.Close()
	gf, err := util.ReadGZ(file)
	if err != nil {
		if conf.Debug() {
			util.Printf("\033[1;31mFailed to extract %s\033[m\n", rpath)
		}
		return
	}
	defer gf.Close()
	tf := util.ReadTAR(gf)
	packages := make(map[string]*db.Package)
	for {
		hdr, e := tf.Next()
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		suffix := path.Base(hdr.Name)
		if suffix != "desc" && suffix != "files" {
			continue
		}
		pn := path.Base(strings.TrimSuffix(hdr.Name, suffix))
		p, ok := packages[pn]
		if !ok {
			p = new(db.Package)
			p.Repository = name
			repo.Add(p)
			packages[pn] = p
		}
		var buf bytes.Buffer
		if _, err = io.Copy(&buf, tf); err != nil {
			break
		}
		sc := bufio.NewScanner(&buf)
		if suffix == "desc" {
			scanDesc(sc, p)
		} else {
			scanFiles(sc, p)
		}
	}
	if conf.Debug() {
		util.Printf("Extract of %s done! (%d packages)\n", name, len(repo))
	}
	return
}

func getRepoNames() (repo []string, err error) {
	files, err := ioutil.ReadDir(basedir())
	if err != nil {
		return
	}
	excl := excludes()
	for _, f := range files {
		fn := f.Name()
		if f.IsDir() && !excl[fn] {
			repo = append(repo, fn)
		}
	}
	return
}

func Update(repos []string) {
	if len(repos) == 0 {
		var err error
		if repos, err = getRepoNames(); err != nil {
			util.Fatalln(err)
		}
	}
	for _, n := range repos {
		repo, err := getRepo(n)
		if err == nil {
			db.SetRepo(n, &repo)
		} else if conf.Debug() {
			util.Printf("Failed to load repo [%s]: %v", n, err)
		}
	}
	errs := db.StorePackages()
	util.Refresh("package")
	if conf.Debug() {
		for _, err := range errs {
			util.Println(err)
		}
	}
}
