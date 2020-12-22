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

var (
	basedir  string
	ext      string
	excludes map[string]bool
)

func init() {
	basedir = conf.Read("repository.basedir")
	ext = "." + conf.Read("repository.extension")
	excludes = make(map[string]bool)
	for _, e := range conf.ReadArray("repository.exclude") {
		excludes[e] = true
	}
}

func rfilespath(repo string) string { return path.Join(basedir, repo, repo+ext) }

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

func getRepo(name string) (repo []db.Package, err error) {
	rpath := rfilespath(name)
	util.Debugf("Extracting %s\n", rpath)
	file, err := os.Open(rpath)
	if err != nil {
		util.Debugf("\033[1;31mFailed to extract %s\033[m\n", rpath)
		return
	}
	defer file.Close()
	gf, err := util.ReadGZ(file)
	if err != nil {
		util.Debugf("\033[1;31mFailed to extract %s\033[m\n", rpath)
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
			repo = append(repo, db.Package{Repository: name})
			p = &repo[len(repo)-1]
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
	util.Debugf("Extract of %s done! (%d packages)\n", name, len(repo))
	return
}

func getRepoNames() (repo []string, err error) {
	files, err := ioutil.ReadDir(basedir)
	if err != nil {
		return
	}
	for _, f := range files {
		fn := f.Name()
		if f.IsDir() && !excludes[fn] {
			repo = append(repo, fn)
		}
	}
	return
}

func Update(repos []string) {
	var refresh_all bool
	if refresh_all = len(repos) == 0; refresh_all {
		var err error
		if repos, err = getRepoNames(); err != nil {
			util.Fatalln(err)
		}
	}
	for _, n := range repos {
		repo, err := getRepo(n)
		if err == nil {
			db.Set(n, repo)
		} else {
			util.Debugf("Failed to load repo [%s]: %v", n, err)
		}
	}
	if refresh_all {
		util.Refresh("package")
	} else {
		for _, r := range repos {
			util.Refresh(r)
		}
	}
}
