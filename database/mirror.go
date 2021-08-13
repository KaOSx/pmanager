package database

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	"pmanager/log"

	"gorm.io/gorm"
	"pmanager/util.new/resource"
)

func parsePacmanConf(uri string) (repos []string, err error) {
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

func newRepos(repos []string) []Repo {
	out := make([]Repo, len(repos))
	for i, r := range repos {
		out[i].Name = r
	}
	return out
}

func parsePacmanMirrors(uri string, repos []string) (countries []Country, err error) {
	var data io.Reader
	if data, err = resource.Open(uri); err != nil {
		return
	}
	var country *Country
	sc := bufio.NewScanner(data)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.Index(line, "Server ="); i >= 0 {
			mirrorName := strings.TrimSpace(line[i+8:])
			mirrorName = strings.Replace(mirrorName, "$repo", "", 1)
			mirror := Mirror{
				Name:  mirrorName,
				Repos: newRepos(repos),
			}
			country.Mirrors = append(country.Mirrors, mirror)
		} else if strings.HasPrefix(line, "#") {
			if country == nil {
				country = new(Country)
			} else if len(country.Mirrors) > 0 {
				countries = append(countries, *country)
				country = new(Country)
			}
			country.Name = strings.TrimSpace(line[1:])
		}
	}
	sort.Slice(countries, func(i, j int) bool {
		c1, c2 := countries[i].Name, countries[j].Name
		if strings.HasPrefix(c1, "Default") {
			return true
		}
		if strings.HasPrefix(c2, "Default") {
			return false
		}
		return c1 < c2
	})
	return
}

func getRepoMd5(mirror, repo string) (md5sum string, err error) {
	url := fmt.Sprintf("%s%s/%s.db.tar.gz", mirror, repo, repo)
	log.Debugf("Begin check md5 from %s\n", url)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("[%i] %s", resp.StatusCode, resp.Status)
			log.Debugf("\033[1;31mfailed to check md5 from %s: %s\n\033[m", url, err)
		} else {
			var b []byte
			if b, err = io.ReadAll(resp.Body); err == nil {
				md5sum = fmt.Sprintf("%x", md5.Sum(b))
			}
			log.Debugf("check md5 from %s successful\n", url)
		}
	}
	log.Debugf("End check md5 from %s\n", url)
	return
}

func getMirrorMd5(mirror *Mirror) {
	if mirror.Online = resource.Exists(mirror.Name); !mirror.Online {
		return
	}
	var wg sync.WaitGroup
	for i := range mirror.Repos {
		wg.Add(1)
		go func(repo *Repo) {
			defer wg.Done()
			if md5, err := getRepoMd5(mirror.Name, repo.Name); err == nil {
				repo.md5 = md5
			}
		}(&mirror.Repos[i])
	}
	wg.Wait()
}

func checkMd5(mirror, mainMirror *Mirror) {
	if !mirror.Online {
		return
	}
	for i := range mirror.Repos {
		repo := &mirror.Repos[i]
		repo.Sync = repo.md5 != "" && repo.md5 == mainMirror.Repos[i].md5
	}
}

func getMirrors(pacmanConf, pacmanMirrors, mainMirrorName string, debug bool) (countries []Country, err error) {
	var repos []string
	if repos, err = parsePacmanConf(pacmanConf); err != nil {
		return
	}
	if debug {
		log.Println("Found repos:")
		for _, r := range repos {
			log.Println(" -", r)
		}
	}
	if countries, err = parsePacmanMirrors(pacmanMirrors, repos); err != nil {
		return
	}
	if debug {
		log.Println("Found mirrors:")
		for _, c := range countries {
			log.Println(" *", c.Name)
			for _, m := range c.Mirrors {
				log.Println("    →", m.Name)
			}
		}
	}
	var mirrors []*Mirror
	var mainMirror *Mirror
	for i := range countries {
		c := &countries[i]
		for j := range c.Mirrors {
			m := &c.Mirrors[j]
			mirrors = append(mirrors, m)
			if m.Name == mainMirrorName {
				mainMirror = m
			}
		}
	}
	var wg sync.WaitGroup
	for _, m := range mirrors {
		wg.Add(1)
		go func(mirror *Mirror) {
			defer wg.Done()
			getMirrorMd5(mirror)
		}(m)
	}
	wg.Wait()
	for _, m := range mirrors {
		checkMd5(m, mainMirror)
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
