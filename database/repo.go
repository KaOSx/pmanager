package database

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"pmanager/log"
	"pmanager/util/conv"
	"pmanager/util/resource"

	"gorm.io/gorm"
)

const (
	bufferSize = 100
)

type packageFiles struct {
	name  string
	files []string
}

func getIncludes(includes, excludes []string) map[string]bool {
	m := make(map[string]bool)

	for _, i := range includes {
		m[i] = true
	}

	for _, e := range excludes {
		delete(m, e)
	}

	return m
}

func getRepoFilePath(base, repoName, extension string) string {
	return path.Join(base, repoName, repoName+"."+extension)
}

func getRepoNames(base string, incl map[string]bool) (repo []string, err error) {
	files, err := os.ReadDir(base)
	if err != nil {
		return
	}

	for _, f := range files {
		fn := f.Name()
		if f.IsDir() && incl[fn] {
			repo = append(repo, fn)
		}
	}

	return
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

func scanDesc(sc *bufio.Scanner, repo, name string, packages chan Package) {
	var (
		p       Package
		section string
	)

	defer func() {
		packages <- p
	}()

	p.Name, p.Repository = name, repo

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

func scanFiles(sc *bufio.Scanner, repo, name string, files chan packageFiles) {
	var (
		f       packageFiles
		section string
	)

	defer func() {
		files <- f
	}()

	f.name = repoName(repo, name)

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
			f.files = append(f.files, line)
		}
	}
}

func scanDescOrFiles(
	content io.Reader,
	repo, file, suffix string,
	packages chan Package,
	files chan packageFiles,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	name := path.Base(strings.TrimSuffix(file, suffix))
	sc := bufio.NewScanner(content)

	switch suffix {
	case "desc":
		scanDesc(sc, repo, name, packages)
	case "files":
		scanFiles(sc, repo, name, files)
	}
}

func readRepoDb(
	base, repo, extension string,
	desc chan Package,
	files chan packageFiles,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	var (
		filePath = getRepoFilePath(base, repo, extension)
		tf, err  = resource.OpenArchive(filePath)
	)

	log.Debugf("Extracting %s\n", filePath)
	if err != nil {
		log.Debugf("\033[1;31mFailed to extract %s: %s\033[m\n", filePath, err)
		return
	}

	var wg2 sync.WaitGroup

	for {
		hdr, err := tf.Next()

		if err != nil {
			if err != io.EOF {
				log.Debugf("\033[1;31mFailed to parse %s: %s\n\033[m", filePath, err)
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

		var buf bytes.Buffer
		if _, err = io.Copy(&buf, tf); err != nil {
			log.Debugf("\033[1;31mFailed to read %s in %s: %s\n\033[m", hdr.Name, filePath, err)
			continue
		}

		wg2.Add(1)
		go scanDescOrFiles(&buf, repo, hdr.Name, suffix, desc, files, &wg2)
	}

	wg2.Wait()
}

func searchPackageUpdate(base, extension string, incl map[string]bool) (packages []Package) {
	repos, err := getRepoNames(base, incl)
	if err != nil {
		log.Fatalln(err)
	}

	var (
		desc   = make(chan Package, bufferSize>>1)
		files  = make(chan packageFiles, bufferSize>>1)
		mfiles = make(map[string][]string)
		done   = make(chan bool, 3)
		wg     sync.WaitGroup
	)

	go func() {
		for p := range desc {
			packages = append(packages, p)
		}
		done <- true
	}()
	go func() {
		for f := range files {
			mfiles[f.name] = f.files
		}
		done <- true
	}()

	wg.Add(len(repos))
	for _, repo := range repos {
		go readRepoDb(base, repo, extension, desc, files, &wg)
	}

	go func() {
		wg.Wait()
		close(desc)
		close(files)
		done <- true
	}()

	for i := 0; i < 3; i++ {
		<-done
	}

	for i := range packages {
		p := &packages[i]
		fn := p.FullName()
		if f, ok := mfiles[fn]; ok {
			p.Files = f
		}
	}

	return
}

func searchGitInfo(base string, p *Package) bool {
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

func unzipPackages(oldPackages, newPackages []Package) (add, update, remove []Package, removeFlags []Flag) {
	if len(oldPackages) == 0 {
		add = newPackages
		return
	}

	packages, repos := make(map[string]Package), make(map[string][]string)
	done := make(map[uint]bool)

	for _, p := range oldPackages {
		packages[p.RepoName()] = p
		repos[p.Name] = append(repos[p.Name], p.Repository)
	}

	for _, np := range newPackages {
		op, ok := packages[np.RepoName()]
		if ok {
			done[op.ID] = true
			np.ID, np.CreatedAt, np.GitID, np.Git = op.ID, op.CreatedAt, op.GitID, op.Git

			if np.Version == op.Version {
				np.FlagID, np.Flag = op.FlagID, op.Flag
			} else if op.FlagID != 0 {
				removeFlags = append(removeFlags, op.Flag)
			}
		}

		if np.GitID == 0 {
			for _, r := range repos[np.Name] {
				if r == np.Repository {
					continue
				}
				if pg := packages[repoName(r, np.Name)]; pg.GitID != 0 {
					np.GitID, np.Git = pg.GitID, pg.Git
					break
				}
			}
		}

		if !ok {
			add = append(add, np)
		} else {
			update = append(update, np)
		}
	}

	for _, p := range oldPackages {
		if !done[p.ID] {
			var rp Package
			p.ID = p.ID
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
				ids := make([]uint, len(r))
				for i, p := range r {
					ids[i] = p.ID
				}
				if err := tx.Unscoped().Delete(&r, ids).Error; err != nil {
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

func updateGit(git *Git, name string) func(*gorm.DB) error {
	return func(tx *gorm.DB) error {
		if err := tx.Create(git).Error; err != nil {
			return err
		}
		return tx.Model(&Package{}).Where("name = ?", name).Update("git_id", git.ID).Error
	}
}

func createFlag(p *Package) func(*gorm.DB) error {
	return func(tx *gorm.DB) error {
		if err := tx.Create(&p.Flag).Error; err != nil {
			return err
		}

		p.FlagID = p.Flag.ID
		return tx.Model(p).Update("flag_id", p.FlagID).Error
	}
}

func deleteFlags(flags []Flag) func(*gorm.DB) error {
	return func(tx *gorm.DB) error {
		if len(flags) == 0 {
			return nil
		}

		ids := make([]uint, len(flags))
		for i, f := range flags {
			ids[i] = f.ID
		}

		if err := tx.Model(&Package{}).Where("flag_id IN ?", ids).Update("flag_id", 0).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&flags).Error
	}
}
