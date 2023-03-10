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

func newRepos(repos []string, mirrorName string) (out []Repo) {
	out = make([]Repo, len(repos))

	for i, r := range repos {
		out[i].Name, out[i].mirrorName = r, mirrorName
	}

	return out
}

func newMirror(name string, repos []string) (mirror Mirror) {
	mirror.Name = strings.Replace(name, "$repo", "", 1)
	mirror.Repos = newRepos(repos, mirror.Name)

	return
}

func checkMirrorIsOnline(mirror *Mirror, repos chan *Repo, mirrors chan *Mirror, wg *sync.WaitGroup) {
	defer wg.Done()

	if mirror.Online = resource.Exists(mirror.Name); !mirror.Online {
		log.Debugf("\033[1;31mMirror %s is not online\n\033[m", mirror.Name)
		return
	}

	log.Debugf("\033[1;32mMirror %s is online\n\033[m", mirror.Name)

	for i := range mirror.Repos {
		repos <- &mirror.Repos[i]
	}
	mirrors <- mirror
}

func getRepoMd5(repo *Repo, wg *sync.WaitGroup) {
	defer wg.Done()

	url := fmt.Sprintf("%s%s/%s.db.tar.gz", repo.mirrorName, repo.Name, repo.Name)

	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		log.Debugf("\033[1;31mFailed to get %s: %s\n[033m", url, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[%d] %s", resp.StatusCode, resp.Status)
		log.Debugf("\033[1;31mfailed to check md5 from %s: %s\n\033[m", url, err)
		return
	}

	var b []byte
	if b, err = io.ReadAll(resp.Body); err != nil {
		log.Debugf("\033[1;31mfailed to parse md5 from %s: %s\n\033[m", url, err)
	}
	repo.md5 = fmt.Sprintf("%x", md5.Sum(b))
	log.Debugf("\033[1;32mcheck md5 from %s successful\n\033[m", url)
}

func checkMirrorMd5(mirror, mainMirror *Mirror, wg *sync.WaitGroup) {
	defer wg.Done()

	if !mirror.Online {
		return
	}

	for i := range mirror.Repos {
		repo := &mirror.Repos[i]
		repo.Sync = repo.md5 != "" && repo.md5 == mainMirror.Repos[i].md5
		if repo.Sync {
			log.Debugf("\033[1;32m%s%s is synced\n\033[m", repo.mirrorName, repo.Name)
		} else {
			log.Debugf("\033[1;31m%s%s is not synced\n\033[m", repo.mirrorName, repo.Name)
		}
	}
}

func sendClose[T any](wg *sync.WaitGroup, c chan T, done chan bool) {
	wg.Wait()
	close(c)
	done <- true
}

func searchMirrorUpdate(pacmanConf, pacmanMirrors, mainMirrorName string) (countries []Country, err error) {
	var repoNames []string
	if repoNames, err = readPacmanConf(pacmanConf); err != nil {
		return
	}

	log.Debugln("Found repos:")
	for _, r := range repoNames {
		log.Debugln(" -", r)
	}

	var data io.Reader
	if data, err = resource.Open(pacmanMirrors); err != nil {
		return
	}

	var (
		sc                           = bufio.NewScanner(data)
		repos                        = make(chan *Repo, bufferSize)
		mirrors                      = make(chan *Mirror, bufferSize)
		done                         = make(chan bool, 2)
		country                      *Country
		mainMirror                   *Mirror
		wgOnline, wgRepos, wgMirrors sync.WaitGroup
	)

	addCountry := func() {
		if country != nil && len(country.Mirrors) > 0 {
			for i := range country.Mirrors {
				mirror := &country.Mirrors[i]
				if mirror.Name == mainMirrorName {
					mainMirror = mirror
				}
				wgOnline.Add(1)
				go checkMirrorIsOnline(mirror, repos, mirrors, &wgOnline)
			}
			countries = append(countries, *country)
		}
	}

	go func() {
		for repo := range repos {
			wgRepos.Add(1)
			go getRepoMd5(repo, &wgRepos)
		}
		done <- true
	}()

	for sc.Scan() {
		line := sc.Text()
		i := strings.Index(line, "Server = ")

		if i >= 0 {
			country.Mirrors = append(country.Mirrors, newMirror(strings.TrimSpace(line[i+8:]), repoNames))
		} else if strings.HasPrefix(line, "#") {
			addCountry()
			country = new(Country)
			country.Name = strings.TrimSpace(line[1:])
		}
	}
	addCountry()

	wgOnline.Wait()

	go sendClose(&wgRepos, repos, done)
	<-done
	<-done

	close(mirrors)
	for mirror := range mirrors {
		wgMirrors.Add(1)
		go checkMirrorMd5(mirror, mainMirror, &wgMirrors)
	}
	wgMirrors.Wait()

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

	log.Debugln("Found mirrors:")
	for _, c := range countries {
		log.Debugln(" *", c.Name)
		for _, m := range c.Mirrors {
			online := "\033[1;32monline\033[m"
			if !m.Online {
				online = "\033[1;31moffline\033[m"
			}
			log.Debugf("    → %s (%s)\n", m.Name, online)
		}
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
