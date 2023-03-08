package database

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"pmanager/log"
	"pmanager/util/resource"
	"sort"
	"strings"
	"sync"

	"gorm.io/gorm"
)

func readPacmanConf(uri string) (repos []string, err error) {
	var data io.Reader
	if data, err = resource.Open(uri); err != nil {
		return
	}

	sc := bufio.NewScanner(data)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}
		l := len(line)
		name := line[1 : l-1]
		if name != "options" {
			repos = append(repos, name)
		}
	}

	for _, r := range repos {
		if r == "build" {
			return
		}
	}

	return append(repos, "build"), nil
}

func newRepos(repos []string) (out []Repo) {
	out = make([]Repo, len(repos))

	for i, r := range repos {
		out[i].Name = r
	}

	return out
}

func newMirror(name string, repos []string) (mirror Mirror) {
	mirror.Name = strings.Replace(name, "$repo", "", 1)
	mirror.Repos = newRepos(repos)

	return
}

func getRepoMd5(mirrorName string, repo *Repo, wg *sync.WaitGroup) {
	defer wg.Done()

	url := fmt.Sprintf("%s%s/%s.db.tar.gz", mirrorName, repo.Name, repo.Name)
	log.Debugf("Begin check md5 from %s\n", url)
	defer log.Debugf("End check md5 from %s\n", url)

	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[%d] %s", resp.StatusCode, resp.Status)
		log.Debugf("\033[1;31mfailed to check md5 from %s: %s\n\033[m", url, err)
		return
	}

	var b []byte
	if b, err = io.ReadAll(resp.Body); err == nil {
		repo.md5 = fmt.Sprintf("%x", md5.Sum(b))
		log.Debugf("\033[1;32mcheck md5 from %s successful\n\033[m", url)
	} else {
		err = fmt.Errorf("[%i] %s", resp.StatusCode, resp.Status)
		log.Debugf("\033[1;31mfailed to parse md5 from %s: %s\n\033[m", url, err)
	}
}

func getMirrorMd5(mirror *Mirror, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	log.Debugf("Begin mirror %s\n", mirror.Name)
	defer log.Debugf("End mirror %s\n", mirror.Name)

	if mirror.Online = resource.Exists(mirror.Name); !mirror.Online {
		log.Debugf("\033[1;31mMirror %s is not online\n\033[m", mirror.Name)
		return
	}

	var wg2 sync.WaitGroup
	wg2.Add(len(mirror.Repos))
	for i := range mirror.Repos {
		go getRepoMd5(mirror.Name, &mirror.Repos[i], &wg2)
	}
	wg2.Wait()
}

func checkMd5(mirror *Mirror, mainMirror Mirror) {
	if !mirror.Online {
		return
	}

	for i := range mirror.Repos {
		repo := &mirror.Repos[i]
		repo.Sync = repo.md5 != "" && repo.md5 == mainMirror.Repos[i].md5
	}
}

func searchMirrorUpdate(pacmanConf, pacmanMirrors, mainMirrorName string) (countries []Country, err error) {
	var repos []string
	if repos, err = readPacmanConf(pacmanConf); err != nil {
		return
	}

	log.Debugln("Found repos:")
	for _, r := range repos {
		log.Debugln(" -", r)
	}

	var data io.Reader
	if data, err = resource.Open(pacmanMirrors); err != nil {
		return
	}

	sc := bufio.NewScanner(data)
	var (
		country    *Country
		ptrs       []*Country
		mirrors    []*Mirror
		mainMirror *Mirror
		wg         sync.WaitGroup
	)

	for sc.Scan() {
		line := sc.Text()
		if i := strings.Index(line, "Server ="); i >= 0 {
			mirror := newMirror(strings.TrimSpace(line[i+8:]), repos)
			country.Mirrors = append(country.Mirrors, mirror)
			j := len(country.Mirrors) - 1
			ptr := &(country.Mirrors[j])
			go getMirrorMd5(ptr, &wg)
			if ptr.Name == mainMirrorName {
				mainMirror = ptr
			}
			mirrors = append(mirrors, ptr)
		} else if strings.HasPrefix(line, "#") {
			if country == nil {
				country = new(Country)
			} else if len(country.Mirrors) > 0 {
				ptrs = append(ptrs, country)
				country = new(Country)
			}
			country.Name = strings.TrimSpace(line[1:])
		}
	}

	if country != nil && len(country.Mirrors) > 0 {
		ptrs = append(ptrs, country)
	}

	sort.Slice(ptrs, func(i, j int) bool {
		c1, c2 := ptrs[i].Name, ptrs[j].Name
		if strings.HasPrefix(c1, "Default") {
			return true
		}
		if strings.HasPrefix(c2, "Default") {
			return false
		}
		return c1 < c2
	})

	wg.Wait()

	log.Debugln("Found mirrors:")
	for _, c := range ptrs {
		log.Debugln(" *", c.Name)
		for _, m := range c.Mirrors {
			log.Debugln("    →", m.Name)
		}
	}

	for _, m := range mirrors {
		checkMd5(m, *mainMirror)
	}

	countries = make([]Country, len(ptrs))
	for i, c := range ptrs {
		countries[i] = *c
	}

	return
}

func updateMirrors(countries []Country) func(*gorm.DB) error {
	return func(tx *gorm.DB) error {
		if len(countries) == 0 {
			return nil
		}
		if err := tx.Where("1 = 1").Unscoped().Delete(&Repo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Unscoped().Delete(&Mirror{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Unscoped().Delete(&Country{}).Error; err != nil {
			return err
		}

		return tx.Create(&countries).Error
	}
}
