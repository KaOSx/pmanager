package mirror

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"pmanager/conf"
	"pmanager/db"
	"strings"
	"sync"

	"pmanager/util"
	"sort"
)

func getRepoFromFile() (repos []string, err error) {
	var f *os.File
	f, err = os.Open(conf.Read("mirror.pacmanconf"))
	if err != nil {
		return
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	for {
		s, e := buf.ReadString('\n')
		s = strings.TrimSpace(s)
		l := len(s)
		if l > 2 && s[0] == '[' && s[l-1] == ']' && s[1:l-1] != "options" {
			repos = append(repos, s[1:l-1])
		}
		if e != nil {
			break
		}
	}
	return
}

func setRepos(repos []string) []*db.Repo {
	out := make([]*db.Repo, len(repos))
	for i, r := range repos {
		out[i] = &db.Repo{Name: r}
	}
	return out
}

func getMirrorFromFile(repos []string) (mirrors db.CountryList, err error) {
	var f *os.File
	f, err = os.Open(conf.Read("mirror.mirrorlist"))
	if err != nil {
		return
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	var country *db.Country
	for {
		s, e := buf.ReadString('\n')
		if i := strings.Index(s, "Server ="); i >= 0 {
			m := strings.TrimSpace(s[i+8:])
			m = strings.Replace(m, "$repo", "", 1)
			mirror := &db.Mirror{
				Name:  m,
				Repos: setRepos(repos),
			}
			country.Mirrors = append(country.Mirrors, mirror)
		} else if strings.HasPrefix(s, "#") {
			if country == nil {
				country = new(db.Country)
			} else if len(country.Mirrors) > 0 {
				mirrors = append(mirrors, country)
				country = new(db.Country)
			}
			country.Name = strings.TrimSpace(s[1:])
		}
		if e != nil {
			break
		}
	}
	sort.Slice(mirrors, func(i, j int) bool {
		c1, c2 := mirrors[i].Name, mirrors[j].Name
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
	return err == nil && resp.StatusCode == 200
}

func getMd5(mirror, repo string) (md5sum string, err error) {
	url := fmt.Sprintf("%s%s/%s.db.tar.gz", mirror, repo, repo)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err == nil {
		if resp.StatusCode != 200 {
			err = fmt.Errorf("[%i] %s", resp.StatusCode, resp.Status)
		} else {
			var b []byte
			if b, err = ioutil.ReadAll(resp.Body); err == nil {
				md5sum = fmt.Sprintf("%x", md5.Sum(b))
			}
		}
	}
	return
}

func getMirrorMd5(mirror *db.Mirror, hash map[string][]string, mx *sync.RWMutex) {
	if mirror.Online = isMirrorOnline(mirror.Name); !mirror.Online {
		return
	}
	var wg sync.WaitGroup
	for i, r := range mirror.Repos {
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

func checkMd5(mirror *db.Mirror, mainmirror string, hash map[string][]string) {
	if !mirror.Online {
		return
	}
	hr := hash[mirror.Name]
	for i, hm := range hash[mainmirror] {
		h := hr[i]
		mirror.Repos[i].Sync = h != "" && h == hm
	}
}

func Update([]string) {
	repos, err := getRepoFromFile()
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
	if conf.ReadBool("main.debug") {
		util.Println("Found repos:")
		for _, r := range repos {
			util.Println(" -", r)
		}
	}
	mirrors, err := getMirrorFromFile(repos)
	if err != nil {
		util.Fatalln(err)
	}
	if conf.ReadBool("main.debug") {
		util.Println("Found mirrors:")
		for _, c := range mirrors {
			util.Println(" *", c.Name)
			for _, m := range c.Mirrors {
				util.Println("    →", m.Name)
			}
		}
	}
	hash := make(map[string][]string)
	l := len(repos)
	for _, c := range mirrors {
		for _, m := range c.Mirrors {
			hash[m.Name] = make([]string, l)
		}
	}
	var wg sync.WaitGroup
	mx := new(sync.RWMutex)
	for _, c := range mirrors {
		for _, m := range c.Mirrors {
			wg.Add(1)
			go func(mirror *db.Mirror) {
				defer wg.Done()
				getMirrorMd5(mirror, hash, mx)
			}(m)
		}
	}
	wg.Wait()
	mm := conf.Read("mirror.main_mirror")
	for _, c := range mirrors {
		for _, m := range c.Mirrors {
			checkMd5(m, mm, hash)
		}
	}
	db.SetMirrors(&mirrors)
	db.StoreMirrors()
}
