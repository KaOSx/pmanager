package mirror

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"pmanager/conf"
	"pmanager/util"
	"sort"
	"strings"
	"sync"

	"pmanager/db.new"
)

var (
	mainMirrorName string
)

func init() {
	mainMirrorName = conf.Read("mirror.main_mirror")
}

func getAvailableRepos() (repos []string, err error) {
	var f *os.File
	f, err = os.Open(conf.Read("mirror.pacmanconf"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
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
	return
}

func newRepos(repos []string) []db.Repo {
	out := make([]db.Repo, len(repos))
	for i, r := range repos {
		out[i] = db.Repo{Name: r}
	}
	return out
}

func getAvailableCountries(repos []string) (countries []db.Country, err error) {
	var f *os.File
	f, err = os.Open(conf.Read("mirror.mirrorlist"))
	if err != nil {
		return
	}
	defer f.Close()
	var country *db.Country
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.Index(line, "Server ="); i >= 0 {
			mirrorName := strings.TrimSpace(line[i+8:])
			mirrorName = strings.Replace(mirrorName, "$repo", "", 1)
			mirror := db.Mirror{
				Name:  mirrorName,
				Repos: newRepos(repos),
			}
			country.Mirrors = append(country.Mirrors, mirror)
		} else if strings.HasPrefix(line, "#") {
			if country == nil {
				country = new(db.Country)
			} else if len(country.Mirrors) > 0 {
				countries = append(countries, *country)
				country = new(db.Country)
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

func isMirrorOnline(url string) bool {
	resp, err := http.Head(url)
	return err == nil && resp.StatusCode == http.StatusOK
}

func getMd5(mirror, repo string) (md5sum string, err error) {
	url := fmt.Sprintf("%s%s/%s.db.tar.gz", mirror, repo, repo)
	util.Debugf("Begin check md5 from %s\n", url)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("[%i] %s", resp.StatusCode, resp.Status)
			util.Debugf("\033[1;mfailed to check md5 from %s: %s\n", url, err)
		} else {
			var b []byte
			if b, err = ioutil.ReadAll(resp.Body); err == nil {
				md5sum = fmt.Sprintf("%x", md5.Sum(b))
			}
			util.Debugf("check md5 from %s successful\n", url)
		}
	}
	util.Debugf("End check md5 from %s\n", url)
	return
}

func getMirrorMd5(mirror *db.Mirror, hash map[string][]string, mx *sync.Mutex) {
	if mirror.Online = isMirrorOnline(mirror.Name); !mirror.Online {
		return
	}
	var wg sync.WaitGroup
	for i := range mirror.Repos {
		r := &mirror.Repos[i]
		wg.Add(1)
		go func(i int, repo *db.Repo) {
			defer wg.Done()
			if md5, err := getMd5(mirror.Name, repo.Name); err == nil {
				mx.Lock()
				hash[mirror.Name][i] = md5
				mx.Unlock()
			}
		}(i, r)
	}
	wg.Wait()
}

func checkMd5(mirror *db.Mirror, hash map[string][]string) {
	if !mirror.Online {
		return
	}
	hr := hash[mirror.Name]
	for i, hm := range hash[mainMirrorName] {
		h := hr[i]
		mirror.Repos[i].Sync = h != "" && h == hm
	}
}

func Update([]string) {
	repos, err := getAvailableRepos()
	if err != nil {
		util.Fatalln(err)
	}
	has_build := false
	for _, r := range repos {
		if has_build = r == "build"; has_build {
			break
		}
	}
	if !has_build {
		repos = append([]string{"build"}, repos...)
	}
	if conf.Debug() {
		util.Println("Found repos:")
		for _, r := range repos {
			util.Println(" -", r)
		}
	}
	countries, err := getAvailableCountries(repos)
	if err != nil {
		util.Fatalln(err)
	}
	if conf.Debug() {
		util.Println("Found mirrors:")
		for _, c := range countries {
			util.Println(" *", c.Name)
			for _, m := range c.Mirrors {
				util.Println("    →", m.Name)
			}
		}
	}
	hash := make(map[string][]string)
	l := len(repos)
	for _, c := range countries {
		for _, m := range c.Mirrors {
			hash[m.Name] = make([]string, l)
		}
	}
	var wg sync.WaitGroup
	mx := new(sync.Mutex)
	for _, c := range countries {
		for i := range c.Mirrors {
			m := &c.Mirrors[i]
			wg.Add(1)
			go func(mirror *db.Mirror) {
				defer wg.Done()
				getMirrorMd5(mirror, hash, mx)
			}(m)
		}
	}
	wg.Wait()
	for _, c := range countries {
		for i := range c.Mirrors {
			m := &c.Mirrors[i]
			checkMd5(m, hash)
		}
	}
	db.Set("mirror", countries)
	util.Refresh("mirror")
}