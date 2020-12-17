package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"pmanager/conf"
	"pmanager/util"
	"strings"
	"sync"
)

func completeName(pkgname, pkgver string) string { return fmt.Sprintf("%s-%s", pkgname, pkgver) }
func repoName(pkgname, pkgver, repo string) string {
	return fmt.Sprintf("%s/%s", repo, completeName(pkgname, pkgver))
}
func fileName(pkgname, pkgver, arch string) string {
	//@TODO modify for tar.zst in 2022
	return fmt.Sprintf("%s-%s.pkg.tar.xz", completeName(pkgname, pkgver), arch)
}

func dbpath() string   { return path.Join(conf.Basedir(), conf.Read("database.subdir")) }
func repopath() string { return path.Join(dbpath(), "repo") }
func ext() string      { return "." + conf.Read("database.extension") }
func mkdir(fp string) error {
	dir := path.Dir(fp)
	return os.MkdirAll(dir, 0755)
}

var flags *Flaglist
var gits *Gitlist
var repos map[string]*Packagelist
var mirrors *CountryList

var lockers = struct {
	flag, git, repo, mirror sync.Mutex
}{}

func LoadFlags(force ...bool) *Flaglist {
	lockers.flag.Lock()
	defer lockers.flag.Unlock()
	if len(force) == 1 && force[0] {
		flags = nil
	}
	if flags != nil {
		return flags
	}
	flags = new(Flaglist)
	fpath := path.Join(dbpath(), "flag"+ext())
	f, err := os.Open(fpath)
	if err != nil {
		return flags
	}
	defer f.Close()
	err = util.ReadJSON(f, flags)
	if err != nil {
		if conf.Debug() {
			util.Printf("\033[1;31mFailed to load %s\033[m\n", fpath)
		}
		return flags
	}
	if conf.Debug() {
		util.Println("Flags database loaded")
	}
	return flags
}

func SetFlags(fl *Flaglist) {
	lockers.flag.Lock()
	defer lockers.flag.Unlock()
	flags = fl
}

func StoreFlags() error {
	lockers.flag.Lock()
	defer lockers.flag.Unlock()
	if flags == nil {
		if conf.Debug() {
			util.Println("No flags to store")
		}
		return nil
	}
	fpath := path.Join(dbpath(), "flag"+ext())
	if conf.Debug() {
		util.Println("Saving flag in", fpath)
	}
	if err := mkdir(fpath); err != nil {
		return err
	}
	f, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer f.Close()
	return util.WriteJSON(f, flags)
}

func GetRepoNames() (names []string) {
	rp := repopath()
	files, err := ioutil.ReadDir(rp)
	if err != nil {
		if conf.Debug() {
			util.Println("\033[1;31mNo repo database found in %s\033[m\n", rp)
		}
		return
	}
	e := ext()
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), e) {
			names = append(names, strings.TrimSuffix(f.Name(), e))
		}
	}
	return
}

func LoadRepo(name string, force ...bool) *Packagelist {
	lockers.repo.Lock()
	defer lockers.repo.Unlock()
	if repos != nil && len(force) == 1 && force[0] {
		delete(repos, name)
	}
	if repos == nil {
		repos = make(map[string]*Packagelist)
	}
	pl, ok := repos[name]
	if !ok {
		pl = new(Packagelist)
		repos[name] = pl
		rpath := path.Join(repopath(), name+ext())
		f, err := os.Open(rpath)
		if err != nil {
			return pl
		}
		defer f.Close()
		err = util.ReadJSON(f, pl)
		if err != nil {
			if conf.Debug() {
				util.Printf("\033[1;31mFailed to load %s\033[m\n", rpath)
			}
			return pl
		}
		if conf.Debug() {
			util.Printf("Repo database [%s] loaded\n", name)
		}
	}
	return pl
}

func SetRepo(name string, pl *Packagelist) {
	lockers.repo.Lock()
	defer lockers.repo.Unlock()
	if repos == nil {
		repos = make(map[string]*Packagelist)
	}
	repos[name] = pl
}

func StoreRepo(name string) error {
	lockers.repo.Lock()
	defer lockers.repo.Unlock()
	if repos == nil {
		if conf.Debug() {
			util.Printf("No repo [%s] to store\n", name)
		}
		return nil
	}
	pl, ok := repos[name]
	if !ok {
		if conf.Debug() {
			util.Printf("No repo [%s] to store\n", name)
		}
		return nil
	}
	rpath := path.Join(repopath(), name+ext())
	if conf.Debug() {
		util.Println("Saving", name, "in", rpath)
	}
	if err := mkdir(rpath); err != nil {
		return err
	}
	f, err := os.Create(rpath)
	if err != nil {
		return err
	}
	defer f.Close()
	return util.WriteJSON(f, pl)
}

func LoadPackages(force ...bool) *Packagelist {
	lockers.repo.Lock()
	defer lockers.repo.Unlock()
	if len(force) == 1 && force[0] {
		repos = nil
	}
	packages := new(Packagelist)
	for _, name := range GetRepoNames() {
		pl := LoadRepo(name)
		packages.Add((*pl)...)
	}
	return packages
}

func StorePackages() (out []error) {
	lockers.repo.Lock()
	defer lockers.repo.Unlock()
	if repos == nil {
		if conf.Debug() {
			util.Println("No repos to store")
		}
		return
	}
	for name := range repos {
		if err := StoreRepo(name); err != nil {
			out = append(out, err)
		}
	}
	return
}

func LoadGits(force ...bool) *Gitlist {
	lockers.git.Lock()
	defer lockers.git.Unlock()
	if len(force) == 1 && force[0] {
		gits = nil
	}
	if gits != nil {
		return gits
	}
	gits = new(Gitlist)
	gpath := path.Join(dbpath(), "git"+ext())
	f, err := os.Open(gpath)
	if err != nil {
		return gits
	}
	defer f.Close()
	err = util.ReadJSON(f, gits)
	if err != nil {
		if conf.Debug() {
			util.Printf("\033[1;31mFailed to load %s\033[m\n", gpath)
		}
		return gits
	}
	if conf.Debug() {
		util.Println("Gits database loaded")
	}
	return gits
}

func SetGits(gl *Gitlist) {
	lockers.git.Lock()
	defer lockers.git.Unlock()
	gits = gl
}

func StoreGits() error {
	lockers.git.Lock()
	defer lockers.git.Unlock()
	if gits == nil {
		if conf.Debug() {
			util.Println("No gits to store")
		}
		return nil
	}
	gpath := path.Join(dbpath(), "git"+ext())
	if conf.Debug() {
		util.Println("Saving git in", gpath)
	}
	if err := mkdir(gpath); err != nil {
		return err
	}
	f, err := os.Create(gpath)
	if err != nil {
		return err
	}
	defer f.Close()
	return util.WriteJSON(f, gits)
}

func LoadMirrors(force ...bool) *CountryList {
	lockers.mirror.Lock()
	defer lockers.mirror.Unlock()
	if len(force) == 1 && force[0] {
		mirrors = nil
	}
	if mirrors != nil {
		return mirrors
	}
	mirrors = new(CountryList)
	fpath := path.Join(dbpath(), "mirror"+ext())
	f, err := os.Open(fpath)
	if err != nil {
		return mirrors
	}
	defer f.Close()
	err = util.ReadJSON(f, mirrors)
	if err != nil {
		if conf.Debug() {
			util.Printf("\033[1;31mFailed to load %s\033[m\n", fpath)
		}
		return mirrors
	}
	if conf.Debug() {
		util.Println("Mirrors database loaded")
	}
	return mirrors
}

func SetMirrors(cl *CountryList) {
	lockers.mirror.Lock()
	defer lockers.mirror.Unlock()
	mirrors = cl
}

func StoreMirrors() error {
	lockers.mirror.Lock()
	defer lockers.mirror.Unlock()
	if mirrors == nil {
		if conf.Debug() {
			util.Println("No mirrors to store")
		}
		return nil
	}
	fpath := path.Join(dbpath(), "mirror"+ext())
	if err := mkdir(fpath); err != nil {
		return err
	}
	if conf.Debug() {
		util.Println("Saving mirror in", fpath)
	}
	f, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer f.Close()
	return util.WriteJSON(f, mirrors)
}
