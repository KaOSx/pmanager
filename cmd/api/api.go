package api

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"pmanager/conf"
	"pmanager/db"
	"pmanager/util"
	"pmanager/util/mail"
	"strings"
	"time"
)

func sendFormSpree(f *db.Flag) {
	pname := f.CompleteName()
	subject := fmt.Sprintf("The package %s has been flagged as outdated", pname)
	body := []string{
		fmt.Sprintf("Package details: %s/view.php?repo=%s&name=%s", conf.Read("main.viewurl"), f.Repository, pname),
		"",
		"",
		"---",
		fmt.Sprintf("The package %s has been flagged as outdated.", pname),
		"by: " + f.Email,
		"",
		"Additional informations:",
		strings.Replace(f.Comment, "\n", "\r\n", -1),
	}
	strbody := strings.Join(body, "\n")
	resp, err := http.PostForm("https://formspree.io/"+conf.Read("smtp.send_to"), url.Values{
		"email":   {f.Email},
		"subject": {subject},
		"message": {strbody},
		"submit":  {"Send"},
	})
	defer resp.Body.Close()
	if conf.Debug() && err != nil {
		util.Println(err)
	}
}

func sendMail(f *db.Flag) {
	server := mail.Server{
		Host:     conf.Read("smtp.host"),
		Port:     conf.Read("smtp.port"),
		TLS:      conf.ReadBool("smtp.use_encryption"),
		User:     conf.Read("smtp.user"),
		Password: conf.Read("smtp.password"),
	}
	pname := f.CompleteName()
	subject := fmt.Sprintf("The package %s has been flagged as outdated", pname)
	body := []string{
		fmt.Sprintf("Package details: %s/view.php?repo=%s&name=%s", conf.Read("main.viewurl"), f.Repository, pname),
		"",
		"",
		"---",
		fmt.Sprintf("The package %s has been flagged as outdated.", pname),
		"by: " + f.Email,
		"",
		"Additional informations:",
		strings.Replace(f.Comment, "\n", "\r\n", -1),
	}
	m := mail.New(server, conf.Read("smtp.send_from"), conf.Read("smtp.send_to"), subject, strings.Join(body, "\r\n"))
	m.AddHeader("Reply-To", conf.Read("smtp.send_to")).
		AddHeader("X-Mailer", "Packages").
		AddHeader("MIME-Version", "1.0").
		AddHeader("Content-Transfer-Encoding", "8bit").
		AddHeader("Content-type", "text/plain; charset=utf-8")
	if err := m.Send(); err != nil {
		util.Println(err)
	}
}

func readPkginfo(sc *bufio.Scanner, g *db.Git) {
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "gitrepo = ") {
			g.Repository = strings.TrimSpace(line[len("gitrepo = "):])
		} else if strings.HasPrefix(line, "gitfolder = ") {
			g.Folder = strings.TrimSpace(line[len("gitfolder = "):])
		}
	}
}

func getGit(p *db.Package) *db.Git {
	g := new(db.Git)
	gpath := path.Join(conf.Read("repository.basedir"), p.Repository, p.FileName())
	f, err := os.Open(gpath)
	if err != nil {
		if conf.Debug() {
			util.Printf("\033[1;31mFailed to load %s: %s\033[m\n", gpath, err)
		}
		return g
	}
	defer f.Close()
	xf, err := util.ReadXZ(f)
	if err != nil {
		util.Printf("\033[1;31mFailed to xunzip %s: %s\033[m\n", gpath, err)
	}
	tf := util.ReadTAR(xf)
	for {
		hdr, err := tf.Next()
		if err != nil {
			break
		}
		if hdr.Name == ".PKGINFO" {
			var buf bytes.Buffer
			if _, err = io.Copy(&buf, tf); err == nil {
				sc := bufio.NewScanner(&buf)
				readPkginfo(sc, g)
			}
			break
		}
	}
	return g
}

func writeResponse(w http.ResponseWriter, data interface{}, codes ...int) {
	code := http.StatusOK
	if len(codes) == 1 {
		code = codes[0]
	}
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
	w.Header().Add("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
	b := util.JSON(data)
	w.Write(b)
	w.WriteHeader(code)
}

func getString(r *http.Request, key string) string  { return r.FormValue(key) }
func getInt(r *http.Request, key string) int64      { return conf.String2Int(getString(r, key)) }
func getBool(r *http.Request, key string) bool      { return conf.String2Bool(getString(r, key)) }
func getDate(r *http.Request, key string) time.Time { return conf.String2Date(getString(r, key)) }

func paginate(r *http.Request, count int64) (pagination conf.Map) {
	pagination = make(conf.Map)
	pagination["total"] = count
	page := getInt(r, "page")
	if page <= 0 {
		page = 1
	}
	pagination["page"] = page
	limit := getInt(r, "limit")
	if limit <= 0 {
		limit = conf.ReadInt("api.pagination")
	}
	if limit == 0 {
		limit = count
	}
	pagination["limit"] = limit
	pagination["offset"] = (page - 1) * limit
	pagination["last"] = (count-1)/limit + 1
	return
}

func sort(r *http.Request) (sorted conf.Map) {
	sorted = make(conf.Map)
	sorted["field"] = getString(r, "sortby")
	sorted["asc"] = getString(r, "sortdir") != "desc"
	return sorted
}

func flagList(w http.ResponseWriter, r *http.Request) {
	flags := db.LoadFlags(true)
	filters := make(conf.Map)
	var funcs []func(*db.Flag) bool
	if e := getString(r, "search"); e != "" {
		filters["search"] = e
		funcs = append(funcs, func(f *db.Flag) bool {
			return strings.Contains(f.CompleteName(), e)
		})
	}
	if e := getString(r, "repo"); e != "" {
		filters["repo"] = e
		funcs = append(funcs, func(f *db.Flag) bool {
			return f.Repository == e
		})
	}
	if e := getString(r, "email"); e != "" {
		filters["email"] = e
		funcs = append(funcs, func(f *db.Flag) bool {
			return strings.Contains(f.Email, e)
		})
	}
	if e := getDate(r, "from"); !e.IsZero() {
		filters["from"] = e
		funcs = append(funcs, func(f *db.Flag) bool {
			return !f.Date.Before(e)
		})
	}
	if e := getDate(r, "to"); !e.IsZero() {
		filters["to"] = e
		funcs = append(funcs, func(f *db.Flag) bool {
			return !f.Date.After(e)
		})
	}
	if getString(r, "flagged") != "" {
		e := getBool(r, "flagged")
		filters["flagged"] = e
		funcs = append(funcs, func(f *db.Flag) bool {
			return f.Flagged == e
		})
	}
	if len(funcs) > 0 {
		flags = flags.Filter(funcs...)
	}
	sorter := sort(r)
	var cmp func(*db.Flag, *db.Flag) int
	if e := sorter["field"]; e != "" {
		switch e {
		case "name":
			cmp = func(f1, f2 *db.Flag) int { return util.CompareString(f1.CompleteName(), f2.CompleteName()) }
		case "repo":
			cmp = func(f1, f2 *db.Flag) int { return util.CompareString(f1.Repository, f2.Repository) }
		case "date":
			cmp = func(f1, f2 *db.Flag) int { return util.CompareDate(f1.Date, f2.Date) }
		case "flagged":
			cmp = func(f1, f2 *db.Flag) int { return util.CompareBool(f1.Flagged, f2.Flagged) }
		}
		if cmp != nil {
			flags = flags.Sort(cmp)
		}
	}
	count := int64(len(*flags))
	paginator := paginate(r, count)
	flags = flags.LimitOffset(paginator.GetInt("limit"), paginator.GetInt("offset"))
	writeResponse(w, conf.Map{
		"data":     *flags,
		"filter":   filters,
		"sort":     sorter,
		"paginate": paginator,
	})
}

func flagAdd(w http.ResponseWriter, r *http.Request) {
	flags := db.LoadFlags(true)
	f := db.Flag{
		Name:       getString(r, "name"),
		Version:    getString(r, "version"),
		Repository: getString(r, "repo"),
		Email:      getString(r, "email"),
		Comment:    getString(r, "comment"),
		Flagged:    true,
		Date:       time.Now(),
	}
	flags.Add(&f)
	if m := strings.ToUpper(r.Method); m == "GET" || m == "POST" {
		if err := db.StoreFlags(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if conf.ReadBool("smtp.use_formspree") {
			sendFormSpree(&f)
		} else {
			sendMail(&f)
		}
	}
	writeResponse(w, conf.Map{
		"data": f,
	})
}

func packageList(w http.ResponseWriter, r *http.Request) {
	packages := db.LoadPackages(true)
	filters := make(conf.Map)
	var funcs []func(*db.Package) bool
	exact := getBool(r, "exact")
	if e := getString(r, "search"); e != "" {
		filters["search"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			if exact {
				return p.Name == e
			}
			return strings.Contains(p.CompleteName(), e)
		})
	}
	if e := getDate(r, "from"); !e.IsZero() {
		filters["from"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			return !p.BuildDate.Before(e)
		})
	}
	if e := getDate(r, "to"); !e.IsZero() {
		filters["to"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			return !p.BuildDate.After(e)
		})
	}
	flagged := false
	if getString(r, "flagged") != "" {
		e := getBool(r, "flagged")
		flagged = true
		db.LoadFlags(true)
		filters["flagged"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			return p.IsFlagged() == e
		})
	}
	if len(funcs) > 0 {
		packages = packages.Filter(funcs...)
	}
	sorter := sort(r)
	var cmp func(*db.Package, *db.Package) int
	if e := sorter["field"]; e != "" {
		switch e {
		case "name":
			cmp = func(p1, p2 *db.Package) int { return util.CompareString(p1.CompleteName(), p2.CompleteName()) }
		case "repo":
			cmp = func(p1, p2 *db.Package) int { return util.CompareString(p1.Repository, p2.Repository) }
		case "date":
			cmp = func(p1, p2 *db.Package) int { return util.CompareDate(p1.BuildDate, p2.BuildDate) }
		case "flagged":
			if !flagged {
				db.LoadFlags(true)
			}
			cmp = func(p1, p2 *db.Package) int { return util.CompareBool(p1.IsFlagged(), p2.IsFlagged()) }
		}
		if cmp != nil {
			if sorter.GetBool("asc") {
				packages = packages.Sort(cmp)
			} else {
				packages = packages.Sort(func(p1, p2 *db.Package) int { return -cmp(p1, p2) })
			}
		}
	}
	var total_size int64
	for _, p := range *packages {
		total_size += p.PackageSize
	}
	count := int64(len(*packages))
	paginator := paginate(r, count)
	packages = packages.LimitOffset(paginator.GetInt("limit"), paginator.GetInt("offset"))
	data := make([]conf.Map, len(*packages))
	for i, p := range *packages {
		m := util.ToMap(p)
		m.Delete("Licenses", "Groups", "Depends", "MakeDepends", "OptDepends", "Files")
		m["PackageSize"] = util.FormatSize(m.GetInt("PackageSize"))
		m["InstalledSize"] = util.FormatSize(m.GetInt("InstalledSize"))
		m["CompleteName"] = p.CompleteName()
		m["RepoName"] = p.RepoName()
		m["FileName"] = p.FileName()
		data[i] = m
	}
	writeResponse(w, conf.Map{
		"data":     data,
		"size":     util.FormatSize(total_size),
		"filter":   filters,
		"sort":     sorter,
		"paginate": paginator,
	})
}

func packageView(w http.ResponseWriter, r *http.Request) {
	repo := getString(r, "repo")
	pkgname := getString(r, "name")
	if repo == "" || pkgname == "" {
		writeResponse(w, conf.Map{"data": nil}, http.StatusNotFound)
		return
	}
	packages := db.LoadRepo(repo, true).Filter(func(p *db.Package) bool { return p.CompleteName() == pkgname })
	if len(*packages) == 0 {
		writeResponse(w, conf.Map{"data": nil}, http.StatusNotFound)
		return
	}
	p := (*packages)[0]
	gits := db.LoadGits(true)
	g := p.GetGit()
	if g == nil {
		g = getGit(p)
		if g.Repository != "" && g.Folder != "" {
			g.Name = p.Name
			gits.Add(g)
			if err := db.StoreGits(); err != nil && conf.Debug() {
				util.Println("\033[1;31mFailed to store git database\033[m")
			}
		}
	}
	db.LoadFlags(true)
	data := util.ToMap(p)
	gurl := conf.Read("main.giturl") + g.Repository
	data["Flagged"] = p.IsFlagged()
	data["PackageSize"] = util.FormatSize(data.GetInt("PackageSize"))
	data["InstalledSize"] = util.FormatSize(data.GetInt("InstalledSize"))
	data["CompleteName"] = p.CompleteName()
	data["RepoName"] = p.RepoName()
	data["FileName"] = p.FileName()
	data["URL"] = conf.Map{
		"Upstream": data["URL"],
		"Download": conf.Read("main.repourl") + p.Repository + "/" + p.FileName(),
		"Bugs":     gurl + "/issues/",
		"Sources":  gurl + "/tree/master/" + g.Folder,
		"PKGBUILD": gurl + "/blob/master/" + g.Folder + "/PKGBUILD",
		"Commits":  gurl + "/commits/master/" + g.Folder + "/PKGBUILD",
	}
	writeResponse(w, conf.Map{"data": data})
}

func repoList(w http.ResponseWriter, r *http.Request) {
	rname := getString(r, "repo")
	if rname == "" {
		writeResponse(w, conf.Map{"data": nil}, http.StatusNotFound)
		return
	}
	packages := db.LoadRepo(rname, true)
	if len(*packages) == 0 {
		writeResponse(w, conf.Map{"data": nil}, http.StatusNotFound)
		return
	}
	filters := make(conf.Map)
	var funcs []func(*db.Package) bool
	exact := getBool(r, "exact")
	if e := getString(r, "search"); e != "" {
		filters["search"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			if exact {
				return p.Name == e
			}
			return strings.Contains(p.CompleteName(), e)
		})
	}
	if e := getDate(r, "from"); !e.IsZero() {
		filters["from"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			return !p.BuildDate.Before(e)
		})
	}
	if e := getDate(r, "to"); !e.IsZero() {
		filters["to"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			return !p.BuildDate.After(e)
		})
	}
	flagged := false
	if getString(r, "flagged") != "" {
		e := getBool(r, "flagged")
		flagged = true
		db.LoadFlags(true)
		filters["flagged"] = e
		funcs = append(funcs, func(p *db.Package) bool {
			return p.IsFlagged() == e
		})
	}
	if len(funcs) > 0 {
		packages = packages.Filter(funcs...)
	}
	sorter := sort(r)
	var cmp func(*db.Package, *db.Package) int
	if e := sorter["field"]; e != "" {
		switch e {
		case "name":
			cmp = func(p1, p2 *db.Package) int { return util.CompareString(p1.CompleteName(), p2.CompleteName()) }
		case "date":
			cmp = func(p1, p2 *db.Package) int { return util.CompareDate(p1.BuildDate, p2.BuildDate) }
		case "flagged":
			if !flagged {
				db.LoadFlags(true)
			}
			cmp = func(p1, p2 *db.Package) int { return util.CompareBool(p1.IsFlagged(), p2.IsFlagged()) }
		}
		if cmp != nil {
			if sorter.GetBool("asc") {
				packages = packages.Sort(cmp)
			} else {
				packages = packages.Sort(func(p1, p2 *db.Package) int { return -cmp(p1, p2) })
			}
		}
	}
	var total_size int64
	for _, p := range *packages {
		total_size += p.PackageSize
	}
	count := int64(len(*packages))
	paginator := paginate(r, count)
	packages = packages.LimitOffset(paginator.GetInt("limit"), paginator.GetInt("offset"))
	data := make([]conf.Map, len(*packages))
	for i, p := range *packages {
		m := util.ToMap(p)
		m.Delete("Licenses", "Groups", "Depends", "MakeDepends", "OptDepends", "Files")
		m["PackageSize"] = util.FormatSize(m.GetInt("PackageSize"))
		m["InstalledSize"] = util.FormatSize(m.GetInt("InstalledSize"))
		m["CompleteName"] = p.CompleteName()
		m["RepoName"] = p.RepoName()
		m["FileName"] = p.FileName()
		data[i] = m
	}
	writeResponse(w, conf.Map{
		"data":     data,
		"size":     util.FormatSize(total_size),
		"filter":   filters,
		"sort":     sorter,
		"paginate": paginator,
	})
}

func mirrorList(w http.ResponseWriter, r *http.Request) {
	mirrors := db.LoadMirrors(true)
	writeResponse(w, mirrors)
}

func Serve([]string) {
	port := ":" + conf.Read("api.port")
	http.HandleFunc("/flag/list", flagList)
	http.HandleFunc("/flag/add", flagAdd)
	http.HandleFunc("/package/list", packageList)
	http.HandleFunc("/package/view", packageView)
	http.HandleFunc("/repo/list", repoList)
	http.HandleFunc("/mirror", mirrorList)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		util.Fatalln(err)
	}
}
