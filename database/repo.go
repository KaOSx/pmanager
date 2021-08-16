package database

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path"
	"strings"

	"pmanager/log"

	"gorm.io/gorm"
	"pmanager/util.new/conv"
	"pmanager/util.new/resource"
)

func getRepoFilePath(base, repoName, extension string) string {
	return path.Join(base, repoName, repoName+"."+extension)
}

func scanDesc(sc *bufio.Scanner, p *Package) {
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
			p.PackageSize = conv.String2Int(line)
		case "ISIZE":
			p.InstalledSize = conv.String2Int(line)
		case "URL":
			p.URL = line
		case "LICENSE":
			p.Licenses = append(p.Licenses, line)
		case "GROUPS":
			p.Groups = append(p.Groups, line)
		case "BUILDDATE":
			unix := conv.String2Int(line)
			p.BuildDate = conv.Int2Date(unix)
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

func scanFiles(sc *bufio.Scanner, p *Package) {
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

func parseRepoFile(base, repoName, extension string) (repo []Package, err error) {
	fp := getRepoFilePath(base, repoName, extension)
	log.Debugf("Extracting %s\n", fp)
	tf, err := resource.OpenArchive(fp)
	if err != nil {
		log.Debugf("\033[1;31mFailed to extract %s: %s\033[m\n", fp, err)
		return
	}
	packages := make(map[string]*Package)
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
			repo = append(repo, Package{Repository: repoName})
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
	log.Debugf("Extract of %s done! (%d packages)\n", repoName, len(repo))
	return
}

func getRepoNames(base string, excludes []string) (repo []string, err error) {
	mexcl := make(map[string]bool)
	for _, e := range excludes {
		mexcl[e] = true
	}
	files, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, f := range files {
		fn := f.Name()
		if f.IsDir() && !mexcl[fn] {
			repo = append(repo, fn)
		}
	}
	return
}

func getPackages(base, extension string, excludes []string) (packages []Package, err error) {
	var repoNames []string
	if repoNames, err = getRepoNames(base, excludes); err != nil {
		log.Fatalln(err)
	}
	for _, rn := range repoNames {
		if r, e := parseRepoFile(base, rn, extension); e == nil {
			packages = append(packages, r...)
		} else {
			log.Debugf("Failed to load repo [%s]: %s", rn, e)
		}
	}
	return
}

func findAllPackages() (packages []Package) {
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	dbsingleton.Find(&packages)
	return
}

func unzipPackages(oldPackages, newPackages []Package) (add, update, remove []Package, removeFlags []Flag) {
	if len(oldPackages) == 0 {
		return newPackages, nil, nil
	}
	mp, mr := make(map[string]Package), make(map[string][]string)
	done := make(map[uint]bool)
	for _, p := range oldPackages {
		mp[p.FullName()] = p
		mr[p.Name] = append(mr[p.Name], p.Repository)
	}
	for _, p := range newPackages {
		op, ok := mp[p.FullName()]
		if ok {
			done[op.ID] = true
			p.ID = op.ID
			p.CreatedAt = op.CreatedAt
			p.GitID = op.GitID
			p.Git = op.Git
			if p.Version == op.Version {
				p.FlagID = op.FlagID
				p.Flag = op.Flag
			} else {
				removeFlags = append(removeFlags, p.Flag)
			}
		}
		if p.GitID == 0 {
			for _, r := range mr[p.Name] {
				if r == p.Repository {
					continue
				}
				if pg := mp[r+"/"+p.Name]; pg.GitID != 0 {
					p.GitID = pg.GitID
					p.Git = pg.Git
					break
				}
			}
		}
		if !ok {
			add = append(add, p)
		} else if p.Version != op.Version || p.GitID != op.GitID || p.Md5Sum != op.Md5Sum || p.Sha256Sum != op.Sha256Sum {
			update = append(update, p)
		}
	}
	for _, p := range oldPackages {
		if !done[p.ID] {
			var rp Package
			rp.ID = p.ID
			remove = append(remove, rp)
		}
	}
	return
}

func updatePackages(add, update, remove []Package, removeFlags []Flag) func(*gorm.DB) error {
	return func(tx *gorm.DB) error {
		if len(remove) > 0 {
			for i := 0; i < 100; i += 100 {
				r := remove[i:]
				if len(r) > 100 {
					r = r[:100]
				}
				if err := tx.Unscoped().Delete(&r).Error; err != nil {
					return err
				}
			}
		}
		if len(update) > 0 {
			for i := 0; i < len(update); i += 100 {
				u := update[i:]
				if len(u) > 100 {
					u = u[:100]
				}
				if err := tx.Save(&u).Error; err != nil {
					return err
				}
			}
		}
		if len(removeFlags) > 0 {
			for i := 0; i < 100; i += 100 {
				r := removeFlags[i:]
				if len(r) > 100 {
					r = r[:100]
				}
				if err := tx.Unscoped().Delete(&r).Error; err != nil {
					return err
				}
			}
		}
		if len(add) > 0 {
			return tx.CreateInBatches(&add, 100).Error
		}
		return nil
	}
}

func scanPkginfo(sc *bufio.Scanner, git *Git) {
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "gitrepo = ") {
			git.Repository = strings.TrimSpace(line[len("gitrepo = "):])
		} else if strings.HasPrefix(line, "gitfolder = ") {
			git.Folder = strings.TrimSpace(line[len("gitfolder = "):])
		}
	}
}

func searchGit(base string, p *Package) bool {
	fp := path.Join(base, p.Repository, p.Filename)
	tf, err := resource.OpenArchive(fp)
	if err != nil {
		log.Debugf("\033[1;31mFailed to load %s: %s\033[m\n", fp, err)
		return false
	}
	for {
		hdr, err := tf.Next()
		if err != nil {
			break
		}
		if hdr.Name == ".PKGINFO" {
			var buf bytes.Buffer
			if _, err = io.Copy(&buf, tf); err == nil {
				sc := bufio.NewScanner(&buf)
				scanPkginfo(sc, &p.Git)
				p.Git.Name = p.Name
				return true
			}
			break
		}
	}
	return false
}

func updateGit(git *Git, name string) func(*gorm.DB) error {
	return func(tx *gorm.DB) error {
		if err := tx.Create(git).Error; err != nil {
			return err
		}
		return tx.Model(&Package{}).Where("name = ?", name).Update("git_id", git.ID).Error
	}
}
